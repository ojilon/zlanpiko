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
