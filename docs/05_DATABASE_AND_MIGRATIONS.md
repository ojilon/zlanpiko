# 05 — Database and Migrations

- Driver: `modernc.org/sqlite` (pure Go, no CGO; see DECISIONS.md D2).
  DSN: `file:<root>/database/zlanpiko.db` with `_pragma=foreign_keys(1)`.
- Single-writer assumption: one process holds the DB; TUI *or* CLI per
  invocation, both short-lived except the TUI session. `busy_timeout = 5000ms`,
  WAL mode enabled at open (`PRAGMA journal_mode=WAL`).

## Schema (v1, migration `0001_init.sql`)

```sql
CREATE TABLE schema_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
-- row: ('schema_version','1')

CREATE TABLE units (
  id TEXT PRIMARY KEY,                       -- 'unit-001'
  name TEXT NOT NULL,
  code TEXT,
  description TEXT NOT NULL DEFAULT '',
  colour TEXT,
  status TEXT NOT NULL DEFAULT 'active'
    CHECK (status IN ('active','archived')),
  created_at TEXT NOT NULL,                  -- UTC RFC3339
  updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX idx_units_name_active ON units(name) WHERE status='active';

CREATE TABLE topics (
  unit_id TEXT NOT NULL REFERENCES units(id) ON DELETE CASCADE,
  id TEXT NOT NULL,                          -- 'topic-001', per-unit scope
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  reading_status TEXT NOT NULL DEFAULT 'unread'
    CHECK (reading_status IN ('unread','pending','read')),
  priority TEXT NOT NULL DEFAULT 'normal'
    CHECK (priority IN ('low','normal','high')),
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (unit_id, id)
);
CREATE INDEX idx_topics_unit ON topics(unit_id);
CREATE INDEX idx_topics_status ON topics(reading_status);

CREATE TABLE tasks (
  unit_id TEXT NOT NULL REFERENCES units(id) ON DELETE CASCADE,
  id TEXT NOT NULL,                          -- 'task-001', per-unit scope
  title TEXT NOT NULL,
  kind TEXT NOT NULL DEFAULT 'assignment'
    CHECK (kind IN ('assignment','coursework','test','examination','project','other')),
  status TEXT NOT NULL DEFAULT 'not_started'
    CHECK (status IN ('not_started','in_progress','completed','submitted')),
  due_at TEXT,                               -- UTC RFC3339, NULL = no deadline
  completed_at TEXT,
  priority TEXT NOT NULL DEFAULT 'normal'
    CHECK (priority IN ('low','normal','high')),
  description TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (unit_id, id)
);
CREATE INDEX idx_tasks_unit ON tasks(unit_id);
CREATE INDEX idx_tasks_due ON tasks(due_at) WHERE due_at IS NOT NULL;
CREATE INDEX idx_tasks_status ON tasks(status);

CREATE TABLE files (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  unit_id TEXT REFERENCES units(id) ON DELETE CASCADE,
  topic_id TEXT,                             -- local id within unit_id
  task_id TEXT,                              -- local id within unit_id
  rel_path TEXT NOT NULL UNIQUE,             -- '/'-separated, root-relative
  size INTEGER NOT NULL DEFAULT 0,
  sha256 TEXT,
  missing INTEGER NOT NULL DEFAULT 0,        -- 0/1
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX idx_files_unit ON files(unit_id);
```

Notes: `topic_id`/`task_id` are validated at the service layer against the
composite keys (SQLite cannot FK two nullable columns cleanly at this scale —
documented trade-off). `overdue` is never stored (derived at read, `docs/09`).

## Initialization

`app.Open()`: ensure `<root>/database/` exists → open DB → `PRAGMA` setup →
read `schema_version` (missing table = fresh DB) → apply pending migrations in
filename order inside one transaction each → `VACUUM` only on explicit
maintenance command, never automatically.

## Migrations

- Files: `migrations/NNNN_name.sql`, forward-only, no destructive
  (`DROP COLUMN`/`TABLE`) migrations in v1 line — archive instead.
- Each migration updates `schema_meta.schema_version` in the same transaction.
- Runner records applied names; unknown/newer version than the binary
  understands → refuse to open with a clear error (never auto-downgrade).
- Every migration ships with a test: migrate a seeded v(N-1) DB in a temp dir,
  assert rows survive (see `docs/13`).

## Transactions and safety

- Related writes (e.g. create topic + sidecar refresh intent) wrap DB changes
  in one `BEGIN IMMEDIATE … COMMIT`; filesystem step follows (order rationale
  in `docs/16`).
- Consistency backup uses the SQLite backup API (`VACUUM INTO`) to a temp file,
  then moves it into `backups/` — never a raw copy of a live DB (see `docs/11`).
- Corruption handling: on open failure, report path + suggest restore from
  latest backup; provide `zlanpiko maintenance verify` (CLI) running
  `PRAGMA integrity_check` plus orphan-file scan (see `docs/08`).

## Recovery considerations

- Fresh install with existing data root: detect `zlanpiko.db`, migrate, keep data.
- Orphan sidecars (DB row missing) and orphan DB rows (file missing, `missing=1`)
  are reported by the verify command, never silently deleted.
