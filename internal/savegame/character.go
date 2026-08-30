package savegame

import (
	"fmt"
	"sort"
	"strings"

	"github.com/SecondPort/mod_space_haven/internal/xmldoc"
)

// Stat names as they appear in a character's block.
const (
	StatHealth = "Health"
	StatMood   = "Mood"
	StatRest   = "Rest"
)

const (
	characterTag = "c"
	traitsTag    = "traits"
	traitTag     = "t"
	skillTag     = "s"
	attributeTag = "a"
	loadoutTag   = "loadout"
	playerSide   = "Player"
)

// Character identifies one crew member. EntID is the entity id, which is the
// only identifier that stays stable across an edit — cid is reused for other
// things, and names are not unique.
type Character struct {
	EntID    string
	CID      string
	Name     string
	LastName string
	Start    int
	End      int
}

// FullName is the crew member's display name.
func (c Character) FullName() string {
	if c.LastName == "" {
		return c.Name
	}
	return c.Name + " " + c.LastName
}

// Stats are the three character values the editor exposes. A value of -1 means
// the save does not record that stat for this character.
type Stats struct {
	Health int
	Mood   int
	Rest   int
}

// SkillLevel is a crew skill's current and maximum level.
type SkillLevel struct {
	Level int
	Max   int
}

// Loadout is a crew member's equipment, by resource id. Zero means the slot is
// empty. It is read-only: crew equip themselves from cargo, so the editor shows
// the loadout and lets you stock the ship rather than forcing an assignment.
type Loadout struct {
	Headgear   int
	Armor      int
	Primary    int
	Attachment int
	Secondary  int
	Pocket1    int
	Pocket2    int
	Pocket3    int
}

