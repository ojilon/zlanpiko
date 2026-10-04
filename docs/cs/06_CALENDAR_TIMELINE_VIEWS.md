# CS-06 — Calendar & Timeline Views

## 1. Two views, one data source

| View | Granularity | Source | Purpose |
| ---- | ----------- | ------ | ------- |
| Timeline | ISO week (Mon–Sun) + overdue anchor | `GetWeek(year, week)` | study-week planning, prev/next navigation |
| Calendar | month grid | `GetMonth(year, month)` | see deadline density, jump to days |

Both render `timeline.Item` projections; neither computes dates. Week parsing
(`2026-W41`) and Monday arithmetic stay in `internal/timeline`.

## 2. Timeline week view

```text
◀ Week 40 (Sep 29 – Oct 05) ▶        [Today]   filter: all ▾
OVERDUE (0) — none, good.
Mon 29 ···  Tue 30 ···  Wed 01 ● class presentation (Periodicity, 00:00, in 2d)
```

- Header: prev/today/next + `YYYY-Www` label + status-kind filter
  (all/assignments/tests/examinations/overdue/completed).
- Columns: 7 day lanes (stack on narrow widths) + a pinned red-tinted overdue
  lane. Items sorted by time then priority (Go order preserved).
- Item chip: time, title, unit code, attention pill; click → task drawer
  (details, edit deadline inline via date input, status buttons, links to unit
  folder and topic). Editing calls `MoveDeadline` and re-renders the lane.
- Keyboard: `←/→` weeks, `T` today, `Enter` opens selected item.

## 3. Calendar month view (new — no TUI equivalent)

```text
┌ October 2026 ──────────────── ◀ ▶ ┐
│ Mo Tu We Th Fr Sa Su             │
│        1  2  3 ●4  5            │
│  ●6  7  8  9 10 11 12  ← ● = due│
└──────────────────────────────────┘  Day agenda → [selected day items]
```

- Grid cells show up to 3 dots + `+n` overflow; dot colour = worst attention
  that day (red > amber > blue); today outlined.
- Click day → agenda panel listing that day's items with the same drawer as §2.
- Month navigation `◀ ▶` + `Today`; keyboard arrows move the selected day.
- `GetMonth` implementation note (Go): bucket the 4–6 ISO weeks overlapping the
  month via existing `timeline.BuildWeek`, then project counts — ~60 lines, no
  new calendar math package.

## 4. Shared task drawer (both views + dashboard)

One `taskDrawer.ts` component everywhere: title, unit link, kind/status/priority,
created/due/completion, description, notes, linked files, deadline editor,
`Complete / Submit / Delete` with the same confirmations as CLI/TUI.
After any mutation the drawer emits `zlanpiko:tasks-changed` handling (CS-02).

## 5. Acceptance visuals

- Your real case must read clearly: `2026-10-06 class presentation` appears as
  an amber tick on the dashboard rail, a chip on week 41, and a dot on Oct 6.
- Overdue items never hide inside a past day — they sit in the anchor lane
  until completed, matching `timeline` package semantics.
