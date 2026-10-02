# 09 — Tasks and Deadlines

## Task model (recap from `docs/04`)

Kinds: `assignment | coursework | test | examination | project | other`.
Stored statuses: `not_started | in_progress | completed | submitted`.
`completed_at` set when entering `completed`/`submitted` (cleared if moved back).

## Overdue is derived, never stored

```text
overdue(task) = task.due_at IS NOT NULL
              AND task.due_at < now
              AND task.status IN (not_started, in_progress)
```

Rationale: a stored `overdue` flag rots (needs a background job to stay true);
a derived value is always correct and needs no migration. Filters and the
timeline treat `overdue` as a pseudo-status computed at read time.

## Timezone policy (applies everywhere)

- **Store** UTC, RFC 3339 (`2026-10-09T23:59:00Z`).
- **Input** parsed as local time unless an offset is supplied
  (`2026-10-09`, `2026-10-09T23:59`).
- **Display** local time, format `2006-01-02 15:04`; week buckets use local
  midnights (see `docs/10` timeline section).
- No academic-calendar assumptions: no semesters/terms in v1; weeks are plain
  ISO weeks.

## Deadline lifecycle

1. Create with or without `due_at` (dateless tasks allowed; they appear under
   "No date" in lists, excluded from timeline buckets).
2. Edit freely (`tasks edit --due …` / timeline `e`); every view recomputes
   from the same row — no caches to invalidate.
3. Past deadlines are allowed (late entry of real work) but flagged in the UI
   ("in the past — still set?"); only incomplete items become overdue.
4. Completion removes overdue state instantly; history keeps the original
   `due_at` for audit.

## Timeline appearance

Each task with `due_at` appears once in its local-date bucket with kind icon,
priority marker, status/overdue marker; completed/submitted render dimmed with
✓. Dateless tasks are listed in a side section, never in day buckets.
Editing from the timeline is the same service call as editing from Tasks —
one code path (`tasks.Update`).

## Reminders

v1 has **no background reminder daemon** (out of scope: local-first, no
services). "Reminders" = attention surfacing on the dashboard + the exportable
report (`docs/10`, `docs/11`). OS notifications are deferred (`docs/17`).