// Characters returns the player's crew in document order.
func (s *Save) Characters() ([]Character, error) {
	var crew []Character

	err := xmldoc.Walk(s.data, func(tok xmldoc.Token, _ []string) error {
		if !isElement(tok) || !tok.NameIs(characterTag) {
			return nil
		}
		side, ok := tok.Attr("side")
		if !ok || side.Value != playerSide {
			return nil
		}
		c, ok := characterFrom(tok)
		if !ok {
			return nil
		}
		end, err := xmldoc.FindElementEnd(s.data, tok)
		if err != nil {
			return err
		}
		c.End = end
		crew = append(crew, c)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("savegame: reading the crew: %w", err)
	}
	return crew, nil
}

// Character looks up any character in the save by entity id, crew or not.
func (s *Save) Character(entID string) (Character, error) {
	tok, end, err := s.characterToken(entID)
	if err != nil {
		return Character{}, err
	}
	c, _ := characterFrom(tok)
	c.End = end
	return c, nil
}

func characterFrom(tok xmldoc.Token) (Character, bool) {
	ent, ok := tok.Attr("entId")
	if !ok {
		return Character{}, false
	}
	c := Character{EntID: ent.Value, Start: tok.Start}
	if a, ok := tok.Attr("cid"); ok {
		c.CID = a.Value
	}
	if a, ok := tok.Attr("name"); ok {
		c.Name = xmldoc.Unescape(a.Value)
	}
	if a, ok := tok.Attr("lname"); ok {
		c.LastName = xmldoc.Unescape(a.Value)
	}
	return c, true
}

// characterToken locates a character's opening tag and the end of its element.
func (s *Save) characterToken(entID string) (xmldoc.Token, int, error) {
	doc := s.region(0, len(s.data))
	tok, ok := doc.firstToken(func(t xmldoc.Token) bool {
		if !isElement(t) || !t.NameIs(characterTag) {
			return false
		}
		a, has := t.Attr("entId")
		return has && a.Value == entID
	})
	if !ok {
		return xmldoc.Token{}, 0, fmt.Errorf("%w: entId %q", ErrCharacterNotFound, entID)
	}
	end, err := xmldoc.FindElementEnd(s.data, tok)
	if err != nil {
		return xmldoc.Token{}, 0, fmt.Errorf("savegame: reading character %q: %w", entID, err)
	}
	return tok, end, nil
}

func (s *Save) characterRegion(entID string) (region, xmldoc.Token, error) {
	tok, end, err := s.characterToken(entID)
	if err != nil {
		return region{}, xmldoc.Token{}, err
	}
	return s.region(tok.Start, end), tok, nil
}

// CharacterStats reads a crew member's health, mood and rest.
func (s *Save) CharacterStats(entID string) (Stats, error) {
	block, _, err := s.characterRegion(entID)
	if err != nil {
		return Stats{}, err
	}

	stats := Stats{Health: -1, Mood: -1, Rest: -1}
	for _, item := range []struct {
		name string
		into *int
	}{
		{StatHealth, &stats.Health},
		{StatMood, &stats.Mood},
		{StatRest, &stats.Rest},
	} {
		if tok, ok := statToken(block, item.name); ok {
			if v, ok := tok.IntAttr("v"); ok {
				*item.into = v
			}
		}
	}
	return stats, nil
}

// SetCharacterStat writes one of a crew member's stats.
func (s *Save) SetCharacterStat(entID, stat string, value int) error {
	block, _, err := s.characterRegion(entID)
	if err != nil {
		return err
	}
	tok, ok := statToken(block, stat)
	if !ok {
		return fmt.Errorf("savegame: character %q has no %s stat", entID, stat)
	}
	patch, err := setIntAttr(tok, "v", value)
	if err != nil {
		return err
	}
	return s.apply(xmldoc.PatchSet{block.shift(patch)})
}

func statToken(block region, stat string) (xmldoc.Token, bool) {
	return block.firstToken(func(t xmldoc.Token) bool {
		if !isElement(t) || !t.NameIs(stat) {
			return false
		}
		_, has := t.Attr("v")
		return has
	})
}

// CharacterSkills reads a crew member's skills, keyed by the save's sk number.
func (s *Save) CharacterSkills(entID string) (map[int]SkillLevel, error) {
	block, _, err := s.characterRegion(entID)
	if err != nil {
		return nil, err
	}

	skills := make(map[int]SkillLevel)
	block.eachToken(func(tok xmldoc.Token) bool {
		if !isElement(tok) || !tok.NameIs(skillTag) {
			return true
		}
		sk, ok := tok.IntAttr("sk")
		if !ok {
			return true
		}
		level, hasLevel := tok.IntAttr("level")
		max, hasMax := tok.IntAttr("mxn")
		if !hasLevel || !hasMax {
			return true
		}
		skills[sk] = SkillLevel{Level: level, Max: max}
		return true
	})
	return skills, nil
}

// SetCharacterSkill writes one skill's level and ceiling.
func (s *Save) SetCharacterSkill(entID string, sk, level, max int) error {
	block, _, err := s.characterRegion(entID)
	if err != nil {
		return err
	}

	tok, ok := block.firstToken(func(t xmldoc.Token) bool {
		if !isElement(t) || !t.NameIs(skillTag) {
			return false
		}
		v, has := t.IntAttr("sk")
		return has && v == sk
	})
	if !ok {
		return fmt.Errorf("savegame: character %q has no skill sk=%d", entID, sk)
	}

	levelPatch, err := setIntAttr(tok, "level", level)
	if err != nil {
		return err
	}
	maxPatch, err := setIntAttr(tok, "mxn", max)
	if err != nil {
		return err
	}
	return s.apply(xmldoc.PatchSet{block.shift(levelPatch), block.shift(maxPatch)})
}

// CharacterAttributes reads a crew member's personality attributes.
func (s *Save) CharacterAttributes(entID string) (map[int]int, error) {
	block, _, err := s.characterRegion(entID)
	if err != nil {
		return nil, err
	}

	attrs := make(map[int]int)
	block.eachToken(func(tok xmldoc.Token) bool {
		if !isElement(tok) || !tok.NameIs(attributeTag) {
			return true
		}
		id, hasID := tok.IntAttr("id")
		points, hasPoints := tok.IntAttr("points")
		if hasID && hasPoints {
			attrs[id] = points
		}
		return true
	})
	return attrs, nil
}

// SetCharacterAttribute writes one personality attribute.
func (s *Save) SetCharacterAttribute(entID string, attrID, points int) error {
	block, _, err := s.characterRegion(entID)
	if err != nil {
		return err
	}

	tok, ok := block.firstToken(func(t xmldoc.Token) bool {
		if !isElement(t) || !t.NameIs(attributeTag) {
			return false
		}
		v, has := t.IntAttr("id")
		return has && v == attrID
	})
	if !ok {
		return fmt.Errorf("savegame: character %q has no attribute id=%d", entID, attrID)
	}

	patch, err := setIntAttr(tok, "points", points)
	if err != nil {
		return err
	}
	return s.apply(xmldoc.PatchSet{block.shift(patch)})
}

// CharacterTraits reads a crew member's traits in document order.
func (s *Save) CharacterTraits(entID string) ([]int, error) {
	block, _, err := s.characterRegion(entID)
	if err != nil {
		return nil, err
	}
	traits, _, _, err := traitList(block)
	if err != nil {
		return nil, err
	}

	ids := make([]int, len(traits))
	for i, t := range traits {
		ids[i] = t.id
	}
	return ids, nil
}

// AddCharacterTrait gives a crew member a trait they do not already have.
func (s *Save) AddCharacterTrait(entID string, traitID int) error {
	block, _, err := s.characterRegion(entID)
	if err != nil {
		return err
	}
	traits, contentEnd, ok, err := traitList(block)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("savegame: character %q has no <traits> block", entID)
	}
	for _, t := range traits {
		if t.id == traitID {
			return fmt.Errorf("savegame: character %q already has trait %d", entID, traitID)
		}
	}

	entry := fmt.Sprintf(`<%s id="%d"/>`, traitTag, traitID)
	var patch xmldoc.Patch
	if n := len(traits); n > 0 {
		indent := leadingWhitespace(block.data, traits[0].start)
		patch = xmldoc.InsertAt(traits[n-1].end, indent+entry)
	} else {
		closing := leadingWhitespace(block.data, contentEnd)
		patch = xmldoc.InsertAt(contentEnd, "\t"+entry+closing)
	}
	return s.apply(xmldoc.PatchSet{block.shift(patch)})
}

