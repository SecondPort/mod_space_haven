package library

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/SecondPort/mod_space_haven/internal/savegame"
)

// BackupPrefix is the name every backup this editor writes starts with.
const BackupPrefix = "game.bak_"

// backupTimestampLayout keeps backups sortable by name.
const backupTimestampLayout = "20060102_150405"

// Slot is one save the game has written: <savegames>/<slot>/save/game.
type Slot struct {
	Name     string // the slot folder, e.g. "slot1"
	Path     string // full path to the game file
	ShipName string
	Modified time.Time
}

// ModifiedLabel renders the save date for a list, or a placeholder when the
// save carries no timestamp.
func (s Slot) ModifiedLabel() string {
	if s.Modified.IsZero() {
		return "—"
	}
	return s.Modified.Format("2006-01-02 15:04")
}

// List returns the saves in a savegames folder, newest first.
func List(dir string) ([]Slot, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("library: reading %s: %w", dir, err)
	}

	var slots []Slot
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		gamePath := filepath.Join(dir, entry.Name(), "save", "game")
		if _, err := os.Stat(gamePath); err != nil {
			continue
		}
		slots = append(slots, Slot{
			Name:     entry.Name(),
			Path:     gamePath,
			ShipName: readShipName(gamePath),
			Modified: readSaveTime(filepath.Join(dir, entry.Name(), "save", "info"), gamePath),
		})
	}

	sort.SliceStable(slots, func(i, j int) bool {
		return slots[i].Modified.After(slots[j].Modified)
	})
	return slots, nil
}

// Load reads a save file into memory.
func Load(path string) (*savegame.Save, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("library: reading the save: %w", err)
	}
	save, err := savegame.Parse(raw)
	if err != nil {
		return nil, err
	}
	return save, nil
}

// Store backs the file up and then replaces it, and returns the backup's name.
//
// The new contents go to a temporary file in the same folder and are renamed
// into place, so an interrupted write cannot leave a half-written save behind.
func Store(path string, save *savegame.Save) (string, error) {
	if save == nil {
		return "", errors.New("library: nothing to store")
	}

	backupPath, err := Backup(path)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".modhaven-*")
	if err != nil {
		return "", fmt.Errorf("library: preparing the write: %w", err)
	}
	tmpName := tmp.Name()

	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(save.Bytes()); err != nil {
		tmp.Close()
		cleanup()
		return "", fmt.Errorf("library: writing the save: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		cleanup()
		return "", fmt.Errorf("library: flushing the save: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", fmt.Errorf("library: closing the save: %w", err)
	}

	if info, err := os.Stat(path); err == nil {
		_ = os.Chmod(tmpName, info.Mode().Perm())
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return "", fmt.Errorf("library: replacing the save: %w", err)
	}

	save.MarkSaved()
	return filepath.Base(backupPath), nil
}

// Backup copies a save next to itself with a timestamped name.
func Backup(path string) (string, error) {
	src, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("library: opening the save to back it up: %w", err)
	}
	defer src.Close()

	backupPath := filepath.Join(filepath.Dir(path), BackupPrefix+time.Now().Format(backupTimestampLayout))
	dst, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("library: creating the backup: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("library: copying the backup: %w", err)
	}
	if err := dst.Sync(); err != nil {
		return "", fmt.Errorf("library: flushing the backup: %w", err)
	}
	return backupPath, nil
}

var realTimeDateRe = regexp.MustCompile(`realTimeDate="(\d+)"`)

// readSaveTime prefers the timestamp the game records, and falls back to the
// file's own modification time.
func readSaveTime(infoPath, gamePath string) time.Time {
	if raw, err := os.ReadFile(infoPath); err == nil {
		if m := realTimeDateRe.FindSubmatch(raw); m != nil {
			if millis, err := strconv.ParseInt(string(m[1]), 10, 64); err == nil {
				return time.UnixMilli(millis)
			}
		}
	}
	if info, err := os.Stat(gamePath); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}

// readShipName pulls the player settlement's ship name out of a save without
// loading the whole file: saves run to tens of megabytes and a slot list would
// otherwise read all of them just to draw one column.
func readShipName(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	name, err := scanShipName(f)
	if err != nil || name == "" {
		return ""
	}
	return name
}

const (
	scanChunk   = 64 * 1024
	scanOverlap = 512              // long enough to hold a marker split across chunks
	scanLimit   = 32 * 1024 * 1024 // stop rather than stream a corrupt file forever
)

var (
	playerMarker = []byte(`isPlayer="true"`)
	shipMarker   = []byte(`shn="`)
)

// scanShipName streams r looking for the player settlement, then for the ship
// name that follows it, keeping a small overlap so a marker straddling two
// reads is still found.
func scanShipName(r io.Reader) (string, error) {
	buf := make([]byte, 0, scanChunk+scanOverlap)
	chunk := make([]byte, scanChunk)
	found := false
	read := 0

	for read < scanLimit {
		n, err := r.Read(chunk)
		if n > 0 {
			read += n
			buf = append(buf, chunk[:n]...)

			if !found {
				if idx := bytes.Index(buf, playerMarker); idx >= 0 {
					found = true
					buf = buf[idx+len(playerMarker):]
				}
			}
			pending := false
			if found {
				if idx := bytes.Index(buf, shipMarker); idx >= 0 {
					rest := buf[idx+len(shipMarker):]
					if end := bytes.IndexByte(rest, '"'); end >= 0 {
						return string(rest[:end]), nil
					}
					// The name straddles this read; keep it whole for the next one.
					pending = true
				}
			}

			if !pending && len(buf) > scanOverlap {
				buf = append(buf[:0], buf[len(buf)-scanOverlap:]...)
			}
		}
		if err == io.EOF {
			return "", nil
		}
		if err != nil {
			return "", err
		}
	}
	return "", nil
}
