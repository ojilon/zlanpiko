# Implementation Status

## Current phase

**Phase 2 — Configuration and storage: DONE, awaiting user review + commit.**

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

## Incomplete tasks / blockers

- None blocking. Open questions for the user (non-blocking, defaults assumed):
  1. `LICENSE` holder — MIT for "zlanpiko contributors" unless told otherwise.
  2. GitHub repo URL — placeholder `TBD` until provided.
  3. `AGENTS.md` still uses the old `academic.*` placeholder names — update
     it too, or leave as the original brief? (Left untouched for now.)

## Test status

- `go test ./...`: 6 test packages green, `migrations` has no test files
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

## Next exact implementation step (Phase 3, after commit)

1. `internal/services` (unit CRUD, rename, archive/unarchive, confirmed
   delete) + `internal/tasks` (task CRUD, deadline edits, derived overdue).
2. Sidecar writes (`unit.json`, `task.json`) in the same service calls.
3. `units|topics|tasks` CLI read/write commands (thin, over services).
4. Service tests (CRUD, archive, confirmed/unconfirmed deletes, overdue
   derivation); green `scripts/test.bat`.

Suggested commit message for this phase:

```text
feat: rename to zlanpiko; implement configuration and SQLite storage
```
