# 01 — Requirements

Legend: **[M]** must-have for v1, **[D]** deferred to a later release (see `docs/17`).

## Functional requirements

### Course units [M]

- FR-U1: Create a unit (name required; code, description, colour optional).
- FR-U2: Rename a unit without breaking folder references (stable folder IDs,
  see `docs/06`).
- FR-U3: Archive / unarchive a unit. Archived units are hidden by default but
  kept with full history.
- FR-U4: Delete a unit only with explicit confirmation when it has topics,
  tasks or files; offer archive as the default alternative.
- FR-U5: No hard limit on unit count (the initial user has 9; the app must not
  assume 9).
- Acceptance: CRUD + archive covered by service tests; delete with children
  requires `confirm=true` (CLI flag / TUI prompt) and removes DB rows and
  folders atomically-as-possible (see `docs/16`).

### Topics and reading progress [M]

- FR-T1: Topics belong to exactly one unit; create, rename, move between units,
  delete (same confirmation rule as units).
- FR-T2: Reading status is exactly one of `Unread | Pending | Read`.
- FR-T3: Status transitions are manual only — opening a folder or file must
  NOT change status.
- FR-T4: Filter topics by status; sort by name / status / updated time.
- FR-T5: Optional per-topic priority (`low | normal | high`) and notes.
- Acceptance: transition test `Unread → Pending → Read → Pending` persists;
  a "folder opened" event leaves status unchanged.

### Assignments, coursework, tests, examinations [M]

- FR-A1: Task record: title, unit, type
  (`assignment | coursework | test | examination | project | other`), status,
  deadline (date, optional time), priority, description, notes.
- FR-A2: Stored statuses: `Not Started | In Progress | Completed | Submitted`.
  `Overdue` is **derived** (deadline passed + not Completed/Submitted), never
  stored — see `docs/09`.
- FR-A3: Deadlines editable; every edit propagates to timeline, dashboard and
  analytics immediately (single recompute path, `docs/10`).
- FR-A4: Completed/Submitted work stays visible in history and counts as done
  in analytics.
- Acceptance: create → set deadline → complete → submit lifecycle test;
  overdue appears automatically after the deadline without user action.

### Files [M]

- FR-F1: Create/rename/move/delete files and folders inside the data root.
- FR-F2: Import files from any local folder (single + bulk with preview and
  conflict report before copying).
- FR-F3: View file metadata; open with the OS default application.
- FR-F4: Full-text filename search across the data root (content search is [D]).
- FR-F5: Never silently overwrite: name collisions require rename/skip/overwrite
  choice; deletes of non-empty folders require confirmation.
- Acceptance: filesystem integration tests in temp dirs (see `docs/13`).

### Deadlines, timeline [M]

- FR-D1: Week view combining all task types; prev/next week navigation.
- FR-D2: Group by date, sort by time then priority; overdue and completed
  markers; select item → detail → edit → jump to unit/folder.
- Acceptance: a task edited from the timeline shows the new date on the
  dashboard without restart.

### Analytics [M]

- FR-N1: Per-unit and overall reading coverage = `read / total × 100`.
  Empty unit displays `—` (no division by zero, never fake 100%).
- FR-N2: Attention labels from transparent rules (`On Track | Attention |
  At Risk | Overdue | Completed`), defined in `docs/10`.
- FR-N3: Every number shown must be reproducible from DB rows via the formula
  in `docs/10`.
- Acceptance: analytics unit tests with fixed fixtures match hand-computed values.

### Reports, backup [M]

- FR-R1: Plain-text academic status report (overall + per-unit), generatable
  from TUI and CLI, suitable for copying to a phone.
- FR-R2: JSON export of the same data for future integrations.
- FR-R3: Backup (DB consistent copy + files + manifest) and restore with
  verification; no silent overwrite of existing backups.
- Acceptance: export → wipe → restore → re-export produces identical report.

### TUI [M]

- FR-I1: Screens: dashboard, units, topics, tasks, files, timeline, analytics,
  settings. Persistent command input + status bar on all screens.
- FR-I2: Full keyboard operation; every destructive action confirmable.
- FR-I3: Usable at 80×24 with 16 colours; adapts to resize.
- Acceptance: TUI model tests for navigation + command parsing (no real
  terminal needed).

### CLI [M]

- FR-C1: Same operations as TUI services, non-interactive with explicit flags.
- FR-C2: Safe bulk import: `zlanpiko .` (or `import <path>`) shows a preview
  (files, destination, collisions) and imports nothing without `--yes`.
- FR-C3: Works from any working directory; never creates a second database
  implicitly (resolves the configured data root, fails loudly if missing).
- Acceptance: CLI tests for help output, error codes, and import-preview flow.

### Installer [M]

- FR-S1: `zlanpiko-installer.exe`: shows identity/version, chooses install
  and data directories, creates structure, initialises DB, optional user-PATH
  entry, re-runnable, preserves existing data.
- Acceptance: installer test in temp dirs: install → reinstall keeps data.

## Non-functional requirements

- NFR-1 Local-first: all features work offline; no network calls in v1.
- NFR-2 Data safety: no silent loss/overwrite; transactions for related DB
  writes; recoverable file ops (see `docs/16`).
- NFR-3 Resource use: both executables start fast on low-end Windows; avoid
  heavy dependencies (stdlib preferred; Charm libs for TUI only).
- NFR-4 Maintainability: small functions, explicit errors, no global mutable
  app state, business logic outside TUI (see `docs/03`).
- NFR-5 Reproducibility: versioned builds via linker flags; release packaging
  scripted (see `docs/14`).
- NFR-6 Time handling: timestamps stored UTC/RFC 3339, displayed in local
  time; policy documented once in `docs/09`.

## Deferred [D] (see docs/17)

Full-text content search, recurring study planner, calendar/cloud sync, rich
dashboards, AI-assisted workflows, auto-updates, portable mode, multi-user.
