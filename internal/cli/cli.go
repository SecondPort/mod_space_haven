// Package cli is the non-interactive face of the editor: the same operations
// the TUI drives, exposed as commands that can be scripted or diffed.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/SecondPort/mod_space_haven/internal/catalog"
	"github.com/SecondPort/mod_space_haven/internal/library"
	"github.com/SecondPort/mod_space_haven/internal/savegame"
)

// Config is what the launcher parsed before handing over.
type Config struct {
	Language catalog.Language
	SavesDir string
	SavePath string
	Out      io.Writer
	Err      io.Writer
}

// ErrUsage means the arguments did not name a command this package knows.
var ErrUsage = errors.New("unknown command")

// Usage is the command reference printed for --help and on a bad command.
const Usage = `Commands:
  list                          list the saves the editor can see
  info                          summarize the selected save
  cargo                         print the player ship's cargo
  crew                          print the crew with stats and skills
  research                      print research progress
  set credits <amount>          set the player's bank balance
  set cargo <id> <amount>       set a resource's total in cargo
  set research <id|all> <done|pending>
                                complete or reset research

Without a command the editor opens its terminal interface.`

// Run executes one command and returns a process exit code.
func Run(cfg Config, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(cfg.Err, Usage)
		return 2
	}

	cat, err := catalog.Embedded()
	if err != nil {
		fmt.Fprintln(cfg.Err, err)
		return 1
	}

	if err := dispatch(cfg, cat, args); err != nil {
		if errors.Is(err, ErrUsage) {
			fmt.Fprintf(cfg.Err, "%v\n\n%s\n", err, Usage)
			return 2
		}
		fmt.Fprintln(cfg.Err, err)
		return 1
	}
	return 0
}

func dispatch(cfg Config, cat *catalog.Catalog, args []string) error {
	switch args[0] {
	case "list":
		return listSaves(cfg)
	case "info":
		return withSave(cfg, func(s *savegame.Save, path string) error {
			return showInfo(cfg, cat, s, path)
		})
	case "cargo":
		return withSave(cfg, func(s *savegame.Save, _ string) error {
			return showCargo(cfg, cat, s)
		})
	case "crew":
		return withSave(cfg, func(s *savegame.Save, _ string) error {
			return showCrew(cfg, cat, s)
		})
	case "research":
		return withSave(cfg, func(s *savegame.Save, _ string) error {
			return showResearch(cfg, cat, s)
		})
	case "set":
		return runSet(cfg, cat, args[1:])
	default:
		return fmt.Errorf("%w: %q", ErrUsage, args[0])
	}
}

// resolveSave picks the save to work on: the explicit one, or the most recently
// written slot in the savegames folder.
func resolveSave(cfg Config) (string, error) {
	if cfg.SavePath != "" {
		return cfg.SavePath, nil
	}

	dir := cfg.SavesDir
	if dir == "" {
		detected, ok := library.Detect()
		if !ok {
			return "", fmt.Errorf("no savegames folder found; set %s or pass --dir", library.EnvOverride)
		}
		dir = detected
	}

	slots, err := library.List(dir)
	if err != nil {
		return "", err
	}
	if len(slots) == 0 {
		return "", fmt.Errorf("no saves in %s", dir)
	}
	return slots[0].Path, nil
}

func withSave(cfg Config, fn func(*savegame.Save, string) error) error {
	path, err := resolveSave(cfg)
	if err != nil {
		return err
	}
	save, err := library.Load(path)
	if err != nil {
		return err
	}
	return fn(save, path)
}

// withWritableSave runs an edit and writes the result back with a backup.
func withWritableSave(cfg Config, fn func(*savegame.Save) (string, error)) error {
	path, err := resolveSave(cfg)
	if err != nil {
		return err
	}
	save, err := library.Load(path)
	if err != nil {
		return err
	}

	summary, err := fn(save)
	if err != nil {
		return err
	}
	if !save.Dirty() {
		fmt.Fprintln(cfg.Out, "nothing changed")
		return nil
	}

	backup, err := library.Store(path, save)
	if err != nil {
		return err
	}
	fmt.Fprintf(cfg.Out, "%s\n%s\nbackup: %s\n", path, summary, backup)
	return nil
}

