# 18 — CLI User Guide (v0.1.0)

The command line is the complete interface: everything the TUI can do (and a
few things it cannot) works here, scriptably, from any directory. This guide
is written for daily use. Design background lives in `docs/08`; known gaps
are collected in `docs/20`.

## Setup and the data root

After installing (see `assets/README.txt`), every command needs to know
where your academic data lives. Resolution order:

1. `--data-root PATH` flag (per command),
2. `$ZLANPIKO_DATA` environment variable,
3. the install pointer (`%AppData%\Zlanpiko\app-config.json`, written by the
   installer or first-run setup).

If none is configured you get an error telling you how to fix it — never a
silent second database. Useful patterns:

```powershell
# one-off against another data root
zlanpiko --data-root D:\School units list

# persist for the session
$env:ZLANPIKO_DATA = "D:\School"
zlanpiko units list

# permanently repoint this machine
zlanpiko config set data_root D:\School
```

`--data-root` works on every command (it is stripped before dispatch).

## Conventions

- `--format text|json` on every list/show/report command. Text is for
  humans, JSON for scripts (stable field names, v1 schema).
- `--yes` is required for destructive and bulk writes. Without it the
  command prints what it *would* do and refuses (exit 1), never hangs.
- Unknown flags are rejected (exit 2) — a typo like `--unt` fails loudly
  instead of silently ignoring your filter.
- Exit codes: `0` success, `1` runtime problem (or "issues remain" from
  `maintenance verify`), `2` usage error. Errors go to stderr, data to
  stdout, diagnostic logs to stderr — safe to pipe stdout into files.
- IDs: units are `unit-001`, `unit-002`, … globally. Topics and tasks are
  numbered **per unit** (`unit-001/topic-003`). IDs never change, even when
  you rename things — use them in scripts, not names.

## Course units

```powershell
zlanpiko units list                        # active units (+ topics/tasks counts)
zlanpiko units list --archived            # include archived
zlanpiko units add --name "Linear Algebra" --code MATH201
zlanpiko units rename --unit unit-001 --name "Advanced Linear Algebra"
zlanpiko units archive --unit unit-001    # hide, keep history
zlanpiko units unarchive --unit unit-001
zlanpiko units show --unit unit-001
zlanpiko units delete --unit unit-001 --yes   # only with --yes when it has content
```

Deleting a unit that still has topics, tasks or files fails with a message
telling you the counts — re-run with `--yes`, or `archive` instead to keep
the history. Creating a unit also creates its folders
(`units/unit-001/` with `topics/` and `notes/`).

## Topics and reading progress

Status is always one of `unread | pending | read`, and only ever changes
when you say so — opening folders never marks anything read.

```powershell
zlanpiko topics list --unit unit-001
zlanpiko topics list --unit unit-001 --status unread
zlanpiko topics add --unit unit-001 --name Eigenvalues --priority high
zlanpiko topics status --unit unit-001 --topic topic-001 --status read
zlanpiko topics rename --unit unit-001 --topic topic-001 --name "Eigenvalues+vectors"
zlanpiko topics move --unit unit-001 --topic topic-001 --to unit-002
zlanpiko topics delete --unit unit-001 --topic topic-001 --yes
```

Priorities: `low | normal | high` (default `normal`). Moving a topic keeps
its number when free in the new unit, otherwise it gets the next free one —
the output tells you. You cannot add topics to an archived unit (unarchive
it first).

## Tasks, deadlines, tests, exams

Kinds: `assignment | coursework | test | examination | project | other`
(default `assignment`). Stored statuses: `not_started | in_progress |
completed | submitted`. `overdue` is never stored — it is derived from
deadline + status whenever you look, so nothing ever goes stale.

Dates accept `YYYY-MM-DD` or `YYYY-MM-DDTHH:MM` (your local time) or full
RFC 3339. Empty `--due` on edit clears the deadline (`--due=`).

```powershell
zlanpiko tasks add --unit unit-001 --title "Problem set 4" --kind assignment --due 2026-10-09T23:59
zlanpiko tasks list --status overdue
zlanpiko tasks list --unit unit-001 --kind test
zlanpiko tasks edit --unit unit-001 --task task-001 --due 2026-10-12 --priority high
zlanpiko tasks edit --unit unit-001 --task task-001 --due=     # clear deadline
zlanpiko tasks complete --unit unit-001 --task task-001
zlanpiko tasks submit --unit unit-001 --task task-001
zlanpiko tasks deadlines                                        # next 14 days
zlanpiko tasks deadlines --days 30 --unit unit-001
zlanpiko tasks delete --unit unit-001 --task task-001 --yes
```

Completing/submitting stamps the completion time and instantly clears any
overdue state; the record stays in history. Every task gets a folder
(`units/unit-001/assignments/task-001/` with `materials/`, `drafts/`,
`submissions/`); changing kind moves the folder with it.

## Timeline and analytics

```powershell
zlanpiko timeline                        # this ISO week
zlanpiko timeline --offset -1            # last week
zlanpiko timeline --week 2026-W41
zlanpiko analytics                       # coverage + attention table
zlanpiko analytics --unit unit-001
```

