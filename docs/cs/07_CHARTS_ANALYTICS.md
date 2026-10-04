# CS-07 — Charts & Analytics (Transparent, Reproducible)

Every chart below renders a Go-computed number; the frontend draws shapes only.
Formulas live in `internal/analytics` today — the GUI adds **no new math**, only
new projections of the same rows.

## 1. Chart set for v0.2 (5 charts, each with a caption of its formula)

1. **Coverage ring** (dashboard header): `read/total×100` overall; caption
   `48/120 read · 40%`. Empty-everything state shows `—`, not 0%.
2. **Read / Pending / Unread stacked bar**: global + per-unit toggle; segments
   labelled `R/P/U` with counts, matching the TUI `R0/P0/U50` line.
3. **Per-unit coverage bars**: one thin bar per unit, sorted worst-first,
   `—` for empty units. Click bar → unit detail.
4. **Tasks by status**: `not_started / in_progress / completed / submitted /
   overdue(derived)` counts; overdue is hatched, never a stored status.
5. **Deadlines next 14 days**: vertical day-columns with counts + worst-attention
   colour; click column → Timeline on that week.

Rendering: hand-rolled SVG in `components/charts.ts` (<300 lines). No chart
dependency — keeps the binary small and works offline. Bars animate width on
data change; charts honour reduced-motion OS settings (no animation then).

## 2. Attention labels (unchanged rules, better surface)

`analytics.Attention` labels (`On Track / Attention / At Risk / Overdue /
Completed`) keep their documented inputs (topic state × deadline proximity ×
completion). The GUI shows the label **plus its reasons** on hover:

```text
At Risk — Periodicity · unread topics + exam in 2d (rule: §docs/10)
```

Never present coverage as pass/fail prediction — captions say
`reading coverage, not mastery`, same wording as the TUI docs.

## 3. Analytics view layout

- Top: coverage ring + stacked bar + tasks-by-status (3 cards).
- Middle: per-unit bars + 14-day deadline columns.
- Bottom: attention table (sortable by unit / days-left / coverage) with
  `Open unit` and `Set reminder note` actions.
- `?` affordance per card opens the formula caption (one sentence + doc link).

## 4. Data plumbing

- New Go projections (small, beside existing functions):
  `analytics.StatusCounts(db, now)` and `analytics.DueHistogram(db, now, 14)`.
- Frontend caches chart DTOs per `zlanpiko:tasks-changed` /
  `zlanpiko:topics-changed` events; stale-while-revalidate with a subtle
  `updating…` shimmer, never a blocking spinner for local reads.
