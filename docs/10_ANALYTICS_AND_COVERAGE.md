# 10 — Analytics and Coverage

All formulas live in `domain`/`analytics` and are shared by dashboard, CLI,
timeline and reports. Nothing here is stored; everything is recomputed on read.

## Reading coverage (the one formula)

```text
coverage = read / total × 100,  rounded to 1 decimal
total    = topics with status in {unread, pending, read}   (= all topics)
```

- Empty unit (total = 0): display `—`, never `0%` or `100%`.
- Archived units excluded from totals by default (flag to include).
- "Read" means marked read by the user — it tracks **reading completion**,
  not mastery; the UI labels it exactly that.

Worked example: 20 topics, 8 read, 5 pending, 7 unread → `8/20 = 40.0%`.

## Task summaries

Per unit and overall: counts by stored status + derived overdue
(`docs/09`); `upcoming N days` (default 14) counts incomplete tasks with
`now ≤ due_at ≤ now+N days`; `dateless` counts tasks with no `due_at`.

## Attention labels (transparent rules, evaluated in order)

For each **unit**:

1. `Overdue` — any overdue task, OR any unread/high-priority topic within
   3 days before a test/examination.
2. `At Risk` — a test/examination within 7 days AND coverage < 50%.
3. `Attention` — a deadline within 14 days AND (coverage < 70% OR any pending
   task), OR any single overdue-free risk factor above at half thresholds.
4. `Completed` — total > 0 AND no incomplete tasks AND coverage = 100%.
5. `On Track` — everything else. Empty units show `No topics yet`, not a label.

For each **topic**: `Overdue`/`At Risk`/`Attention` derive from its unit's
upcoming assessments + its own status (unread + near test = At Risk or worse);
read topics near tests show `Review` hint (informational, not a warning).

These are planning indicators, never performance predictions. The TUI
Analytics screen prints the rule summary under each table; thresholds
(3/7/14 days, 50/70%) are constants in `domain` with names, changeable in one
place. Limitations are stated in-app: rules see deadlines + reading flags only.

## Timeline calculations (package `timeline`, the only date-math owner)

- Week = ISO week, local midnights; `WeekOffset(offset)` navigates; buckets
  `Mon…Sun`, tasks sorted by `(due_at, priority high→low, title)`.
- Overdue-incomplete items anchor to a leading "Overdue" section, not to their
  original day, so they stay visible. Completed items render in their day,
  dimmed.

## Reproducibility contract

Every screen/CLI/report shows inputs (counts) alongside outputs (percentages,
labels). `analytics` unit tests use frozen `now` fixtures; fixtures in
`testdata/` double as the documented examples. Any two fronts showing the same
unit at the same data state must show identical numbers.
