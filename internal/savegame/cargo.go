package savegame

import (
	"fmt"

	"github.com/SecondPort/mod_space_haven/internal/xmldoc"
)

// A cargo hold is an <inv> that hangs directly off a <feat>. The same <inv>
// element also appears one level deeper, inside a machine's <prod> buffer,
// holding the inputs and outputs the machine is working on. Those units are not
// storage: rewriting them desynchronizes the machine from its job, so only
// direct children of <feat> are ever counted or edited.
const (
	cargoContainerTag = "inv"
	cargoParentTag    = "feat"
	cargoSlotTag      = "s"
)

// cargoSlot is one <s elementaryId=... inStorage=.../> entry in a cargo hold.
type cargoSlot struct {
	resourceID int
	amount     int
	amountAttr xmldoc.Attr
}

// cargoContainer is one cargo hold and the place a new resource would go.
type cargoContainer struct {
	contentEnd int // offset of '<' in the closing </inv>
	slots      int
	insertAt   int
	indent     string
}

// Cargo totals every resource held in the player ship's cargo containers.
func (s *Save) Cargo() (map[int]int, error) {
	slots, _, _, err := s.cargoLayout()
	if err != nil {
		return nil, err
	}

	totals := make(map[int]int, len(slots))
	for _, slot := range slots {
		totals[slot.resourceID] += slot.amount
	}
	return totals, nil
}

// SetCargo sets the total amount of one resource across the player ship's cargo
// containers, splitting it as evenly as the containers allow. It reports
// whether the resource had to be inserted because the ship was not carrying any.
func (s *Save) SetCargo(resourceID, amount int) (inserted bool, err error) {
	if amount < 0 {
		return false, fmt.Errorf("savegame: cargo amount cannot be negative (got %d)", amount)
	}

	slots, containers, block, err := s.cargoLayout()
	if err != nil {
		return false, err
	}

	var matching []cargoSlot
	for _, slot := range slots {
		if slot.resourceID == resourceID {
			matching = append(matching, slot)
		}
	}

	if len(matching) == 0 {
		patch, err := insertCargoPatch(block, containers, resourceID, amount)
		if err != nil {
			return false, err
		}
		return true, s.apply(xmldoc.PatchSet{block.shift(patch)})
	}

	base, remainder := amount/len(matching), amount%len(matching)
	patches := make(xmldoc.PatchSet, 0, len(matching))
	for i, slot := range matching {
		value := base
		if i < remainder {
			value++
		}
		patches = append(patches, block.shift(xmldoc.SetAttrRaw(slot.amountAttr, fmt.Sprint(value))))
	}
	return false, s.apply(patches)
}

// cargoLayout walks the player ship once and returns its cargo slots and the
// containers holding them, in document order.
func (s *Save) cargoLayout() ([]cargoSlot, []cargoContainer, region, error) {
	ship, err := s.PlayerShip()
	if err != nil {
		return nil, nil, region{}, err
	}
	block := s.region(ship.Start, ship.End)

	var (
		slots      []cargoSlot
		containers []cargoContainer
		open       []int // indices of the cargo containers currently entered
	)

	walkErr := xmldoc.Walk(block.data, func(tok xmldoc.Token, stack []string) error {
		switch {
		case tok.Kind == xmldoc.KindEnd && tok.NameIs(cargoContainerTag):
			if n := len(open); n > 0 && containers[open[n-1]].contentEnd == tok.Start {
				open = open[:n-1]
			}

		case tok.Kind == xmldoc.KindStart && tok.NameIs(cargoContainerTag):
			if !isCargoContainer(stack) {
				return nil
			}
			end, err := xmldoc.FindElementEnd(block.data, tok)
			if err != nil {
				return err
			}
			contentEnd := end - len("</"+cargoContainerTag+">")
			containers = append(containers, cargoContainer{
				contentEnd: contentEnd,
				insertAt:   contentEnd,
			})
			open = append(open, len(containers)-1)

		case isElement(tok) && tok.NameIs(cargoSlotTag):
			if len(open) == 0 || !isCargoSlot(stack) {
				return nil
			}
			id, ok := tok.IntAttr("elementaryId")
			if !ok {
				return nil
			}
			amountAttr, ok := tok.Attr("inStorage")
			if !ok {
				return nil
			}
			amount, ok := tok.IntAttr("inStorage")
			if !ok {
				return nil
			}

			idx := open[len(open)-1]
			c := &containers[idx]
			if c.slots == 0 {
				c.indent = leadingWhitespace(block.data, tok.Start)
			}
			c.slots++
			c.insertAt = tok.End

			slots = append(slots, cargoSlot{
				resourceID: id,
				amount:     amount,
				amountAttr: amountAttr,
			})
		}
		return nil
	})
	if walkErr != nil {
		return nil, nil, region{}, fmt.Errorf("savegame: reading cargo: %w", walkErr)
	}

	return slots, containers, block, nil
}

// isCargoContainer reports whether an <inv> opened with this ancestor stack is
// a storage hold rather than a machine's buffer.
func isCargoContainer(stack []string) bool {
	return len(stack) > 0 && stack[len(stack)-1] == cargoParentTag
}

// isCargoSlot reports whether an <s> with this ancestor stack sits directly in
// a cargo hold.
func isCargoSlot(stack []string) bool {
	n := len(stack)
	return n >= 2 && stack[n-1] == cargoContainerTag && stack[n-2] == cargoParentTag
}

// insertCargoPatch adds a resource the ship is not carrying to its fullest
// cargo hold, which is the one most likely to be a general store.
func insertCargoPatch(block region, containers []cargoContainer, resourceID, amount int) (xmldoc.Patch, error) {
	if len(containers) == 0 {
		return xmldoc.Patch{}, fmt.Errorf("savegame: the player ship has no cargo container to add resource %d to", resourceID)
	}

	best := 0
	for i, c := range containers {
		if c.slots > containers[best].slots {
			best = i
		}
	}

	c := containers[best]
	entry := fmt.Sprintf(`<s elementaryId="%d" inStorage="%d" onTheWayIn="0" onTheWayOut="0"/>`, resourceID, amount)

	if c.slots > 0 {
		// Land next to the existing entries, matching their indentation.
		return xmldoc.InsertAt(c.insertAt, c.indent+entry), nil
	}
	// An empty hold: indent one level past its closing tag.
	closing := leadingWhitespace(block.data, c.contentEnd)
	return xmldoc.InsertAt(c.contentEnd, "\t"+entry+closing), nil
}
