# CS-01 — Architecture: Go Backend + Wails Window + TS Frontend

## 1. Target process model

```text
┌──────────────────────────── zlanpiko-gui.exe (Wails) ────────────────────────────┐
│  frontend/ (Vite+TS, served from embedded dist/)                                 │
│   views/*.html · assets/*.css · src/*.ts ──calls──► window.go.zlanpiko.GuiApi.*  │
│                                                                                  │
│  Go backend (same process, wails bindings)                                        │
│   internal/gui/api.go  ──thin──► internal/services · tasks · timeline ·          │
│                          analytics · filesystem · importer · exporter · backup ·  │
│                          config · database (SQLite, schema v1)                     │
└──────────────────────────────────────────────────────────────────────────────────┘
         ▲ same data root / same zlanpiko.db as CLI+TUI
         │
zlanpiko.exe (CLI + Bubble Tea TUI) ······ unchanged, shares internal/*
zlanpiko-installer.exe ··················· unchanged flow + GUI exe install (CS-09)
```

One Wails process = Go runtime + embedded webview. No localhost server, no second
DB connection owner: open SQLite once per process with the same `app.Context`
resolution (`--data-root` → `$ZLANPIKO_DATA` → install pointer) as CLI/TUI.

## 2. Package rules

```text
cmd/
  zlanpiko/          CLI+TUI entry (existing, untouched)
  installer/         existing installer + new GUI payload step (CS-09)
  gui/               NEW: wails entry, ~50 lines: ctx resolve, logging, Wails run

internal/gui/        NEW, the only package Wails binds. Owns:
  api.go             GuiApi struct + bound methods (CS-02)
  dto.go             JSON DTOs (dashboard, week, charts, files, events)
  commands.go        /command string parser → same handlers as TUI/CLI
  events.go          event names + payloads (tasks changed, backup progress…)

internal/*           existing services stay authoritative. gui/ may call them,
                     never the reverse. No imports from internal/tui or
                     internal/cli into gui/ (would drag Bubble Tea into Wails).
```

Dependency direction: `cmd/gui → internal/gui → internal/{services,tasks,timeline,
analytics,filesystem,importer,exporter,backup,config,database,domain,app}`.
`frontend/src/*` may only use DTOs from CS-02 — never SQL, never file paths raw.

## 3. Why three executables

| Exe | Built by | Role |
| --- | -------- | ---- |
| `zlanpiko.exe` | `go build ./cmd/zlanpiko` | CLI + TUI, scripting, offline fallback, headless environments |
| `zlanpiko-gui.exe` | `wails build` (embeds `frontend/dist`) | daily driver window: dashboard, calendar, charts, command bar |
| `zlanpiko-installer.exe` | `go build ./cmd/installer` | first install + safe update of **both** exes, DB-preserving (CS-09) |

Do not merge CLI into the GUI exe. Keeping them separate means: (a) PATH scripting
keeps working when the window toolkit breaks; (b) release can ship GUI-optional
packages; (c) tests stay honest — services are exercised by two fronts.

## 4. Data-flow principles

1. **Single source of truth stays Go.** Coverage, attention labels
   (`On Track/Attention/At Risk/Overdue/Completed`), week buckets, search ranking
   are computed in Go and shipped as DTOs. TS renders; it never re-derives.
2. **Reads are cheap JSON, writes return the new truth.** Every mutating call
   (`SetTopicStatus`, `UpdateDeadline`, …) returns the affected DTO + dashboard
   delta so the UI updates without a second round-trip.
3. **Long work emits events.** Import/scan/backup/restore run on Go background
   goroutines with `runtime.EventsEmit("zlanpiko:progress", …)`; TS shows progress
   bars and never blocks the window.
4. **Files stay on disk.** The frontend never receives file bytes except via
   explicit export/download paths; "open file" asks Go to call the OS opener.
5. **Errors are typed.** Every bound method returns `(T, *GuiError)` with
   `{code, message, hint}` so the UI can show actionable toasts (CS-02 §5).

## 5. What explicitly does NOT move to TypeScript

- Deadline arithmetic, ISO-week logic, overdue derivation.
- Coverage/attention formulas.
- Path validation / traversal guards (`filesystem` owns the jail).
- Import conflict decisions, backup integrity checks, migration application.

If a rule exists in `docs/04/05/09/10`, its implementation stays in Go.
TS duplication of any such rule is a review-blocking defect.
