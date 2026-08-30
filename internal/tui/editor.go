package tui

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/SecondPort/mod_space_haven/internal/catalog"
	"github.com/SecondPort/mod_space_haven/internal/savegame"
	"github.com/charmbracelet/bubbles/table"
)

// section is one entry in the sidebar.
type section int

const (
	sectionCargo section = iota
	sectionWeapons
	sectionCrew
	sectionResearch
	sectionCount
)

const sidebarWidth = 20

// editorState holds the loaded save and every panel drawn from it.
type editorState struct {
	cat  *catalog.Catalog
	lang catalog.Language
	txt  uiText

	save     *savegame.Save
	path     string
	shipName string

	credits       int
	researchDone  int
	researchTotal int
	crew          []savegame.Character
	cargo         map[int]int

	section    section
	weaponPane int // 0 weapons, 1 defense

	cargoGrid    grid
	weaponGrid   grid
	defenseGrid  grid
	loadoutTable table.Model
	crewGrid     grid
	crewIDs      []string
	researchGrid grid
}

func newEditor(cat *catalog.Catalog, lang catalog.Language, txt uiText, save *savegame.Save, path string) (editorState, error) {
	e := editorState{cat: cat, lang: lang, txt: txt, save: save, path: path}

	ship, err := save.PlayerShip()
	if err != nil {
		return editorState{}, err
	}
	e.shipName = ship.Name
	if e.shipName == "" {
		e.shipName = "sid " + ship.SID
	}

	e.cargoGrid = newGrid([]table.Column{
		{Title: txt.ColID, Width: 7},
		{Title: txt.ColName, Width: 40},
		{Title: txt.ColAmount, Width: 12},
	}, 12)
	e.weaponGrid = newGrid([]table.Column{
		{Title: txt.ColID, Width: 6},
		{Title: txt.ColWeapon, Width: 30},
		{Title: txt.ColInCargo, Width: 10},
	}, 12)
	e.defenseGrid = newGrid([]table.Column{
		{Title: txt.ColID, Width: 6},
		{Title: txt.ColItem, Width: 28},
		{Title: txt.ColInCargo, Width: 10},
	}, 12)
	e.loadoutTable = newTable([]table.Column{
		{Title: txt.ColCrew, Width: 22},
		{Title: txt.ColPrimary, Width: 20},
		{Title: txt.ColSecond, Width: 20},
		{Title: txt.ColArmor, Width: 20},
		{Title: txt.ColHelmet, Width: 18},
	}, 6)
	e.loadoutTable.Blur()
	e.crewGrid = newGrid([]table.Column{
		{Title: txt.ColCrew, Width: 26},
		{Title: txt.ColHealth, Width: 8},
		{Title: txt.ColMood, Width: 8},
		{Title: txt.ColRest, Width: 10},
		{Title: txt.ColSkills, Width: 40},
	}, 12)
	e.researchGrid = newGrid([]table.Column{
		{Title: txt.ColID, Width: 8},
		{Title: txt.ColTech, Width: 42},
		{Title: txt.ColStatus, Width: 14},
	}, 12)

	if err := e.reload(); err != nil {
		return editorState{}, err
	}
	return e, nil
}

// reload rebuilds every panel from the save. Reading the whole document back is
// what keeps the panels honest after an edit: nothing is cached behind the
// bytes on the way to disk.
func (e *editorState) reload() error {
	cargo, err := e.save.Cargo()
	if err != nil {
		return err
	}
	e.cargo = cargo

	crew, err := e.save.Characters()
	if err != nil {
		return err
	}
	e.crew = crew

	research, err := e.save.Research()
	if err != nil {
		return err
	}
	e.researchTotal = len(research)
	e.researchDone = 0
	for _, done := range research {
		if done {
			e.researchDone++
		}
	}
	e.credits = e.save.Credits()

	e.fillCargo()
	e.fillWeapons()
	e.fillCrew()
	e.fillResearch(research)
	return nil
}

func (e *editorState) fillCargo() {
	var (
		rows []table.Row
		ids  []int
	)

	appendGroup := func(label string, group []int) {
		var present []int
		for _, id := range group {
			if _, ok := e.cargo[id]; ok {
				present = append(present, id)
			}
		}
		if len(present) == 0 {
			return
		}
		rows = append(rows, table.Row{"", heading(label), ""})
		ids = append(ids, headingID)
		for _, id := range present {
			rows = append(rows, table.Row{strconv.Itoa(id), e.cat.ResourceName(id, e.lang), humanInt(e.cargo[id], e.lang)})
			ids = append(ids, id)
		}
	}

	appendGroup(e.txt.SecFood, e.cat.Group(catalog.GroupFood, e.lang))
	appendGroup(e.txt.SecFuel, e.cat.Group(catalog.GroupFuel, e.lang))
	appendGroup(e.txt.SecMedical, e.cat.Group(catalog.GroupMedical, e.lang))
	appendGroup(e.txt.SecWeaponry, e.cat.Group(catalog.GroupWeaponry, e.lang))
	appendGroup(e.txt.SecGear, e.cat.Group(catalog.GroupGear, e.lang))

	var materials []int
	for id := range e.cargo {
		if !e.cat.IsClassified(id) {
			materials = append(materials, id)
		}
	}
	sort.Slice(materials, func(i, j int) bool {
		return e.cat.ResourceName(materials[i], e.lang) < e.cat.ResourceName(materials[j], e.lang)
	})
	appendGroup(e.txt.SecMaterials, materials)

	e.cargoGrid.setRows(rows, ids)
}

