# 08 — CLI Specification

Binary: `academic.exe`. No arguments launches the TUI. All commands work from
any working directory (config locates the data root; never creates one
implicitly — missing root is a fatal error with setup instructions).

Conventions: `--format text|json` on reads; `--yes` required for destructive
or bulk writes; exit `0` ok, `2` usage error, `1` runtime error; errors go to
stderr as `academic: <command>: <message>`; `--help` on every node.

## Command tree (v1)

```text
academic                                   # launch TUI
academic units list [--archived] [--format]
academic units add --name N [--code C] [--colour C]
academic units rename --unit U --name N
academic units archive|unarchive --unit U
academic units delete --unit U --yes
academic units show --unit U
academic topics list --unit U [--status unread|pending|read]
academic topics add --unit U --name N [--priority P]
academic topics status --unit U --topic T --status S
academic topics move --unit U --topic T --to UNIT
academic topics delete --unit U --topic T --yes
academic tasks list [--unit U] [--status ...|overdue] [--kind K]
academic tasks add --unit U --title T --kind K --due DATE [--priority P]
academic tasks edit --unit U --task T [--title|--due|--status|--priority|--kind]
academic tasks complete|submit --unit U --task T
academic tasks deadlines [--days N] [--unit U]
academic timeline [--week YYYY-Www | --offset N]
academic analytics [--unit U] [--format]
academic files list [PATH] | import SRC --to DEST [--yes] | move S D | rename P N | delete P --yes | open P | search Q
academic . [--to DEST] [--yes]            # context import of CWD with preview
academic export [--unit U] [--format txt|json] [--out FILE]
academic backup create [--full] [--out FILE] | list | verify FILE | restore FILE --yes
academic maintenance verify [--repair]
academic config show | config set KEY VALUE | config path
academic version | help
```

IDs: `--unit unit-001`, `--topic topic-003` (unit-scoped, see `docs/04`).
Dates: `YYYY-MM-DD` with optional `THH:MM` (local time, stored UTC — `docs/09`).

## Examples

```powershell
academic units add --name "Linear Algebra" --code MATH201
academic topics add --unit unit-001 --name Eigenvalues
academic topics status --unit unit-001 --topic topic-001 --status read
academic tasks add --unit unit-001 --title "Problem set 4" --kind assignment --due 2026-10-09T23:59
academic tasks deadlines --days 14
academic export --format txt --out $HOME\Desktop\status.txt
academic backup create --full
academic . --to unit-001 --yes
```

## `academic .` (context import)

1. Scan CWD (non-recursive by default; `--recursive` opts in; skips executables
   and hidden files, lists them as skipped).
2. Print preview: file list with sizes, destination, collisions.
3. Import nothing unless `--yes` (or interactive confirm when a TTY is present).
   Default destination: `inbox/<date>/` unless `--to` names a unit/topic/task.

## Non-interactive behaviour

- No prompts when stdin is not a TTY: missing `--yes` = error, not a hang.
- `--format json` emits stable schemas (same structs as `exporter`, `docs/11`)
  for scripting; `text` is human-oriented and may evolve.
-stdout` carries data; logs go to the log file, never stdout.

## TUI command input (mirrors CLI, `/`-prefixed)

`/help`, `/dashboard`, `/units [list|add|open|rename|archive]`,
`/topics …`, `/tasks …`, `/timeline [next|prev|today]`, `/files …`,
`/search <q>`, `/analytics`, `/export`, `/backup`, `/settings`, `/quit`.
Parsed by a small dispatcher in `tui/` that calls the **same service
functions** as `cli/` (never Cobra imports in TUI). Unknown command →
"suggestion" message (closest match), not silent failure.

## Cobra decision

v1 uses stdlib `flag` + a hand-written dispatcher: the tree is shallow and the
extra dependency buys little. Revisit only if subcommand count or flag
complexity grows (recorded in DECISIONS.md D3).
