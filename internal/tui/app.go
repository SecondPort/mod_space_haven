// Package tui is the terminal interface: a Bubble Tea program over the editing
// operations in internal/savegame. It owns presentation and key handling only —
// every change it makes goes through the domain package, so the rules about
// what a save may contain live in one place.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/SecondPort/mod_space_haven/internal/catalog"
	"github.com/SecondPort/mod_space_haven/internal/library"
)

// stage is the screen the program is on.
type stage int

const (
	stagePicker stage = iota
	stageEditor
	stageDetail
)

// statusLine is the one-line message under the content area.
type statusLine struct {
	text    string
	isError bool
}

func (s *statusLine) ok(format string, args ...any) {
	s.text, s.isError = fmt.Sprintf(format, args...), false
}

func (s *statusLine) fail(err error) {
	if err == nil {
		return
	}
	s.text, s.isError = err.Error(), true
}

func (s *statusLine) clear() { s.text, s.isError = "", false }

// Model is the root Bubble Tea model.
type Model struct {
	cat  *catalog.Catalog
	lang catalog.Language
	txt  uiText

	width, height int

	stage   stage
	picker  pickerState
	editor  editorState
	detail  detailState
	overlay overlayState

	status   statusLine
	quitting bool
}

// Options configure a run of the editor.
type Options struct {
	// Language selects the interface language.
	Language catalog.Language
	// SavesDir overrides save discovery. Empty means detect.
	SavesDir string
	// SavePath opens one save directly, skipping the picker.
	SavePath string
}

// New builds the root model.
func New(cat *catalog.Catalog, opts Options) Model {
	m := Model{
		cat:    cat,
		lang:   opts.Language,
		txt:    textFor(opts.Language),
		stage:  stagePicker,
		width:  100,
		height: 30,
	}
	m.overlay = newOverlay()

	if opts.SavePath != "" {
		if err := m.openSave(opts.SavePath); err != nil {
			m.status.fail(err)
		}
		return m
	}

	dir := opts.SavesDir
	if dir == "" {
		if detected, ok := library.Detect(); ok {
			dir = detected
		}
	}
	m.picker = newPicker(dir, m.txt)
	return m
}

// Run starts the interactive editor.
func Run(cat *catalog.Catalog, opts Options) error {
	p := tea.NewProgram(New(cat, opts), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()
		return m, nil

	case tea.KeyMsg:
		if m.overlay.active() {
			return m.updateOverlay(msg)
		}
		switch m.stage {
		case stagePicker:
			return m.updatePicker(msg)
		case stageEditor:
			return m.updateEditor(msg)
		case stageDetail:
			return m.updateDetail(msg)
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var body string
	switch m.stage {
	case stagePicker:
		body = m.viewPicker()
	case stageEditor:
		body = m.viewEditor()
	case stageDetail:
		body = m.viewDetail()
	}

	if m.overlay.active() {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.viewOverlay())
	}
	return body
}

// contentHeight is the room left for a panel once the chrome is drawn.
func (m Model) contentHeight() int {
	const chrome = 7 // header, subheader, status, footer and their padding
	if h := m.height - chrome; h > 4 {
		return h
	}
	return 4
}

func (m *Model) resize() {
	m.picker.resize(m.contentHeight())
	m.editor.resize(m.width, m.contentHeight())
	m.detail.resize(m.width, m.contentHeight())
}

// openSave loads a save and builds the editor panels around it.
func (m *Model) openSave(path string) error {
	save, err := library.Load(path)
	if err != nil {
		return err
	}
	editor, err := newEditor(m.cat, m.lang, m.txt, save, path)
	if err != nil {
		return err
	}
	m.editor = editor
	m.editor.resize(m.width, m.contentHeight())
	m.stage = stageEditor
	m.status.clear()
	return nil
}

// header renders the title bar and the summary line under it.
func (m Model) header() string {
	title := styleHeader.Width(m.width).Render(m.txt.AppTitle)
	if m.stage == stagePicker {
		return title
	}

	summary := fmt.Sprintf("%s  ·  %s: %s  ·  %s: %d/%d  ·  %s: %d",
		m.editor.shipName,
		m.txt.Credits, humanInt(m.editor.credits, m.lang),
		m.txt.Research, m.editor.researchDone, m.editor.researchTotal,
		m.txt.CrewCount, len(m.editor.crew),
	)
	if m.editor.save.Dirty() {
		summary += "  " + styleDirty.Render("● "+m.txt.Unsaved)
	}
	return lipgloss.JoinVertical(lipgloss.Left, title, styleSubHeader.Width(m.width).Render(summary))
}

// footer renders the status line and the keys available right now.
func (m Model) footer(keys string) string {
	status := ""
	if m.status.text != "" {
		style := styleStatusOK
		if m.status.isError {
			style = styleStatusErr
		}
		status = style.Width(m.width).Render(truncate(m.status.text, m.width-2))
	}
	return lipgloss.JoinVertical(lipgloss.Left, status, styleFooter.Width(m.width).Render(keys))
}

// saveToDisk writes the save out, keeping a backup.
func (m *Model) saveToDisk() {
	if m.editor.save == nil {
		return
	}
	if !m.editor.save.Dirty() {
		m.status.ok("%s", m.txt.StatusNoChanges)
		return
	}
	backup, err := library.Store(m.editor.path, m.editor.save)
	if err != nil {
		m.status.fail(err)
		return
	}
	m.status.ok(m.txt.StatusSaved, backup)
}

// requestQuit leaves straight away unless there is unsaved work.
func (m Model) requestQuit() (tea.Model, tea.Cmd) {
	if m.editor.save == nil || !m.editor.save.Dirty() {
		m.quitting = true
		return m, tea.Quit
	}
	m.overlay.openConfirm(m.txt.ConfirmQuitTitle, m.txt.ConfirmQuitBody, []string{
		m.txt.ConfirmSave, m.txt.ConfirmDiscard, m.txt.ConfirmCancel,
	})
	return m, nil
}
