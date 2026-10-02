# Implementation Status

## Current phase

**Phase 4 — Filesystem integration: DONE, awaiting user review + commit.**

- `internal/filesystem` extended: `WriteFile` (no-overwrite), `Mkdir`,
  `Rename` (same-dir, no separators), `Move` (protected sources refused;
  moving *into* inbox etc. allowed), `Delete` (protected root/skeleton/
  unit-roots refused, `--recursive` needed for trees), `Stat`, `List`,
  filename `Search` (units/ + inbox/), OS-default `Open` (files only),
  `Scan` (refreshes `missing` flags, reports unindexed; sidecars ignored).
- `internal/importer` (new): `Preview` (skips hidden/executables/oversize/
  links, collision rename|skip|overwrite, default dest `inbox/<date>`,
  record-link dests `unit`, `unit/topic-*`, `unit/task-*`, bare `inbox`) +
  `Execute` (context-cancelled, per-file journal, DB-row-first with
  compensation, sha256 recorded).
- `internal/cli`: `files` commands (list/import/move/rename/delete/open/
  search, mixed positional+flag grammar) and `zlanpiko .` context import;
  preview-first with `--yes` / TTY prompt / non-TTY refusal (D9).
- Bugs caught by tests and fixed: `Move` refusing protected *destinations*,
  `Scan` deadlock (UPDATE while rows open on the single-connection DB).
- Verification 2026-10-02: `gofmt`/`vet` clean, `go test ./...` green
  (11 test packages), `scripts/test.bat` exit 0, plus a live exe smoke test:
  unit → bulk import (preview refusal, then `--yes` with `.exe` skipped) →
  `files list` → `files search` (workflow step 7 covered).

- `internal/domain`: JSON tags on entities, UTC helpers (`Now/FormatTime/
  ParseTime`), `ConfirmRequiredError` + `ChildCounts`.
- `internal/filesystem` (new, minimal per D8): safe `Join` (escapes, drive
  letters, UNC, reserved names rejected), record dirs, unit/topic/task
  skeletons, atomic `WriteJSON` sidecars, collision-safe `MoveDir`.
- `internal/services`: unit CRUD + rename + archive + confirmed delete;
  topic CRUD + status transitions + rename + cross-unit move (id kept when
  free, fresh id on collision) + confirmed delete; per-unit sequential IDs
  never reused; archived units reject new topics/tasks.
- `internal/tasks`: task CRUD, `ParseDue` (local date/datetime/RFC3339),
  `Update` with tri-state due, `completed_at` follows stored status, kind
  change moves folders, derived-overdue `Filter`, confirmed delete.
- `internal/cli` (new, thin): `units|topics|tasks` subsets of `docs/08`
  (text via tabwriter + `--format json`), exit codes 0/1/2; `cmd/zlanpiko`
  routes non-help commands through `app.Open` (`--data-root` supported).
- Verification 2026-10-02: `gofmt`/`vet` clean, `go test ./...` green
  (10 test packages), `scripts/test.bat` exit 0, plus a live exe smoke test:
  unit → topic → status → task → `tasks deadlines` → `units list` all behave
  (9 workflow steps 1–6 covered; 7–10 need Phase 4/8).

