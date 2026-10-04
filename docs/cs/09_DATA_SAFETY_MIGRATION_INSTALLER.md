# CS-09 — Data Safety, Migration & Installer (Never Wipe the DB)

You are already using the app (11 units, 50 topics, live deadlines) — an update
that wipes or strands `zlanpiko.db` is the one unforgivable failure. This doc
makes preservation mechanical.

## 1. Compatibility guarantee

- GUI v0.2 opens **schema v1** databases read/write with zero migration.
  `SchemaVersion` stays `1`; `migrations/` gains nothing unless a bug forces it.
- Data-root resolution is identical across exes: `--data-root` flag →
  `$ZLANPIKO_DATA` → install pointer file → default. The GUI adds a Settings
  picker (`SetDataRoot`) that validates then re-opens — it never moves files.
- JSON sidecars (`unit.json`, `topic.json`, `task.json`) remain best-effort
  mirrors; SQLite stays the source of truth. GUI edits write DB first, sidecar
  second, and log sidecar failures without failing the edit.

## 2. Pre-update safety flow (installer + GUI)

1. Installer detects existing install (pointer file + `zlanpiko.db` present).
2. Automatic pre-update backup: SQLite-consistent copy + manifest
   `{appVersion, schemaVersion, timestamp}` into `backups/pre-update-<ts>/`.
   Abort the update if this backup fails — say so explicitly.
3. Replace program files only: `zlanpiko.exe`, `zlanpiko-gui.exe` (new),
   shortcuts, PATH entry. Never touch `AcademicData/` content.
4. Post-install verify: open DB, run `maintenance verify`, report
   `installed X → Y · data intact · N units · M topics`.
5. Rollback: installer keeps the previous exes as `*.prev.exe` one generation
   back; docs tell the user the one-click revert.

## 3. Installer changes (native-framed, as before)

- The installer exe keeps its existing console/native UI — **frameless does not
  apply to it**. Add exactly two screens: (a) `Install GUI? [yes]` (disk cost
  shown), (b) pre-update backup confirmation showing the backup path.
- PATH integration covers both exes (`zlanpiko`, `zlanpiko-gui`); re-running
  the installer repairs entries idempotently.
- Portable mode: if a `portable.json` sits beside the exes, all roots resolve
  relative — supported but not the default.

## 4. First-run with existing data

- If a DB exists, the GUI opens straight to the dashboard — no setup wizard,
  no "create storage" prompt. Wizard appears only on truly fresh machines.
- If the DB is from a newer schema (user downgraded), refuse to open with
  `DB_TOO_NEW + "install vX or restore backup Y"` instead of migrating down.

## 5. Tests that must pass before any release (also CS-10)

- `TestGuiOpensV1Database`: fixture DB with your shape (11 units/50 topics/
  mixed statuses/deadlines) opens, dashboard counts match CLI output exactly.
- `TestUpdatePreservesData`: simulate old→new exe swap + backup/restore cycle;
  checksum academic files before/after.
- `TestDowngradeRefuses`: newer-schema DB is rejected with actionable error.
