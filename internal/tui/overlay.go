package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/SecondPort/mod_space_haven/internal/catalog"
)

type overlayKind int

const (
	overlayNone overlayKind = iota
	overlayPrompt
	overlaySearch
	overlayConfirm
)

// intentKind says what the value coming out of a prompt is for. Bubble Tea
// models are values, so the pending action is recorded as data rather than as a
// closure over a model that is about to be copied.
type intentKind int

const (
	intentNone intentKind = iota
	intentCredits
	intentCargo
	intentStat
	intentSkill
	intentAttribute
	intentRename
)

type intent struct {
	kind       intentKind
	resourceID int
	entID      string
	stat       string
	skillID    int
	attrID     int
	previous   int
	firstName  string
	askingLast bool
}

// overlayState is the modal layer: a prompt, a resource search, or a
// confirmation. Only one is ever open.
type overlayState struct {
	kind overlayKind

	title   string
	hint    string
	input   textinput.Model
	numeric bool
	act     intent

	searchGrid    grid
	searchMatches []catalog.Resource

	confirmBody    string
	confirmOptions []string
	confirmIndex   int
}

func newOverlay() overlayState {
	in := textinput.New()
	in.CharLimit = 64
	in.Width = 40
	return overlayState{input: in}
}

func (o *overlayState) active() bool { return o.kind != overlayNone }

func (o *overlayState) close() {
	o.kind = overlayNone
	o.act = intent{}
	o.input.SetValue("")
	o.input.Blur()
}

// openPrompt asks for a single value.
func (o *overlayState) openPrompt(title, hint, initial string, numeric bool, act intent) {
	o.kind = overlayPrompt
	o.title = title
	o.hint = hint
	o.numeric = numeric
	o.act = act
	o.input.Placeholder = ""
	o.input.SetValue(initial)
	o.input.CursorEnd()
	o.input.Focus()
}

// openSearch opens the resource picker.
func (o *overlayState) openSearch(cat *catalog.Catalog, lang catalog.Language, txt uiText) {
	o.kind = overlaySearch
	o.title = txt.SearchTitle
	o.hint = txt.KeysModal
	o.numeric = false
	o.act = intent{}
	o.input.SetValue("")
	o.input.Placeholder = txt.PromptSearch
	o.input.Focus()

	o.searchGrid = newGrid([]table.Column{
		{Title: txt.ColID, Width: 8},
		{Title: txt.ColName, Width: 44},
	}, 12)
	o.applySearch(cat, lang)
}

func (o *overlayState) applySearch(cat *catalog.Catalog, lang catalog.Language) {
	o.searchMatches = cat.SearchResources(o.input.Value())

	rows := make([]table.Row, 0, len(o.searchMatches))
	ids := make([]int, 0, len(o.searchMatches))
	for _, r := range o.searchMatches {
		rows = append(rows, table.Row{strconv.Itoa(r.ID), r.Name(lang)})
		ids = append(ids, r.ID)
	}
	o.searchGrid.setRows(rows, ids)
	o.searchGrid.table.SetCursor(0)
}

// openConfirm asks a question with a fixed set of answers.
func (o *overlayState) openConfirm(title, body string, options []string) {
	o.kind = overlayConfirm
	o.title = title
	o.confirmBody = body
	o.confirmOptions = options
	o.confirmIndex = 0
	o.input.Blur()
}

// updateOverlay routes keys while a modal is open.
func (m Model) updateOverlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.overlay.kind {
	case overlayConfirm:
		return m.updateConfirm(msg)
	case overlaySearch:
		return m.updateSearch(msg)
	default:
		return m.updatePrompt(msg)
	}
}

func (m Model) updatePrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.overlay.close()
		return m, nil
	case "enter":
		value := strings.TrimSpace(m.overlay.input.Value())
		act := m.overlay.act
		m.overlay.close()
		m.applyIntent(act, value)
		return m, nil
	}

	if m.overlay.numeric && !acceptableNumeric(msg) {
		return m, nil
	}

	var cmd tea.Cmd
	m.overlay.input, cmd = m.overlay.input.Update(msg)
	return m, cmd
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.overlay.close()
		return m, nil

	case "enter":
		id, ok := m.overlay.searchGrid.selected()
		m.overlay.close()
		if !ok {
			return m, nil
		}
		current := m.editor.cargo[id]
		m.overlay.openPrompt(
			m.cat.ResourceLabel(id, m.lang),
			m.txt.KeysModal,
			strconv.Itoa(current),
			true,
			intent{kind: intentCargo, resourceID: id, previous: current},
		)
		return m, nil

	case "up", "down", "pgup", "pgdown", "home", "end":
		return m, m.overlay.searchGrid.update(msg)
	}

	var cmd tea.Cmd
	m.overlay.input, cmd = m.overlay.input.Update(msg)
	m.overlay.applySearch(m.cat, m.lang)
	return m, cmd
}

func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.overlay.close()
		return m, nil

	case "left", "h", "shift+tab":
		if m.overlay.confirmIndex > 0 {
			m.overlay.confirmIndex--
		}
		return m, nil

	case "right", "l", "tab":
		if m.overlay.confirmIndex < len(m.overlay.confirmOptions)-1 {
			m.overlay.confirmIndex++
		}
		return m, nil

	case "enter":
		choice := m.overlay.confirmIndex
		m.overlay.close()
		switch choice {
		case 0: // save and quit
			m.saveToDisk()
			if m.status.isError {
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case 1: // quit without saving
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}
	return m, nil
}

// acceptableNumeric filters a numeric prompt down to digits, a leading sign and
// the editing keys, so a stray letter cannot become part of an amount.
func acceptableNumeric(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	default:
		return true
	}
}

func (m Model) viewOverlay() string {
	switch m.overlay.kind {
	case overlayConfirm:
		return m.viewConfirm()
	case overlaySearch:
		return m.viewSearch()
	default:
		return m.viewPrompt()
	}
}

func (m Model) viewPrompt() string {
	body := lipgloss.JoinVertical(lipgloss.Left,
		styleModalTitle.Render(m.overlay.title),
		"",
		m.overlay.input.View(),
		"",
		styleHint.Render(m.overlay.hint),
	)
	return styleModal.Width(m.modalWidth(52)).Render(body)
}

func (m Model) viewSearch() string {
	body := lipgloss.JoinVertical(lipgloss.Left,
		styleModalTitle.Render(m.overlay.title),
		"",
		m.overlay.input.View(),
		"",
		m.overlay.searchGrid.view(),
		"",
		styleHint.Render(m.overlay.hint),
	)
	return styleModal.Width(m.modalWidth(62)).Render(body)
}

func (m Model) viewConfirm() string {
	options := make([]string, 0, len(m.overlay.confirmOptions))
	for i, opt := range m.overlay.confirmOptions {
		style := styleNavItem
		if i == m.overlay.confirmIndex {
			style = styleNavActive
		}
		options = append(options, style.Render(opt))
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		styleModalTitle.Render(m.overlay.title),
		"",
		m.overlay.confirmBody,
		"",
		lipgloss.JoinHorizontal(lipgloss.Top, options...),
		"",
		styleHint.Render("←/→ · enter"),
	)
	return styleModal.Width(m.modalWidth(58)).Render(body)
}

func (m Model) modalWidth(preferred int) int {
	if m.width > 0 && preferred > m.width-6 {
		return m.width - 6
	}
	return preferred
}
