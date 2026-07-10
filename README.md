# gsd — get stuff done tracker

A token-lean project state tracker for AI agents (and the humans working
with them). Instead of a fresh agent session burning thousands of tokens
re-reading the repo, git log, and old conversations to figure out where
things stand, it runs one command:

```
$ gsd status
gsd: 2 active, 1 blocked, 3 done | 4 decisions
DOING
  T-x7k2  Add auth middleware  (2d)
TODO
  T-m3qa  Rate limiting  needs: T-x7k2
BLOCKED
  T-p9f2  Deploy pipeline  — waiting on auth middleware
DONE (last 3 of 3)
  T-k2d8  Session store  (1d)
DECISIONS (last 2 of 4)
  D-a4f1  SQLite over Postgres
  D-b2c3  Event log as source of truth
→ gsd show <ID> | gsd decisions | gsd help
```

~200–400 tokens for a typical project. Details are pull, not push:
`gsd show T-x7k2` drills into one task; the digest stays small.

## Install

```sh
go build -o gsd .   # then put the binary on PATH
```

Single static binary, no runtime dependencies (SQLite is embedded via
pure-Go modernc.org/sqlite).

## Usage

```
gsd init                              set up .gsd/ in the project root
gsd status                            compact digest (run at session start)
gsd task add <title> [--dep <ID>]     create a task
gsd task start|done <ID>              mark doing / done
gsd task block <ID> [reason]          mark blocked
gsd task unblock <ID>                 back to todo
gsd task edit <ID> [--title <t>] [--dep <ID>]   edit task (--dep none clears)
gsd show <ID>                         full detail + history (T-… or D-…)
gsd handoff <text> [--task <ID>]      record where-I-left-off; no args: show recent
gsd log decision <text> [--why <r>]   record a decision
gsd decision supersede <ID> <text>    replace a decision (old stays in history)
gsd decisions                         list decisions
```

IDs are forgiving: `t-x7k2` and bare `x7k2` both resolve. Set `GSD_ACTOR`
so multi-agent history shows who did what (falls back to git user.name).

## Architecture

```
.gsd/
  log.jsonl       committed — append-only event log, the source of truth
  cache.db        gitignored — derived SQLite view, rebuilt on log change
  .gitattributes  log.jsonl merge=union
  .gitignore      cache.db
```

**Event log as truth, SQLite as cache.** Every write appends one
self-contained JSON line (`task_created`, `task_status`,
`decision_logged`, `decision_superseded`) carrying a timestamp, actor, and
a time-prefixed event ID. Reads replay the log into current state; the
replay result is cached in SQLite keyed by the log's sha256, so unchanged
logs load without re-parsing.

**Merges are conflict-free by construction.** `merge=union` tells git to
keep both sides' appended lines. That is safe because line order carries
no meaning: replay sorts events by `(timestamp, event ID)`, so any
interleaving converges to the same state. Concurrent updates to the same
task resolve last-write-wins by timestamp; both events remain in history.
Duplicate lines after a merge are deduplicated by event ID.

**IDs can't collide across branches.** Task/decision IDs (`T-x7k2`,
`D-a4f1`) use a random 4-char suffix instead of a sequence number, so two
branches minting IDs concurrently never fight over "the next number".

**Staleness fighting.** A `doing` task is flagged `stale?` in status when
untouched for 72h, or when ≥10 commits landed since it was last touched —
an active repo with an untouched task means the tracker drifted from
reality. Status also leads with the latest handoff note, so a fresh
session sees "where the last one left off" first.

**Known limits (accepted for v1):** last-write-wins trusts wall clocks —
skewed clocks across machines can pick the "wrong" winner, with worst case
a briefly stale status.

## Agent integration

`.claude/skills/gsd/SKILL.md` teaches agents the protocol: `gsd status`
at session start instead of exploring, `gsd decisions` before relitigating
settled questions, one-line writes at the moment state changes. Copy the
skill directory into any repo that uses gsd (or install it globally in
`~/.claude/skills/`).