- Rename applied first: app `zlanpiko`; module `zlanpiko`; `zlanpiko.exe` /
  `zlanpiko-installer.exe`; DB `zlanpiko.db`; `%LocalAppData%\Zlanpiko`,
  `%AppData%\Zlanpiko\app-config.json`; env `ZLANPIKO_DATA`; `AcademicData`
  storage-dir name kept (descriptive, not brand-derived). All docs, configs,
  scripts and code updated; `AGENTS.md` intentionally untouched (user's file).
- `internal/config`: pointer load/save, `ResolveDataRoot`
  (explicit > `ZLANPIKO_DATA` > pointer > error, never implicit), user prefs
  with defaults; malformed JSON is an error, never silently replaced.
- `internal/database` (sole driver importer, `modernc.org/sqlite` v1.60.1):
  open with busy_timeout/FK/WAL pragmas, single-writer, forward-only
  migration runner, newer-schema refusal both directions, `Verify`
  (integrity_check + version sanity).
- `migrations/0001_init.sql` + `migrations` embed package (schema v1,
  exactly per `docs/05`).
- `app.Open`: resolve → ensure skeleton (`database config backups exports
  reports units inbox`) → open → migrate → default `user.json`; `Close`.
- Go floor now `go 1.26.0` (raised by the sqlite dependency; toolchain 1.27).
- Verification 2026-10-02: `gofmt -l` clean, `go vet ./...` clean,
  `go test ./...` green (7 packages, incl. config round-trip/precedence,
  migrate-fresh, reopen-keeps-data, newer-schema refusal, FK enforcement,
  Open-creates-skeleton, unconfigured-root error); `scripts/test.bat` exit 0;
  `scripts/build.bat` produces `dist\zlanpiko.exe`, live `version` shows
  `zlanpiko 0.1.0-dev (commit …)`, unknown command exits 2.

## Completed tasks

- [x] Phase 0: repo + toolchain recon, 18 planning docs, consistency review.
- [x] Phase 1: module, metadata, logging, domain, entry point, scripts,
  LICENSE/README/CHANGELOG; verification green.
- [x] Rename to `zlanpiko` across code, configs, scripts and docs (D7).
- [x] Phase 2: config, database + migrations, `app.Open`; full verification
  green (see above).
- [x] Phase 3: domain tags/time/confirm-error, filesystem record API,
  services + tasks + CLI subset; full verification green (see above).
- [x] Phase 4: user file ops, consistency scan, preview-first importer,
  `files` CLI + `zlanpiko .`; full verification green (see above).

## Incomplete tasks / blockers

- None blocking. Open questions for the user (non-blocking, defaults assumed):
  1. `LICENSE` holder — MIT for "zlanpiko contributors" unless told otherwise.
  2. GitHub repo URL — placeholder `TBD` until provided.
  3. `AGENTS.md` still uses the old `academic.*` placeholder names — update
     it too, or leave as the original brief? (Left untouched for now.)

## Test status

- `go test ./...`: 11 test packages green, `migrations` has no test files
  (by design — exercised through `database` tests via the embedded FS).
- Manual-on-Windows items (installer PATH/shortcuts, OS `open`) are pre-listed
  in `docs/13` for later phases.

## Consistency review (Phase 0, still valid after rename)

Checked 2026-10-02 across all docs (re-swept after rename):

- Identity values (`zlanpiko.exe`, `AcademicData`, schema `1`, version
  `0.1.0-dev`) identical in 00/05/08/12/14/15. ✅
- Overdue derived-not-stored: 01/04/05/09/10 agree. ✅
- Empty-unit coverage `—`: 01/07/10 agree; no fake 100%. ✅
- DB-first crash order: 02/05/06/16 agree. ✅
- Sidecars = cache, DB wins: 04/05/06/16 agree. ✅
- stdlib CLI, no Cobra in v1: 02/03/08 agree (revisit trigger defined). ✅
- `modernc.org/sqlite` driver: 02/05 agree, rationale in DECISIONS.md D2. ✅
- No `Makefile` (`.bat` only): 03/14 agree, deviation from AGENTS.md sketch
  recorded in 03 + DECISIONS.md D5. ✅
- Time policy (store UTC RFC3339, local display): 01/05/09/10/11 agree. ✅
- `zlanpiko .` preview-first: 01/06/08/11/16 agree. ✅

## Next exact implementation step (Phase 5, after commit)

1. `internal/tui`: root Bubble Tea model, navigation registry, styles
   (Lipgloss, 16-colour fallback), persistent command input + status bar.
2. Screens: dashboard, units, topics, tasks, files (over the Phase 3–4
   services; read-only first, then mutations with confirmations).
3. TUI model tests (navigation, input persistence, no real TTY);
   `go get` Charm libs (bubbletea, lipgloss, bubbles); green `test.bat`.

Suggested commit message for this phase:

```text
feat: add file management, scan and preview-first import
```
