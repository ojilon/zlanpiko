# 15 — Implementation Roadmap

Phases are sequential; each ends with tests + docs update + user commit.
Do not start the next phase with a red suite. Status tracked in
`IMPLEMENTATION_STATUS.md`.

## Phase 0 — Repository and documentation ✅ (this batch)

Inspect repo/toolchain; write `docs/00–17` + status/decision trackers; review
consistency. Acceptance: 18 docs + 2 trackers present, cross-references valid,
no contradictions. Next: user reviews and commits.

## Phase 1 — Go foundation

`go.mod` (module `github.com/example/academic-manager`, go ≥1.24), `configs/`,
`internal/app` (Version vars), `internal/logging`, `internal/domain` (enums +
validation), `cmd/academic` (`--help`/`version` only), `.gitignore`,
`scripts/test.bat`. Tests: domain validation. Acceptance: `go build ./...`
+ `go test ./...` green; exe prints help/version.

## Phase 2 — Configuration and storage

`internal/config`, `internal/database` (open + runner + `0001_init.sql`),
first-run init in `app.Open`, `maintenance verify` skeleton.
Tests: config round-trip, migrate-fresh, reopen-idempotent, newer-schema
refusal. Acceptance: init → close → reopen keeps data; no phantom second DB.

## Phase 3 — Core academic entities

`services` (units/topics), `tasks`, sidecar writes, `units/topics/tasks/*`
CLI read/write commands. Tests: CRUD, archive, confirmed delete, overdue
derivation. Acceptance: full lifecycle via CLI on a temp root.

## Phase 4 — Filesystem integration

`filesystem` ops + skeleton creation on record create, `importer` preview/
execute, `files/*` CLI, missing-file scan. Tests: collisions, traversal
rejection, preview-makes-no-changes. Acceptance: import/move/rename/delete
without DB/FS divergence.

## Phase 5 — Initial TUI

Root model + navigation + styles + command input + dashboard/units/topics/
tasks/files screens. TUI model tests. Acceptance: manage units/topics/tasks
entirely in-TUI on temp data.

## Phase 6 — Timeline and analytics

`timeline` + `analytics` packages, timeline + analytics screens, dashboard
attention section, inline deadline editing. Frozen-time tests.
Acceptance: identical numbers across dashboard/CLI/report for one fixture.

## Phase 7 — CLI and command system

Full command tree (`docs/08`) incl. `academic .`, TUI `/` dispatcher wired to
the same services, history. CLI tests incl. exit codes.
Acceptance: every TUI mutation has a CLI equivalent and vice versa.

## Phase 8 — Export, import polish and backup

`exporter` (txt+json), `backup` (create/verify/restore/list), golden-file
report tests, 10-step e2e script. Acceptance: export→wipe→restore→identical.

## Phase 9 — Installer

`cmd/installer` + `internal/installer` (setup, adopt-existing, PATH,
shortcuts), first-run-in-`academic.exe` path, temp-dir installer tests.
Acceptance: fresh install on Windows without Go toolchain; reinstall keeps data.

## Phase 10 — Release preparation

Linker-flag builds, `package.bat`/`release.bat`, `README.txt`/`CHANGELOG.txt`,
checksums, full suite + manual checklist, sample `release\`.
Acceptance: reproducible `release\` per `docs/14` and green suite.

## Dependencies between phases

```text
0 → 1 → 2 → 3 → 4 → 5 ┐
                      ├→ 8 → 9 → 10
3 → 6 → 7 ────────────┘   (6 needs 3; 7 needs 3+5+6)
```

## Explicitly deferred (see docs/17)

Content search, recurring planner, calendar/cloud sync, rich dashboards, AI
workflows, auto-update, portable mode, multi-user, background reminders.
If a phase threatens the MVP, cut the deferred-adjacent polish first and note
it in `IMPLEMENTATION_STATUS.md` — never cut tests or data-safety work.
