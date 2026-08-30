package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SecondPort/mod_space_haven/internal/catalog"
)

// savesDir lays out a savegames folder with one slot holding the fixture.
func savesDir(t *testing.T) (root, gamePath string) {
	t.Helper()

	fixture, err := os.ReadFile("testdata/sample_save.xml")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	root = t.TempDir()
	saveDir := filepath.Join(root, "slot1", "save")
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		t.Fatalf("creating slot: %v", err)
	}
	gamePath = filepath.Join(saveDir, "game")
	if err := os.WriteFile(gamePath, fixture, 0o644); err != nil {
		t.Fatalf("writing save: %v", err)
	}
	return root, gamePath
}

func run(t *testing.T, cfg Config, args ...string) (stdout, stderr string, code int) {
	t.Helper()

	var out, errOut bytes.Buffer
	cfg.Out, cfg.Err = &out, &errOut
	if cfg.Language == "" {
		cfg.Language = catalog.Spanish
	}
	code = Run(cfg, args)
	return out.String(), errOut.String(), code
}

func TestNoCommandPrintsUsage(t *testing.T) {
	_, stderr, code := run(t, Config{})

	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "Commands:") {
		t.Errorf("stderr = %q, want the usage text", stderr)
	}
}

func TestUnknownCommandIsAUsageError(t *testing.T) {
	_, stderr, code := run(t, Config{}, "frobnicate")

	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "frobnicate") {
		t.Errorf("stderr = %q, want it to name the unknown command", stderr)
	}
}

func TestListShowsTheSlot(t *testing.T) {
	root, _ := savesDir(t)

	stdout, stderr, code := run(t, Config{SavesDir: root}, "list")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "slot1") {
		t.Errorf("stdout = %q, want the slot name", stdout)
	}
	if !strings.Contains(stdout, "Nostromo") {
		t.Errorf("stdout = %q, want the ship name", stdout)
	}
}

func TestInfoSummarizesTheSave(t *testing.T) {
	_, path := savesDir(t)

	stdout, stderr, code := run(t, Config{SavePath: path}, "info")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	for _, want := range []string{"Nostromo", "12500", "crew", "research"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestCargoListsResourcesByName(t *testing.T) {
	_, path := savesDir(t)

	stdout, _, code := run(t, Config{SavePath: path, Language: catalog.English}, "cargo")
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stdout, "Base Metals") {
		t.Errorf("stdout = %q, want the English resource name", stdout)
	}
	if !strings.Contains(stdout, "150") {
		t.Errorf("stdout = %q, want the cargo total", stdout)
	}
}

func TestCrewListsThePlayerCrew(t *testing.T) {
	_, path := savesDir(t)

	stdout, _, code := run(t, Config{SavePath: path}, "crew")
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stdout, "Ellen Ripley") {
		t.Errorf("stdout = %q, want the crew member", stdout)
	}
	if strings.Contains(stdout, "Xeno") {
		t.Error("the enemy character should not be listed as crew")
	}
}

func TestResearchListsProgress(t *testing.T) {
	_, path := savesDir(t)

	stdout, _, code := run(t, Config{SavePath: path}, "research")
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stdout, "Scanner") {
		t.Errorf("stdout = %q, want a technology name", stdout)
	}
	if !strings.Contains(stdout, "pending") || !strings.Contains(stdout, "done") {
		t.Errorf("stdout = %q, want both statuses", stdout)
	}
}

func TestSetCreditsWritesAndBacksUp(t *testing.T) {
	root, path := savesDir(t)

	stdout, stderr, code := run(t, Config{SavePath: path}, "set", "credits", "77777")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "12500 -> 77777") {
		t.Errorf("stdout = %q, want the before and after", stdout)
	}
	if !strings.Contains(stdout, "backup:") {
		t.Errorf("stdout = %q, want the backup name", stdout)
	}

	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(written), `ca="77777"`) {
		t.Error("the edit did not reach the file")
	}

	entries, _ := os.ReadDir(filepath.Join(root, "slot1", "save"))
	backups := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "game.bak_") {
			backups++
		}
	}
	if backups != 1 {
		t.Errorf("backups = %d, want 1", backups)
	}
}

func TestSetCreditsRejectsNonNumbers(t *testing.T) {
	_, path := savesDir(t)

	_, stderr, code := run(t, Config{SavePath: path}, "set", "credits", "lots")
	if code == 0 {
		t.Error("a non-numeric amount should fail")
	}
	if !strings.Contains(stderr, "lots") {
		t.Errorf("stderr = %q, want it to quote the bad value", stderr)
	}
}

func TestSetCargoUpdatesAndInserts(t *testing.T) {
	_, path := savesDir(t)

	stdout, stderr, code := run(t, Config{SavePath: path, Language: catalog.English}, "set", "cargo", "157", "500")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "updated") {
		t.Errorf("stdout = %q, want it to report an update", stdout)
	}

	stdout, stderr, code = run(t, Config{SavePath: path, Language: catalog.English}, "set", "cargo", "2475", "10")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "inserted") {
		t.Errorf("stdout = %q, want it to report an insertion", stdout)
	}

	stdout, _, _ = run(t, Config{SavePath: path, Language: catalog.English}, "cargo")
	if !strings.Contains(stdout, "500") || !strings.Contains(stdout, "Fertilizer") {
		t.Errorf("cargo did not reflect both edits:\n%s", stdout)
	}
}

func TestSetResearchOneAndAll(t *testing.T) {
	_, path := savesDir(t)

	if _, stderr, code := run(t, Config{SavePath: path}, "set", "research", "2532", "done"); code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	stdout, _, _ := run(t, Config{SavePath: path}, "research")
	if strings.Count(stdout, "pending") != 1 {
		t.Errorf("expected one pending technology left:\n%s", stdout)
	}

	if _, stderr, code := run(t, Config{SavePath: path}, "set", "research", "all", "done"); code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	stdout, _, _ = run(t, Config{SavePath: path}, "research")
	if strings.Contains(stdout, "pending") {
		t.Errorf("all technologies should be done:\n%s", stdout)
	}

	if _, _, code := run(t, Config{SavePath: path}, "set", "research", "all", "pending"); code != 0 {
		t.Fatal("resetting everything failed")
	}
	stdout, _, _ = run(t, Config{SavePath: path}, "research")
	if strings.Contains(stdout, "\tdone") || strings.Contains(stdout, " done") {
		t.Errorf("all technologies should be pending:\n%s", stdout)
	}
}

func TestSetResearchRejectsABadState(t *testing.T) {
	_, path := savesDir(t)

	_, stderr, code := run(t, Config{SavePath: path}, "set", "research", "2532", "maybe")
	if code != 2 {
		t.Errorf("exit code = %d, want a usage error", code)
	}
	if !strings.Contains(stderr, "maybe") {
		t.Errorf("stderr = %q, want it to quote the bad state", stderr)
	}
}

func TestSetReportsWhenNothingChanged(t *testing.T) {
	_, path := savesDir(t)

	stdout, _, code := run(t, Config{SavePath: path}, "set", "credits", "12500")
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stdout, "nothing changed") {
		t.Errorf("stdout = %q, want it to say nothing changed", stdout)
	}
	if strings.Contains(stdout, "backup:") {
		t.Error("a no-op should not write a backup")
	}
}

func TestMissingSaveIsReported(t *testing.T) {
	_, stderr, code := run(t, Config{SavesDir: t.TempDir()}, "info")

	if code == 0 {
		t.Error("an empty savegames folder should fail")
	}
	if !strings.Contains(stderr, "no saves") {
		t.Errorf("stderr = %q, want it to say there are no saves", stderr)
	}
}
