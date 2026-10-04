# CS-11 — Implementation Steps (Ordered, Checkpointed)

Execute phases in order; stop at each checkpoint for review + commit. Do not
skip the safety tests — the user has live data.

## Phase A — Scaffold (no behaviour change)

1. `wails init` into a temp dir, move `wails.json`/`frontend/` skeleton in,
   set `frameless:true`, wire `package.json` scripts (CS-03 §4).
2. Add `cmd/gui/main.go` (embed `dist-frontend`, resolve `app.Context`, run Wails).
3. `go build ./...` + `wails build` produce a blank frameless window with the
   custom titlebar (CS-03 §3). **Checkpoint:** drag/min/max/close work.

## Phase B — Read-only dashboard

4. `internal/gui/dto.go` + `GetDashboard` + `GetUnitCards` + `GetWeek`.
5. `frontend` shell (titlebar/sidebar/status/command bar) + dashboard view:
   header, rail, unit cards, upcoming list (CS-04/05).
6. Golden tests: dashboard JSON for a fixture == CLI numbers.
   **Checkpoint:** your 11-unit DB renders correctly, counts match TUI exactly.

## Phase C — Deadline distance + timeline

7. `GetDeadlineDistance` + distance bars; `GetMonth` + Timeline + Calendar
   views + task drawer (read-only first).
8. `vitest` fixtures for bars/strip from Go goldens.
   **Checkpoint:** `2026-10-06 class presentation` visible on rail, week 41,
   and Oct 6 with consistent days-left everywhere.

## Phase D — Mutations + commands

9. Topic/task/unit mutations + deadline editing from drawer/calendar.
10. `RunCommand` + command bar (history, autocomplete); extract shared
    `internal/cmdparse` if TUI/GUI parsing diverges.
11. Parity test: every `/command` vs CLI twin on same fixture.
    **Checkpoint:** full CRUD from GUI; TUI/CLI outputs unchanged.

## Phase E — Files, reports, backup

12. File cards + OS-open + preview-first import; plain-text + JSON reports;
    backup list/verify/restore with progress events.
    **Checkpoint:** import/backup/restore E2E green.

## Phase F — Charts + polish

13. `StatusCounts`/`DueHistogram` + 5 charts (CS-07); attention reasons;
    light mode; empty states; reduced-motion.
    **Checkpoint:** analytics view matches CLI `analytics` numbers.

## Phase G — Installer + release

14. Installer GUI payload + pre-update backup + verify (CS-09); update-sim
    and downgrade-refusal tests; `build-gui/package` scripts; release notes.
    **Checkpoint:** CS-09 §5 + CS-10 suites green; user commits release.

Deferred past v0.2 GUI: reminders/notifications, study planner, phone sync,
richer editors, plugin system. Log them in `docs/17` instead of building them.
