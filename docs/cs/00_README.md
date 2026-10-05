# CS-00 — Wails GUI Port: Overview & Ground Rules

`docs/cs/` = **Client Shell** plan: porting zlanpiko from a Bubble Tea TUI to a
Wails desktop GUI (Go backend + TS/HTML/CSS frontend) without losing data or CLI power.

## 1. Why this port

The current TUI home dashboard (11 units · 50 topics · 1 active task) proves the
data layer works, but it cannot show what you asked for:

- a thin linear week/day timeline strip,
- a "how far am I from deadline" distance graphic,
- calendars, charts, cards, statistics.

Terminals cannot do these well. Wails can: it carries a real OS window while the
existing Go code stays the backend.

## 2. Toolchain — verified 2026-10-04 on this machine

| Tool | Result | Note |
| ---- | ------ | ---- |
| Go | `go1.27.0 windows/amd64`, CGO enabled | `go.mod` says `go 1.26.0` — keep, Go 1.27 builds it fine |
| Wails | `v2.15.0` | working |
| Node | `v24.19.0` | working |
| pnpm | `11.25.0` | works via `cmd /c`, blocked in PowerShell by `ExecutionPolicy` (`.ps1` ban) |
| npm | `12.0.2` | same `cmd /c` workaround |
| TypeScript | **not installed yet** | `npx tsc` resolves to the wrong `tsc@2.0.4` stub; install real `typescript` per CS-03 |

PowerShell rule for this project going forward: run all `pnpm/npm/npx` through
`cmd /c "..."` or document the `Set-ExecutionPolicy -Scope CurrentUser` opt-in.
Never assume `pnpm` works raw in PowerShell 5.1.

## 3. What stays, what changes

**Stays (untouched semantics):**

- `internal/domain` vocabularies (unit/topic/task statuses, overdue-is-derived rule).
- `internal/analytics` coverage formula (`read/total×100`, empty unit = `—`, never 0/100%).
- `internal/timeline` ISO-week arithmetic (Monday–Sunday, local time, overdue anchor section).
- SQLite schema v1 + `migrations/` + `modernc.org/sqlite` driver (pure Go, no CGO pain).
- Filesystem layout under the data root (`units/`, `backups/`, `exports/`, `reports/`, `inbox/`).
- CLI grammar in `internal/cli` (`units/topics/tasks/timeline/analytics/export/backup/files/.`)
  and the TUI `/command` set — the GUI command bar must stay a superset (see CS-08).
- Installer philosophy: never wipe academic data on update (see CS-09).

**Changes:**

- New 3rd executable: `zlanpiko-gui.exe` (Wails). `zlanpiko.exe` (CLI+TUI) and
  `zlanpiko-installer.exe` keep working.
- New `frontend/` tree (Vite + TS + HTML + CSS + `node_modules`, pnpm-managed).
- New thin Go binding layer `internal/gui/` that exposes **read models + actions**
  to the frontend — no business logic moves into TS.
- Visual concept is deliberately **not** a TUI clone (see CS-04): cards with thin
  borders, calendar, charts, linear timeline, terminal-bordered command input.

## 4. Non-goals for the port

1. No cloud sync, accounts, or network services — local-first stays.
2. No schema v2 unless forced; the port must open existing `zlanpiko.db` files read/write.
3. No rewrite of `services/tasks/filesystem/importer/exporter/backup` — wrap them.
4. No heavy frontend framework (Next/Nuxt/Electron). Target: Vite + TS + hand-rolled
   CSS + tiny chart rendering (SVG/canvas), friendly to low-end Windows PCs.
5. No removal of TUI/CLI — they are the fallback and scripting surface.

## 5. Document map

| File | Question it answers |
| ---- | ------------------- |
| CS-01 | How do Go backend and TS frontend split? What are the 3 exes? |
| CS-02 | What exact Go methods does the UI call? DTOs, events, errors? |
| CS-03 | How is `frontend/` set up with pnpm + Vite + TS? |
| CS-04 | What does the new window look like (concept, not TUI copy)? |
| CS-05 | How do dashboard, deadline-distance graphic, thin timeline work? |
| CS-06 | How do calendar + weekly timeline views work and edit deadlines? |
| CS-07 | Which charts, from which Go formulas, with which rendering? |
| CS-08 | How do `/commands` survive in a GUI command bar? |
| CS-09 | How do installer + update preserve the existing DB? |
| CS-10 | How do we build, test, and release 3 exes reproducibly? |
| CS-11 | In what order do we execute all of this, with what acceptance checks? |

Read CS-11 last for the ordered build sequence; read CS-01/CS-02 first before writing code.
