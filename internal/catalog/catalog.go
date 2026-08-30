// Package catalog holds the reference data that turns raw save-file numbers
// into something a human can read: resource ids, skills, attributes, traits and
// research technologies.
//
// The tables were extracted from the game's own files (library/haven,
// library/texts and decompiled class constants) and are embedded in the binary,
// so the editor ships as a single file with no data directory to install.
package catalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Language selects which of the two name columns the catalog reports.
type Language string

const (
	// Spanish is the project's default interface language.
	Spanish Language = "es"
	// English mirrors the names the game itself displays.
	English Language = "en"
)

// ParseLanguage maps a user-supplied code to a Language, defaulting to Spanish.
func ParseLanguage(code string) Language {
	if strings.EqualFold(strings.TrimSpace(code), string(English)) {
		return English
	}
	return Spanish
}

// Resource is a storable item: cargo, weapons, gear, medicine and so on.
type Resource struct {
	ID int    `json:"id"`
	ES string `json:"es"`
	EN string `json:"en"`
}

// Name returns the resource name in the requested language.
func (r Resource) Name(lang Language) string {
	if lang == English && r.EN != "" {
		return r.EN
	}
	return r.ES
}

// Skill is a crew skill, addressed in the save file by its sk number.
type Skill struct {
	ID int    `json:"id"`
	ES string `json:"es"`
}

// Attribute is a crew personality attribute.
type Attribute struct {
	ID int    `json:"id"`
	EN string `json:"en"`
}

// Trait is a crew trait.
type Trait struct {
	ID int    `json:"id"`
	EN string `json:"en"`
}

// Tech is a research technology. LabPoints holds the level 1, 2 and 3 block
// counts the game expects once the technology is finished.
type Tech struct {
	ID        int    `json:"id"`
	EN        string `json:"en"`
	LabPoints [3]int `json:"labPoints"`
}

// Group names for the resource groupings the interface presents.
const (
	GroupFood        = "food"
	GroupFuel        = "fuel"
	GroupMedical     = "medical"
	GroupWeaponry    = "weaponry"
	GroupGear        = "gear"
	GroupWeapons     = "weapons"
	GroupAttachments = "attachments"
	GroupArmor       = "armor"
	GroupHeadgear    = "headgear"
	GroupSurvival    = "survival"
)

//go:embed data/resources.json
var resourcesJSON []byte

//go:embed data/skills.json
var skillsJSON []byte

//go:embed data/attributes.json
var attributesJSON []byte

//go:embed data/traits.json
var traitsJSON []byte

//go:embed data/techs.json
var techsJSON []byte

//go:embed data/groups.json
var groupsJSON []byte

// Catalog is an immutable lookup over the embedded reference tables.
type Catalog struct {
	resources  map[int]Resource
	skills     map[int]Skill
	attributes map[int]Attribute
	traits     map[int]Trait
	techs      map[int]Tech
	groups     map[string][]int
	groupSets  map[string]map[int]bool

	sortedResources []Resource
	sortedTraits    []Trait
}

var embedded = sync.OnceValues(load)

// Embedded returns the catalog built from the tables compiled into the binary.
func Embedded() (*Catalog, error) { return embedded() }

// MustEmbedded is Embedded for callers that cannot proceed without the catalog.
func MustEmbedded() *Catalog {
	c, err := Embedded()
	if err != nil {
		panic(err)
	}
	return c
}

func load() (*Catalog, error) {
	var (
		resources  []Resource
		skills     []Skill
		attributes []Attribute
		traits     []Trait
		techs      []Tech
		groups     map[string][]int
	)

	for _, table := range []struct {
		name string
		raw  []byte
		into any
	}{
		{"resources", resourcesJSON, &resources},
		{"skills", skillsJSON, &skills},
		{"attributes", attributesJSON, &attributes},
		{"traits", traitsJSON, &traits},
		{"techs", techsJSON, &techs},
		{"groups", groupsJSON, &groups},
	} {
		if err := json.Unmarshal(table.raw, table.into); err != nil {
			return nil, fmt.Errorf("catalog: reading the %s table: %w", table.name, err)
		}
	}

	c := &Catalog{
		resources:  make(map[int]Resource, len(resources)),
		skills:     make(map[int]Skill, len(skills)),
		attributes: make(map[int]Attribute, len(attributes)),
		traits:     make(map[int]Trait, len(traits)),
		techs:      make(map[int]Tech, len(techs)),
		groups:     groups,
		groupSets:  make(map[string]map[int]bool, len(groups)),
	}

	for _, r := range resources {
		c.resources[r.ID] = r
	}
	for _, s := range skills {
		c.skills[s.ID] = s
	}
	for _, a := range attributes {
		c.attributes[a.ID] = a
	}
	for _, t := range traits {
		c.traits[t.ID] = t
	}
	for _, t := range techs {
		c.techs[t.ID] = t
	}
	for name, ids := range groups {
		set := make(map[int]bool, len(ids))
		for _, id := range ids {
			set[id] = true
		}
		c.groupSets[name] = set
	}

	c.sortedResources = append(c.sortedResources, resources...)
	sort.Slice(c.sortedResources, func(i, j int) bool {
		return strings.ToLower(c.sortedResources[i].ES) < strings.ToLower(c.sortedResources[j].ES)
	})

	c.sortedTraits = append(c.sortedTraits, traits...)
	sort.Slice(c.sortedTraits, func(i, j int) bool {
		return c.sortedTraits[i].EN < c.sortedTraits[j].EN
	})

	return c, nil
}

