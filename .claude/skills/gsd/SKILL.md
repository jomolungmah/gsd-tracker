---
name: gsd
description: Project status protocol for any repo containing a .gsd/ directory. Run `gsd status` at session start instead of exploring the repo to learn project state; check `gsd decisions` before re-deciding a settled question; record task starts/completions/blocks and non-obvious decisions with one-line gsd commands as you work.
---

# gsd protocol

`gsd` tracks project state (tasks + decisions) in `.gsd/` so agents don't
burn tokens rediscovering it. If the repo has a `.gsd/` directory, follow
this protocol. If `gsd` isn't installed, `.gsd/log.jsonl` is readable
directly (one JSON event per line, newest last) — but prefer the CLI.

## Read (cheap — do this first)

- **Session start**: run `gsd status` before exploring the repo. It lists
  active/blocked/done tasks and recent decisions in a few hundred tokens.
- **Drill down, don't scan**: `gsd show T-xxxx` for one task's full detail
  and history. Never read `.gsd/log.jsonl` when the CLI is available.
- **Before proposing a design choice**: run `gsd decisions`. If the
  question is already settled (e.g. "D-a4f1 SQLite over Postgres"), follow
  it or explicitly supersede it — don't silently relitigate.

## Write (one-liners — do these as you work)

| Moment | Command |
|---|---|
| Picking up a task | `gsd task start T-xxxx` |
| Finishing a task | `gsd task done T-xxxx` |
| Hitting a blocker | `gsd task block T-xxxx <reason>` |
| Blocker cleared | `gsd task unblock T-xxxx` |
| New work identified | `gsd task add "<title>" [--dep T-xxxx]` |
| Non-obvious choice made | `gsd log decision "<what>" --why "<rationale>"` |
| Reversing a decision | `gsd decision supersede D-xxxx "<new>" --why "<r>"` |

Set `GSD_ACTOR` to your agent/session name so history shows who did what.

## Rules

- Record decisions when the *why* isn't obvious from the code — that's
  what saves the next agent from re-deriving it. Skip trivial choices.
- Update status at the moment it changes, not in a batch at the end.
- Never edit `.gsd/log.jsonl` by hand; it's append-only and the CLI owns
  the format. `.gsd/cache.db` is a derived cache — ignore it entirely.
- Commit `.gsd/log.jsonl` together with the work it describes.
