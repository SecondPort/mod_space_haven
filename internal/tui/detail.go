package tui

import (
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/SecondPort/mod_space_haven/internal/savegame"
)

// detailPane is the panel the keyboard drives inside the crew detail screen.
type detailPane int

const (
	paneSkills detailPane = iota
	paneAttributes
	paneTraits
	paneCount
)

// detailState is one crew member's editable sheet.
type detailState struct {
	entID string
	name  string
	stats savegame.Stats

	skills     grid
	attributes grid
	traits     grid
	pane       detailPane
	ready      bool
}

func (m *Model) openDetail(entID string) {
	character, err := m.editor.save.Character(entID)
	if err != nil {
		m.status.fail(err)
		return
	}

	d := detailState{entID: entID, name: character.FullName()}
	d.skills = newGrid([]table.Column{
		{Title: "sk", Width: 5},
		{Title: m.txt.ColSkill, Width: 20},
		{Title: m.txt.ColLevel, Width: 7},
		{Title: m.txt.ColMax, Width: 6},
	}, 10)
	d.attributes = newGrid([]table.Column{
		{Title: "id", Width: 5},
		{Title: m.txt.ColAttr, Width: 18},
		{Title: m.txt.ColPoints, Width: 8},
	}, 6)
	d.traits = newGrid([]table.Column{
		{Title: "", Width: 3},
		{Title: "id", Width: 6},
		{Title: m.txt.ColTrait, Width: 24},
	}, 10)
	d.ready = true

	m.detail = d
	m.stage = stageDetail
	m.refreshDetail()
	m.detail.resize(m.width, m.contentHeight())
	m.status.clear()
}

// refreshDetail reloads the sheet from the save.
func (m *Model) refreshDetail() {
	if !m.detail.ready {
		return
	}

	character, err := m.editor.save.Character(m.detail.entID)
	if err != nil {
		m.status.fail(err)
		return
	}
	m.detail.name = character.FullName()

	stats, err := m.editor.save.CharacterStats(m.detail.entID)
	if err != nil {
		m.status.fail(err)
		return
	}
	m.detail.stats = stats

	skills, err := m.editor.save.CharacterSkills(m.detail.entID)
	if err != nil {
		m.status.fail(err)
		return
	}
	skillIDs := savegame.SortedSkillIDs(skills)
	skillRows := make([]table.Row, 0, len(skillIDs))
	for _, id := range skillIDs {
		s := skills[id]
		skillRows = append(skillRows, table.Row{
			strconv.Itoa(id), m.cat.Skill(id), strconv.Itoa(s.Level), strconv.Itoa(s.Max),
		})
	}
	m.detail.skills.setRows(skillRows, skillIDs)

	attrs, err := m.editor.save.CharacterAttributes(m.detail.entID)
	if err != nil {
		m.status.fail(err)
		return
	}
	attrIDs := savegame.SortedAttributeIDs(attrs)
	attrRows := make([]table.Row, 0, len(attrIDs))
	for _, id := range attrIDs {
		attrRows = append(attrRows, table.Row{
			strconv.Itoa(id), m.cat.Attribute(id), strconv.Itoa(attrs[id]),
		})
	}
	m.detail.attributes.setRows(attrRows, attrIDs)

	owned, err := m.editor.save.CharacterTraits(m.detail.entID)
	if err != nil {
		m.status.fail(err)
		return
	}
	has := make(map[int]bool, len(owned))
	for _, id := range owned {
		has[id] = true
	}

	catalogTraits := m.cat.Traits()
	traitRows := make([]table.Row, 0, len(catalogTraits))
	traitIDs := make([]int, 0, len(catalogTraits))
	for _, t := range catalogTraits {
		mark := " "
		if has[t.ID] {
			mark = "✓"
		}
		traitRows = append(traitRows, table.Row{mark, strconv.Itoa(t.ID), t.EN})
		traitIDs = append(traitIDs, t.ID)
	}
	// Traits the save carries that the catalog does not know about still belong
	// on the sheet — a mod or a newer game version can add them.
	for _, id := range owned {
		if containsID(traitIDs, id) {
			continue
		}
		traitRows = append(traitRows, table.Row{"✓", strconv.Itoa(id), m.cat.Trait(id)})
		traitIDs = append(traitIDs, id)
	}
	m.detail.traits.setRows(traitRows, traitIDs)
}

func containsID(ids []int, want int) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func (d *detailState) resize(width, height int) {
	if height <= 0 {
		return
	}
	panelHeight := height - 4
	if panelHeight < 4 {
		panelHeight = 4
	}
	d.skills.setHeight(panelHeight)
	d.traits.setHeight(panelHeight)

	attrHeight := panelHeight / 2
	if attrHeight < 3 {
		attrHeight = 3
	}
	d.attributes.setHeight(attrHeight)
}

func (d *detailState) activeGrid() *grid {
	switch d.pane {
	case paneSkills:
		return &d.skills
	case paneAttributes:
		return &d.attributes
	default:
		return &d.traits
	}
}