// RemoveCharacterTrait takes a trait away, along with the whitespace that laid
// it out, so the block does not accumulate blank lines.
func (s *Save) RemoveCharacterTrait(entID string, traitID int) error {
	block, _, err := s.characterRegion(entID)
	if err != nil {
		return err
	}
	traits, _, _, err := traitList(block)
	if err != nil {
		return err
	}

	for _, t := range traits {
		if t.id != traitID {
			continue
		}
		indent := leadingWhitespace(block.data, t.start)
		patch := xmldoc.Delete(t.start-len(indent), t.end)
		return s.apply(xmldoc.PatchSet{block.shift(patch)})
	}
	return fmt.Errorf("savegame: character %q does not have trait %d", entID, traitID)
}

type traitEntry struct {
	id         int
	start, end int
}

// traitList returns a character's trait entries and the offset just before the
// closing </traits> tag.
func traitList(block region) (entries []traitEntry, contentEnd int, found bool, err error) {
	open, ok := block.firstToken(func(t xmldoc.Token) bool {
		return t.Kind == xmldoc.KindStart && t.NameIs(traitsTag)
	})
	if !ok {
		return nil, 0, false, nil
	}

	end, err := xmldoc.FindElementEnd(block.data, open)
	if err != nil {
		return nil, 0, false, fmt.Errorf("savegame: reading traits: %w", err)
	}
	contentEnd = end - len("</"+traitsTag+">")

	sc := xmldoc.NewScannerAt(block.data, open.End)
	for {
		tok, ok := sc.Next()
		if !ok || tok.Start >= contentEnd {
			break
		}
		if !isElement(tok) || !tok.NameIs(traitTag) {
			continue
		}
		if id, ok := tok.IntAttr("id"); ok {
			entries = append(entries, traitEntry{id: id, start: tok.Start, end: tok.End})
		}
	}
	return entries, contentEnd, true, nil
}

// SetCharacterName renames a crew member.
func (s *Save) SetCharacterName(entID, name, lastName string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("savegame: a character needs a first name")
	}
	lastName = strings.TrimSpace(lastName)

	tok, _, err := s.characterToken(entID)
	if err != nil {
		return err
	}

	var patches xmldoc.PatchSet
	nameAttr, ok := tok.Attr("name")
	if !ok {
		return fmt.Errorf("savegame: character %q has no name attribute", entID)
	}
	patches = append(patches, xmldoc.SetAttr(nameAttr, name))

	if lastAttr, ok := tok.Attr("lname"); ok {
		patches = append(patches, xmldoc.SetAttr(lastAttr, lastName))
	}
	return s.apply(patches)
}

// CharacterLoadout reads a crew member's equipment slots. A character with no
// loadout element reports empty slots rather than an error — not every entity
// in a save carries equipment.
func (s *Save) CharacterLoadout(entID string) (Loadout, error) {
	block, _, err := s.characterRegion(entID)
	if err != nil {
		return Loadout{}, err
	}

	tok, ok := block.firstToken(func(t xmldoc.Token) bool {
		return isElement(t) && t.NameIs(loadoutTag)
	})
	if !ok {
		return Loadout{}, nil
	}

	var lo Loadout
	for _, slot := range []struct {
		attr string
		into *int
	}{
		{"headgear", &lo.Headgear},
		{"armor", &lo.Armor},
		{"primary", &lo.Primary},
		{"attachment", &lo.Attachment},
		{"secondary", &lo.Secondary},
		{"pocket1", &lo.Pocket1},
		{"pocket2", &lo.Pocket2},
		{"pocket3", &lo.Pocket3},
	} {
		if v, ok := tok.IntAttr(slot.attr); ok {
			*slot.into = v
		}
	}
	return lo, nil
}

// SortedSkillIDs returns a skill map's keys in ascending order, which is how the
// interface lists them.
func SortedSkillIDs(skills map[int]SkillLevel) []int {
	ids := make([]int, 0, len(skills))
	for id := range skills {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// SortedAttributeIDs returns an attribute map's keys in ascending order.
func SortedAttributeIDs(attrs map[int]int) []int {
	ids := make([]int, 0, len(attrs))
	for id := range attrs {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}
