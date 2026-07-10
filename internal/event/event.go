// Package event defines the wire format of .gsd/log.jsonl.
//
// Each line is one self-contained Event. Lines are append-only and
// order-independent: replay sorts by (TS, ID), so a union merge of two
// branches' logs always converges to the same state (last write wins).
package event

type Type string

const (
	TaskCreated        Type = "task_created"
	TaskStatus         Type = "task_status"
	TaskUpdated        Type = "task_updated"
	DecisionLogged     Type = "decision_logged"
	DecisionSuperseded Type = "decision_superseded"
)

// Task statuses carried by TaskStatus events.
const (
	StatusTodo    = "todo"
	StatusDoing   = "doing"
	StatusDone    = "done"
	StatusBlocked = "blocked"
)

type Event struct {
	V     int    `json:"v"`
	ID    string `json:"id"` // time-sortable: <unix-milli hex>-<rand>
	TS    string `json:"ts"` // RFC3339 UTC
	Actor string `json:"actor,omitempty"`
	Type  Type   `json:"type"`

	Task      string   `json:"task,omitempty"`     // T-xxxx this event applies to
	Decision  string   `json:"decision,omitempty"` // D-xxxx this event applies to
	Title     string   `json:"title,omitempty"`
	Status    string   `json:"status,omitempty"`
	Reason    string   `json:"reason,omitempty"` // block reason
	Deps      []string `json:"deps,omitempty"`
	ClearDeps bool     `json:"clear_deps,omitempty"` // task_updated: drop all deps
	Text      string   `json:"text,omitempty"`       // decision summary
	Why       string   `json:"why,omitempty"`        // decision rationale
	By        string   `json:"by,omitempty"`         // superseding decision id
}

func ValidStatus(s string) bool {
	switch s {
	case StatusTodo, StatusDoing, StatusDone, StatusBlocked:
		return true
	}
	return false
}
