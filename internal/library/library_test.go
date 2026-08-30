package library

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const miniSave = `<?xml version="1.0"?>
<game>
	<playerBank ca="500"/>
	<settlements>
		<settlement isPlayer="false" shn="Pirate Den" createdShipId="1"/>
		<settlement isPlayer="true" shn="Nostromo" createdShipId="38"/>
	</settlements>
	<ships><ship sid="38"><e><feat><inv/></feat></e></ship></ships>
</game>`

func writeSlot(t *testing.T, root, slot, contents string, realTime int64) string {
	t.Helper()
	dir := filepath.Join(root, slot, "save")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating slot: %v", err)
	}
	gamePath := filepath.Join(dir, "game")
	if err := os.WriteFile(gamePath, []byte(contents), 0o644); err != nil {
		t.Fatalf("writing game: %v", err)
	}
	if realTime > 0 {
		info := `<info realTimeDate="` + strconv.FormatInt(realTime, 10) + `"/>`
		if err := os.WriteFile(filepath.Join(dir, "info"), []byte(info), 0o644); err != nil {
			t.Fatalf("writing info: %v", err)
		}
	}
	return gamePath
}

func TestCandidateDirsPutsTheOverrideFirst(t *testing.T) {
	t.Setenv(EnvOverride, "/tmp/some-custom-place")

	dirs := CandidateDirs()
	if len(dirs) == 0 {
		t.Fatal("no candidate directories")
	}
	if dirs[0] != "/tmp/some-custom-place" {
		t.Errorf("first candidate = %q, want the override", dirs[0])
	}
}

func TestCandidateDirsExpandsTilde(t *testing.T) {
	t.Setenv(EnvOverride, "~/saves-here")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory in this environment")
	}
	want := filepath.Join(home, "saves-here")
	if got := CandidateDirs()[0]; got != want {
		t.Errorf("expanded override = %q, want %q", got, want)
	}
}

func TestCandidateDirsHasNoDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, dir := range CandidateDirs() {
		if seen[dir] {
			t.Errorf("candidate %q appears twice", dir)
		}
		seen[dir] = true
	}
}

func TestDetectFindsTheOverriddenFolder(t *testing.T) {
	root := t.TempDir()
	t.Setenv(EnvOverride, root)

	dir, ok := Detect()
	if !ok {
		t.Fatal("Detect did not find the overridden folder")
	}
	if dir != root {
		t.Errorf("Detect = %q, want %q", dir, root)
	}
}

func TestDetectReportsAMissingFolder(t *testing.T) {
	t.Setenv(EnvOverride, filepath.Join(t.TempDir(), "not-created"))

	// The machine running the tests may still have a real Steam install, so only
	// assert that the missing override was not returned.
	if dir, ok := Detect(); ok && strings.Contains(dir, "not-created") {
		t.Errorf("Detect returned a folder that does not exist: %q", dir)
	}
}

func TestListReadsSlotsNewestFirst(t *testing.T) {
	root := t.TempDir()
	writeSlot(t, root, "slot1", miniSave, 1_600_000_000_000)
	writeSlot(t, root, "slot2", miniSave, 1_700_000_000_000)
	// A folder with no save inside must be ignored.
	if err := os.MkdirAll(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	slots, err := List(root)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(slots) != 2 {
		t.Fatalf("slots = %d, want 2", len(slots))
	}
	if slots[0].Name != "slot2" {
		t.Errorf("first slot = %q, want the newest one", slots[0].Name)
	}
	if slots[0].ShipName != "Nostromo" {
		t.Errorf("ship name = %q, want %q", slots[0].ShipName, "Nostromo")
	}
	if slots[0].ModifiedLabel() == "—" {
		t.Error("a slot with a timestamp should render a date")
	}
}

func TestListFallsBackToTheFileTimeWithoutAnInfoFile(t *testing.T) {
	root := t.TempDir()
	writeSlot(t, root, "slot1", miniSave, 0)

	slots, err := List(root)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(slots) != 1 {
		t.Fatalf("slots = %d, want 1", len(slots))
	}
	if slots[0].Modified.IsZero() {
		t.Error("expected the file modification time as a fallback")
	}
}

func TestListReportsAMissingFolder(t *testing.T) {
	if _, err := List(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("List accepted a folder that does not exist")
	}
}

func TestShipNameIsFoundAcrossAChunkBoundary(t *testing.T) {
	// Pad the document so the markers land far apart in the stream.
	padded := strings.Replace(miniSave,
		`<settlement isPlayer="true"`,
		strings.Repeat("<pad/>", 20000)+`<settlement isPlayer="true"`, 1)

	name, err := scanShipName(strings.NewReader(padded))
	if err != nil {
		t.Fatalf("scanShipName: %v", err)
	}
	if name != "Nostromo" {
		t.Errorf("ship name = %q, want %q", name, "Nostromo")
	}
}

func TestShipNameIgnoresNonPlayerSettlements(t *testing.T) {
	name, err := scanShipName(strings.NewReader(miniSave))
	if err != nil {
		t.Fatalf("scanShipName: %v", err)
	}
	if name != "Nostromo" {
		t.Errorf("ship name = %q, want the player's ship, not the pirate den", name)
	}
}

func TestShipNameOnADocumentWithoutAPlayer(t *testing.T) {
	name, err := scanShipName(strings.NewReader(`<game><settlements/></game>`))
	if err != nil {
		t.Fatalf("scanShipName: %v", err)
	}
	if name != "" {
		t.Errorf("ship name = %q, want empty", name)
	}
}

func TestLoadAndStoreRoundTrip(t *testing.T) {
	root := t.TempDir()
	path := writeSlot(t, root, "slot1", miniSave, 1_700_000_000_000)

	save, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := save.Credits(); got != 500 {
		t.Fatalf("credits = %d, want 500", got)
	}
	if err := save.SetCredits(4242); err != nil {
		t.Fatalf("SetCredits: %v", err)
	}

	backup, err := Store(path, save)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	if !strings.HasPrefix(backup, BackupPrefix) {
		t.Errorf("backup name = %q, want the %q prefix", backup, BackupPrefix)
	}
	if save.Dirty() {
		t.Error("the save should be clean after a successful store")
	}

	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if !strings.Contains(string(written), `ca="4242"`) {
		t.Error("the edit did not reach the file")
	}

	original, err := os.ReadFile(filepath.Join(root, "slot1", "save", backup))
	if err != nil {
		t.Fatalf("reading the backup: %v", err)
	}
	if string(original) != miniSave {
		t.Error("the backup does not hold the original save")
	}
}

func TestStoreLeavesNoTemporaryFilesBehind(t *testing.T) {
	root := t.TempDir()
	path := writeSlot(t, root, "slot1", miniSave, 0)

	save, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := save.SetCredits(1); err != nil {
		t.Fatalf("SetCredits: %v", err)
	}
	if _, err := Store(path, save); err != nil {
		t.Fatalf("Store: %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".modhaven-") {
			t.Errorf("a temporary file survived: %q", e.Name())
		}
	}
}

func TestStoreReportsAMissingOriginal(t *testing.T) {
	save, err := Load(writeSlot(t, t.TempDir(), "slot1", miniSave, 0))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := Store(filepath.Join(t.TempDir(), "gone", "game"), save); err == nil {
		t.Error("Store accepted a path that does not exist")
	}
}

func TestLoadReportsACorruptSave(t *testing.T) {
	root := t.TempDir()
	path := writeSlot(t, root, "slot1", "not xml at all", 0)

	if _, err := Load(path); err == nil {
		t.Error("Load accepted a file with no XML in it")
	}
}
