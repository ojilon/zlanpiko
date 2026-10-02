-- 0001_init: initial schema (schema_version 1).
-- Forward-only migration. Source of truth for v1 tables; see docs/05.

CREATE TABLE schema_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE units (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  code TEXT,
  description TEXT NOT NULL DEFAULT '',
  colour TEXT,
  status TEXT NOT NULL DEFAULT 'active'
    CHECK (status IN ('active', 'archived')),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX idx_units_name_active ON units(name) WHERE status = 'active';

CREATE TABLE topics (
  unit_id TEXT NOT NULL REFERENCES units(id) ON DELETE CASCADE,
  id TEXT NOT NULL,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  reading_status TEXT NOT NULL DEFAULT 'unread'
    CHECK (reading_status IN ('unread', 'pending', 'read')),
  priority TEXT NOT NULL DEFAULT 'normal'
    CHECK (priority IN ('low', 'normal', 'high')),
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (unit_id, id)
);
CREATE INDEX idx_topics_unit ON topics(unit_id);
CREATE INDEX idx_topics_status ON topics(reading_status);

CREATE TABLE tasks (
  unit_id TEXT NOT NULL REFERENCES units(id) ON DELETE CASCADE,
  id TEXT NOT NULL,
  title TEXT NOT NULL,
  kind TEXT NOT NULL DEFAULT 'assignment'
    CHECK (kind IN ('assignment', 'coursework', 'test', 'examination', 'project', 'other')),
  status TEXT NOT NULL DEFAULT 'not_started'
    CHECK (status IN ('not_started', 'in_progress', 'completed', 'submitted')),
  due_at TEXT,
  completed_at TEXT,
  priority TEXT NOT NULL DEFAULT 'normal'
    CHECK (priority IN ('low', 'normal', 'high')),
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
  topic_id TEXT,
  task_id TEXT,
  rel_path TEXT NOT NULL UNIQUE,
  size INTEGER NOT NULL DEFAULT 0,
  sha256 TEXT,
  missing INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX idx_files_unit ON files(unit_id);
