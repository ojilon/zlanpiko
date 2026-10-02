# Implementation Status

## Current phase

**Phase 0 — Repository and documentation: DONE, awaiting user review + commit.**

- Repo inspected: only `AGENTS.md` + `README.md` present; no code, no `go.mod`.
- Toolchain: `go1.27.0 windows/amd64`; `GOARCH=amd64`; module proxy reachable.
- Wrote `docs/00_PROJECT_OVERVIEW.md` … `docs/17_FUTURE_EXTENSIONS.md`
  (18 docs) + this file + `docs/DECISIONS.md`. No code changes.

## Completed tasks

- [x] Repo + toolchain recon (Go 1.27, empty repo, git branch `main`).
- [x] All 18 planning docs drafted per AGENTS.md §5 requirements.
- [x] Roadmap phases 0–10 with acceptance criteria (`docs/15`).
- [x] Consistency self-review (see bottom of this file).

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

## Next exact implementation step (Phase 1, after commit)

1. `go mod init github.com/example/academic-manager` (go ≥ 1.24).
2. Add `configs/app.json`, `configs/build.json`, `.gitignore`, `scripts/test.bat`.
3. Implement `internal/app` version vars + `internal/logging` + `internal/domain`.
4. Thin `cmd/academic` (`help`, `version`); domain unit tests; green
   `go build ./...` + `go test ./...` + `go vet ./...`.

Suggested commit message for this phase:

```text
docs: establish project architecture and roadmap
```