// Resource looks up one resource.
func (c *Catalog) Resource(id int) (Resource, bool) {
	r, ok := c.resources[id]
	return r, ok
}

// Resources returns every known resource, ordered by Spanish name.
func (c *Catalog) Resources() []Resource {
	out := make([]Resource, len(c.sortedResources))
	copy(out, c.sortedResources)
	return out
}

// ResourceName returns a display name, falling back to the raw id for items the
// tables do not cover — a save can always contain something newer than these.
func (c *Catalog) ResourceName(id int, lang Language) string {
	if r, ok := c.resources[id]; ok {
		return r.Name(lang)
	}
	if lang == English {
		return "Item #" + strconv.Itoa(id)
	}
	return "Elemento #" + strconv.Itoa(id)
}

// ResourceLabel returns the name in the requested language with the other
// language in parentheses, which is how the editor disambiguates similar items.
func (c *Catalog) ResourceLabel(id int, lang Language) string {
	r, ok := c.resources[id]
	if !ok {
		return c.ResourceName(id, lang)
	}
	primary, secondary := r.ES, r.EN
	if lang == English {
		primary, secondary = r.EN, r.ES
	}
	if secondary == "" || secondary == primary {
		return primary
	}
	return primary + "  (" + secondary + ")"
}

// SearchResources matches a query against either language or the numeric id.
// An empty query returns everything.
func (c *Catalog) SearchResources(query string) []Resource {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return c.Resources()
	}
	var out []Resource
	for _, r := range c.sortedResources {
		if strings.Contains(strings.ToLower(r.ES), q) ||
			strings.Contains(strings.ToLower(r.EN), q) ||
			strings.Contains(strconv.Itoa(r.ID), q) {
			out = append(out, r)
		}
	}
	return out
}

// Group returns the resource ids in a named group, ordered by display name.
func (c *Catalog) Group(name string, lang Language) []int {
	ids := append([]int(nil), c.groups[name]...)
	sort.Slice(ids, func(i, j int) bool {
		return strings.ToLower(c.ResourceName(ids[i], lang)) < strings.ToLower(c.ResourceName(ids[j], lang))
	})
	return ids
}

// InGroup reports whether a resource belongs to a named group.
func (c *Catalog) InGroup(name string, id int) bool {
	return c.groupSets[name][id]
}

// IsClassified reports whether a resource belongs to any of the top-level cargo
// groups. Everything else is shown under the generic materials heading.
func (c *Catalog) IsClassified(id int) bool {
	for _, g := range []string{GroupFood, GroupFuel, GroupMedical, GroupWeaponry, GroupGear} {
		if c.groupSets[g][id] {
			return true
		}
	}
	return false
}

// Skill looks up a crew skill name, falling back to its raw sk number.
func (c *Catalog) Skill(id int) string {
	if s, ok := c.skills[id]; ok {
		return s.ES
	}
	return "sk=" + strconv.Itoa(id)
}

// Attribute looks up a crew attribute name.
func (c *Catalog) Attribute(id int) string {
	if a, ok := c.attributes[id]; ok {
		return a.EN
	}
	return "id=" + strconv.Itoa(id)
}

// Trait looks up a trait name.
func (c *Catalog) Trait(id int) string {
	if t, ok := c.traits[id]; ok {
		return t.EN
	}
	return strconv.Itoa(id)
}

// Traits returns every known trait, ordered by name.
func (c *Catalog) Traits() []Trait {
	out := make([]Trait, len(c.sortedTraits))
	copy(out, c.sortedTraits)
	return out
}

// Tech looks up a research technology.
func (c *Catalog) Tech(id int) (Tech, bool) {
	t, ok := c.techs[id]
	return t, ok
}

// TechName looks up a technology name, falling back to its raw id.
func (c *Catalog) TechName(id int) string {
	if t, ok := c.techs[id]; ok {
		return t.EN
	}
	return "Tech #" + strconv.Itoa(id)
}

// TechLabPoints returns the completed block counts to write for a technology.
// Unknown technologies get the minimal completion the game accepts.
func (c *Catalog) TechLabPoints(id int) [3]int {
	if t, ok := c.techs[id]; ok {
		return t.LabPoints
	}
	return [3]int{1, 0, 0}
}
