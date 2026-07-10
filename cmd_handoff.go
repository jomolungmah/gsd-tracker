package main

import (
	"fmt"
	"strings"

	"gsd/internal/event"
	"gsd/internal/store"
)

// cmdHandoff records a "where I left off" note, or with no args shows
// the most recent ones in full.
func cmdHandoff(args []string) error {
	flags, pos, err := splitFlags(args, map[string]bool{"task": true})
	if err != nil {
		return err
	}
	root, st, err := mustRootAndState()
	if err != nil {
		return err
	}

	text := strings.TrimSpace(strings.Join(pos, " "))
	if text == "" {
		return showHandoffs(st)
	}

	ev := newEvent(event.HandoffLogged)
	ev.Text = text
	if ts := flags["task"]; len(ts) > 0 {
		id, err := resolveID(st, ts[len(ts)-1])
		if err != nil {
			return fmt.Errorf("--task: %w", err)
		}
		if !strings.HasPrefix(id, "T-") {
			return fmt.Errorf("--task: %s is not a task", id)
		}
		ev.Task = id
	}
	if err := store.Append(root, ev); err != nil {
		return err
	}
	fmt.Println("handoff recorded")
	return nil
}

const recentHandoffs = 3

func showHandoffs(st *stateAlias) error {
	hs := st.Handoffs
	if len(hs) == 0 {
		fmt.Println("no handoffs. gsd handoff \"<done / tried / next step>\" [--task T-xxxx]")
		return nil
	}
	if len(hs) > recentHandoffs {
		hs = hs[len(hs)-recentHandoffs:]
	}
	for i := len(hs) - 1; i >= 0; i-- {
		h := hs[i]
		ref := ""
		if h.Task != "" {
			ref = "  re: " + h.Task
		}
		fmt.Printf("%s ago, %s%s\n  %s\n", rel(h.Created), h.Actor, ref, h.Text)
	}
	return nil
}
