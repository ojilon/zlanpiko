# Decisions

Log of architecture decisions (newest last). Each: context, choice, rationale,
alternatives, revisit trigger.

## D1 — SQLite as primary store; sidecars as cache (Phase 0)

- Context: need relations/status/deadlines/analytics + real user files.
- Choice: SQLite is the source of truth; `unit.json`/`topic.json`/`task.json`
  are regenerated mirrors.
- Rationale: single queryable truth; portable sidecars aid recovery without
  dual-truth bugs (staleness rule: DB `updated_at` wins).
- Alternatives: JSON-files-only (no queries, corruption-prone), embedded
  KV (loses relational queries).
- Revisit: only if DB proves unmaintainable — unlikely at this scale.

## D2 — `modernc.org/sqlite` driver (Phase 0)

- Context: Windows dev box, plain Go toolchain, no gcc guaranteed.
- Choice: pure-Go `modernc.org/sqlite` over `mattn/go-sqlite3` (CGO).
- Rationale: builds anywhere with `go build`; no CGO setup for the student or
  CI. Cost: larger binary (~acceptable for a local tool).
- Alternatives: `mattn/go-sqlite3` (smaller/faster, needs gcc on Windows).
- Revisit: if binary size or a specific SQLite feature becomes a problem.

## D3 — stdlib `flag` CLI, no Cobra in v1 (Phase 0)

- Context: shallow command tree (~30 leaf commands), single maintainer.
- Choice: hand-written dispatcher on stdlib `flag`.
- Rationale: one less dependency, full control of help/exit codes; Cobra's
  value (large generated trees, plugins) isn't needed yet.
- Alternatives: Cobra (revisit if the tree grows or shell completion is wanted).
- Revisit: subcommand count × flags gets unwieldy, or completion demanded.

## D4 — Per-unit sequential IDs, composite keys (Phase 0)

- Context: folder names must survive renames; bones must stay readable.
- Choice: `unit-001` global; `topic-001`/`task-001` per-unit with
  PK `(unit_id, id)`; folders use the same IDs.
- Rationale: short, stable, human-debuggable; avoids UUID folder soup.
- Alternatives: UUIDs (globally unique but unreadable), name-based folders
  (break on rename).
- Revisit: if cross-unit topic moves make per-unit sequences confusing in
  practice (mitigation: display `unit/topic` qualified IDs).

## D5 — `.bat` scripts instead of Makefile (Phase 0)

- Context: AGENTS.md sketch shows a `Makefile`; target is Windows-only,
  low-end, no extra tooling.
- Choice: `scripts/*.bat` as the canonical automation; no Makefile in v1.
- Rationale: zero-dependency on Windows; maintainer already in PowerShell/cmd.
- Revisit: add a thin Makefile later if *nix CI is introduced.

## D6 — Overdue derived, deadlines are task attributes (Phase 0)
- Context: spec asks for first-class deadlines + timeline.
- Choice: no `deadlines` table in v1; `tasks.due_at` + derived `overdue`;
  no standalone dateless events.
- Rationale: one task = one date covers all v1 use cases; derived overdue
  can't rot; fewer tables = fewer migrations.
- Alternatives: separate events table (more flexible, more UI + migration cost).
- Revisit: when user-defined dateless reminders are requested (see docs/17).

## D7 — Application and module renamed to zlanpiko (Phase 2)

