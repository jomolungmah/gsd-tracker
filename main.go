// gsd — a token-lean project state tracker for AI agents and humans.
//
// Source of truth is .gsd/log.jsonl (append-only, committed, merges via
// union); .gsd/cache.db is a derived SQLite view, rebuilt on log change.
package main

import (
	"fmt"
	"os"
)

const usage = `gsd — project state tracker for agents & humans

  gsd init                              set up .gsd/ in current directory
  gsd status                            compact project digest (run at session start)
  gsd task add <title> [--dep <ID>]     create task (repeat --dep as needed)
  gsd task start|done <ID>              mark task doing / done
  gsd task block <ID> [reason]          mark task blocked
  gsd task unblock <ID>                 back to todo
  gsd task edit <ID> [--title <t>] [--dep <ID>]   edit task (--dep none clears)
  gsd show <ID>                         full detail + history (T-… or D-…)
  gsd handoff <text> [--task <ID>]      record where-I-left-off; no args: show recent
  gsd log decision <text> [--why <r>]   record a decision
  gsd decision supersede <ID> <text> [--why <r>]
  gsd decisions                         list decisions

State lives in .gsd/: log.jsonl is committed (source of truth), cache.db is not.`

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println(usage)
		os.Exit(2)
	}
	var err error
	switch args[0] {
	case "init":
		err = cmdInit()
	case "status":
		err = cmdStatus()
	case "task":
		err = cmdTask(args[1:])
	case "show":
		err = cmdShow(args[1:])
	case "handoff":
		err = cmdHandoff(args[1:])
	case "log":
		err = cmdLog(args[1:])
	case "decision":
		err = cmdDecision(args[1:])
	case "decisions":
		err = cmdDecisions()
	case "help", "--help", "-h":
		fmt.Println(usage)
	default:
		fmt.Fprintf(os.Stderr, "gsd: unknown command %q\n\n%s\n", args[0], usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "gsd: %v\n", err)
		os.Exit(1)
	}
}