func (e *editorState) fillWeapons() {
	var (
		wRows []table.Row
		wIDs  []int
	)
	appendTo := func(rows *[]table.Row, ids *[]int, label string, group []int) {
		if len(group) == 0 {
			return
		}
		*rows = append(*rows, table.Row{"", heading(label), ""})
		*ids = append(*ids, headingID)
		for _, id := range group {
			*rows = append(*rows, table.Row{
				strconv.Itoa(id),
				e.cat.ResourceName(id, e.lang),
				amountCell(e.cargo[id], e.lang),
			})
			*ids = append(*ids, id)
		}
	}

	appendTo(&wRows, &wIDs, e.txt.SecWeapons, e.cat.Group(catalog.GroupWeapons, e.lang))
	appendTo(&wRows, &wIDs, e.txt.SecAttachments, e.cat.Group(catalog.GroupAttachments, e.lang))
	e.weaponGrid.setRows(wRows, wIDs)

	var (
		dRows []table.Row
		dIDs  []int
	)
	appendTo(&dRows, &dIDs, e.txt.SecArmor, e.cat.Group(catalog.GroupArmor, e.lang))
	appendTo(&dRows, &dIDs, e.txt.SecHeadgear, e.cat.Group(catalog.GroupHeadgear, e.lang))
	appendTo(&dRows, &dIDs, e.txt.SecSurvival, e.cat.Group(catalog.GroupSurvival, e.lang))
	e.defenseGrid.setRows(dRows, dIDs)

	slotName := func(id int) string {
		if id == 0 {
			return e.txt.EmptySlot
		}
		return e.cat.ResourceName(id, e.lang)
	}
	loadoutRows := make([]table.Row, 0, len(e.crew))
	for _, c := range e.crew {
		lo, err := e.save.CharacterLoadout(c.EntID)
		if err != nil {
			continue
		}
		loadoutRows = append(loadoutRows, table.Row{
			c.FullName(),
			slotName(lo.Primary),
			slotName(lo.Secondary),
			slotName(lo.Armor),
			slotName(lo.Headgear),
		})
	}
	e.loadoutTable.SetRows(loadoutRows)
}

func (e *editorState) fillCrew() {
	rows := make([]table.Row, 0, len(e.crew))
	ids := make([]int, 0, len(e.crew))
	e.crewIDs = e.crewIDs[:0]

	for i, c := range e.crew {
		stats, err := e.save.CharacterStats(c.EntID)
		if err != nil {
			continue
		}
		skills, err := e.save.CharacterSkills(c.EntID)
		if err != nil {
			continue
		}
		rows = append(rows, table.Row{
			c.FullName(),
			statCell(stats.Health),
			statCell(stats.Mood),
			statCell(stats.Rest),
			e.topSkills(skills),
		})
		ids = append(ids, i)
		e.crewIDs = append(e.crewIDs, c.EntID)
	}
	e.crewGrid.setRows(rows, ids)
}

// topSkills summarizes a crew member's four strongest skills for the list view.
func (e *editorState) topSkills(skills map[int]savegame.SkillLevel) string {
	type entry struct {
		id    int
		level int
	}
	var ranked []entry
	for id, s := range skills {
		if s.Level > 0 {
			ranked = append(ranked, entry{id, s.Level})
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].level != ranked[j].level {
			return ranked[i].level > ranked[j].level
		}
		return ranked[i].id < ranked[j].id
	})
	if len(ranked) > 4 {
		ranked = ranked[:4]
	}

	out := ""
	for _, r := range ranked {
		if out != "" {
			out += "  "
		}
		out += fmt.Sprintf("%s:%d", e.cat.Skill(r.id), r.level)
	}
	return out
}

func (e *editorState) fillResearch(research map[int]bool) {
	ids := make([]int, 0, len(research))
	for id := range research {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	rows := make([]table.Row, 0, len(ids))
	for _, id := range ids {
		status := e.txt.Pending
		if research[id] {
			status = e.txt.Done
		}
		rows = append(rows, table.Row{strconv.Itoa(id), e.cat.TechName(id), status})
	}
	e.researchGrid.setRows(rows, ids)
}

func statCell(v int) string {
	if v < 0 {
		return "—"
	}
	return strconv.Itoa(v)
}

func (e *editorState) resize(width, height int) {
	if height <= 0 {
		return
	}
	e.cargoGrid.setHeight(height)
	e.crewGrid.setHeight(height)
	e.researchGrid.setHeight(height)

	loadoutHeight := 6
	if height < 14 {
		loadoutHeight = 3
	}
	e.loadoutTable.SetHeight(loadoutHeight)
	paneHeight := height - loadoutHeight - 4
	if paneHeight < 3 {
		paneHeight = 3
	}
	e.weaponGrid.setHeight(paneHeight)
	e.defenseGrid.setHeight(paneHeight)

	if width > sidebarWidth+20 {
		content := width - sidebarWidth - 4
		e.cargoGrid.setColumns([]table.Column{
			{Title: e.txt.ColID, Width: 7},
			{Title: e.txt.ColName, Width: max(20, content-24)},
			{Title: e.txt.ColAmount, Width: 12},
		})
	}
}

// activeGrid is the panel the keyboard is driving right now.
func (e *editorState) activeGrid() *grid {
	switch e.section {
	case sectionCargo:
		return &e.cargoGrid
	case sectionWeapons:
		if e.weaponPane == 1 {
			return &e.defenseGrid
		}
		return &e.weaponGrid
	case sectionCrew:
		return &e.crewGrid
	default:
		return &e.researchGrid
	}
}