func listSaves(cfg Config) error {
	dir := cfg.SavesDir
	if dir == "" {
		detected, ok := library.Detect()
		if !ok {
			fmt.Fprintf(cfg.Err, "No savegames folder found. Looked in:\n")
			for _, candidate := range library.CandidateDirs() {
				fmt.Fprintf(cfg.Err, "  %s\n", candidate)
			}
			return fmt.Errorf("set %s to the right folder", library.EnvOverride)
		}
		dir = detected
	}

	slots, err := library.List(dir)
	if err != nil {
		return err
	}

	fmt.Fprintln(cfg.Out, dir)
	w := table(cfg.Out)
	fmt.Fprintln(w, "SLOT\tSHIP\tDATE\tPATH")
	for _, s := range slots {
		ship := s.ShipName
		if ship == "" {
			ship = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, ship, s.ModifiedLabel(), s.Path)
	}
	return w.Flush()
}

func showInfo(cfg Config, cat *catalog.Catalog, save *savegame.Save, path string) error {
	ship, err := save.PlayerShip()
	if err != nil {
		return err
	}
	cargo, err := save.Cargo()
	if err != nil {
		return err
	}
	crew, err := save.Characters()
	if err != nil {
		return err
	}
	research, err := save.Research()
	if err != nil {
		return err
	}

	done := 0
	for _, isDone := range research {
		if isDone {
			done++
		}
	}

	w := table(cfg.Out)
	fmt.Fprintf(w, "save\t%s\n", path)
	fmt.Fprintf(w, "ship\t%s (sid %s)\n", ship.Name, ship.SID)
	fmt.Fprintf(w, "credits\t%d\n", save.Credits())
	fmt.Fprintf(w, "cargo\t%d resource types\n", len(cargo))
	fmt.Fprintf(w, "crew\t%d\n", len(crew))
	fmt.Fprintf(w, "research\t%d/%d done\n", done, len(research))
	fmt.Fprintf(w, "size\t%d bytes\n", save.Size())
	return w.Flush()
}

func showCargo(cfg Config, cat *catalog.Catalog, save *savegame.Save) error {
	cargo, err := save.Cargo()
	if err != nil {
		return err
	}

	ids := make([]int, 0, len(cargo))
	for id := range cargo {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return cat.ResourceName(ids[i], cfg.Language) < cat.ResourceName(ids[j], cfg.Language)
	})

	w := table(cfg.Out)
	fmt.Fprintln(w, "ID\tRESOURCE\tAMOUNT")
	for _, id := range ids {
		fmt.Fprintf(w, "%d\t%s\t%d\n", id, cat.ResourceName(id, cfg.Language), cargo[id])
	}
	return w.Flush()
}

func showCrew(cfg Config, cat *catalog.Catalog, save *savegame.Save) error {
	crew, err := save.Characters()
	if err != nil {
		return err
	}

	w := table(cfg.Out)
	fmt.Fprintln(w, "ENTID\tNAME\tHEALTH\tMOOD\tREST\tSKILLS")
	for _, c := range crew {
		stats, err := save.CharacterStats(c.EntID)
		if err != nil {
			return err
		}
		skills, err := save.CharacterSkills(c.EntID)
		if err != nil {
			return err
		}

		var parts []string
		for _, id := range savegame.SortedSkillIDs(skills) {
			if skills[id].Level > 0 {
				parts = append(parts, fmt.Sprintf("%s:%d", cat.Skill(id), skills[id].Level))
			}
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%s\n",
			c.EntID, c.FullName(), stats.Health, stats.Mood, stats.Rest, strings.Join(parts, " "))
	}
	return w.Flush()
}

