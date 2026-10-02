# 17 — Future Extensions

Deferred items — documented so the architecture stays hospitable, **not**
implemented in v1. Each notes its hook point.

## Richer analytics

Per-topic study time, grade tracking, trend charts, custom dashboards.
Hook: `analytics` already isolates formulas; add new pure functions + screens
without touching storage. Constraint: every new metric needs a documented,
reproducible formula (`docs/10` contract).

## Advanced scheduling / recurring planner

Recurring study sessions, timetable grid, workload balancing.
Hook: `timeline` owns date math; a `plans` table + recurrence expansion would
slot beside task-derived events. Needs background reminders first (below).

## Background reminders / OS notifications

Toast on approaching deadlines, tray presence. Requires a resident process or
scheduled task — conflicts with today's short-lived CLI model; design as an
opt-in `academic watch` command later.

## Calendar and cloud sync

ICS export, CalDAV/Google Calendar, file sync backends.
Hook: `exporter` JSON is the sync payload seed; keep field names stable
(`docs/11`). Cloud sync needs conflict resolution — currently out of scope by
design (local-first, single writer).

## Content search and AI-assisted workflows

Full-text/PDF indexing, summarisation helpers, "what should I study next".
Hook: `files` metadata table gains a content index; AI features must stay
offline-optional and never auto-modify records (suggest → user confirms).
No AI in v1; the `domain` purity rule keeps future inference code separable.

## Portable mode

Run from USB with data root beside the exe. Almost free today (data root is
configurable); needs: pointer-file discovery order update in `config` +
installer opt-out. Deferred only to avoid splitting Phase 9 focus.

## Multi-user / shared machines

Per-user data roots already supported via pointer files; true sharing needs
locking beyond single-writer + access control — not planned before v2.

## Remote build integration

If releases outgrow `scripts/*.bat`, the `configs/build.json` contract
(`docs/14`) lets an external manager take over packaging without code changes.

## Non-goals (recorded to prevent drift)

Web frontend, mobile app, server component, plugin system, scripting language
inside the app. If requested, they become new top-level docs, not scope creep
inside existing phases.
