# Changelog

Format follows Keep a Changelog; versions follow semantic versioning.

## [Unreleased]

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
