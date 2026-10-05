# 22 — Testing the GUI Build (Go + Frontend + Installer)

How to run every check for the Wails GUI. Run from the repository root on
Windows. PowerShell 5.1 blocks the `.ps1` shims of Node tools, so all
`pnpm`/`npx` calls below go through `cmd /c` (or use `*.cmd` directly).
Fresh terminals pick up `D:\Dev\pnpm\global` from the user PATH; already-open
ones need `set PATH=%PATH%;D:\Dev\pnpm\global` first (cmd) for global tools.

## 1. The one command: `scripts\test.bat`

```cmd
scripts\test.bat
```

It runs, in order: `go vet ./...`, a `gofmt` cleanliness gate
(`internal`, `cmd`, root `main.go`), the full `go test ./...`, then in
`frontend/`: `pnpm install`, `pnpm typecheck` (`tsc --noEmit`), `pnpm test`
(`vitest run`). Anything red aborts the script. This is the gate for every
commit on the GUI branch and for every release (see `docs/23`).

## 2. Go tests for the GUI layer (`internal/gui`)

```cmd
go test ./internal/gui/ -count=1 -v
```

What they cover:

| Test | What it proves |
| ---- | -------------- |
| `TestGetDashboardMatchesAnalytics` | dashboard counts equal `analytics.Summarize`/`UnitCoverages`/`Upcoming` recomputed independently; fixture expectations (Periodicity R1/P1/U3 = 20%, empty Ecology = `—`, `in 2d`, overdue list) |
| `TestGetWeekMatchesTimeline` | week items equal `timeline.Build`; bad weeks rejected; `GetWeekOffset` lands on the right ISO week; completed tasks read `done` |
| `TestGetMonthGrid` | 42 Monday-start cells, in-month count, item counts, overdue total, bad-month validation, `GetMonthOffset(0)` |
| `TestGetDayItems` | day bucket + overdue-anchored attribution, date validation |
| `TestGetTaskDetail` | drawer fields, `NOT_FOUND` code for unknown IDs |
| `TestRunCommandNav` / `TestRunCommandMutations` | nav parity with the TUI grammar, help text, typo suggestion, topic/task mutations round-trip through commands |
| `TestFilesRoundTrip` | list/search/mkdir/rename/delete against a temp data root |
| `TestImportPreviewConfirm` | preview-then-confirm copies one file; plan tokens are single-use |
| `TestReportAndBackup` | text+JSON reports mention fixture data; db-only backup creates, lists and verifies; `SetDataRoot` rejects missing dirs |

The fixture (`seedPhaseB`) pins `now` to 2026-10-04 12:00 local and pins
`created_at` via SQL, so `progress_pct`, distances and week buckets are
deterministic.

### Golden files

`internal/gui/testdata/{dashboard,week,month,analytics}.json` freeze the exact
DTOs the TypeScript side mirrors (`frontend/src/types.ts`). Times serialize
with the machine's local UTC offset, so goldens are host-TZ specific.

- After a **deliberate** DTO change: `$env:UPDATE_GOLDEN="1"; go test ./internal/gui/`,
  inspect the diff, then re-run without the flag.
- After an **accidental** mismatch: do not regenerate — fix the code.
- `frontend/src/types.ts` must be updated by hand in the same commit; the
  goldens are the machine-readable contract reviewers diff against.

## 3. Frontend checks

```cmd
cmd /c "cd frontend & pnpm typecheck & pnpm test & pnpm build"
```

- `typecheck`: strict `tsc --noEmit` (unused locals fail the build).
- `test`: vitest pure-function tests (`dashboard.test.ts`: HTML escaping,
  status-line format). Backend numbers are covered by Go goldens, not here.
- `build`: Vite production build into `../dist-frontend/` (+ `.gitkeep`
  restore via `postbuild` so plain `go build ./...` keeps working).
- First run ever needs `pnpm approve-builds esbuild` (committed in
  `frontend/pnpm-workspace.yaml`; one-time per machine).

## 4. GUI smoke test (no test framework — a real window)

```powershell
$tmp = Join-Path $env:TEMP "zlanpiko-smoke"
.\build\bin\zlanpiko-gui.exe --data-root $tmp
```

Expect: frameless window (custom titlebar, drag works, `— ▢ ✕` work),
dashboard numbers matching `zlanpiko --data-root $tmp analytics`, rail ticks
on deadline days, timeline/calendar navigation, working command bar
(`/help`, `/timeline next`, `/topics status …`). Close with ✕; data stays in
`$tmp`. Headless check used in automation: start with stderr redirected,
sleep 12 s, assert the process is alive and the skeleton
(`database/`, `units/`, `backups/` …) was created.

Seed data for a meaningful smoke run:

```cmd
go run ./cmd/zlanpiko --data-root %TEMP%\zlanpiko-smoke units add --name Periodicity --code SED:2103
go run ./cmd/zlanpiko --data-root %TEMP%\zlanpiko-smoke tasks add --unit unit-001 --title "class presentation" --kind assignment --due 2026-10-06
```

## 5. Installer tests

```cmd
go test ./internal/installer/ -count=1 -v
```

`gui_install_test.go` proves the data-safety contract: reinstall writes
`backups/pre-update-<stamp>.zip` **before** touching exes, rotates previous
exes to `*.prev.exe`, preserves user files, installs `zlanpiko-gui.exe`
unless `--no-gui`, and skips the GUI cleanly when asked.

## 6. What is NOT automated yet

- Pixel-level window assertions (titlebar drag, button focus, DPI) — manual
  checklist in §4.
- `wails build` itself (needs the Wails CLI + WebView2) — covered by
  `docs/23`; CI should run `scripts\build-gui.bat` on a Windows runner.
- Restore-over-open-DB in GUI tests (replaces the file under the test
  handle) — covered by the CLI backup/restore suite instead.
