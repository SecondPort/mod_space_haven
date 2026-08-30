package savegame

import (
	"fmt"

	"github.com/SecondPort/mod_space_haven/internal/xmldoc"
)

const (
	researchEntryTag = "l"
	blocksDoneTag    = "blocksDone"
	doneAttr         = "done"
)

// Research reports every technology in the save and whether it is finished.
func (s *Save) Research() (map[int]bool, error) {
	entries, err := s.researchEntries()
	if err != nil {
		return nil, err
	}

	out := make(map[int]bool, len(entries))
	for _, e := range entries {
		out[e.techID] = e.done
	}
	return out, nil
}

// SetTechDone marks a technology finished or pending. blocks are the level 1, 2
// and 3 lab point counts to record on completion; a reset writes zeroes.
//
// The save keeps a technology's state on elements nested inside its entry
// rather than on the entry itself, and an entry can carry several of them, so
// every done flag inside the entry is written.
func (s *Save) SetTechDone(techID int, done bool, blocks [3]int) error {
	entries, err := s.researchEntries()
	if err != nil {
		return err
	}

	var entry *researchEntry
	for i := range entries {
		if entries[i].techID == techID {
			entry = &entries[i]
			break
		}
	}
	if entry == nil {
		return fmt.Errorf("%w: techId %d", ErrTechNotFound, techID)
	}

	if !done {
		blocks = [3]int{0, 0, 0}
	}
	flag := "false"
	if done {
		flag = "true"
	}

	block := s.region(entry.start, entry.end)
	var patches xmldoc.PatchSet

	block.eachToken(func(tok xmldoc.Token) bool {
		if !isElement(tok) {
			return true
		}
		if a, ok := tok.Attr(doneAttr); ok {
			patches = append(patches, block.shift(xmldoc.SetAttrRaw(a, flag)))
		}
		if tok.NameIs(blocksDoneTag) {
			for i, name := range []string{"level1", "level2", "level3"} {
				if a, ok := tok.Attr(name); ok {
					patches = append(patches, block.shift(xmldoc.SetAttrRaw(a, fmt.Sprint(blocks[i]))))
				}
			}
		}
		return true
	})

	if len(patches) == 0 {
		return fmt.Errorf("savegame: research entry %d records no state to change", techID)
	}
	return s.apply(patches)
}

type researchEntry struct {
	techID     int
	done       bool
	start, end int
}

func (s *Save) researchEntries() ([]researchEntry, error) {
	var entries []researchEntry

	sc := xmldoc.NewScanner(s.data)
	for {
		tok, ok := sc.Next()
		if !ok {
			break
		}
		if !isElement(tok) || !tok.NameIs(researchEntryTag) {
			continue
		}
		techID, ok := tok.IntAttr("techId")
		if !ok {
			continue
		}
		end, err := xmldoc.FindElementEnd(s.data, tok)
		if err != nil {
			return nil, fmt.Errorf("savegame: reading research entry %d: %w", techID, err)
		}

		entries = append(entries, researchEntry{
			techID: techID,
			done:   readDoneFlag(s.region(tok.Start, end), tok),
			start:  tok.Start,
			end:    end,
		})
	}
	return entries, nil
}

// readDoneFlag takes the first done flag inside the entry, falling back to one
// on the entry element itself.
func readDoneFlag(block region, entry xmldoc.Token) bool {
	var (
		found bool
		done  bool
	)
	block.eachToken(func(tok xmldoc.Token) bool {
		if tok.Start == 0 || !isElement(tok) {
			return true
		}
		if a, ok := tok.Attr(doneAttr); ok {
			done, found = a.Value == "true", true
			return false
		}
		return true
	})
	if found {
		return done
	}
	if a, ok := entry.Attr(doneAttr); ok {
		return a.Value == "true"
	}
	return false
}