func (m Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.stage = stageEditor
		m.status.clear()
		return m, nil

	case "q", "ctrl+c":
		return m.requestQuit()

	case "s":
		m.saveToDisk()
		return m, nil

	case "tab":
		m.detail.pane = (m.detail.pane + 1) % paneCount
		return m, nil

	case "shift+tab":
		m.detail.pane = (m.detail.pane + paneCount - 1) % paneCount
		return m, nil

	case "n":
		character, err := m.editor.save.Character(m.detail.entID)
		if err != nil {
			m.status.fail(err)
			return m, nil
		}
		m.overlay.openPrompt(m.txt.PromptFirst, m.txt.KeysModal, character.Name, false,
			intent{kind: intentRename, entID: m.detail.entID})
		return m, nil

	case "H", "M", "R":
		m.promptForStat(msg.String())
		return m, nil

	case "enter":
		return m.activateDetailRow()
	}

	return m, m.detail.activeGrid().update(msg)
}

// promptForStat opens the editor for one of the three character stats.
func (m *Model) promptForStat(key string) {
	var (
		stat    string
		current int
		ceiling int
	)
	switch key {
	case "H":
		stat, current, ceiling = savegame.StatHealth, m.detail.stats.Health, maxHealth
	case "M":
		stat, current, ceiling = savegame.StatMood, m.detail.stats.Mood, maxMood
	default:
		stat, current, ceiling = savegame.StatRest, m.detail.stats.Rest, maxRest
	}

	m.overlay.openPrompt(
		stat+" (0-"+strconv.Itoa(ceiling)+")",
		m.txt.KeysModal,
		strconv.Itoa(maxInt(current, 0)),
		true,
		intent{kind: intentStat, entID: m.detail.entID, stat: stat, previous: current},
	)
}

func (m Model) activateDetailRow() (tea.Model, tea.Cmd) {
	id, ok := m.detail.activeGrid().selected()
	if !ok {
		return m, nil
	}

	switch m.detail.pane {
	case paneSkills:
		skills, err := m.editor.save.CharacterSkills(m.detail.entID)
		if err != nil {
			m.status.fail(err)
			return m, nil
		}
		m.overlay.openPrompt(
			m.cat.Skill(id)+" (0-"+strconv.Itoa(maxSkill)+")",
			m.txt.KeysModal,
			strconv.Itoa(skills[id].Level),
			true,
			intent{kind: intentSkill, entID: m.detail.entID, skillID: id, previous: skills[id].Level},
		)

	case paneAttributes:
		attrs, err := m.editor.save.CharacterAttributes(m.detail.entID)
		if err != nil {
			m.status.fail(err)
			return m, nil
		}
		m.overlay.openPrompt(
			m.cat.Attribute(id)+" ("+strconv.Itoa(minAttribute)+"-"+strconv.Itoa(maxAttribute)+")",
			m.txt.KeysModal,
			strconv.Itoa(attrs[id]),
			true,
			intent{kind: intentAttribute, entID: m.detail.entID, attrID: id, previous: attrs[id]},
		)

	case paneTraits:
		m.toggleTrait(id)
	}
	return m, nil
}

func (m *Model) toggleTrait(traitID int) {
	owned, err := m.editor.save.CharacterTraits(m.detail.entID)
	if err != nil {
		m.status.fail(err)
		return
	}

	name := m.cat.Trait(traitID)
	if containsID(owned, traitID) {
		m.applyEdit(m.editor.save.RemoveCharacterTrait(m.detail.entID, traitID), m.txt.StatusRemoved, name)
		return
	}
	m.applyEdit(m.editor.save.AddCharacterTrait(m.detail.entID, traitID), m.txt.StatusAdded, name)
}

func (m Model) viewDetail() string {
	title := lipgloss.JoinHorizontal(lipgloss.Top,
		styleSection.Render(m.detail.name),
		styleHint.Render("   "+m.txt.ColHealth+" "+statCell(m.detail.stats.Health)+
			" · "+m.txt.ColMood+" "+statCell(m.detail.stats.Mood)+
			" · "+m.txt.ColRest+" "+statCell(m.detail.stats.Rest)),
	)

	label := func(text string, pane detailPane) string {
		if m.detail.pane == pane {
			return styleSection.Render("▸ " + text)
		}
		return styleMuted.Render("  " + text)
	}

	left := lipgloss.JoinVertical(lipgloss.Left,
		label(m.txt.SecSkills, paneSkills), m.detail.skills.view())
	middle := lipgloss.JoinVertical(lipgloss.Left,
		label(m.txt.SecAttributes, paneAttributes), m.detail.attributes.view())
	right := lipgloss.JoinVertical(lipgloss.Left,
		label(m.txt.SecTraits, paneTraits), m.detail.traits.view())

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		left,
		lipgloss.NewStyle().PaddingLeft(2).Render(middle),
		lipgloss.NewStyle().PaddingLeft(2).Render(right),
	)

	keys := m.txt.KeysDetail + " · H/M/R " + m.txt.ColHealth + "/" + m.txt.ColMood + "/" + m.txt.ColRest + " · s"
	return lipgloss.JoinVertical(lipgloss.Left,
		m.header(),
		lipgloss.NewStyle().Padding(0, 1).Render(lipgloss.JoinVertical(lipgloss.Left, title, "", body)),
		m.footer(keys),
	)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
