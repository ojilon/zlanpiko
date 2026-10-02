# Implementation Status

## Current phase

**Phase 1 — Go foundation: DONE, awaiting user review + commit.**

- `go.mod`: module `github.com/example/academic-manager`, floor `go 1.24`
  (toolchain 1.27; no dependency needs newer stdlib yet).
- New packages: `internal/app` (metadata + ldflags vars), `internal/logging`
  (slog bootstrap), `internal/domain` (enums, validation, overdue rule).
- Entry point `cmd/academic`: `help` (incl. `--help`/`-h`/`/help`) and
  `version`; unknown command → stderr + exit 2.
- `configs/app.json` + `configs/build.json`, `.gitignore`,
  `scripts/test.bat` + `scripts/build.bat`, MIT `LICENSE`, README, CHANGELOG.
- Verification 2026-10-02: `gofmt -l` clean, `go vet ./...` clean,
  `go test ./...` green (4 packages), `scripts/test.bat` exit 0,
  `scripts/build.bat` produces `dist/academic.exe` with commit SHA injected;
  live run of `version` / bare / unknown-command behaves as specified.

## Completed tasks

- [x] Phase 0: repo + toolchain recon, 18 planning docs, consistency review.
- [x] Phase 1: module, metadata, logging, domain, entry point, scripts,
  LICENSE/README/CHANGELOG; full verification green (see above).

## Incomplete tasks / blockers

- None blocking. Open questions for the user (non-blocking, defaults assumed):
  1. `LICENSE` choice — defaulting to MIT in Phase 1 unless told otherwise.
  2. GitHub repo URL — placeholder `TBD` until provided.
  3. Final app name — keeping "Academic Manager" placeholder.

## Test status

- No code yet: `go test ./...` not applicable. First tests arrive in Phase 1
  (domain validation) and Phase 2 (config + migrations).
- Manual-on-Windows items (installer PATH/shortcuts, OS `open`) are pre-listed
  in `docs/13` for later phases.

## Consistency review (Phase 0)

Checked 2026-10-02 across all docs:

- Identity values (`academic.exe`, `AcademicData`, schema `1`, version
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
- `academic .` preview-first: 01/06/08/11/16 agree. ✅

## Next exact implementation step (Phase 2, after commit)

1. `internal/config` (load/save JSON, data-root resolution per `docs/12`).
2. `internal/database` (open + migration runner + `migrations/0001_init.sql`
   per `docs/05`); wire first-run init into `internal/app`.
3. Config round-trip + migrate-fresh + reopen-idempotent + newer-schema
   refusal tests; green `scripts/test.bat`.

Suggested commit message for this phase:

```text
feat: implement Go foundation with domain and CLI entry point
```
