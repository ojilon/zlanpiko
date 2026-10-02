# 21 — How v0.1.0 Works (Maintainer's Map)

What the released code actually does, package by package, so you can
maintain it without re-reading everything. Design rationale lives in
`docs/02` and `docs/DECISIONS.md`; this is the ground truth of the
implementation. Module `zlanpiko`, Go floor 1.26, Windows-only paths.

## 1. The two fronts and the shared core

```
cmd/zlanpiko  →  app.Open  →  cli.Run  →  services/tasks/timeline/...
                          →  tui.NewModel (Bubble Tea, alt screen)
cmd/installer →  installer.Run (installer pkg, plain stdio + one Tea picker)
```

Dependency rules (enforced by review, visible in imports):

- `tui/*` and `cli` call `services`, `tasks`, `timeline`, `analytics`,
  `importer`, `exporter`, `backup`, `maintenance`, `domain`, `config` —
  never `database` or `filesystem` directly (one exception: the CLI restore
  flow re-opens the DB handle after replacing the file).
- `services`, `tasks`, `timeline`, `analytics`, `importer`, `exporter`,
  `backup`, `maintenance` call `domain` + `database`/`filesystem` as needed.
- `domain` is stdlib-only: entities, vocabularies, validation, overdue and
  priority-rank math. `internal/config`, `database`, `filesystem`, `logging`
  are infrastructure. `migrations/` embeds `0001_init.sql` (schema v1).

## 2. Startup sequences

**CLI command** (`cmd/zlanpiko/main.go`): `help`/`version` need no storage.
Everything else: strip `--data-root` → if none given, `ensureConfigured`
(resolve: flag > `$ZLANPIKO_DATA` > pointer file; unconfigured + TTY prompts
for a data dir and saves the pointer; unconfigured + piped errors) →
`app.Open` (ensure skeleton dirs, open `database/zlanpiko.db` with
busy_timeout/FK/WAL pragmas, single connection, run pending migrations,
write default `user.json`) → `cli.Run` dispatch → exit 0/1/2.

**TUI** (bare `zlanpiko` on a TTY; piped stdout prints help instead):
same open sequence, then `tea.NewProgram(tui.NewModel(ctx),
tea.WithAltScreen())`. The root model owns tabs, status bar, persistent
input + history (`config/history.json`, cap 100), and `/` dispatch; each
screen reloads from services on activation and after every mutation.

**Installer** (`cmd/installer` → `installer.Run`): identity banner →
install/data directory resolution (flags, or prompt: default / typed path /
`c` for the Tea drive→folder→confirm picker) → copy `zlanpiko.exe` from
beside itself → `app.Open` init → save pointer → Start-Menu `.lnk` via
stock `cscript` → user-PATH via one idempotent PowerShell/.NET call.
Re-runs adopt existing data; PATH/shortcut failures print manual steps and
continue. `SkipOS`/`StartMenuDir` seams keep tests off the real machine.

## 3. Data: what lives where

- **SQLite** (`<root>/database/zlanpiko.db`) is the source of truth:
  `units`, `topics` (PK `(unit_id, id)`), `tasks` (same), `files`
  (`rel_path` unique, `missing` flag), `schema_meta`. CHECK constraints mirror
  the domain vocabularies; FK cascades delete children rows (services still
  confirm first and remove folders themselves).
- **Folders** (`units/unit-001/{unit.json,topics/,assignments/,notes/}`,
  `topics/topic-001/{topic.json,reading/,research/,summaries/}`,
  `tasks/task-001/{task.json,materials/,drafts/,submissions/}`, `inbox/`,
  `backups/`, `exports/`, `reports/` reserved). Folder names are stable IDs:
  `unit-NNN` global, `topic-NNN`/`task-NNN` per unit, `max+1` allocation,
  never reused. Task group dirs pluralise the kind (`assignments/`,
  `coursework/`, `tests/`, `examinations/`, `projects/`, `other/`).
- **Sidecars** (`unit.json` etc.) mirror rows, rewritten on every mutation
  after the DB commit (DB-first crash order: a crash leaves a detectable
  `missing` row, never an invisible orphan). `maintenance verify` compares
  `updated_at` and rewrites with `--repair`.
- **IDs in UIs**: `unit-001/topic-003` display where ambiguous; scripts
  should use IDs, never names.

## 4. Rules the code enforces (and where)

- Statuses: units `active|archived`; topics `unread|pending|read` (manual
  transitions only — folder opens never call `SetTopicStatus`); tasks
  `not_started|in_progress|completed|submitted`, `overdue` derived
  (`IsOverdue`: dated + past + still open). `completed_at` follows stored
  status automatically.
