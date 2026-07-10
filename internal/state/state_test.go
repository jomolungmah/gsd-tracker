package state

import (
	"testing"

	"gsd/internal/event"
)

func ev(id, ts string, typ event.Type, mut func(*event.Event)) event.Event {
	e := event.Event{V: 1, ID: id, TS: ts, Type: typ}
	if mut != nil {
		mut(&e)
	}
	return e
}

func TestReplayOrdersByTimestampNotFilePosition(t *testing.T) {
	// Simulate a union merge: the later event appears earlier in the file.
	events := []event.Event{
		ev("e2", "2026-07-02T00:00:00Z", event.TaskStatus, func(e *event.Event) {
			e.Task, e.Status = "T-aaaa", event.StatusDone
		}),
		ev("e1", "2026-07-01T00:00:00Z", event.TaskCreated, func(e *event.Event) {
			e.Task, e.Title, e.Status = "T-aaaa", "thing", event.StatusTodo
		}),
	}
	st := Replay(events)
	if got := st.Tasks["T-aaaa"].Status; got != event.StatusDone {
		t.Fatalf("status = %q, want done (last write by timestamp must win)", got)
	}
	if got := st.Tasks["T-aaaa"].Title; got != "thing" {
		t.Fatalf("title = %q, want thing", got)
	}
}

func TestConcurrentStatusLastWriteWins(t *testing.T) {
	// Alice marks done at 10:00, Bob blocks at 11:00 on another branch.
	events := []event.Event{
		ev("e1", "2026-07-01T00:00:00Z", event.TaskCreated, func(e *event.Event) {
			e.Task, e.Title = "T-aaaa", "thing"
		}),
		ev("e2", "2026-07-02T10:00:00Z", event.TaskStatus, func(e *event.Event) {
			e.Task, e.Status = "T-aaaa", event.StatusDone
		}),
		ev("e3", "2026-07-02T11:00:00Z", event.TaskStatus, func(e *event.Event) {
			e.Task, e.Status, e.Reason = "T-aaaa", event.StatusBlocked, "regression"
		}),
	}
	st := Replay(events)
	task := st.Tasks["T-aaaa"]
	if task.Status != event.StatusBlocked || task.Reason != "regression" {
		t.Fatalf("got %s/%q, want blocked/regression", task.Status, task.Reason)
	}
}

func TestTimestampTieBrokenByEventID(t *testing.T) {
	ts := "2026-07-01T00:00:00Z"
	events := []event.Event{
		ev("e1", "2026-06-30T00:00:00Z", event.TaskCreated, func(e *event.Event) {
			e.Task, e.Title = "T-aaaa", "thing"
		}),
		ev("e9", ts, event.TaskStatus, func(e *event.Event) {
			e.Task, e.Status = "T-aaaa", event.StatusDone
		}),
		ev("e2", ts, event.TaskStatus, func(e *event.Event) {
			e.Task, e.Status = "T-aaaa", event.StatusDoing
		}),
	}
	st := Replay(events)
	if got := st.Tasks["T-aaaa"].Status; got != event.StatusDone {
		t.Fatalf("status = %q, want done (higher event ID wins ties)", got)
	}
}

func TestDuplicateEventIDsAppliedOnce(t *testing.T) {
	// A union merge can duplicate identical lines.
	e := ev("e1", "2026-07-01T00:00:00Z", event.TaskCreated, func(e *event.Event) {
		e.Task, e.Title = "T-aaaa", "thing"
	})
	st := Replay([]event.Event{e, e, e})
	if len(st.Tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(st.Tasks))
	}
}

func TestUnblockClearsReason(t *testing.T) {
	events := []event.Event{
		ev("e1", "2026-07-01T00:00:00Z", event.TaskCreated, func(e *event.Event) {
			e.Task, e.Title = "T-aaaa", "thing"
		}),
		ev("e2", "2026-07-02T00:00:00Z", event.TaskStatus, func(e *event.Event) {
			e.Task, e.Status, e.Reason = "T-aaaa", event.StatusBlocked, "waiting"
		}),
		ev("e3", "2026-07-03T00:00:00Z", event.TaskStatus, func(e *event.Event) {
			e.Task, e.Status = "T-aaaa", event.StatusTodo
		}),
	}
	st := Replay(events)
	task := st.Tasks["T-aaaa"]
	if task.Status != event.StatusTodo || task.Reason != "" {
		t.Fatalf("got %s/%q, want todo with cleared reason", task.Status, task.Reason)
	}
}

func TestDecisionSupersede(t *testing.T) {
	events := []event.Event{
		ev("e1", "2026-07-01T00:00:00Z", event.DecisionLogged, func(e *event.Event) {
			e.Decision, e.Text, e.Why = "D-aaaa", "use X", "fast"
		}),
		ev("e2", "2026-07-02T00:00:00Z", event.DecisionLogged, func(e *event.Event) {
			e.Decision, e.Text = "D-bbbb", "use Y"
		}),
		ev("e3", "2026-07-02T00:00:01Z", event.DecisionSuperseded, func(e *event.Event) {
			e.Decision, e.By = "D-aaaa", "D-bbbb"
		}),
	}
	st := Replay(events)
	if got := st.Decisions["D-aaaa"].SupersededBy; got != "D-bbbb" {
		t.Fatalf("SupersededBy = %q, want D-bbbb", got)
	}
	if st.Decisions["D-bbbb"].SupersededBy != "" {
		t.Fatal("new decision must not be superseded")
	}
}
