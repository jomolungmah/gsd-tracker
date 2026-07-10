package store

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"gsd/internal/event"
	"gsd/internal/state"
)

const schema = `
CREATE TABLE IF NOT EXISTS meta(k TEXT PRIMARY KEY, v TEXT);
CREATE TABLE IF NOT EXISTS events(
  id TEXT PRIMARY KEY, ts TEXT, type TEXT, task TEXT, decision TEXT, raw TEXT
);
CREATE INDEX IF NOT EXISTS events_task ON events(task);
CREATE TABLE IF NOT EXISTS tasks(
  id TEXT PRIMARY KEY, title TEXT, status TEXT, reason TEXT,
  deps TEXT, actor TEXT, created TEXT, updated TEXT
);
CREATE TABLE IF NOT EXISTS decisions(
  id TEXT PRIMARY KEY, text TEXT, why TEXT, actor TEXT,
  superseded_by TEXT, created TEXT
);
`

func openDB(root string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", filepath.Join(root, Dir, "cache.db"))
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// LoadState returns current state, from the cache when it matches the
// log hash, otherwise by replaying the log and rebuilding the cache.
// A cache failure never blocks reads: we fall back to a pure replay.
func LoadState(root string) (*state.State, error) {
	hash, err := HashLog(root)
	if err != nil {
		return nil, err
	}
	db, err := openDB(root)
	if err != nil {
		return replayOnly(root)
	}
	defer db.Close()

	var cur string
	_ = db.QueryRow(`SELECT v FROM meta WHERE k='log_hash'`).Scan(&cur)
	if cur == hash {
		if st, err := loadFromDB(db); err == nil {
			return st, nil
		}
	}

	evs, err := ReadAll(root)
	if err != nil {
		return nil, err
	}
	st := state.Replay(evs)
	_ = rebuild(db, st, evs, hash) // best-effort; state is already in hand
	return st, nil
}

func replayOnly(root string) (*state.State, error) {
	evs, err := ReadAll(root)
	if err != nil {
		return nil, err
	}
	return state.Replay(evs), nil
}

// EventsFor returns the event history for one task or decision ID,
// oldest first. Reads the indexed cache; falls back to a log scan.
func EventsFor(root, id string) ([]event.Event, error) {
	db, err := openDB(root)
	if err == nil {
		defer db.Close()
		rows, qerr := db.Query(
			`SELECT raw FROM events WHERE task=? OR decision=? ORDER BY ts, id`, id, id)
		if qerr == nil {
			defer rows.Close()
			var out []event.Event
			ok := true
			for rows.Next() {
				var raw string
				if rows.Scan(&raw) != nil {
					ok = false
					break
				}
				var ev event.Event
				if json.Unmarshal([]byte(raw), &ev) != nil {
					ok = false
					break
				}
				out = append(out, ev)
			}
			if ok && rows.Err() == nil {
				return out, nil
			}
		}
	}
	// Fallback: scan the log directly.
	evs, err := ReadAll(root)
	if err != nil {
		return nil, err
	}
	var out []event.Event
	for _, ev := range evs {
		if ev.Task == id || ev.Decision == id {
			out = append(out, ev)
		}
	}
	return out, nil
}

func rebuild(db *sql.DB, st *state.State, evs []event.Event, hash string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, table := range []string{"events", "tasks", "decisions"} {
		if _, err := tx.Exec("DELETE FROM " + table); err != nil {
			return err
		}
	}
	for _, ev := range evs {
		raw, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		// OR IGNORE: a union merge can leave duplicate event IDs in the log.
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO events(id, ts, type, task, decision, raw) VALUES(?,?,?,?,?,?)`,
			ev.ID, ev.TS, string(ev.Type), ev.Task, ev.Decision, string(raw)); err != nil {
			return err
		}
	}
	for _, t := range st.TasksSorted() {
		if _, err := tx.Exec(
			`INSERT INTO tasks(id, title, status, reason, deps, actor, created, updated) VALUES(?,?,?,?,?,?,?,?)`,
			t.ID, t.Title, t.Status, t.Reason, strings.Join(t.Deps, " "), t.Actor,
			t.Created.Format(time.RFC3339), t.Updated.Format(time.RFC3339)); err != nil {
			return err
		}
	}
	for _, d := range st.DecisionsSorted() {
		if _, err := tx.Exec(
			`INSERT INTO decisions(id, text, why, actor, superseded_by, created) VALUES(?,?,?,?,?,?)`,
			d.ID, d.Text, d.Why, d.Actor, d.SupersededBy,
			d.Created.Format(time.RFC3339)); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(
		`INSERT INTO meta(k, v) VALUES('log_hash', ?) ON CONFLICT(k) DO UPDATE SET v=excluded.v`,
		hash); err != nil {
		return err
	}
	return tx.Commit()
}

func loadFromDB(db *sql.DB) (*state.State, error) {
	st := state.New()

	rows, err := db.Query(`SELECT id, title, status, reason, deps, actor, created, updated FROM tasks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t state.Task
		var deps, created, updated string
		if err := rows.Scan(&t.ID, &t.Title, &t.Status, &t.Reason, &deps, &t.Actor, &created, &updated); err != nil {
			return nil, err
		}
		if deps != "" {
			t.Deps = strings.Fields(deps)
		}
		t.Created, _ = time.Parse(time.RFC3339, created)
		t.Updated, _ = time.Parse(time.RFC3339, updated)
		st.Tasks[t.ID] = &t
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	drows, err := db.Query(`SELECT id, text, why, actor, superseded_by, created FROM decisions`)
	if err != nil {
		return nil, err
	}
	defer drows.Close()
	for drows.Next() {
		var d state.Decision
		var created string
		if err := drows.Scan(&d.ID, &d.Text, &d.Why, &d.Actor, &d.SupersededBy, &created); err != nil {
			return nil, err
		}
		d.Created, _ = time.Parse(time.RFC3339, created)
		st.Decisions[d.ID] = &d
	}
	return st, drows.Err()
}
