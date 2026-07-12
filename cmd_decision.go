package main

import (
	"fmt"
	"strings"

	"github.com/jomolungmah/gsd-tracker/internal/event"
	"github.com/jomolungmah/gsd-tracker/internal/ids"
	"github.com/jomolungmah/gsd-tracker/internal/store"
)

// cmdLog handles `gsd log decision <text> [--why <rationale>]`.
func cmdLog(args []string) error {
	if len(args) == 0 || args[0] != "decision" {
		return fmt.Errorf("usage: gsd log decision <text> [--why <rationale>]")
	}
	flags, pos, err := splitFlags(args[1:], map[string]bool{"why": true})
	if err != nil {
		return err
	}
	text := strings.TrimSpace(strings.Join(pos, " "))
	if text == "" {
		return fmt.Errorf("usage: gsd log decision <text> [--why <rationale>]")
	}
	root, st, err := mustRootAndState()
	if err != nil {
		return err
	}
	ev := newEvent(event.DecisionLogged)
	ev.Decision = ids.NewDecision(st.Exists)
	ev.Text = text
	ev.Why = strings.Join(flags["why"], " ")
	if err := store.Append(root, ev); err != nil {
		return err
	}
	fmt.Printf("%s logged: %s\n", ev.Decision, text)
	return nil
}

// cmdDecision handles `gsd decision supersede <ID> <text> [--why <r>]`:
// logs a new decision and marks the old one superseded by it.
func cmdDecision(args []string) error {
	if len(args) < 2 || args[0] != "supersede" {
		return fmt.Errorf("usage: gsd decision supersede <ID> <new text> [--why <rationale>]")
	}
	flags, pos, err := splitFlags(args[1:], map[string]bool{"why": true})
	if err != nil {
		return err
	}
	root, st, err := mustRootAndState()
	if err != nil {
		return err
	}
	oldID, err := resolveID(st, pos[0])
	if err != nil {
		return err
	}
	if !strings.HasPrefix(oldID, "D-") {
		return fmt.Errorf("%s is not a decision", oldID)
	}
	text := strings.TrimSpace(strings.Join(pos[1:], " "))
	if text == "" {
		return fmt.Errorf("supersede needs the text of the new decision")
	}

	newEv := newEvent(event.DecisionLogged)
	newEv.Decision = ids.NewDecision(st.Exists)
	newEv.Text = text
	newEv.Why = strings.Join(flags["why"], " ")
	if err := store.Append(root, newEv); err != nil {
		return err
	}
	supEv := newEvent(event.DecisionSuperseded)
	supEv.Decision = oldID
	supEv.By = newEv.Decision
	if err := store.Append(root, supEv); err != nil {
		return err
	}
	fmt.Printf("%s logged: %s (supersedes %s)\n", newEv.Decision, text, oldID)
	return nil
}

func cmdDecisions() error {
	_, st, err := mustRootAndState()
	if err != nil {
		return err
	}
	ds := st.DecisionsSorted()
	if len(ds) == 0 {
		fmt.Println("no decisions logged. gsd log decision \"chose X\" --why \"because Y\"")
		return nil
	}
	for _, d := range ds {
		mark := ""
		if d.SupersededBy != "" {
			mark = fmt.Sprintf("  [superseded by %s]", d.SupersededBy)
		}
		fmt.Printf("%s  %s%s\n", d.ID, d.Text, mark)
	}
	fmt.Println("→ gsd show <ID> for rationale")
	return nil
}
