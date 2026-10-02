# 19 — TUI User Guide (v0.1.0)

The terminal UI is the visual front over the exact same data and rules as
the CLI (`docs/18`). Start it with bare `zlanpiko` on a terminal (needs at
least 80×24; piped output prints help instead of hanging). Where the two
fronts differ, this guide says so — the differences are collected as
limitations in `docs/20`.

## The screen at a glance

```text
┌ zlanpiko                      Tue 2026-10-06 · week 41 · 0.1.0 ─┐
│ [1] Dashboard  [2] Units ...                                    │ tabs
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  content: tables, dashboard, browser, dialogs                   │
│                                                                │
├────────────────────────────────────────────────────────────────┤
│ keys for this screen (e.g. a add · r rename · ...)             │ help line
│ Status: 9 units · 42 topics (R8/P5/U7 40%) · ...               │ status bar
│ > Type a command or /help                         [command]    │ input
└────────────────────────────────────────────────────────────────┘
```

- **Header**: app, date, ISO week, version.
- **Tabs**: screens `1` Dashboard `2` Units `3` Topics `4` Tasks `5` Timeline
  `6` Files `7` Analytics `8` Settings (compact number-only strip on narrow
  terminals).
- **Help line**: the keys that work *right now*, on this screen.
- **Status bar**: live counts, or the latest message (errors show here too).
- **Command input**: persistent across screens — half-typed text, history and
  errors survive tab switches.

## Global keys (work everywhere, except while typing)

| Key | Action |
| --- | ------ |
| `1`–`8` | Jump to a screen |
| `/` | Focus the command input |
| `esc` | Blur input / close dialog / back out |
| `?` | Keys + commands cheat sheet (any key closes) |
| `q`, `ctrl+c` | Quit |

While the input is focused: `enter` runs the command, `esc` blurs (keeping
your text), `↑`/`↓` walks command history (kept across sessions, last 100).

## Commands (type after `/`)

```
/dashboard /units /topics [unit] /tasks [unit] /timeline [next|prev|today|YYYY-Www]
/files /analytics /settings /config /verify /export /backup /version /quit
```

Notes: `/topics unit-001` and `/tasks unit-001` pre-filter those screens.
`/export` and `/backup` jump to Settings, where `e` / `B` do the work.
`/verify` prints a one-line consistency report in the status bar. Unknown
commands get a closest-match suggestion (`/topix` → did you mean `/topics`?).
`/search` is reserved for a later release.

## Dialogs: forms and confirmations

- **Forms** (adding/renaming/editing): `tab` moves between fields, `enter`
  on the last field submits, `enter` elsewhere moves on, `esc` cancels.
  Required fields block empty submits and say which field is missing.
- **Confirmations** (deletes): default to **No**. `←`/`→` toggle, `y`
  accepts, `n`/`esc` cancels, `enter` confirms the highlighted choice.
- Your typed input is never lost by switching screens mid-form — cancelling
  is always explicit with `esc`.

## Screen by screen

### 1 · Dashboard (read-only)

Counts, per-unit coverage bars, a "Needs attention" section (units flagged
Overdue / At Risk / Attention with reasons), overdue items, and the next 14
days of deadlines. Scroll with `↑`/`↓`. If you have no units yet it tells
you to press `2` then `a`. Coverage math is identical to the CLI: see
`docs/10`.

### 2 · Units

Table: ID, name, code, status, topic count, coverage, active tasks.

| Key | Action |
| --- | ------ |
| `a` | Add (name required; code, colour optional) |
| `r` | Rename selected (folders keep stable IDs, nothing breaks) |
| `A` | Archive / unarchive (archived units reject new topics/tasks) |
| `D` | Delete: instant when empty, confirmation dialog with counts otherwise |
| `enter` | Open the unit's topics (jumps to Topics, pre-filtered) |

Unlike the CLI form, the add dialog has no *description* field in v0.1.0
(use `units add --description` or edit later — see `docs/20`).

