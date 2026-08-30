// Package library is the filesystem side of the editor: it finds the game's
// savegames folder, lists the slots inside it, and reads and writes save files
// with a backup. Keeping this apart from internal/savegame means the editing
// rules can be tested without ever touching a disk.
package library

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

// EnvOverride names the environment variable that pins the savegames folder
// when the game is installed somewhere unusual.
const EnvOverride = "SPACEHAVEN_SAVEGAMES_DIR"

// steamRelativePath is where Steam puts the game's saves under any library root.
var steamRelativePath = filepath.Join("steamapps", "common", "SpaceHaven", "savegames")

// CandidateDirs lists the folders the editor looks in, in priority order. The
// environment override always comes first.
func CandidateDirs() []string {
	var candidates []string

	if override := os.Getenv(EnvOverride); override != "" {
		if expanded, err := expandHome(override); err == nil {
			candidates = append(candidates, expanded)
		}
	}

	home, err := os.UserHomeDir()
	if err == nil {
		for _, root := range steamRoots(home) {
			candidates = append(candidates, filepath.Join(root, steamRelativePath))
		}
	}
	candidates = append(candidates, mountedDriveCandidates()...)

	return dedupe(candidates)
}

// steamRoots returns the Steam library roots worth trying on this platform.
func steamRoots(home string) []string {
	switch runtime.GOOS {
	case "windows":
		return []string{
			filepath.Join(`C:\`, "Program Files (x86)", "Steam"),
			filepath.Join(`C:\`, "Program Files", "Steam"),
			filepath.Join(home, "Steam"),
		}
	case "darwin":
		return []string{
			filepath.Join(home, "Library", "Application Support", "Steam"),
		}
	default:
		return []string{
			filepath.Join(home, ".local", "share", "Steam"),
			filepath.Join(home, ".steam", "steam"),
			filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam"),
		}
	}
}

// mountedDriveCandidates covers a second Steam library on another drive, which
// is where large game installs usually end up.
func mountedDriveCandidates() []string {
	var roots []string

	switch runtime.GOOS {
	case "windows":
		for _, drive := range []string{"D:", "E:", "F:"} {
			roots = append(roots, filepath.Join(drive+`\`, "SteamLibrary"), filepath.Join(drive+`\`, "Steam"))
		}
	case "darwin":
		// Steam libraries on external volumes.
		entries, err := os.ReadDir("/Volumes")
		if err != nil {
			return nil
		}
		for _, e := range entries {
			roots = append(roots, filepath.Join("/Volumes", e.Name(), "Steam"))
		}
	default:
		for _, mount := range []string{"/mnt", "/media", "/run/media"} {
			entries, err := os.ReadDir(mount)
			if err != nil {
				continue
			}
			for _, e := range entries {
				roots = append(roots, filepath.Join(mount, e.Name(), "Steam"), filepath.Join(mount, e.Name(), "SteamLibrary"))
			}
		}
	}

	sort.Strings(roots)
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		out = append(out, filepath.Join(root, steamRelativePath))
	}
	return out
}

// Detect returns the first candidate folder that exists.
func Detect() (string, bool) {
	for _, dir := range CandidateDirs() {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir, true
		}
	}
	return "", false
}

func expandHome(path string) (string, error) {
	if path == "~" || len(path) > 1 && path[0] == '~' && (path[1] == '/' || path[1] == filepath.Separator) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func dedupe(paths []string) []string {
	seen := make(map[string]bool, len(paths))
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}
