package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jomolungmah/gsd-tracker/internal/event"
	"github.com/jomolungmah/gsd-tracker/internal/ids"
)

func initRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, Dir), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func mkEvent(typ event.Type, ts time.Time) event.Event {
	return event.Event{V: 1, ID: ids.NewEvent(ts), TS: ts.Format(time.RFC3339), Type: typ}
}

func TestAppendReadRoundtrip(t *testing.T) {
	root := initRoot(t)
	e1 := mkEvent(event.TaskCreated, time.Now().UTC())
	e1.Task, e1.Title, e1.Status = "T-aaaa", "first task", event.StatusTodo
	e2 := mkEvent(event.DecisionLogged, time.Now().UTC().Add(time.Second))
	e2.Decision, e2.Text, e2.Why = "D-bbbb", "use sqlite", "cache speed"

	for _, e := range []event.Event{e1, e2} {
		if err := Append(root, e); err != nil {
			t.Fatal(err)
		}
	}
	got, err := ReadAll(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Title != "first task" || got[1].Why != "cache speed" {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
}

func TestCacheRebuildAndReuse(t *testing.T) {
	root := initRoot(t)
	e := mkEvent(event.TaskCreated, time.Now().UTC())
	e.Task, e.Title, e.Status = "T-aaaa", "cached task", event.StatusTodo
	if err := Append(root, e); err != nil {
		t.Fatal(err)
	}

	// First load rebuilds the cache from the log.
	st1, err := LoadState(root)
	if err != nil {
		t.Fatal(err)
	}
	if st1.Tasks["T-aaaa"].Title != "cached task" {
		t.Fatalf("replay path: %+v", st1.Tasks)
	}

	// Second load must hit the cache (same hash) and agree.
	st2, err := LoadState(root)
	if err != nil {
		t.Fatal(err)
	}
	if st2.Tasks["T-aaaa"].Title != "cached task" || st2.Tasks["T-aaaa"].Status != event.StatusTodo {
		t.Fatalf("cache path: %+v", st2.Tasks["T-aaaa"])
	}

	// Handoffs must survive the cache roundtrip too.
	h := mkEvent(event.HandoffLogged, time.Now().UTC().Add(500*time.Millisecond))
	h.Text, h.Task = "left off here", "T-aaaa"
	if err := Append(root, h); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadState(root); err != nil { // rebuild
		t.Fatal(err)
	}
	stc, err := LoadState(root) // cache path
	if err != nil {
		t.Fatal(err)
	}
	if len(stc.Handoffs) != 1 || stc.Handoffs[0].Text != "left off here" {
		t.Fatalf("handoff lost in cache roundtrip: %+v", stc.Handoffs)
	}

	// New append invalidates the hash; next load must see the change.
	e2 := mkEvent(event.TaskStatus, time.Now().UTC().Add(time.Second))
	e2.Task, e2.Status = "T-aaaa", event.StatusDone
	if err := Append(root, e2); err != nil {
		t.Fatal(err)
	}
	st3, err := LoadState(root)
	if err != nil {
		t.Fatal(err)
	}
	if st3.Tasks["T-aaaa"].Status != event.StatusDone {
		t.Fatalf("stale cache served after log change: %+v", st3.Tasks["T-aaaa"])
	}
}

func TestEventsForFiltersByID(t *testing.T) {
	root := initRoot(t)
	base := time.Now().UTC()
	for i, task := range []string{"T-aaaa", "T-bbbb", "T-aaaa"} {
		e := mkEvent(event.TaskStatus, base.Add(time.Duration(i)*time.Second))
		e.Task, e.Status = task, event.StatusDoing
		if err := Append(root, e); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := LoadState(root); err != nil { // populate cache
		t.Fatal(err)
	}
	evs, err := EventsFor(root, "T-aaaa")
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 {
		t.Fatalf("got %d events for T-aaaa, want 2", len(evs))
	}
}

func TestMalformedLineSkipped(t *testing.T) {
	root := initRoot(t)
	e := mkEvent(event.TaskCreated, time.Now().UTC())
	e.Task, e.Title = "T-aaaa", "good"
	if err := Append(root, e); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(LogPath(root), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{not json\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()

	got, err := ReadAll(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Title != "good" {
		t.Fatalf("want the one good event, got %+v", got)
	}
}
