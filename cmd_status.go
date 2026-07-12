package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/jomolungmah/gsd-tracker/internal/event"
	"github.com/jomolungmah/gsd-tracker/internal/state"
)

// stateAlias keeps command files readable without importing state everywhere.
type stateAlias = state.State

const (
	recentDone      = 5
	recentDecisions = 5
	staleAfter      = 72 * time.Hour
	staleCommits    = 10 // repo commits since last touch that mark a doing task stale
)

func cmdStatus() error {
	root, st, err := mustRootAndState()
	if err != nil {
		return err
	}

	tasks := st.TasksSorted()
	if len(tasks) == 0 && len(st.Decisions) == 0 {
		fmt.Println("empty tracker. gsd task add \"<title>\" to start")
		return nil
	}

	var doing, todo, blocked, done []*state.Task
	for _, t := range tasks {
		switch t.Status {
		case event.StatusDoing:
			doing = append(doing, t)
		case event.StatusBlocked:
			blocked = append(blocked, t)
		case event.StatusDone:
			done = append(done, t)
		default:
			todo = append(todo, t)
		}
	}

	fmt.Printf("gsd: %d active, %d blocked, %d done | %d decisions\n",
		len(doing)+len(todo), len(blocked), len(done), len(st.Decisions))

	if n := len(st.Handoffs); n > 0 {
		h := st.Handoffs[n-1]
		text := h.Text
		if len(text) > 140 {
			text = text[:139] + "…"
		}
		ref := ""
		if h.Task != "" {
			ref = " re: " + h.Task
		}
		fmt.Printf("HANDOFF (%s, %s%s)  %s\n", rel(h.Created), h.Actor, ref, text)
	}

	section("DOING", doing, func(t *state.Task) string {
		flag := ""
		if time.Since(t.Updated) > staleAfter {
			flag = ", stale?"
		} else if n, ok := commitsSince(root, t.Updated); ok && n >= staleCommits {
			flag = fmt.Sprintf(", stale? repo +%d commits since touch", n)
		}
		return fmt.Sprintf("(%s%s)", rel(t.Updated), flag)
	}, st)
	section("TODO", todo, func(t *state.Task) string { return "" }, st)
	section("BLOCKED", blocked, func(t *state.Task) string {
		if t.Reason != "" {
			return "— " + t.Reason
		}
		return ""
	}, st)

	if len(done) > 0 {
		n := len(done)
		show := done
		if n > recentDone {
			show = done[n-recentDone:]
		}
		fmt.Printf("DONE (last %d of %d)\n", len(show), n)
		for i := len(show) - 1; i >= 0; i-- {
			t := show[i]
			fmt.Printf("  %s  %s  (%s)\n", t.ID, t.Title, rel(t.Updated))
		}
	}

	ds := st.DecisionsSorted()
	var live []*state.Decision
	for _, d := range ds {
		if d.SupersededBy == "" {
			live = append(live, d)
		}
	}
	if len(live) > 0 {
		n := len(live)
		show := live
		if n > recentDecisions {
			show = live[n-recentDecisions:]
		}
		fmt.Printf("DECISIONS (last %d of %d)\n", len(show), n)
		for i := len(show) - 1; i >= 0; i-- {
			fmt.Printf("  %s  %s\n", show[i].ID, show[i].Text)
		}
	}

	fmt.Println("→ gsd show <ID> | gsd decisions | gsd help")
	return nil
}

func section(name string, tasks []*state.Task, note func(*state.Task) string, st *state.State) {
	if len(tasks) == 0 {
		return
	}
	fmt.Println(name)
	for _, t := range tasks {
		extra := note(t)
		if unmet := unmetDeps(t, st); unmet != "" {
			extra = strings.TrimSpace(extra + "  needs: " + unmet)
		}
		if extra != "" {
			extra = "  " + extra
		}
		fmt.Printf("  %s  %s%s\n", t.ID, t.Title, extra)
	}
}

func unmetDeps(t *state.Task, st *state.State) string {
	var unmet []string
	for _, dep := range t.Deps {
		if d, ok := st.Tasks[dep]; !ok || d.Status != event.StatusDone {
			unmet = append(unmet, dep)
		}
	}
	return strings.Join(unmet, ",")
}