The timeline groups by day, sorts by time then priority, anchors overdue
items in a leading section, and skips dateless tasks. Analytics prints
`read/total` coverage (`—` for units with no topics, never a fake number)
plus one of `Overdue | At Risk | Attention | On Track | Completed` with the
rule inputs documented in `docs/10` — planning hints, not predictions.

## Files

All paths are **root-relative** (`inbox`, `units/unit-001/notes`). Only
`import` reads from outside the data root.

```powershell
zlanpiko files list inbox
zlanpiko files list units/unit-001
zlanpiko files search eigenvalues --limit 20
zlanpiko files import C:\Downloads\paper.pdf --to unit-001
zlanpiko files import C:\Downloads\reader --to unit-001 --recursive --yes
zlanpiko files move inbox/paper.pdf units/unit-001/notes
zlanpiko files rename inbox/paper.pdf paper-final.pdf
zlanpiko files open units/unit-001/notes/paper-final.pdf   # default app
zlanpiko files delete inbox/draft.txt --yes
zlanpiko files delete inbox/old --recursive --yes
```

There is **no `files mkdir`** in v0.1.0: new folders come from creating
units/topics/tasks, or appear as import destinations. (Tracked as a v0.1.1
item in `docs/20`.)

### Import rules (preview-first, always)

Bulk import never copies blindly. Without `--yes` (and without an
interactive terminal to confirm) it prints the plan — every file, its
destination, collisions and skips — and copies nothing. Skipped, with
reasons: hidden files, executables (`.exe/.msi/.bat/.cmd/.com/.scr/.ps1/
.dll`), files over 100 MB (override with `--include-large`), and links.
Name collisions default to `name-2.ext` renaming; `--collision skip` skips
them; `--collision overwrite` replaces but additionally requires `--yes`.

`DEST` is a unit id, `unit/topic-*`, `unit/task-*`, a directory
(`inbox/2026-10-03`), bare `inbox`, or omitted (`inbox/<today>`).

### The dot command

```powershell
cd C:\Downloads\reader
zlanpiko . --to unit-001 --yes
```

imports the **current directory** (non-recursive unless `--recursive`),
with the same preview/confirm rules. Without `--yes` on a non-interactive
terminal it prints the preview and refuses.

## Reports, backups, maintenance, config

```powershell
zlanpiko export                                  # phone-friendly text to stdout
zlanpiko export --unit unit-001 --format json
zlanpiko export --out $HOME\Desktop\status.txt   # files are never overwritten

zlanpiko backup create --full                    # timestamped zip in backups/
zlanpiko backup create                           # database only
zlanpiko backup list
zlanpiko backup verify backups\backup-full-20261003-120000.zip
zlanpiko backup restore <file> --yes                        # empty target only
zlanpiko backup restore <file> --yes --overwrite-data       # safety backup first

zlanpiko maintenance verify                      # exit 1 while issues remain
zlanpiko maintenance verify --repair             # rewrites sidecars, clears temps

zlanpiko config show
zlanpiko config set theme 16                     # auto | 16 | none
zlanpiko config set data_root D:\School          # repoints only: move files yourself
zlanpiko config path                             # prints the root (for scripts)
zlanpiko version
zlanpiko version --format json
```

Restore details that bite: the target database must be absent/empty or you
pass `--overwrite-data` (then a safety backup is recorded automatically);
newer-schema backups are refused; the report in `export` is generated from
live rows at that instant.

## Worked session: setting up a semester

```powershell
$env:ZLANPIKO_DATA = "$HOME\AcademicData"
zlanpiko units add --name "Linear Algebra" --code MATH201
zlanpiko units add --name "Mechanics" --code PHYS110
zlanpiko topics add --unit unit-001 --name Eigenvalues --priority high
zlanpiko topics add --unit unit-001 --name "Systems of equations"
zlanpiko tasks add --unit unit-001 --title "Problem set 4" --kind assignment --due 2026-10-09T23:59
zlanpiko tasks add --unit unit-001 --title Midterm --kind test --due 2026-10-20
zlanpiko files import $HOME\Downloads\linalg-notes --to unit-001 --recursive --yes
zlanpiko tasks deadlines --days 21
zlanpiko backup create --full
```

## Troubleshooting

| Symptom | Meaning / fix |
| ------- | ------------- |
| `config: no data root configured` | Set `--data-root`, `$ZLANPIKO_DATA`, or run the installer / first-run setup |
| `unit "unit-009" not found` | Check `units list` — IDs are per the messages that created them |
| `unit ... is archived` | `units unarchive` first; archived units reject new topics/tasks |
| import copies nothing | Read the preview: skips list reasons; add `--yes` |
| `refusing to overwrite ...` | Rename `--out`/backup target, or delete the old file deliberately |
| `schema v2 is newer than this build` | The data was touched by a newer app — update, never downgrade |
| `database: ... is locked`/busy | Only one writer at a time — close the other TUI/CLI first |
