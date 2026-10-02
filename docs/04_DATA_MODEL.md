# 04 — Data Model

Source of truth: **SQLite** for all relationships, statuses, deadlines and
computed views. The filesystem holds user files; JSON sidecars (`unit.json`,
`topic.json`, `task.json`) are a portable cache regenerated from the DB —
readable when the DB is present, rebuildable when it is not (see `docs/06`).

## Entities

### Unit

| Field | Type | Notes |
| ----- | ---- | ----- |
| `id` | TEXT PK | stable, e.g. `unit-001` (never reused, survives renames) |
| `name` | TEXT UNIQUE (among active) | required, 1–120 chars |
| `code` | TEXT NULL | optional course code, e.g. `MATH201` |
| `description` | TEXT | default `""` |
| `colour` | TEXT NULL | `blue\|grey\|green\|...`, display hint only |
| `status` | TEXT | `active \| archived` |
| `created_at` / `updated_at` | TEXT | UTC RFC 3339 |

### Topic

| Field | Type | Notes |
| ----- | ---- | ----- |
| `id` | TEXT PK | `topic-001`… scoped counter per unit for readability; globally unique by prefix `unitID:` — stored as e.g. `unit-001/topic-003`? **Decision:** single TEXT PK `topic-<unitseq>-<seq>` is unreadable. Simpler: PK is `unit_id + local_id` composite in folders, but DB PK is a surrogate `t_XXXXXX`? |

**Simplification (final):** IDs are short human-stable strings:

- Unit: `unit-001`, `unit-002`, … (global sequence, never reused).
- Topic: `topic-001`, … **unique per unit**; DB PK is `(unit_id, id)`, folder
  name is the local id. Display shows `unit-001/topic-003` where ambiguous.
- Task: `task-001`, … same per-unit scoping, PK `(unit_id, id)`.
- File record: `file-<n>` global sequence (internal; users see filenames).

This keeps folder names short and renames free, at the cost of composite keys —
acceptable for this scale (hundreds of rows).

| Topic field | Notes |
| ----------- | ----- |
| `unit_id` FK | `ON DELETE CASCADE` gated by service confirmation (see below) |
| `name` required, unique within unit |
| `reading_status`: `unread \| pending \| read` (default `unread`) |
| `priority`: `low \| normal \| high` (default `normal`) |
| `notes` TEXT, `created_at/updated_at` UTC RFC 3339 |

### Task (assignment / coursework / test / examination / project / other)

| Field | Notes |
| ----- | ----- |
| `unit_id` FK, `id` per-unit seq; `title` required |
| `kind`: `assignment \| coursework \| test \| examination \| project \| other` |
| `status`: `not_started \| in_progress \| completed \| submitted` (stored). `overdue` is derived — see `docs/09`. Default `not_started`. |
| `due_at` TEXT NULL (UTC RFC 3339, optional time); `completed_at` NULL |
| `priority`, `description`, `notes`, timestamps as above |

### Deadline

Deadlines are **attributes of tasks**, not a separate table in v1: one task =
one due date. User-defined date-only events without a task are [D]
(see `docs/17`). The `timeline` package projects task rows into week buckets.

### File record

| Field | Notes |
| ----- | ----- |
| `id`, `unit_id` NULL-able, `topic_id`/`task_id` NULL-able (one attachment link) |
| `rel_path` (relative to data root, `/`-separated), `size`, `sha256` (filled on backup/import, lazy otherwise), `missing` BOOL (set by scan) |
| Files link to at most one of topic/task; folders themselves are implied by unit/topic/task rows. Unlinked files live under `inbox/`. |

## Relationships

```text
unit 1───* topic
unit 1───* task
topic 1───* file_record
task  1───* file_record
(unit archived ⇒ topics/tasks hidden by default, never deleted)
```

## What lives where

| Information | SQLite | Filesystem/sidecar |
| ----------- | ------ | ------------------ |
| names, statuses, deadlines, priorities, relations | ✅ truth | mirrored for portability |
| user files (PDFs, notes, submissions) | path index only | ✅ truth (bytes) |
| `unit.json` / `topic.json` / `task.json` | regenerated after writes | cache; stale copies ignored when DB newer (compare `updated_at`) |
| analytics, coverage | computed on read, never stored | reports are snapshots |

## Integrity rules

- Deleting a unit/topic/task with children (topics, tasks, files) requires
  explicit confirmation; otherwise the service refuses. Archive is offered first.
- `updated_at` changes on every mutation; sidecars rewritten in the same
  service call (DB write first, then sidecar — see `docs/16` for crash order).
- Status/deadline values validated in `domain`; DB CHECK constraints mirror
  them as a second line of defence (defined in `docs/05`).

## Example (illustrative)

```json
{ "id": "unit-001", "name": "Linear Algebra", "code": "MATH201",
  "status": "active", "colour": "blue",
  "created_at": "2026-09-20T08:00:00Z" }
{ "unit_id": "unit-001", "id": "topic-003", "name": "Eigenvalues",
  "reading_status": "pending", "priority": "high" }
{ "unit_id": "unit-001", "id": "task-001", "title": "Problem set 4",
  "kind": "assignment", "status": "in_progress", "due_at": "2026-10-09T23:59:00Z" }
```
