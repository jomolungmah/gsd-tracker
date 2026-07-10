package main

import (
	"fmt"
	"strings"

	"gsd/internal/event"
	"gsd/internal/ids"
	"gsd/internal/store"
)

func cmdTask(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: gsd task add|start|done|block|unblock|show …")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "add":
		return taskAdd(rest)
	case "start":
		return taskStatus(rest, event.StatusDoing, "started")
	case "done":
		return taskStatus(rest, event.StatusDone, "done")
	case "block":
		return taskStatus(rest, event.StatusBlocked, "blocked")
	case "unblock":
		return taskStatus(rest, event.StatusTodo, "unblocked (now todo)")
	case "show":
		return cmdShow(rest)
	default:
		return fmt.Errorf("unknown task subcommand %q", sub)
	}
}

func taskAdd(args []string) error {
	flags, pos, err := splitFlags(args, map[string]bool{"dep": true})
	if err != nil {
		return err
	}
	title := strings.TrimSpace(strings.Join(pos, " "))
	if title == "" {
		return fmt.Errorf("usage: gsd task add <title> [--dep <ID>]")
	}
	root, st, err := mustRootAndState()
	if err != nil {
		return err
	}
	var deps []string
	for _, d := range flags["dep"] {
		id, err := resolveID(st, d)
		if err != nil {
			return fmt.Errorf("--dep: %w", err)
		}
		deps = append(deps, id)
	}
	ev := newEvent(event.TaskCreated)
	ev.Task = ids.NewTask(st.Exists)
	ev.Title = title
	ev.Status = event.StatusTodo
	ev.Deps = deps
	if err := store.Append(root, ev); err != nil {
		return err
	}
	fmt.Printf("%s added: %s\n", ev.Task, title)
	return nil
}

func taskStatus(args []string, status, verb string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: gsd task %s <ID>", verbToSub(status))
	}
	root, st, err := mustRootAndState()
	if err != nil {
		return err
	}
	id, err := resolveID(st, args[0])
	if err != nil {
		return err
	}
	if !strings.HasPrefix(id, "T-") {
		return fmt.Errorf("%s is not a task", id)
	}
	ev := newEvent(event.TaskStatus)
	ev.Task = id
	ev.Status = status
	if status == event.StatusBlocked {
		ev.Reason = strings.TrimSpace(strings.Join(args[1:], " "))
	}
	if err := store.Append(root, ev); err != nil {
		return err
	}
	fmt.Printf("%s %s: %s\n", id, verb, st.Tasks[id].Title)
	return nil
}

func verbToSub(status string) string {
	switch status {
	case event.StatusDoing:
		return "start"
	case event.StatusDone:
		return "done"
	case event.StatusBlocked:
		return "block"
	default:
		return "unblock"
	}
}
