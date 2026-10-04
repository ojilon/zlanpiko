# CS-05 — Dashboard & Deadline-Distance Visuals

Fixes the exact pain in your screenshot: 50 unread topics at 0% with one bare
`Attention (deadline 2026-10-06 at 0%)` line and an `Upcoming` list with no sense
of urgency.

## 1. Dashboard composition (single screen, top→bottom)

1. **Header strip**: `Academic overview — 11 units · 50 topics · 1 active task`
   + global coverage ring + week label (`Sun 2026-10-04 · week 40`).
2. **Thin linear timeline rail** (§2): 14 days, deadlines as ticks.
3. **Attention row**: up to 3 cards (`At Risk` red, `Attention` amber,
   `Overdue` red-outline) — each shows unit → topic/task, days-left, coverage.
4. **Two columns**: left = unit cards with coverage bars; right =
   deadline-distance list (§3) + overdue box + upcoming-14-days agenda.
5. **Status line + command bar** persist (CS-04).

All numbers come from one `GetDashboard()` call; cards reuse `GetUnitCards()`.

## 2. Thin linear week/day rail

```text
Sep 29 ─────●──────○──────●──────┬──────○──────●──────○───── Oct 12
Mon    Tue  Wed  Thu      Sun    Mon    Tue    Wed     Fri
            ▲pres. ▲test         today
```

- A 6px horizontal track spanning today−7 … today+7 (scrollable to any week,
  arrows jump ±7 days, click a tick → Timeline view on that week).
- Ticks: amber dot = due that day, red = overdue anchor, green check = completed
  that day, hollow = planned study. Today gets a vertical marker + label.
- Tooltip per tick: `class presentation · Periodicity · 2026-10-06 00:00 · in 2d`.
- Implementation: pure SVG in `timelineStrip.ts`, data = `GetWeek` × 2.
  No date math in TS — Go supplies ISO dates + labels.

## 3. Deadline-distance graphic ("how far am I")

Per active task a horizontal bar maps **created → due**:

```text
class presentation · Periodicity · due 2026-10-06 00:00 · in 2d  [■■■■■■■───] 78%
```

- Fill = `pctElapsed` from `GetDeadlineDistance` (clamped 0–100); colour follows
  attention label: blue `<70%`, amber `70–90%`, red `>90%` or overdue.
- A `▼ you are here` tick marks today; overdue bars render full-red with
  `overdue by 1d` text. Dateless tasks show `no date — set one` + quick edit.
- Clicking the bar opens the task drawer: edit deadline, change status, jump to
  unit folder, mark complete/submit. Edits call `MoveDeadline`/`SetTaskStatus`
  and the bar animates to its new position.

Formula (Go, documented next to `analytics`):

```text
pctElapsed = (now - created) / (due - created), clamped [0,100]
daysLeft   = ceil((due - now) / 24h)   // negative = overdue
```

Single-day deadlines (created≈due) show `due today` instead of a misleading bar.

## 4. Unit cards (replace the ASCII table)

```text
┌ Periodicity [SED:2103] ──────────┐  coverage bar  0% · R0/P0/U5
│ ● Attention · due 2026-10-06     │  1 active · 0 overdue
└──────────────────────────────────┘
```

Grid of cards (responsive 1–3 columns), each with: name, code, thin coverage
bar (`—` when empty), `R/P/U` counts, active/overdue badges, attention pill,
click → unit detail (topics, tasks, files, per-unit analytics).

## 5. Empty/edge states

- All-read unit: green `Completed` pill, full bar.
- Empty unit (no topics): `—` glyph + `add topics` affordance, never 0%/100%.
- No deadlines: rail shows hollow `no deadlines this fortnight — enjoy it`.
- Overdue empty: `Overdue: none` (keep the TUI's reassuring explicitness).
