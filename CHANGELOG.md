# Changelog

Format follows Keep a Changelog; versions follow semantic versioning.

## [0.1.3] - 2026-10-05

Frameless desktop GUI (Wails port): same academic data, same CLI/TUI, plus a
desktop window with calendars, charts, a deadline timeline and a
terminal-style command bar. Schema stays v1; updating never wipes data.

- Frameless `zlanpiko-gui.exe` (Go backend + TypeScript frontend): custom
  titlebar, sidebar, status line, collapsible persistent command input with
  history, Tab-complete, `/clear` and TUI-parity grammar.
- Dashboard: academic overview, 3-week day rail, per-task deadline-distance
  bars, attention cards, overdue + 14-day lists.
- Units/Topics/Tasks views with full CRUD (archive preserves history,
  double-confirm deletes, one-click topic status cycling, task filters).
- Timeline (ISO-week lanes, overdue anchor) and month calendar (worst-of-day
  dots, agendas) with a task drawer: deadline editor (`YYYY-MM-DD`,
  `…THH:MM`, `… HH:MM`), status controls, delete.
- Analytics: coverage ring, R/P/U stack, tasks-by-status, 14-day deadline
  columns, per-unit bars, attention table (formulas stated on every card).
- Files: browse/search/open, mkdir/rename/delete, preview-first import.
- Settings: storage + version, plain-text phone report, backup
  create/verify/restore, in-app user guide (`F1`).
- Installer: optional GUI payload, pre-update backup (update aborts on
  failure), previous exes kept as `*.prev.exe`.
- Tooling: `scripts/build-gui.bat`, `scripts/test.bat` (Go + frontend),
  `scripts/package.bat` 3-exe release, `docs/cs/00-11` plan, `docs/22`
  testing and `docs/23` build/packaging guides.
- Fixes: Wails runtime bridge path and `--wails-draggable` resize/drag,
  `build-gui.bat` Wails CLI resolution without ambient PATH.
- Known limitations: global `/search` stubbed (as in TUI); restore replaces
  the database file (close other fronts first).

## [0.1.0] - 2026-10-03

First usable release: local-first academic management for Windows.

- Installer interactive drive/folder chooser (pick a drive, a folder, the
  drive root, or type a name/subpath; app folder created inside after
  confirmation), installer usage guide, first-run prompt fixes.

- Phase 1: Go module, app metadata, logging, domain vocabularies and
  validation, `zlanpiko help` / `zlanpiko version` entry point, docs.
- Rename: application `Academic Manager` -> `zlanpiko`; module
  `github.com/example/academic-manager` -> `zlanpiko`; executables
  `zlanpiko.exe` / `zlanpiko-installer.exe`; database `zlanpiko.db`;
  install paths under `%LocalAppData%\Zlanpiko` and `%AppData%\Zlanpiko`;
  env override `ZLANPIKO_DATA`.
- Phase 2: configuration (install pointer, data-root resolution, user prefs),
  SQLite storage (`modernc.org/sqlite` v1.60.1, schema v1, forward-only
  migrations, integrity check) and `app.Open` first-run initialisation.
- Phase 3: unit/topic/task services with folder skeletons and sidecars,
  `units|topics|tasks` CLI commands (text/json), overdue derivation,
  confirmation guards on deletes.
- Phase 4: user file operations (create/move/rename/delete guarded,
  metadata, search, OS-default open, DB/filesystem consistency scan),
  preview-first bulk importer with collision policies, `files` CLI commands
  and `zlanpiko .` context import.
- Phase 5: Bubble Tea TUI (dashboard, units, topics, tasks, files, settings;
  persistent command input with history; auto/16/none themes), shared
  coverage package, no-args launches TUI on a terminal.
- Phase 6: ISO-week timeline package, attention-label rules, timeline and
  analytics screens (with inline deadline editing), dashboard attention
  section, `timeline`/`analytics` CLI commands, cross-front parity tests.
- Phase 7: `config` and `maintenance verify [--repair]` commands, JSON
  `version`, strict flag validation, TUI `/verify` + `/config` parity and
  persisted command history.
- Phase 8: plain-text + JSON status reports, full/db-only backups with
  manifests, verify/restore/list commands, export→wipe→restore e2e test.
- Phase 9: `zlanpiko-installer.exe` (setup, adoption, shortcut, PATH),
  in-app first-run data-root setup.
