# CS-10 — Build, Test & Release (3 Exes, Reproducible)

## 1. Build matrix

| Artifact | Command | Output |
| -------- | ------- | ------ |
| CLI+TUI | `go build ./cmd/zlanpiko` (via `scripts/build.bat`) | `dist/zlanpiko.exe` |
| Installer | `go build ./cmd/installer` | `dist/zlanpiko-installer.exe` |
| GUI frontend | `cmd /c "cd frontend & pnpm install & pnpm build"` | `dist-frontend/` (git-ignored) |
| GUI exe | `cmd /c "wails build -o zlanpiko-gui.exe"` (frameless per CS-03) | `dist/zlanpiko-gui.exe` |

Version injection (existing `configs/build.json` + `-ldflags`) extends to the
GUI: `app.Version/Commit/BuildDate` shown in titlebar + Settings + report
headers. `wails.json` version is generated from the same source — one bump
script, never three manual edits.

Release package:

```text
release/
  zlanpiko.exe  zlanpiko-gui.exe  zlanpiko-installer.exe
  README.txt  CHANGELOG.txt  checksums.txt
```

## 2. Test layers

- `go test ./...` — existing suites must stay green; new `internal/gui/*_test.go`
  covers DTO shapes, `RunCommand` parity, `GetMonth` bucketing, deadline-distance
  clamping. Golden-file one JSON sample per DTO to catch TS drift.
- `pnpm typecheck` (`tsc --noEmit`) + `vitest run` for strip/bar/chart pure
  functions (date formatting is display-only; logic fixtures come from Go goldens).
- E2E (manual checklist until automated): fresh install → import fixture →
  dashboard counts = CLI counts → edit deadline in calendar → backup/restore →
  update-simulation preserves DB (CS-09 §5).
- `go vet ./...` + `wails doctor` in CI/release script; record versions.

## 3. Scripts (Windows-first)

- `scripts/build-gui.bat`: pnpm install → pnpm build → wails build.
- `scripts/test.bat`: extends to `go test` + `pnpm typecheck/test` (via `cmd /c`).
- `scripts/package.bat`: assembles `release/` + checksums.
- All scripts assume PowerShell 5.1: shell out to `cmd /c` for Node tooling and
  print the ExecutionPolicy hint on failure.