func showResearch(cfg Config, cat *catalog.Catalog, save *savegame.Save) error {
	research, err := save.Research()
	if err != nil {
		return err
	}

	ids := make([]int, 0, len(research))
	for id := range research {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	w := table(cfg.Out)
	fmt.Fprintln(w, "ID\tTECHNOLOGY\tSTATUS")
	for _, id := range ids {
		status := "pending"
		if research[id] {
			status = "done"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\n", id, cat.TechName(id), status)
	}
	return w.Flush()
}

func runSet(cfg Config, cat *catalog.Catalog, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: set needs a target", ErrUsage)
	}

	switch args[0] {
	case "credits":
		if len(args) != 2 {
			return fmt.Errorf("%w: set credits <amount>", ErrUsage)
		}
		amount, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("credits: %q is not a number", args[1])
		}
		return withWritableSave(cfg, func(s *savegame.Save) (string, error) {
			before := s.Credits()
			if err := s.SetCredits(amount); err != nil {
				return "", err
			}
			return fmt.Sprintf("credits: %d -> %d", before, amount), nil
		})

	case "cargo":
		if len(args) != 3 {
			return fmt.Errorf("%w: set cargo <id> <amount>", ErrUsage)
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("cargo: %q is not a resource id", args[1])
		}
		amount, err := strconv.Atoi(args[2])
		if err != nil {
			return fmt.Errorf("cargo: %q is not a number", args[2])
		}
		return withWritableSave(cfg, func(s *savegame.Save) (string, error) {
			cargo, err := s.Cargo()
			if err != nil {
				return "", err
			}
			before := cargo[id]
			inserted, err := s.SetCargo(id, amount)
			if err != nil {
				return "", err
			}
			verb := "updated"
			if inserted {
				verb = "inserted"
			}
			return fmt.Sprintf("%s (%d) %s: %d -> %d",
				cat.ResourceName(id, cfg.Language), id, verb, before, amount), nil
		})

	case "research":
		if len(args) != 3 {
			return fmt.Errorf("%w: set research <id|all> <done|pending>", ErrUsage)
		}
		done, err := parseDone(args[2])
		if err != nil {
			return err
		}
		return setResearch(cfg, cat, args[1], done)

	default:
		return fmt.Errorf("%w: set %s", ErrUsage, args[0])
	}
}

func setResearch(cfg Config, cat *catalog.Catalog, target string, done bool) error {
	return withWritableSave(cfg, func(s *savegame.Save) (string, error) {
		research, err := s.Research()
		if err != nil {
			return "", err
		}

		if target != "all" {
			id, err := strconv.Atoi(target)
			if err != nil {
				return "", fmt.Errorf("research: %q is not a technology id", target)
			}
			if err := s.SetTechDone(id, done, cat.TechLabPoints(id)); err != nil {
				return "", err
			}
			return fmt.Sprintf("%s (%d): %s", cat.TechName(id), id, doneLabel(done)), nil
		}

		changed := 0
		ids := make([]int, 0, len(research))
		for id := range research {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		for _, id := range ids {
			if research[id] == done {
				continue
			}
			if err := s.SetTechDone(id, done, cat.TechLabPoints(id)); err != nil {
				return "", err
			}
			changed++
		}
		return fmt.Sprintf("%d technologies set to %s", changed, doneLabel(done)), nil
	})
}

func parseDone(value string) (bool, error) {
	switch strings.ToLower(value) {
	case "done", "true", "1", "complete", "completed":
		return true, nil
	case "pending", "false", "0", "reset", "incomplete":
		return false, nil
	default:
		return false, fmt.Errorf("%w: expected done or pending, got %q", ErrUsage, value)
	}
}

func doneLabel(done bool) string {
	if done {
		return "done"
	}
	return "pending"
}

func table(out io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
}

// Flags registers the shared options on a flag set.
func Flags(fs *flag.FlagSet, lang *string, dir *string, save *string) {
	fs.StringVar(lang, "lang", "es", "interface language: es or en")
	fs.StringVar(dir, "dir", "", "savegames folder (overrides detection)")
	fs.StringVar(save, "save", "", "path to a save file, e.g. .../slot1/save/game")
}
