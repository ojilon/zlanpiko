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
