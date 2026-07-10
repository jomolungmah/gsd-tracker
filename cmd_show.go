package main

import (
	"fmt"
	"strings"
	"time"

	"gsd/internal/event"
	"gsd/internal/store"
)

func cmdShow(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: gsd show <ID>")
	}
	root, st, err := mustRootAndState()
	if err != nil {
		return err
	}
	id, err := resolveID(st, args[0])
	if err != nil {
		return err
	}
	if strings.HasPrefix(id, "T-") {
		showTask(root, st, id)
	} else {
		showDecision(st, id)
	}
	return nil
}

func showTask(root string, st *stateAlias, id string) {
	t := st.Tasks[id]
	fmt.Printf("%s  %s\n", t.ID, t.Title)
	fmt.Printf("status: %s", t.Status)
	if t.Reason != "" {
		fmt.Printf(" — %s", t.Reason)
	}
	fmt.Printf("  (updated %s ago by %s)\n", rel(t.Updated), t.Actor)
	if len(t.Deps) > 0 {
		var parts []string
		for _, dep := range t.Deps {
			mark := "?"
			if d, ok := st.Tasks[dep]; ok {
				mark = d.Status
			}
			parts = append(parts, fmt.Sprintf("%s(%s)", dep, mark))
		}
		fmt.Printf("deps: %s\n", strings.Join(parts, ", "))
	}
	printHistory(root, id)
}

func showDecision(st *stateAlias, id string) {
	d := st.Decisions[id]
	fmt.Printf("%s  %s\n", d.ID, d.Text)
	if d.Why != "" {
		fmt.Printf("why: %s\n", d.Why)
	}
	fmt.Printf("logged %s by %s\n", d.Created.Format("2006-01-02"), d.Actor)
	if d.SupersededBy != "" {
		fmt.Printf("SUPERSEDED by %s\n", d.SupersededBy)
	}
}

func printHistory(root, id string) {
	evs, err := store.EventsFor(root, id)
	if err != nil || len(evs) == 0 {
		return
	}
	fmt.Println("history:")
	for _, ev := range evs {
		ts, _ := time.Parse(time.RFC3339, ev.TS)
		line := string(ev.Type)
		switch ev.Type {
		case event.TaskCreated:
			line = "created"
		case event.TaskStatus:
			line = ev.Status
			if ev.Reason != "" {
				line += " — " + ev.Reason
			}
		}
		fmt.Printf("  %s  %-12s %s\n", ts.Format("2006-01-02 15:04"), ev.Actor, line)
	}
}
