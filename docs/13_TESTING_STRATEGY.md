# 13 — Testing Strategy

Run: `.\scripts\test.bat` (= `go test ./...`), plus `go vet ./...` and
`gofmt -l .` before every checkpoint. Windows-only paths are tested on Windows;
pure logic stays OS-independent.

## Test layers

| Layer | Where | How |
| ----- | ----- | --- |
| Unit (domain/analytics/timeline) | `*_test.go` beside code | frozen `now`, table-driven; coverage math checked against hand-computed fixtures in `testdata/` |
| DB integration | `internal/database/*_test.go` | temp dir per test (`t.TempDir()`), migrate fresh, seed, assert; transaction rollback case included |
| Filesystem | `internal/filesystem/*_test.go`, `importer/*_test.go` | temp dirs only; collisions, `..` rejection, cross-volume-ish move (mocked by copy path), missing-file scan |
| Service | `internal/services/*_test.go`, `tasks/*_test.go` | DB+FS temp pair; full CRUD, archive, confirmed/unconfirmed deletes |
| CLI | `internal/cli/*_test.go` | invoke `Run(args)` with buffers; assert stdout/stderr + exit codes; import-preview asserts nothing copied without `--yes` |
| TUI model | `internal/tui/.../*_test.go` | drive `Update` with key msgs + `WindowSizeMsg`; assert screen switches, input persistence, no real TTY |
| Installer | `internal/installer/*_test.go` | temp install+data dirs; install → reinstall preserves DB rows and files; PATH edit against a fake env block (never the real registry in tests) |
| Migration | `database/migrate_test.go` | build v(N-1) DB from SQL seed, run runner, assert data + version bump |
| End-to-end | `testdata/e2e_*.bat` + Go test | the 10-step workflow below on a temp root |

## The 10-step acceptance workflow (scripted in Phase 8)

1. Create unit → 2. add topics → 3. change statuses → 4. assert coverage
   (`read/total`) → 5. create assignment → 6. set + edit deadline →
   7. import files → 8. check dashboard + timeline entries → 9. export report →
   10. backup → wipe → restore → re-export identical.

## Failure-matrix (must-have negative tests)

Invalid config JSON; missing/unwritable data root; migration from newer schema
(refusal); duplicate unit name / topic name / filename; missing linked file
(`missing=1` + report); interrupted copy (temp file left, no partial record
corruption); bad CLI args (exit 2); malformed JSON import; past-date deadline
(allowed-but-flagged, never an error).

## Fixtures

`testdata/`: `seed_9units.sql` (9 units mirroring the real user shape),
`import_sample/` (mixed extensions incl. skipped `.exe`), `malformed/` inputs,
`reports/expected_*.txt` for golden-file export tests. Golden files regenerate
via `UPDATE_GOLDEN=1` and are diffed in review, never silently.

## What cannot run here

Installer PATH/shortcut tests touching `HKCU`/Start Menu, and the OS-default
`open` action, are manual on a real Windows box and marked `// Manual:` in the
test file plus listed in `IMPLEMENTATION_STATUS.md`. Everything else runs in
`go test ./...`.
