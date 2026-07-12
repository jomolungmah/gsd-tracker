// Package store handles the on-disk layout: the committed append-only
// log (.gsd/log.jsonl, source of truth) and the gitignored SQLite cache
// (.gsd/cache.db, derived, rebuilt whenever the log hash changes).
package store

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jomolungmah/gsd-tracker/internal/event"
)

const Dir = ".gsd"

var ErrNoRoot = errors.New("no .gsd directory found (run `gsd init` in the project root)")

// FindRoot walks up from cwd to the directory containing .gsd/.
func FindRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if fi, err := os.Stat(filepath.Join(dir, Dir)); err == nil && fi.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNoRoot
		}
		dir = parent
	}
}

func LogPath(root string) string {
	return filepath.Join(root, Dir, "log.jsonl")
}

// Append writes one event as a single JSON line. O_APPEND keeps concurrent
// same-machine writers from interleaving within a line for writes of this size.
func Append(root string, ev event.Event) error {
	line, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(LogPath(root), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

// ReadAll parses the log. Malformed lines are skipped with a warning on
// stderr rather than failing the whole command: a merged log with one bad
// line should not brick the tracker.
func ReadAll(root string) ([]event.Event, error) {
	f, err := os.Open(LogPath(root))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []event.Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		raw := sc.Bytes()
		if len(raw) == 0 {
			continue
		}
		var ev event.Event
		if err := json.Unmarshal(raw, &ev); err != nil {
			fmt.Fprintf(os.Stderr, "gsd: warning: skipping malformed log line %d: %v\n", lineNo, err)
			continue
		}
		out = append(out, ev)
	}
	return out, sc.Err()
}

// HashLog returns the sha256 of the log file; a missing log hashes as empty.
func HashLog(root string) (string, error) {
	h := sha256.New()
	f, err := os.Open(LogPath(root))
	if err != nil {
		if os.IsNotExist(err) {
			return hex.EncodeToString(h.Sum(nil)), nil
		}
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
