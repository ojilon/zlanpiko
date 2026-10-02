# 08 — CLI Specification

Binary: `zlanpiko.exe`. No arguments launches the TUI. All commands work from
any working directory (config locates the data root; never creates one
implicitly — missing root is a fatal error with setup instructions).

Conventions: `--format text|json` on reads; `--yes` required for destructive
or bulk writes; exit `0` ok, `2` usage error, `1` runtime error; errors go to
stderr as `zlanpiko: <command>: <message>`; `--help` on every node.

## Command tree (v1)

```text
zlanpiko                                   # launch TUI
zlanpiko units list [--archived] [--format]
zlanpiko units add --name N [--code C] [--colour C]
zlanpiko units rename --unit U --name N
zlanpiko units archive|unarchive --unit U
zlanpiko units delete --unit U --yes
zlanpiko units show --unit U
zlanpiko topics list --unit U [--status unread|pending|read]
zlanpiko topics add --unit U --name N [--priority P]
zlanpiko topics status --unit U --topic T --status S
zlanpiko topics move --unit U --topic T --to UNIT
zlanpiko topics delete --unit U --topic T --yes
zlanpiko tasks list [--unit U] [--status ...|overdue] [--kind K]
zlanpiko tasks add --unit U --title T --kind K --due DATE [--priority P]
zlanpiko tasks edit --unit U --task T [--title|--due|--status|--priority|--kind]
zlanpiko tasks complete|submit --unit U --task T
zlanpiko tasks deadlines [--days N] [--unit U]
zlanpiko timeline [--week YYYY-Www | --offset N]
zlanpiko analytics [--unit U] [--format]
zlanpiko files list [PATH] | import SRC --to DEST [--yes] | move S D | rename P N | delete P --yes | open P | search Q
zlanpiko . [--to DEST] [--yes]            # context import of CWD with preview
zlanpiko export [--unit U] [--format txt|json] [--out FILE]
zlanpiko backup create [--full] [--out FILE] | list | verify FILE | restore FILE --yes
zlanpiko maintenance verify [--repair]
zlanpiko config show | config set KEY VALUE | config path
zlanpiko version | help
```

IDs: `--unit unit-001`, `--topic topic-003` (unit-scoped, see `docs/04`).
Dates: `YYYY-MM-DD` with optional `THH:MM` (local time, stored UTC — `docs/09`).

## Examples

```powershell
zlanpiko units add --name "Linear Algebra" --code MATH201
zlanpiko topics add --unit unit-001 --name Eigenvalues
zlanpiko topics status --unit unit-001 --topic topic-001 --status read
zlanpiko tasks add --unit unit-001 --title "Problem set 4" --kind assignment --due 2026-10-09T23:59
zlanpiko tasks deadlines --days 14
zlanpiko export --format txt --out $HOME\Desktop\status.txt
zlanpiko backup create --full
zlanpiko . --to unit-001 --yes
```

## `zlanpiko .` (context import)

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
