package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"gsd/internal/event"
	"gsd/internal/ids"
	"gsd/internal/state"
	"gsd/internal/store"
)

// newEvent fills the envelope fields shared by every event.
func newEvent(typ event.Type) event.Event {
	now := time.Now().UTC()
	return event.Event{
		V:     1,
		ID:    ids.NewEvent(now),
		TS:    now.Format(time.RFC3339),
		Actor: actor(),
		Type:  typ,
	}
}

// actor identifies who wrote an event: GSD_ACTOR (lets agents label
// themselves), else git user.name, else $USER.
func actor() string {
	if a := os.Getenv("GSD_ACTOR"); a != "" {
		return a
	}
	if out, err := exec.Command("git", "config", "user.name").Output(); err == nil {
		if name := strings.TrimSpace(string(out)); name != "" {
			return name
		}
	}
	return os.Getenv("USER")
}

// resolveID normalizes user input to a canonical ID: fixes case
// ("t-x7k2" → "T-x7k2") and accepts a bare suffix ("x7k2").
func resolveID(st *state.State, arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	candidates := []string{arg}
	if len(arg) > 2 && (arg[1] == '-') {
		candidates = append(candidates, strings.ToUpper(arg[:1])+arg[1:])
	} else {
		candidates = append(candidates, "T-"+arg, "D-"+arg)
	}
	for _, c := range candidates {
		if st.Exists(c) {
			return c, nil
		}
	}
	return "", fmt.Errorf("no task or decision %q (see `gsd status`)", arg)
}

// splitFlags extracts "--name value" pairs anywhere in args; everything
// else is returned as positional. Stdlib flag stops at the first
// positional, which is hostile to `gsd task add "title" --dep T-x`.
func splitFlags(args []string, known map[string]bool) (map[string][]string, []string, error) {
	flags := map[string][]string{}
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if name, isFlag := strings.CutPrefix(a, "--"); isFlag {
			var val string
			if eq := strings.IndexByte(name, '='); eq >= 0 {
				name, val = name[:eq], name[eq+1:]
			} else {
				if i+1 >= len(args) {
					return nil, nil, fmt.Errorf("flag --%s needs a value", name)
				}
				i++
				val = args[i]
			}
			if !known[name] {
				return nil, nil, fmt.Errorf("unknown flag --%s", name)
			}
			flags[name] = append(flags[name], val)
			continue
		}
		pos = append(pos, a)
	}
	return flags, pos, nil
}

// rel renders a compact relative age: 5m, 3h, 2d.
func rel(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "<1m"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func mustRootAndState() (string, *state.State, error) {
	root, err := store.FindRoot()
	if err != nil {
		return "", nil, err
	}
	st, err := store.LoadState(root)
	if err != nil {
		return "", nil, err
	}
	return root, st, nil
}
