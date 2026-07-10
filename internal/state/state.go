// Package state materializes the event log into current project state.
package state

import (
	"sort"
	"time"

	"gsd/internal/event"
)

type Task struct {
	ID      string
	Title   string
	Status  string
	Reason  string // block reason, set while Status == blocked
	Deps    []string
	Actor   string // actor of the last event touching this task
	Created time.Time
	Updated time.Time
}

type Decision struct {
	ID           string
	Text         string
	Why          string
	Actor        string
	SupersededBy string
	Created      time.Time
}

type State struct {
	Tasks     map[string]*Task
	Decisions map[string]*Decision
}

func New() *State {
	return &State{
		Tasks:     map[string]*Task{},
		Decisions: map[string]*Decision{},
	}
}

// Replay folds events into state. Events are sorted by (TS, ID) so replay
// is deterministic regardless of file order after a union merge, and
// concurrent updates to the same entity resolve last-write-wins by
// timestamp. Duplicate event IDs (possible after a merge) are applied once.
func Replay(events []event.Event) *State {
	evs := make([]event.Event, len(events))
	copy(evs, events)
	sort.SliceStable(evs, func(i, j int) bool {
		if evs[i].TS != evs[j].TS {
			return evs[i].TS < evs[j].TS
		}
		return evs[i].ID < evs[j].ID
	})

	st := New()
	seen := map[string]bool{}
	for _, ev := range evs {
		if ev.ID != "" {
			if seen[ev.ID] {
				continue
			}
			seen[ev.ID] = true
		}
		st.apply(ev)
	}
	return st
}

func (st *State) apply(ev event.Event) {
	ts, _ := time.Parse(time.RFC3339, ev.TS)

	switch ev.Type {
	case event.TaskCreated:
		t, ok := st.Tasks[ev.Task]
		if !ok {
			t = &Task{ID: ev.Task, Status: event.StatusTodo, Created: ts}
			st.Tasks[ev.Task] = t
		}
		t.Title = ev.Title
		if len(ev.Deps) > 0 {
			t.Deps = ev.Deps
		}
		if ev.Status != "" {
			t.Status = ev.Status
		}
		t.Actor = ev.Actor
		t.Updated = ts

	case event.TaskStatus:
		t, ok := st.Tasks[ev.Task]
		if !ok {
			// Status event for a task whose creation we haven't seen
			// (shouldn't happen, but logs are merged files — degrade gracefully).
			t = &Task{ID: ev.Task, Title: "(unknown)", Created: ts}
			st.Tasks[ev.Task] = t
		}
		if event.ValidStatus(ev.Status) {
			t.Status = ev.Status
		}
		if t.Status == event.StatusBlocked {
			t.Reason = ev.Reason
		} else {
			t.Reason = ""
		}
		t.Actor = ev.Actor
		t.Updated = ts

	case event.TaskUpdated:
		t, ok := st.Tasks[ev.Task]
		if !ok {
			t = &Task{ID: ev.Task, Status: event.StatusTodo, Created: ts}
			st.Tasks[ev.Task] = t
		}
		if ev.Title != "" {
			t.Title = ev.Title
		}
		switch {
		case ev.ClearDeps:
			t.Deps = nil
		case len(ev.Deps) > 0:
			t.Deps = ev.Deps
		}
		t.Actor = ev.Actor
		t.Updated = ts

	case event.DecisionLogged:
		st.Decisions[ev.Decision] = &Decision{
			ID:      ev.Decision,
			Text:    ev.Text,
			Why:     ev.Why,
			Actor:   ev.Actor,
			Created: ts,
		}

	case event.DecisionSuperseded:
		if d, ok := st.Decisions[ev.Decision]; ok {
			d.SupersededBy = ev.By
		}
	}
}

// Exists reports whether an ID is already taken by a task or decision.
func (st *State) Exists(id string) bool {
	if _, ok := st.Tasks[id]; ok {
		return true
	}
	_, ok := st.Decisions[id]
	return ok
}

// TasksSorted returns tasks ordered by creation time (ID as tiebreak).
func (st *State) TasksSorted() []*Task {
	out := make([]*Task, 0, len(st.Tasks))
	for _, t := range st.Tasks {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Created.Equal(out[j].Created) {
			return out[i].Created.Before(out[j].Created)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// DecisionsSorted returns decisions ordered by creation time.
func (st *State) DecisionsSorted() []*Decision {
	out := make([]*Decision, 0, len(st.Decisions))
	for _, d := range st.Decisions {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Created.Equal(out[j].Created) {
			return out[i].Created.Before(out[j].Created)
		}
		return out[i].ID < out[j].ID
	})
	return out
}