- Coverage (`analytics`): `read/total × 100`, empty unit → `—`.
- Attention (`analytics.AssessUnit`, thresholds 3/7/14 d, 50/70%):
  overdue-task or unread-high-topic-before-exam → Overdue; nothing open +
  100% → Completed; exam ≤7 d + <50% → At Risk; deadline ≤14 d +
  (<70% or open work) → Attention; else On Track; no topics → None.
  Topic-level (`AssessTopic`, pure): unread/high near exams escalate, read
  near exams shows Review. Half-threshold clause deliberately unimplemented
  (D11).
- Timeline (`timeline.Build`): ISO weeks, local midnights; overdue items
  anchor in a leading section (deduped against in-week hits); sort by due,
  priority rank, title; dateless excluded.
- Import (`importer.Preview` pure plan → `Execute` journal): skips hidden,
  executables (`.exe/.msi/.bat/.cmd/.com/.scr/.ps1/.dll`), >100 MB
  (opt-out), links; collisions rename-`2` by default (`skip` drops,
  `overwrite` needs `--yes` even on TTY); copies stream to temp + hash,
  insert DB row, then rename into place. Confirmation model (D9): preview
  always, `--yes` or TTY prompt, refusal otherwise.
- Backup (`backup.Create`): `VACUUM INTO` snapshot (never raw-copy),
  zip of `database/` + `files/<units,inbox,user.json>` + `manifest.json`
  (hashes, versions, kind); self-verifies, refuses existing names.
  `Restore`: verify-first, newer-schema refusal, zip-slip guard, same-volume
  staging, stale WAL cleanup; empty DB counts as empty target, else
  `--overwrite-data` with a live-handle safety backup first.
- Deletes: counts checked, `ConfirmRequiredError` without explicit consent;
  archive offered by message; protected roots (`/`, skeleton dirs, unit
  folders) refused by `filesystem` (record subtrees go through services).
- Time: UTC RFC 3339 in storage, local in display/input; weeks are local.

## 5. Formats, codes, files

- CLI text tables via `text/tabwriter`; `--format json` marshals the domain
  structs (snake_case tags) — v1 additive-stability promise covers exports
  and list/show JSON. Text report: 72 columns, CRLF-safe plain text.
- Exit codes 0/1/2 (usage errors print the command's help to stderr).
- No persistent log file in v0.1.0: `slog` text logs go to stderr only.
- TUI themes `auto|16|none` (`styles.Build`; `NO_COLOR` honoured); table
  selection is reverse-video when colourless; <90 cols drops unit/kind
  columns; <80×24 shows a resize notice.

## 6. Scripts reference (`scripts\`)

All run from the repo root on stock Windows (Go toolchain for build/test;
only PowerShell-out-of-the-box beyond that for checksums/dates).

| Script | Does |
| ------ | ---- |
| `test.bat` | `go vet ./...`, `gofmt -l` gate, `go test ./...`. Exit non-zero on any failure. The pre-commit gate. |
| `build.bat [version]` | Dev builds of **both** exes into `dist\` (default `0.1.0-dev`), injecting version + git SHA via ldflags. `dist\` is git-ignored. |
| `package.bat [version]` | Full release packaging: wipes `dist\`+`release\`, versioned builds with version+SHA+ISO date, assembles `release\` (`zlanpiko.exe`, `zlanpiko-installer.exe`, `README.txt` from `assets\`, `CHANGELOG.md` → `CHANGELOG.txt`, PowerShell SHA-256 `checksums.txt`), lists contents. Default version `0.1.0`. |
| `release.bat [version]` | `test.bat` → `package.bat` → prints the manual publish checklist. **Never tags, pushes or uploads by itself.** |

`configs/build.json` is the machine-readable twin of the build settings
(module, targets, ldflags vars, outputs, package contents) for any future
external build manager.

## 7. Pushing a release (manual, step by step)

Prerequisites: clean tree, `CHANGELOG.md` section for the version,
`configs/app.json` version bumped if releasing a new number.

1. `.\scripts\release.bat 0.1.0` — must exit 0 end to end.
2. Verify: `certutil -hashfile release\zlanpiko.exe SHA256` against
   `release\checksums.txt`.
3. Rehearse on a machine without Go (or isolated temp dirs + temp
   `%APPDATA%`): install from `release\`, then CRUD, timeline, export,
   `backup create --full` → wipe → `backup restore --yes` → data intact.
4. Commit anything left over; check `git status` clean.
5. `git tag v0.1.0` (annotated `-a` with the changelog line if you like).
6. `git push origin main v0.1.0`.
7. GitHub → Releases → Draft new release from tag `v0.1.0`: title the
   version, paste the `CHANGELOG.md` section as notes, attach **all five**
   `release\` files, publish.
8. After release: bump `configs/app.json` version to the next `-dev` (code
   default is `0.1.0-dev` until then) and open a new `[Unreleased]`
   changelog section.

Versioning is semver (`docs/14`); the in-app `version`/`version --format
json` commands report name, version, commit, build date and schema version
for any build.
