# 23 — Building and Packaging (CLI, GUI, Installer, Release)

How to build each executable, assemble the release, and publish. All commands
run from the repository root on Windows. Toolchain: Go ≥ 1.26
(`D:\Dev\go`), Wails CLI v2.15.0, Node 24 + pnpm 11 (`D:\Dev\nodejs`,
store in `D:\Dev\pnpm\store`). See `docs/cs/10` for design rationale and
`configs/build.json` for the machine-readable manifest.

## 1. Version numbers

One source of truth: the `[version]` argument to the scripts below, injected
via `-ldflags` into `zlanpiko/internal/app` (`Version`, `Commit` from
`git rev-parse --short HEAD`, `BuildDate`). The same version lands in the
titlebar, Settings view, `wails.json` product version and report headers.
To cut version `0.2.0`:

1. Update `CHANGELOG.md` (`[Unreleased]` → `0.2.0`, date, notes).
2. Run the release (below) with `0.2.0`.
3. `git tag v0.2.0`, push, publish the GitHub Release with `release\*`.

Never hand-edit version strings in source; the `-dev` default
(`0.1.0-dev`) marks local builds.

## 2. Dev builds (fast iteration)

```cmd
:: CLI + TUI and installer (plain Go, console)
scripts\build.bat [version]

:: Frontend only (Vite dev server or production preview)
cmd /c "cd frontend & pnpm install & pnpm dev"
cmd /c "cd frontend & pnpm build & pnpm preview"

:: Full GUI executable (frameless)
scripts\build-gui.bat [version]
```

`build-gui.bat` does: `pnpm install` → `pnpm typecheck` → `pnpm build`
(output `dist-frontend/`) → `wails build -skipbindings -o zlanpiko-gui.exe`
with version ldflags → copies `build\bin\zlanpiko-gui.exe` to
`dist\zlanpiko-gui.exe`. Layout notes you must know:

- The Wails entry is root `main.go` (not `cmd/gui`): `wails build` compiles
  the project-root main package; a library package there produced a 52 KB
  dud during development.
- Always `-skipbindings`: the bindings helper exe is incompatible with this
  Windows setup. `frontend/src/types.ts` is hand-kept against
  `internal/gui/testdata/*.json` goldens instead (see `docs/22`).
- Plain `go build .` also works (embeds the last `dist-frontend/` build)
  but yields a **console** window; `wails build`/​`build-gui.bat` produce the
  proper GUI-subsystem exe.

## 3. Tests before packaging

```cmd
scripts\test.bat
```

Must be green: `go vet`, `gofmt` gate, `go test ./...`, frontend
typecheck + vitest. Never package on red.

## 4. Release packaging

```cmd
scripts\package.bat 0.2.0
```

What it does: wipes `dist/` + `release/`, version-builds `zlanpiko.exe`
and `zlanpiko-installer.exe`, calls `build-gui.bat`, then assembles:

```text
release/
  zlanpiko.exe
  zlanpiko-installer.exe
  zlanpiko-gui.exe
  README.txt
  CHANGELOG.txt
  checksums.txt   (SHA-256 of the five files above)
```

`scripts\release.bat 0.2.0` chains test → package → prints the manual
publish checklist (checksum spot-check, fresh-install rehearsal on a
machine **without** Go, smoke test, tag, GitHub Release). It never tags,
pushes or uploads by itself.

## 5. Installing / updating (what the installer does)

```cmd
release\zlanpiko-installer.exe
release\zlanpiko-installer.exe --install-dir D:\Tools\Zlanpiko --data-dir D:\AcademicData --yes
release\zlanpiko-installer.exe --yes --no-gui   :: skip the desktop payload
```

Behavior, in order: choose directories → adopt existing data (never wipe)
→ **pre-update backup** `backups/pre-update-<stamp>.zip` (update aborts if
it fails) → rotate old exes to `*.prev.exe` (one generation) → copy
`zlanpiko.exe` + `zlanpiko-gui.exe` (unless `--no-gui`; interactive runs
confirm `[Y/n]`) → init/migrate DB → save pointer → shortcut + user-PATH.
Safe to re-run. The installer itself stays a native console UI; only
`zlanpiko-gui.exe` is frameless.

## 6. Troubleshooting

| Symptom | Cause / fix |
| ------- | ----------- |
| `pnpm`/`tsc` “not recognized” in PowerShell | Use `cmd /c "…"`; `.ps1` shims are blocked by ExecutionPolicy. Global TS lives in `D:\Dev\pnpm\global` (`pnpm add -g typescript`). |
| `[ERR_PNPM_IGNORED_BUILDS] esbuild` | Run `pnpm approve-builds esbuild` once per machine (choice is committed in `frontend/pnpm-workspace.yaml`). |
| `wails build` bindings error (`wailsbindings.exe … not compatible`) | Expected here — always build with `-skipbindings` / `build-gui.bat`. |
| `'wails' is not recognized` from `build-gui.bat` | `go install` does not put `wails.exe` on PATH. The script now self-resolves via `go env GOBIN` → `GOPATH\bin` → `D:\Dev\go-workspace\bin`; if all fail it prints the one-time install command. For bare `wails …` calls, add `D:\Dev\go-workspace\bin` to your user PATH (Windows Settings → Environment Variables). |
| `wails.json … frontend:dev:watcher … type string` | That key was removed; dev-watcher config is not used (`wails dev` not required for this project). |
| `go build` embed error on fresh clone | Run `pnpm build` in `frontend/` first (or `build-gui.bat`); the embed needs `dist-frontend/` content. |
| GUI shows wrong/old UI after code change | `wails build` re-runs `pnpm build`; plain `go build .` does not — rebuild the frontend. |
| `package.bat` checksum step fails | A `release\` file is missing (often `assets\README.txt`); the script names it. |
