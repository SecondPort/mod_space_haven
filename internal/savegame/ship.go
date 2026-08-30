package savegame

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/SecondPort/mod_space_haven/internal/xmldoc"
)

// Ship is the player's vessel and the byte range its element occupies.
type Ship struct {
	SID   string
	Name  string
	Start int
	End   int
}

// Credits returns the player's bank balance.
func (s *Save) Credits() int {
	doc := s.region(0, len(s.data))
	tok, ok := doc.firstToken(func(t xmldoc.Token) bool {
		if !t.NameIs("playerBank") {
			return false
		}
		_, has := t.Attr("ca")
		return has
	})
	if !ok {
		return 0
	}
	n, _ := tok.IntAttr("ca")
	return n
}

// SetCredits writes the player's bank balance.
func (s *Save) SetCredits(amount int) error {
	if amount < 0 {
		return fmt.Errorf("savegame: credits cannot be negative (got %d)", amount)
	}

	doc := s.region(0, len(s.data))
	tok, ok := doc.firstToken(func(t xmldoc.Token) bool {
		if !t.NameIs("playerBank") {
			return false
		}
		_, has := t.Attr("ca")
		return has
	})
	if !ok {
		return errors.New("savegame: this save has no player bank")
	}

	patch, err := setIntAttr(tok, "ca", amount)
	if err != nil {
		return err
	}
	return s.apply(xmldoc.PatchSet{patch})
}

// PlayerShip locates the ship the player's settlement created.
//
// The save marks the player's settlement with isPlayer="true" and records the
// hull it owns in createdShipId; the ship element itself carries no such flag,
// which is why the lookup starts from the settlement.
func (s *Save) PlayerShip() (Ship, error) {
	sid, name, err := s.playerShipRef()
	if err != nil {
		return Ship{}, err
	}

	doc := s.region(0, len(s.data))
	tok, ok := doc.firstToken(func(t xmldoc.Token) bool {
		if !t.NameIs("ship") || !isElement(t) {
			return false
		}
		a, has := t.Attr("sid")
		return has && a.Value == sid
	})
	if !ok {
		return Ship{}, fmt.Errorf("%w: settlement points at ship %s, which is not in the save", ErrPlayerShipNotFound, sid)
	}

	end, err := xmldoc.FindElementEnd(s.data, tok)
	if err != nil {
		return Ship{}, fmt.Errorf("savegame: reading the player ship: %w", err)
	}

	if name == "" {
		if a, has := tok.Attr("sname"); has {
			name = xmldoc.Unescape(a.Value)
		}
	}
	return Ship{SID: sid, Name: name, Start: tok.Start, End: end}, nil
}

// playerShipRef reads the ship id and name recorded on the player's settlement.
func (s *Save) playerShipRef() (sid, name string, err error) {
	found := false

	walkErr := xmldoc.Walk(s.data, func(tok xmldoc.Token, _ []string) error {
		if !found {
			a, ok := tok.Attr("isPlayer")
			if !ok || a.Value != "true" {
				return nil
			}
			found = true
		}
		if sid == "" {
			if a, ok := tok.Attr("createdShipId"); ok {
				sid = a.Value
			}
		}
		if name == "" {
			if a, ok := tok.Attr("shn"); ok {
				name = xmldoc.Unescape(a.Value)
			}
		}
		if sid != "" && name != "" {
			return xmldoc.ErrStopWalk
		}
		return nil
	})
	if walkErr != nil {
		return "", "", walkErr
	}

	if !found {
		return "", "", fmt.Errorf("%w: no settlement is flagged isPlayer=\"true\"", ErrPlayerShipNotFound)
	}
	if sid == "" {
		return "", "", fmt.Errorf("%w: the player settlement records no createdShipId", ErrPlayerShipNotFound)
	}
	if _, convErr := strconv.Atoi(sid); convErr != nil {
		return "", "", fmt.Errorf("%w: createdShipId %q is not a number", ErrPlayerShipNotFound, sid)
	}
	return sid, name, nil
}

// ShipName returns the player ship's name, or an empty string when the save
// cannot be read.
func (s *Save) ShipName() string {
	ship, err := s.PlayerShip()
	if err != nil {
		return ""
	}
	return ship.Name
}
