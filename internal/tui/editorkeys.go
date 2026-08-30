package tui

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) updateEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m.requestQuit()

	case "s":
		m.saveToDisk()
		return m, nil

	case "c":
		m.overlay.openPrompt(m.txt.PromptCredits, m.txt.KeysModal,
			strconv.Itoa(m.editor.credits), true,
			intent{kind: intentCredits, previous: m.editor.credits})
		return m, nil

	case "tab":
		m.editor.section = (m.editor.section + 1) % sectionCount
		m.status.clear()
		return m, nil

	case "shift+tab":
		m.editor.section = (m.editor.section + sectionCount - 1) % sectionCount
		m.status.clear()
		return m, nil

	case "1", "2", "3", "4":
		n, _ := strconv.Atoi(msg.String())
		m.editor.section = section(n - 1)
		m.status.clear()
		return m, nil
	}

	switch m.editor.section {
	case sectionCargo:
		return m.updateCargoSection(msg)
	case sectionWeapons:
		return m.updateWeaponsSection(msg)
	case sectionCrew:
		return m.updateCrewSection(msg)
	default:
		return m.updateResearchSection(msg)
	}
}

func (m Model) updateCargoSection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "a":
		m.overlay.openSearch(m.cat, m.lang, m.txt)
		return m, nil
	case "enter":
		if id, ok := m.editor.cargoGrid.selected(); ok {
			m.promptForResource(id)
		}
		return m, nil
	}
	return m, m.editor.cargoGrid.update(msg)
}

func (m Model) updateWeaponsSection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "h":
		m.editor.weaponPane = 0
		return m, nil
	case "right", "l":
		m.editor.weaponPane = 1
		return m, nil
	case "a":
		m.overlay.openSearch(m.cat, m.lang, m.txt)
		return m, nil
	case "enter":
		if id, ok := m.editor.activeGrid().selected(); ok {
			m.promptForResource(id)
		}
		return m, nil
	}
	return m, m.editor.activeGrid().update(msg)
}

func (m Model) updateCrewSection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" {
		index, ok := m.editor.crewGrid.selected()
		if !ok || index >= len(m.editor.crewIDs) {
			return m, nil
		}
		m.openDetail(m.editor.crewIDs[index])
		return m, nil
	}
	return m, m.editor.crewGrid.update(msg)
}

func (m Model) updateResearchSection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		id, ok := m.editor.researchGrid.selected()
		if !ok {
			return m, nil
		}
		research, err := m.editor.save.Research()
		if err != nil {
			m.status.fail(err)
			return m, nil
		}
		done := !research[id]
		err = m.editor.save.SetTechDone(id, done, m.cat.TechLabPoints(id))
		format := m.txt.StatusReset
		if done {
			format = m.txt.StatusCompleted
		}
		m.applyEdit(err, format, m.cat.TechName(id))
		return m, nil

	case "A":
		m.setAllResearch(true)
		return m, nil

	case "R":
		m.setAllResearch(false)
		return m, nil
	}
	return m, m.editor.researchGrid.update(msg)
}

// setAllResearch completes or resets every technology in the save.
func (m *Model) setAllResearch(done bool) {
	research, err := m.editor.save.Research()
	if err != nil {
		m.status.fail(err)
		return
	}

	changed := 0
	for id, current := range research {
		if current == done {
			continue
		}
		if err := m.editor.save.SetTechDone(id, done, m.cat.TechLabPoints(id)); err != nil {
			m.status.fail(err)
			return
		}
		changed++
	}
	if changed == 0 {
		return
	}

	format := m.txt.StatusAllReset
	if done {
		format = m.txt.StatusAllDone
	}
	m.applyEdit(nil, format, changed)
}

func (m *Model) promptForResource(id int) {
	current := m.editor.cargo[id]
	m.overlay.openPrompt(
		m.cat.ResourceLabel(id, m.lang),
		m.txt.KeysModal,
		strconv.Itoa(current),
		true,
		intent{kind: intentCargo, resourceID: id, previous: current},
	)
}

func (m Model) viewEditor() string {
	content := ""
	keys := m.txt.KeysGlobal

	switch m.editor.section {
	case sectionCargo:
		content = m.editor.cargoGrid.view()
		keys = m.txt.KeysCargo + " · " + keys
	case sectionWeapons:
		content = m.viewWeapons()
		keys = "←/→ " + m.txt.SecDefense + " · " + m.txt.KeysCargo + " · " + keys
	case sectionCrew:
		content = m.editor.crewGrid.view()
		keys = m.txt.KeysCrew + " · " + keys
	case sectionResearch:
		content = m.editor.researchGrid.view()
		keys = m.txt.KeysResear + " · " + keys
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		styleSidebar.Height(m.contentHeight()).Width(sidebarWidth).Render(m.sidebar()),
		lipgloss.NewStyle().Padding(0, 1).Render(content),
	)
	return lipgloss.JoinVertical(lipgloss.Left, m.header(), body, m.footer(keys))
}

func (m Model) viewWeapons() string {
	weapons := m.editor.weaponGrid.view()
	defense := m.editor.defenseGrid.view()

	label := func(text string, active bool) string {
		if active {
			return styleSection.Render("▸ " + text)
		}
		return styleMuted.Render("  " + text)
	}

	panes := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.JoinVertical(lipgloss.Left,
			label(m.txt.SecWeapons, m.editor.weaponPane == 0), weapons),
		lipgloss.NewStyle().PaddingLeft(2).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				label(m.txt.SecDefense, m.editor.weaponPane == 1), defense)),
	)

	loadouts := lipgloss.JoinVertical(lipgloss.Left,
		styleSection.Render(m.txt.SecLoadouts)+" "+styleHint.Render("("+m.txt.LoadoutNote+")"),
		m.editor.loadoutTable.View(),
	)
	return lipgloss.JoinVertical(lipgloss.Left, panes, "", loadouts)
}

func (m Model) sidebar() string {
	items := []struct {
		key   string
		label string
		sec   section
	}{
		{"1", m.txt.NavCargo, sectionCargo},
		{"2", m.txt.NavWeapons, sectionWeapons},
		{"3", m.txt.NavCrew, sectionCrew},
		{"4", m.txt.NavResearch, sectionResearch},
	}

	rendered := make([]string, 0, len(items))
	for _, item := range items {
		style := styleNavItem
		if item.sec == m.editor.section {
			style = styleNavActive
		}
		rendered = append(rendered, style.Width(sidebarWidth-2).Render(item.key+"  "+item.label))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rendered...)
}