- Context: user chose the final name; placeholder identity retired.
- Choice: app `zlanpiko`, module `zlanpiko` (single-element path),
  `zlanpiko.exe` / `zlanpiko-installer.exe`, DB `zlanpiko.db`,
  `%LocalAppData%\Zlanpiko`, `%AppData%\Zlanpiko\app-config.json`,
  env `ZLANPIKO_DATA`. Kept generic: `AcademicData` default storage dir
  (descriptive, user-changeable) and the `academic` adjective in prose.
  `AGENTS.md` left untouched (user's file; placeholders illustrative).
- Alternatives: `github.com/<user>/zlanpiko` module path — adopt if/when a
  forge host is chosen (mechanical change via `go mod edit -module`).
- Revisit: only when a repository host is selected.

## D8 — Record folders and sidecars in Phase 3 via a minimal filesystem API

- Context: Phase 3 services need unit/topic/task folders + sidecars, but full
  file operations belong to Phase 4.
- Choice: `internal/filesystem` gains only safe `Join`, record-dir helpers,
  skeleton creators, atomic `WriteJSON`, `MoveDir` and `RemoveDir` now;
  import/rename/search/open/delete UX stays Phase 4. Services and tasks call
  these helpers — no direct `os` calls outside `filesystem`/`config`/`database`.
- Alternatives: `os.MkdirAll` inline in services (faster now, rework later).
- Revisit: Phase 4 extends the package; the Phase 3 API surface is frozen
  unless a safety issue demands changes.

## D9 — Preview-first imports with TTY-aware confirmation (Phase 4)

- Context: bulk imports must never surprise; CLI must also never hang
  waiting for input when scripted.
- Choice: every import prints its plan (files, destination, collisions,
  skips) and copies nothing without `--yes`; without `--yes` it prompts
  only when stdin is a character device, otherwise refuses with exit 1.
  `--collision overwrite` additionally requires `--yes` even on a TTY.
  Default collision policy is rename; executables/hidden/oversize are
  skipped and reported.
- Alternatives: always-prompt (hangs scripts), always-require-`--yes`
  (hostile interactively). The TTY check uses `ModeCharDevice` (stdlib
  only) with a test seam (`stdinIsTerminal`) so tests never block.
- Revisit: if richer progress UI is wanted in the TUI (Phase 5).

## D10 — TUI as a thin Elm front over services (Phase 5)

- Context: the TUI must reuse service logic with no SQL in screens, stay
  testable without a terminal, and degrade on small/limited terminals.
- Choice: root Bubble Tea model (tabs, status bar, persistent input with
  history, `/` command dispatch with typo suggestions) + one screen struct
  per tab sharing a `Shared` struct (filters, nav requests, flash messages).
  Reusable form/confirm widgets consume `tea.KeyMsg` directly (unit-tested).
  Coverage math lives in `internal/analytics` from day one (dashboard,
  status bar and later reports share it). Themes auto/16/none via a
  renderer switch + a `Plain` flag (reverse video marks selection).
  Long imports run as `tea.Cmd` with a result message; everything else is
  synchronous local SQLite. No-args launches the TUI only on a TTY,
  otherwise prints help (script-safe).
- Alternatives: per-screen TUIs or a retained-mode framework (heavier,
  harder to test).
- Revisit: timeline/analytics screens and richer widgets in Phase 6.

## D11 — Attention rules quantified; single-connection DB discipline (Phase 6)

- Context: docs/10's "half thresholds" clause is unquantifiable, and the
  single-writer DB deadlocks when queries nest inside open row iteration
  (bit twice: filesystem Scan, analytics coverages).
- Choice: implement the 3/7/14-day and 50/70% rules exactly as written and
  drop the half-threshold clause (documented in code); rule: never issue
  queries while rows are open — drain into slices, close, then follow up.
- Alternatives: background connection pool (breaks the single-writer
  simplicity for no v1 benefit).
- Revisit: thresholds become user-configurable only if requested.

## D12 — Verify repairs sidecars, never file divergence (Phase 7)

- Context: `maintenance verify` finds three independent problem classes.
- Choice: `--repair` rewrites missing/stale/unreadable sidecars from DB
  rows (ensuring skeletons) and deletes `.tmp-*` leftovers; missing files
  and unindexed files are reported only — linking or deleting user content
  needs a human. Verify exits 1 while issues remain (script-friendly).
- Alternatives: auto-import unindexed files (violates never-auto-link),
  silent temp cleanup during verify (hides crash evidence).
- Revisit: none expected.

## D13 — Backup format and restore gating (Phase 8)

- Context: backups must verify, restore safely onto live roots, and round-trip.
- Choice: zip with `database/zlanpiko.db` (VACUUM INTO snapshot) +
  `files/<root-rel>` entries + `manifest.json` (hashes, versions, kind).
  Restore verifies hashes first, refuses newer schemas, replaces files via
  same-volume staging, and drops stale WAL sidecars. An *empty* database
  counts as an empty target (app.Open recreates the file anyway); otherwise
  restore needs `--overwrite-data` plus a safety backup taken with the live
  handle before closing it.
- Alternatives: raw file copy of the live DB (unsafe under WAL), tar
  (worse Windows tooling).
- Revisit: none expected.

## D14 — Installer via stock Windows tooling, no new dependencies (Phase 9)

- Context: shortcuts and user-PATH need OS integration; new Go deps would
  bloat the installer for one-time actions.
- Choice: Start-Menu `.lnk` via `cscript` + temp VBS (stock Windows),
  user-PATH via one idempotent PowerShell/.NET call (registry + broadcast,
  no setx truncation). Both are advisory (setup continues + prints manual
  steps on failure) and injectable/skippable in tests; real PATH writes are
  manual-checklist items. First-run setup lives in the main exe (TTY prompt
  → pointer), never inventing storage on pipes.
- Alternatives: go-ole shortcut lib, x/sys registry writes without
  broadcast (leaves stale sessions), always-prompt (hangs scripts).
- Revisit: none expected.