### 3 · Topics

Table: unit, ID, name, reading status, priority, attention hint.

| Key | Action |
| --- | ------ |
| `x` | Cycle the reading status (`Unread → Pending → Read → Unread`) |
| `f` | Cycle the status filter (all → unread → pending → read) |
| `u` | Clear the unit filter (show all units) |
| `a` | Add (unit, name, priority) |
| `r` | Rename |
| `m` | Move to another unit (keeps its number when free) |
| `D` | Delete (confirmation only when files are linked) |

Tip: from Units, `enter` on a unit lands here already filtered; `u` gets
you back to everything.

### 4 · Tasks

Table: unit, ID, title, kind, status (derived `overdue` shown automatically),
due date, priority.

| Key | Action |
| --- | ------ |
| `f` | Cycle the status filter (all → not started → in progress → overdue → completed → submitted) |
| `u` | Clear the unit filter |
| `a` | Add (unit, title, kind, due `YYYY-MM-DD[THH:MM]`, priority) |
| `e` | Edit title, due (empty clears), priority, description |
| `c` / `s` | Mark completed / submitted (clears overdue instantly) |
| `D` | Delete (confirmation only when files are linked) |

The edit dialog cannot change a task's *kind* in v0.1.0 (CLI `tasks edit
--kind` can) — see `docs/20`.

### 5 · Timeline

One ISO week per screen: overdue section first, then day rows.

| Key | Action |
| --- | ------ |
| `n` / `p` | Next / previous week |
| `t` | Back to this week |
| `enter` | Open the selected item's detail |
| (in detail) `e` | Edit the deadline |
| (in detail) `c` | Mark completed |
| (in detail) `g` | Jump to the unit's topics |
| (in detail) `esc` | Back to the week |

Under 90 columns the table drops the unit and kind columns automatically.

### 6 · Files

A browser rooted at your data root. `enter` descends, `backspace`/`←`/`h`
goes up; the header always shows where you are.

| Key | Action |
| --- | ------ |
| `i` | Import (source path, destination defaulting to here, recursive y/N) — shows a preview with counts, confirms, then imports in the background |
| `o` | Open the file in its default app |
| `r` | Rename (same folder, collisions refused) |
| `D` | Delete (confirmation; folders deleted recursively) |

There is **no create-folder key** in v0.1.0 — new folders arrive with new
units/topics/tasks, or as import destinations (same gap as the CLI; see
`docs/20`). Protected locations (the root itself, `database/`, unit folders)
refuse deletion with an explanation.

### 7 · Analytics (read-only)

Per-unit coverage with `R/P/U` counts, task counts and attention labels,
plus the overall summary. The formulas used are printed under the table —
same numbers as `zlanpiko analytics`, reproducible from your rows.

### 8 · Settings

Shows version, data root, database path, unit counts, current theme.

| Key | Action |
| --- | ------ |
| `t` | Cycle theme `auto → 16 → none` (saved; `none` disables all colour) |
| `v` | Verify the database and file consistency (like `maintenance verify`) |
| `e` | Export the text status report into `exports/` |
| `B` | Take a full backup into `backups/` |

## Theme and terminals

`auto` adapts colour to your terminal; `16` forces the 16-colour palette;
`none` (or `NO_COLOR` in the environment) disables colour while keeping
layout, progress bars and the reverse-video selection marker. Small
terminals get a compact tab strip and a compact timeline; below 80×24 the
app shows a polite resize notice instead of garbled output.

## Practical notes

- The status bar counts refresh after every action — if two windows are
  open on the same data, switch screens to refresh (single-writer rule:
  close the other window if the database reports busy).
- Bulk imports run without freezing the interface; `esc` does not cancel a
  running import in v0.1.0 — wait for its summary line.
- Logs go to stderr (visible if you launch from a console with redirection);
  there is no persistent log file in v0.1.0.
