# CS-04 — UI Concept (Frameless, Not a TUI Clone)

## 1. Shell layout

```text
┌─ custom titlebar (drag) ─────────────────────────────────────────────┐
│ ● zlanpiko        Sun 2026-10-04 · week 40 · 0.1.0        — ▢ ✕      │
├──────────┬───────────────────────────────────────────────────────────┤
│ sidebar  │  VIEW (dashboard / units / calendar / analytics …)        │
│          │  ┌ cards with thin 1px borders, generous whitespace ┐     │
│ Dash     │  │ thin week/day strip (CS-05)                       │     │
│ Units    │  │ deadline-distance bars (CS-05)                    │     │
│ Topics   │  │ attention cards · overdue · upcoming              │     │
│ Tasks    │  └───────────────────────────────────────────────────┘    │
│ Timeline │  STATUS LINE: 11 units · 50 topics (R0/P0/U50 0%) · …      │
│ Calendar │  ┌ terminal-bordered command bar ───────────────────┐      │
│ Files    │  │ > /units list                        ⌄ history    │      │
│ Analytics│  └───────────────────────────────────────────────────┘      │
│ Settings │                                                            │
└──────────┴───────────────────────────────────────────────────────────┘
```

Rules:

- **Titlebar is custom** (CS-03 §3): drag anywhere except the three buttons;
  no native menu bar. Window chrome must still expose minimise/maximise/close
  with tooltips + keyboard focus.
- **Sidebar** (icons + labels, collapsible to icons at <1100px): Dashboard,
  Units, Topics, Tasks, Timeline, Calendar, Files, Analytics, Settings.
  Number badges for overdue (red) and due-this-week (amber).
- **Status line** above the command bar mirrors the TUI status:
  `11 units · 50 topics (R0/P0/U50 0%) · 1 active · 0 overdue`.
- **Command bar is persistent** across all views, keeps partial input on view
  switch, `Ctrl+K` or `/` focuses it (CS-08). Its border is styled like a
  terminal frame (see §3).
- Minimum 960×640; below that the view scrolls, the titlebar/sidebar/command
  bar never collapse. Support 100%/125%/150% DPI.

## 2. Design language

- Light-blue/grey identity on dark slate (`--accent #7fb6e8`), thin `1px`
  boundaries, `10px` radius, compact density (the TUI dashboard fits one screen —
  the GUI dashboard must too, without scrolling on 1080p).
- Status colours used sparingly: blue = info/progress, amber = attention/due,
  red = overdue/at-risk, green = completed/read. Never colour alone — always pair
  with a label or icon for colour-blind safety.
- Progress bars are thin (6px) with `—` for empty units (same rule as TUI).
- Fonts: system stack only (`Segoe UI`, Consolas for the command bar). No webfont
  downloads (offline-first).

## 3. Terminal-bordered command input (signature element)

```text
┌─ terminal ─────────────────────────────────────────┐
│ > /topics status topic-003 read          [↵ run]    │
│ hint: /timeline next · /tasks deadlines · /help    │
└────────────────────────────────────────────────────┘
```

- Monospace, prompt glyph `>`, blinking caret, command history (`↑↓`), inline
  autocomplete dropdown, `Esc` clears, `Enter` runs via `RunCommand`.
- Success → green toast + view refresh; error → red inline message with `hint`.
- The frame label `terminal` signals "this speaks the same language as the CLI".

## 4. New ideas beyond the TUI (committed for v0.2)

1. **Deadline-distance bars** (CS-05): per-task bar from created→due with a
   "you are here" tick — answers "how far am I from the deadline" at a glance.
2. **Thin week/day strip**: 14-day linear rail under the dashboard header with
   dots for deadlines, draggable to the Timeline view.
3. **Attention cards**: `At Risk / Attention / Overdue` cards group the exact
   TUI "Needs attention" rows but with unit, topic state, and days-left inline.
4. **Calendar month grid** (new view, no TUI equivalent) with per-day
   deadline dots + click-to-day agenda (CS-06).
5. **Analytics charts** (CS-07): stacked read/pending/unread, tasks-by-status,
   per-unit coverage bars — all from existing Go formulas.
6. **File cards** with type icons, size/mtime, missing-file badges (from
   `maintenance verify`), and OS-open on double-click.
7. **Global search** (`Ctrl+K`): units, topics, tasks, files in one ranked list.

## 5. What we deliberately do NOT copy from the TUI

- No full-screen text tables, no ASCII progress bars (`░░░`), no screen-per-key
  modal stacks. Lists become cards/rows with hover actions.
- No hidden keyboard-only flows: every command-bar action has a clickable twin
  (button, context menu, or row action).
