# 12 — Configuration and Installer

## Application identity (single source: `configs/app.json`)

```json
{
  "name": "zlanpiko",
  "exe": "zlanpiko.exe",
  "installer_exe": "zlanpiko-installer.exe",
  "id": "com.local.zlanpiko",
  "version": "0.1.0",
  "schema_version": 1,
  "default_storage_dir": "AcademicData",
  "repository": "TBD",
  "channel": "dev"
}
```

Code reads these at build time (version injected via linker flags, `docs/14`)
and at runtime; the name/exe never appear as literals outside `configs/` and
tests. Renaming = edit `app.json` + `build.json` + rebuild (procedure in
`docs/14`). Build settings (`GOOS/GOARCH`, flags, outputs, package contents)
live in `configs/build.json`.

## Configuration locations (four distinct roots, never mixed)

| What | Where | Example |
| ---- | ----- | ------- |
| Program install | user-chosen, default `%LocalAppData%\Zlanpiko\` | `zlanpiko.exe` |
| App config (install pointer) | `%AppData%\Zlanpiko\app-config.json` → `{data_root, install_dir}` | written by installer |
| Academic data root | user-chosen, default `%USERPROFILE%\AcademicData\` | layout in `docs/06` |
| Temp work | `os.TempDir()` / `%TEMP%` staging, cleaned after ops | — |

`internal/config`: `Load()` (install pointer → data root), `Save()`,
`ResolveDataRoot()` (explicit flag > env `ZLANPIKO_DATA` > pointer file >
error, never auto-create elsewhere). Data-local prefs
(`<root>/config/user.json`: theme, last screen) are separate from the install
pointer so reinstalls keep preferences.

## First-run flow

`zlanpiko.exe` with no pointer file (or missing data root) starts guided setup
in-place: show identity/version → pick data root (default suggested, free
choice incl. other drives) → create skeleton (`docs/06`) → init DB + migrate →
write pointer → continue into the app. No admin rights needed (user scope only).

## Installer (`zlanpiko-installer.exe`, same repo)

Interactive flow: identity/version → install dir → data dir (existing data
detected → **preserve + adopt**, never wipe) → create structure + DB →
optional Start-Menu shortcut + user-PATH entry → completion summary.

Each directory prompt accepts three answers:

1. **Enter** — take the shown default.
2. **A full path** — used as-is (e.g. `D:\Dev\Zlanpiko`).
3. **`c`** — open the interactive drive/folder chooser (needs a terminal):
   pick a drive (type and free space shown, unready drives refused), then
   pick one of the first 50 folders (`r` = drive root itself), or press
   `t` and type a folder name (covers folders beyond the first 50) or a
   deeper relative subpath like `Dev\Projects`. Typed input must already
   exist — the installer verifies it and only ever *creates* the final app
   folder (`Zlanpiko` for the program, `AcademicData` for data) inside your
   choice, after a `y/n` confirmation showing the full path.

Non-interactive use: `--install-dir`, `--data-dir` with `--yes`
(plus `--no-path` / `--no-shortcut` to skip OS integration). Without a
terminal and without flags the installer errors instead of hanging.

Update = run newer installer: detects versions, replaces program files only,
runs migrations, keeps data root, takes a pre-update safety backup, reports
old→new versions. No auto-update service in v1 (manual re-run; `docs/17`).
Implementation: `internal/installer` (flow), drive listing via
`golang.org/x/sys/windows`, chooser UI as a Bubble Tea model over
`ListDrives`/`ListRootDirs`/`ResolveTypedChoice` (all unit-tested without a
terminal).

## PATH integration

User-level `HKCU\Environment\PATH` prepend/append of the install dir (never
system-wide without explicit elevation + consent); installer prints the
"restart terminal" notice and is idempotent (re-run adds no duplicates,
repairs missing entries, preserves data).

## Existing-installation safety

Installer and first-run both check for: existing pointer, existing DB (adopt +
migrate), non-empty data dir without DB (ask: adopt as files vs. choose
another dir — never delete). Every destructive-adjacent choice defaults to the
preserving option.
