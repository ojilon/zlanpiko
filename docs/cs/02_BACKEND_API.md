# CS-02 — Backend API: Wails-Bound Service Surface

This is the contract `frontend/src/*` programs against. All methods live on
`internal/gui.GuiApi` and are bound via Wails. No SQL, no Bubble Tea, no CLI
parsing leaks through — only these methods + the command entrypoint (CS-08).

## 1. Api struct shape

```go
// internal/gui/api.go
type GuiApi struct {
  ctx *app.Context      // resolved data root, db handle, logger
  // services wired at startup; events via wails runtime
}
```

Constructor `NewGuiApi(ctx *app.Context) *GuiApi` opens the DB once (same
`database.Open` path as CLI/TUI), applies pending migrations, and is the **only**
place allowed to hold the `*sql.DB` for the GUI process.

## 2. Method catalogue (final for v0.2 GUI)

Dashboard + attention (powers CS-05):

- `GetDashboard() (DashboardDTO, *GuiError)` — units/topics/tasks counts,
  coverage rollup, attention list, overdue list, next-14-days. One call boots
  the whole home view. Reuses `analytics.UnitCoverages` + `analytics.Attention`.
- `GetUnitCards() ([]UnitCardDTO, *GuiError)` — per-unit: name, code, coverage,
  coverage text (`—` for empty), active tasks, next due, attention label.
- `GetDeadlineDistance(taskID string) (DeadlineDistanceDTO, *GuiError)` —
  `{due, now, daysLeft, hoursLeft, pctElapsed, state}` where `pctElapsed` is the
  position on the created→due segment (clamped 0–100). Pure derivation in Go.

Units / topics / tasks (CRUD wraps `services`, `tasks`):

- `ListUnits(includeArchived bool)` / `CreateUnit(name, code)` /
  `RenameUnit(id, name)` / `ArchiveUnit(id)` / `UnarchiveUnit(id)` /
  `DeleteUnit(id, confirm bool)`
- `ListTopics(unitID, statusFilter)` / `CreateTopic(unitID, name)` /
  `SetTopicStatus(topicID, status)` — validates via `domain.CanTransitionReading`.
- `ListTasks(filter TaskFilterDTO)` / `CreateTask(input TaskInputDTO)` /
  `UpdateDeadline(taskID, isoLocal string)` / `SetTaskStatus(taskID, status)` /
  `DeleteTask(taskID, confirm bool)`

Timeline + calendar (powers CS-06):

- `GetWeek(year, week int) (WeekDTO, *GuiError)` — `timeline.BuildWeek` output:
  Monday, 7 days, per-day items, leading overdue section.
- `GetMonth(year, month int) (MonthDTO, *GuiError)` — day→counts + attention dots,
  built by bucketing 4–6 `timeline` weeks; no new date logic in TS.
- `MoveDeadline(taskID, newDateLocal string)` — same validator as CLI `tasks edit`.

Files (list/open/import only; heavy ops stay preview-first):

- `ListFiles(unitID)` / `SearchFiles(query)` — metadata only (name, size, mtime).
- `OpenFile(pathID)` — Go calls the OS default app; TS never touches bytes.
- `PreviewImport(dirPath, targetUnitID)` → conflict list; `ConfirmImport(token)`
  executes. Mirrors the CLI `.` flow: never silently import.

Reports / backup / config:

- `GetReport(unitID) (txt string)` / `GetReportJSON(unitID) (json string)` —
  reuse `exporter`.
- `CreateBackup(includeFiles bool)` + progress events; `ListBackups()` /
  `VerifyBackup(id)` / `RestoreBackup(id, confirm bool)` — reuse `backup`.
- `GetConfig()` / `SetDataRoot(path)` (validates, re-opens DB) / `GetVersion()`.

Command bridge (CS-08):

- `RunCommand(input string) (CommandResultDTO, *GuiError)` — parses `/units list`,
  `/topics status …`, etc. and dispatches to the methods above. This is how the
  GUI command bar keeps 100% parity with TUI/CLI without duplicating handlers.

## 3. DTO conventions

- Times: RFC 3339 local (`2026-10-06T00:00:00+01:00`) + pre-formatted
  `display` strings (`"2026-10-06 00:00"`, `"in 2d 4h"`, `"overdue by 1d"`).
  TS never formats raw timestamps for deadlines.
- IDs: existing stable `unit-001`-style strings; folder names never derived in TS.
- Enums: exact `domain` strings (`unread|pending|read`,
  `not_started|in_progress|completed|submitted`, plus derived `overdue` flag).
- Empty-unit coverage: `{hasTopics:false, coverage:0, coverageText:"—"}`.

## 4. Events emitted by Go

| Event | Payload | UI use |
| ----- | ------- | ------ |
| `zlanpiko:tasks-changed` | `{touchedIDs}` | refresh dashboard/timeline/charts |
| `zlanpiko:topics-changed` | `{unitID}` | refresh unit cards + coverage |
| `zlanpiko:progress` | `{op, done, total, label}` | import/backup/restore progress bar |
| `zlanpiko:toast` | `{level, message}` | user-facing confirmations/errors |

## 5. Error shape

```ts
interface GuiError { code: string; message: string; hint?: string }
// codes: VALIDATION, NOT_FOUND, CONFLICT, STORAGE, DB, BACKUP, CANCELLED
```

Every failure returns a `GuiError`, never a Go stack trace. `message` is
user-readable; `hint` suggests the fix ("target folder exists — pick a new name").

## 6. Implementation order (also CS-11 Phase D)

1. `dto.go` + `GuiError` first (unblocks frontend types).
2. Read methods (`GetDashboard`, `GetUnitCards`, `GetWeek`, `GetMonth`).
3. Mutating methods with `touchedIDs` events.
4. `RunCommand` last, dispatching to 2–3 (proves parity).
