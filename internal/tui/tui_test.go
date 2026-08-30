package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/SecondPort/mod_space_haven/internal/catalog"
)

func fixtureSave(t *testing.T) string {
	t.Helper()

	raw, err := os.ReadFile("testdata/sample_save.xml")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	dir := filepath.Join(t.TempDir(), "slot1", "save")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating slot: %v", err)
	}
	path := filepath.Join(dir, "game")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("writing save: %v", err)
	}
	return path
}

func newTestModel(t *testing.T) Model {
	t.Helper()

	cat, err := catalog.Embedded()
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	m := New(cat, Options{Language: catalog.Spanish, SavePath: fixtureSave(t)})
	if m.stage != stageEditor {
		t.Fatalf("model did not reach the editor: %s", m.status.text)
	}
	return send(m, tea.WindowSizeMsg{Width: 120, Height: 40})
}

func send(m Model, msgs ...tea.Msg) Model {
	for _, msg := range msgs {
		updated, _ := m.Update(msg)
		m = updated.(Model)
	}
	return m
}

func runes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func special(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }

func TestHumanIntGroupsByLanguage(t *testing.T) {
	cases := []struct {
		n      int
		lang   catalog.Language
		expect string
	}{
		{0, catalog.Spanish, "0"},
		{999, catalog.Spanish, "999"},
		{1000, catalog.Spanish, "1.000"},
		{1234567, catalog.Spanish, "1.234.567"},
		{1234567, catalog.English, "1,234,567"},
		{-4500, catalog.Spanish, "-4.500"},
	}
	for _, c := range cases {
		if got := humanInt(c.n, c.lang); got != c.expect {
			t.Errorf("humanInt(%d, %s) = %q, want %q", c.n, c.lang, got, c.expect)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("short string was changed: %q", got)
	}
	if got := truncate("hello world", 8); len([]rune(got)) > 8 || !strings.HasSuffix(got, "…") {
		t.Errorf("truncate = %q, want it clipped with an ellipsis", got)
	}
	if got := truncate("hello", 0); got != "" {
		t.Errorf("truncate to zero width = %q, want empty", got)
	}
}

func TestGridSkipsHeadingRows(t *testing.T) {
	g := newGrid([]table.Column{{Title: "a", Width: 10}}, 5)
	g.setRows([]table.Row{
		{heading("GROUP")}, {"one"}, {"two"}, {heading("OTHER")}, {"three"},
	}, []int{headingID, 1, 2, headingID, 3})

	if id, ok := g.selected(); !ok || id != 1 {
		t.Fatalf("initial selection = %d (%v), want the first data row", id, ok)
	}

	g.move(1)
	if id, _ := g.selected(); id != 2 {
		t.Errorf("after one step = %d, want 2", id)
	}

	g.move(1) // must jump over the second heading
	if id, _ := g.selected(); id != 3 {
		t.Errorf("after the heading = %d, want 3", id)
	}

	g.move(1) // already at the end
	if id, _ := g.selected(); id != 3 {
		t.Errorf("past the end = %d, want to stay on 3", id)
	}

	g.move(-1)
	if id, _ := g.selected(); id != 2 {
		t.Errorf("stepping back = %d, want 2", id)
	}
}

func TestGridOnAnEmptyTable(t *testing.T) {
	g := newGrid([]table.Column{{Title: "a", Width: 10}}, 5)
	g.setRows(nil, nil)

	if _, ok := g.selected(); ok {
		t.Error("an empty grid reported a selection")
	}
	g.move(1) // must not panic
}

func TestOpeningASaveFillsThePanels(t *testing.T) {
	m := newTestModel(t)

	if m.editor.shipName != "Nostromo" {
		t.Errorf("ship = %q, want Nostromo", m.editor.shipName)
	}
	if m.editor.credits != 12500 {
		t.Errorf("credits = %d, want 12500", m.editor.credits)
	}
	if len(m.editor.crew) != 2 {
		t.Errorf("crew = %d, want 2", len(m.editor.crew))
	}
	if m.editor.researchTotal != 3 || m.editor.researchDone != 1 {
		t.Errorf("research = %d/%d, want 1/3", m.editor.researchDone, m.editor.researchTotal)
	}
	if !strings.Contains(m.View(), "Nostromo") {
		t.Error("the header does not show the ship name")
	}
}

func TestOpeningAMissingSaveReportsTheError(t *testing.T) {
	cat, err := catalog.Embedded()
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	m := New(cat, Options{SavePath: filepath.Join(t.TempDir(), "nope")})

	if m.stage != stagePicker {
		t.Error("a failed open should not enter the editor")
	}
	if m.status.text == "" || !m.status.isError {
		t.Error("a failed open should show an error in the status line")
	}
}

func TestCreditsPromptWritesThroughToTheSave(t *testing.T) {
	m := newTestModel(t)

	m = send(m, runes("c"))
	if !m.overlay.active() {
		t.Fatal("pressing c did not open the credits prompt")
	}

	// The prompt starts on the current value; appending a zero multiplies by ten.
	m = send(m, runes("0"), special(tea.KeyEnter))

	if m.editor.credits != 125000 {
		t.Errorf("credits = %d, want 125000", m.editor.credits)
	}
	if !m.editor.save.Dirty() {
		t.Error("the save should be dirty after an edit")
	}
	if !strings.Contains(m.View(), m.txt.Unsaved) {
		t.Error("the header should flag unsaved work")
	}
}

func TestNumericPromptRejectsLetters(t *testing.T) {
	m := newTestModel(t)

	m = send(m, runes("c"), runes("x"), special(tea.KeyEnter))

	if m.editor.credits != 12500 {
		t.Errorf("credits = %d, want them untouched", m.editor.credits)
	}
}

func TestEscapeClosesAPromptWithoutEditing(t *testing.T) {
	m := newTestModel(t)

	m = send(m, runes("c"), runes("9"), special(tea.KeyEsc))

	if m.overlay.active() {
		t.Error("escape should close the prompt")
	}
	if m.editor.credits != 12500 {
		t.Errorf("credits = %d, want them untouched", m.editor.credits)
	}
}

func TestSectionNavigation(t *testing.T) {
	m := newTestModel(t)

	m = send(m, special(tea.KeyTab))
	if m.editor.section != sectionWeapons {
		t.Errorf("section = %d, want weapons", m.editor.section)
	}

	m = send(m, runes("4"))
	if m.editor.section != sectionResearch {
		t.Errorf("section = %d, want research", m.editor.section)
	}

	m = send(m, special(tea.KeyTab))
	if m.editor.section != sectionCargo {
		t.Errorf("section wrapped to %d, want cargo", m.editor.section)
	}
}

func TestCargoRowEditsTheResource(t *testing.T) {
	m := newTestModel(t)

	id, ok := m.editor.cargoGrid.selected()
	if !ok {
		t.Fatal("no selectable cargo row")
	}
	before := m.editor.cargo[id]

	m = send(m, special(tea.KeyEnter), runes("0"), special(tea.KeyEnter))

	if got := m.editor.cargo[id]; got != before*10 {
		t.Errorf("resource %d = %d, want %d", id, got, before*10)
	}
}

func TestAddResourceSearchInsertsIntoCargo(t *testing.T) {
	m := newTestModel(t)

	m = send(m, runes("a"))
	if m.overlay.kind != overlaySearch {
		t.Fatal("a did not open the resource search")
	}

	m = send(m, runes("2475")) // fertilizer, not in the fixture's cargo
	if len(m.overlay.searchMatches) == 0 {
		t.Fatal("the search found nothing for a known id")
	}

	m = send(m, special(tea.KeyEnter))
	if m.overlay.kind != overlayPrompt {
		t.Fatal("picking a resource should ask for an amount")
	}

	m = send(m, runes("7"), special(tea.KeyEnter))
	if got := m.editor.cargo[2475]; got != 7 {
		t.Errorf("fertilizer = %d, want 7", got)
	}
}

func TestResearchToggleAndBulkActions(t *testing.T) {
	m := newTestModel(t)
	m = send(m, runes("4"))

	m = send(m, special(tea.KeyEnter))
	if m.editor.researchDone != 2 {
		t.Errorf("done = %d, want 2 after completing one", m.editor.researchDone)
	}

	m = send(m, runes("A"))
	if m.editor.researchDone != m.editor.researchTotal {
		t.Errorf("done = %d, want all %d", m.editor.researchDone, m.editor.researchTotal)
	}

	m = send(m, runes("R"))
	if m.editor.researchDone != 0 {
		t.Errorf("done = %d, want 0 after a reset", m.editor.researchDone)
	}
}

func TestCrewDetailOpensAndReturns(t *testing.T) {
	m := newTestModel(t)
	m = send(m, runes("3"), special(tea.KeyEnter))

	if m.stage != stageDetail {
		t.Fatalf("stage = %d, want the crew detail", m.stage)
	}
	if !strings.Contains(m.detail.name, "Ripley") {
		t.Errorf("detail is showing %q", m.detail.name)
	}
	if !strings.Contains(m.View(), "Ripley") {
		t.Error("the detail view does not render the crew member")
	}

	m = send(m, special(tea.KeyEsc))
	if m.stage != stageEditor {
		t.Error("escape should return to the editor")
	}
}

func TestCrewDetailEditsASkill(t *testing.T) {
	m := newTestModel(t)
	m = send(m, runes("3"), special(tea.KeyEnter))

	id, ok := m.detail.skills.selected()
	if !ok {
		t.Fatal("no skill row to edit")
	}

	m = send(m, special(tea.KeyEnter), runes("9"), special(tea.KeyEnter))

	skills, err := m.editor.save.CharacterSkills(m.detail.entID)
	if err != nil {
		t.Fatalf("CharacterSkills: %v", err)
	}
	// The prompt opened on level 3, appending 9 gives 39, clamped to the cap.
	if skills[id].Level != maxSkill {
		t.Errorf("skill %d level = %d, want it clamped to %d", id, skills[id].Level, maxSkill)
	}
	if skills[id].Max < skills[id].Level {
		t.Errorf("skill ceiling %d is below the level %d", skills[id].Max, skills[id].Level)
	}
}

func TestUnknownTraitsStillAppearOnTheSheet(t *testing.T) {
	m := newTestModel(t)

	// 4242 is in no catalog table; the sheet must still show it as owned.
	if err := m.editor.save.AddCharacterTrait("1001", 4242); err != nil {
		t.Fatalf("AddCharacterTrait: %v", err)
	}
	m = send(m, runes("3"), special(tea.KeyEnter))

	if !containsID(m.detail.traits.ids, 4242) {
		t.Error("a trait the catalog does not know was dropped from the sheet")
	}
}

func TestCrewDetailTogglesATrait(t *testing.T) {
	m := newTestModel(t)
	m = send(m, runes("3"), special(tea.KeyEnter), special(tea.KeyTab), special(tea.KeyTab))

	if m.detail.pane != paneTraits {
		t.Fatalf("pane = %d, want traits", m.detail.pane)
	}

	id, ok := m.detail.traits.selected()
	if !ok {
		t.Fatal("no trait row")
	}
	before, err := m.editor.save.CharacterTraits(m.detail.entID)
	if err != nil {
		t.Fatalf("CharacterTraits: %v", err)
	}
	had := containsID(before, id)

	m = send(m, special(tea.KeyEnter))

	after, err := m.editor.save.CharacterTraits(m.detail.entID)
	if err != nil {
		t.Fatalf("CharacterTraits: %v", err)
	}
	if containsID(after, id) == had {
		t.Errorf("trait %d did not toggle", id)
	}
}

func TestRenameAsksForBothNames(t *testing.T) {
	m := newTestModel(t)
	m = send(m, runes("3"), special(tea.KeyEnter), runes("n"))

	if m.overlay.kind != overlayPrompt {
		t.Fatal("n did not open the rename prompt")
	}

	m = send(m, special(tea.KeyEnter)) // keep the first name
	if m.overlay.kind != overlayPrompt || !m.overlay.act.askingLast {
		t.Fatal("the rename did not move on to the last name")
	}

	m = send(m, runes("son"), special(tea.KeyEnter))

	character, err := m.editor.save.Character(m.detail.entID)
	if err != nil {
		t.Fatalf("Character: %v", err)
	}
	if character.LastName != "Ripleyson" {
		t.Errorf("last name = %q, want %q", character.LastName, "Ripleyson")
	}
}

func TestQuittingWithUnsavedWorkAsksFirst(t *testing.T) {
	m := newTestModel(t)
	m = send(m, runes("c"), runes("0"), special(tea.KeyEnter))

	m = send(m, runes("q"))
	if m.overlay.kind != overlayConfirm {
		t.Fatal("quitting with unsaved work should ask")
	}
	if m.quitting {
		t.Error("the program should not be quitting while the question is open")
	}

	// Cancel.
	m = send(m, special(tea.KeyEsc))
	if m.overlay.active() || m.quitting {
		t.Error("escape should dismiss the question and stay in the editor")
	}
}

func TestSavingWritesTheFileAndClearsTheFlag(t *testing.T) {
	m := newTestModel(t)
	m = send(m, runes("c"), runes("0"), special(tea.KeyEnter), runes("s"))

	if m.editor.save.Dirty() {
		t.Error("the save should be clean after writing")
	}
	if !strings.Contains(m.status.text, "backup") && !strings.Contains(m.status.text, "Backup") {
		t.Errorf("status = %q, want the backup name", m.status.text)
	}

	written, err := os.ReadFile(m.editor.path)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if !strings.Contains(string(written), `ca="125000"`) {
		t.Error("the edit did not reach the file")
	}
}

func TestSavingWithNothingToDoSaysSo(t *testing.T) {
	m := newTestModel(t)
	m = send(m, runes("s"))

	if m.status.text != m.txt.StatusNoChanges {
		t.Errorf("status = %q, want %q", m.status.text, m.txt.StatusNoChanges)
	}
}

func TestViewSurvivesSmallTerminals(t *testing.T) {
	m := newTestModel(t)

	for _, size := range []tea.WindowSizeMsg{
		{Width: 40, Height: 10},
		{Width: 20, Height: 6},
		{Width: 200, Height: 60},
	} {
		m = send(m, size)
		for _, sec := range []section{sectionCargo, sectionWeapons, sectionCrew, sectionResearch} {
			m.editor.section = sec
			if m.View() == "" {
				t.Errorf("empty view at %dx%d in section %d", size.Width, size.Height, sec)
			}
		}
	}
}

func TestPickerReportsAMissingFolder(t *testing.T) {
	cat, err := catalog.Embedded()
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	m := New(cat, Options{SavesDir: filepath.Join(t.TempDir(), "gone")})
	m = send(m, tea.WindowSizeMsg{Width: 100, Height: 30})

	if view := m.View(); view == "" {
		t.Fatal("the picker rendered nothing")
	}
	// Selecting nothing must not panic.
	m = send(m, special(tea.KeyEnter))
	if m.stage != stagePicker {
		t.Error("an empty picker should stay put")
	}
}

func TestPickerListsAndOpensASave(t *testing.T) {
	path := fixtureSave(t)
	root := filepath.Dir(filepath.Dir(filepath.Dir(path)))

	cat, err := catalog.Embedded()
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	m := New(cat, Options{SavesDir: root})
	m = send(m, tea.WindowSizeMsg{Width: 100, Height: 30})

	if len(m.picker.slots) != 1 {
		t.Fatalf("picker listed %d saves, want 1", len(m.picker.slots))
	}

	m = send(m, special(tea.KeyEnter))
	if m.stage != stageEditor {
		t.Fatalf("enter did not open the save: %s", m.status.text)
	}
	if m.editor.shipName != "Nostromo" {
		t.Errorf("ship = %q", m.editor.shipName)
	}
}
