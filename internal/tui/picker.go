package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/SecondPort/mod_space_haven/internal/library"
)

// pickerState is the save selection screen.
type pickerState struct {
	dir     string
	slots   []library.Slot
	table   table.Model
	loadErr error
}

func newPicker(dir string, txt uiText) pickerState {
	p := pickerState{dir: dir}
	p.table = newTable([]table.Column{
		{Title: txt.ColSlot, Width: 16},
		{Title: txt.ColShip, Width: 30},
		{Title: txt.ColDate, Width: 18},
	}, 12)

	if dir == "" {
		return p
	}

	slots, err := library.List(dir)
	if err != nil {
		p.loadErr = err
		return p
	}
	p.slots = slots

	rows := make([]table.Row, 0, len(slots))
	for _, s := range slots {
		ship := s.ShipName
		if ship == "" {
			ship = txt.EmptySlot
		}
		rows = append(rows, table.Row{s.Name, ship, s.ModifiedLabel()})
	}
	p.table.SetRows(rows)
	return p
}

func (p *pickerState) resize(height int) {
	if height > 0 {
		p.table.SetHeight(height)
	}
}

func (m Model) updatePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "enter":
		if len(m.picker.slots) == 0 {
			return m, nil
		}
		slot := m.picker.slots[m.picker.table.Cursor()]
		if err := m.openSave(slot.Path); err != nil {
			m.status.fail(err)
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.picker.table, cmd = m.picker.table.Update(msg)
	return m, cmd
}

func (m Model) viewPicker() string {
	var parts []string
	parts = append(parts, m.header())
	parts = append(parts, styleWarn.Padding(0, 1).Render(m.txt.PickWarning))
	parts = append(parts, styleSection.Padding(0, 1).Render(m.txt.PickTitle))

	switch {
	case m.picker.dir == "":
		parts = append(parts, m.missingDirectoryHelp())
	case m.picker.loadErr != nil:
		parts = append(parts, styleStatusErr.Render(m.picker.loadErr.Error()))
	case len(m.picker.slots) == 0:
		parts = append(parts,
			styleMuted.Padding(0, 1).Render(m.picker.dir),
			styleMuted.Padding(1, 1).Render(m.txt.NoSaves))
	default:
		parts = append(parts,
			styleMuted.Padding(0, 1).Render(m.picker.dir),
			lipgloss.NewStyle().Padding(0, 1).Render(m.picker.table.View()))
	}

	parts = append(parts, m.footer(m.txt.KeysPicker))
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// missingDirectoryHelp explains where the editor looked, which is the only
// useful thing to say when the game is installed somewhere unexpected.
func (m Model) missingDirectoryHelp() string {
	var b strings.Builder
	b.WriteString(m.txt.NoSaveDir + "\n\n" + m.txt.SearchedIn + "\n")
	for _, dir := range library.CandidateDirs() {
		b.WriteString("  • " + dir + "\n")
	}
	b.WriteString("\n" + m.txt.DirTip)
	return lipgloss.NewStyle().Padding(1, 1).Foreground(colorMuted).Render(b.String())
}
