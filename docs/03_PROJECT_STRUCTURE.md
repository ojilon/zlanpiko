# 03 — Project Structure

Module path (placeholder until a repo is chosen): `github.com/example/academic-manager`.
Go minimum: 1.24 (developed with 1.27). All paths Windows-compatible.

```text
academic-manager/
├── AGENTS.md
├── README.md
├── CHANGELOG.md
├── LICENSE            # to be chosen in Phase 1 (MIT recommended, TBD)
├── go.mod / go.sum
├── .gitignore
│
├── cmd/
│   ├── academic/main.go    # thin: flags → app.Open → cli.Run or tui.Run
│   └── installer/main.go   # thin: → installer.Run
│
├── internal/
│   ├── app/        # Context wiring: config+DB+FS+logger; Open/Close
│   ├── config/     # JSON config load/save, data-root resolution
│   ├── database/   # connection, migrations runner, query helpers
│   ├── domain/     # entities, enums, validation, coverage/attention rules (no I/O)
│   ├── services/   # unit/topic use cases (CRUD, archive, delete rules)
│   ├── tasks/      # task CRUD, deadline edits, derived overdue
│   ├── timeline/   # week buckets, grouping/sorting (date logic lives HERE only)
│   ├── analytics/  # coverage + summary computations over domain types
│   ├── filesystem/ # safe paths, CRUD, import copy, metadata, missing-file scan
│   ├── importer/   # bulk-import preview + execute
│   ├── exporter/   # text + JSON reports
│   ├── backup/     # backup manifest, create/verify/restore
│   ├── cli/        # command tree, flag parsing, stdout/stderr, exit codes
│   ├── tui/
│   │   ├── app/        # root Bubble Tea model, screen switching
│   │   ├── screens/    # dashboard, units, topics, tasks, files, timeline, analytics, settings
│   │   ├── components/ # status bar, command input, tables, progress bars
│   │   ├── styles/     # Lipgloss theme (light-blue/grey), 16-colour fallback
│   │   └── navigation/ # routes, history stack, shortcut registry
│   ├── installer/  # setup steps, PATH integration, shortcuts
│   └── logging/    # slog bootstrap (file + stderr)
│
├── migrations/         # 0001_init.sql, 0002_....sql (forward-only, see docs/05)
├── configs/
│   ├── app.json        # placeholder identity: name, exe, version, schema, channel
│   └── build.json      # build flags, GOOS/GOARCH, outputs, package contents
├── scripts/
│   ├── test.bat build.bat package.bat release.bat
├── docs/               # this documentation + IMPLEMENTATION_STATUS.md + DECISIONS.md
├── assets/             # icons, sample files for tests/manuals (no code)
├── testdata/           # fixtures: sample DB seeds, import folders, malformed inputs
└── dist/               # git-ignored build outputs
```

## Package ownership (one-line contracts)

- `app`: owns process-wide wiring; no business rules.
- `domain`: owns entity shapes + pure rules; imports stdlib only.
- `services`, `tasks`: own validation-to-storage use cases; only place that
  writes units/topics/tasks.
- `timeline`: owns all week/day bucketing and date arithmetic for display.
- `analytics`: owns coverage/attention math; reads via `database` queries.
- `filesystem`: owns every `os.*` call under the data root.
- `database`: owns every `sql.*` call.
- `importer/exporter/backup`: own their flows end-to-end (plan → execute →
  verify), calling `filesystem`/`database`, never each other's internals.
- `cli`/`tui`: own presentation; call services, never `database`/`filesystem`.
- `installer`: owns setup; may call `config`, `database` (migrate), `filesystem`
  (mkdir) but not academic services.

## Dependency rules

1. `cmd/*` → `app` → (`cli`|`tui`|`installer`); `cmd` contains no logic.
2. `cli`, `tui/*` → `services`, `tasks`, `timeline`, `analytics`,
   `importer`, `exporter`, `backup`, `domain`, `config` — never `database`,
   `filesystem` directly.
3. `services`, `tasks`, `timeline`, `analytics`, `importer`, `exporter`,
   `backup` → `domain`, `database`, `filesystem`, `config`, `logging`.
4. `domain` → stdlib only. `database`, `filesystem`, `config`, `logging`
   → stdlib (+ sqlite driver for `database`) only.
5. `tui/screens` → `tui/components`, `tui/styles`, `tui/navigation`;
   screens never import each other (navigation registry wires them).

## Size guards

- No package > ~800 lines without a split proposal in DECISIONS.md.
- `tui/screens`: one file per screen. `services`: one file per entity.
- Do not create packages before they have a caller (AGENTS.md: no empty
  packages to match a diagram).

## Deviations from the AGENTS.md sketch

- `Makefile` dropped: Windows-first repo uses `scripts/*.bat`; a Makefile can
  be added later if a *nix CI needs it (recorded in DECISIONS.md).
- `LICENSE` content and `README.md` body are Phase 1 tasks (see roadmap).
