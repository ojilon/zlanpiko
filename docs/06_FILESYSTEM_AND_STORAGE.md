# 06 — Filesystem and Storage

## Data-root layout (authoritative)

```text
<AcademicData>/                       # user-chosen, e.g. D:\AcademicData
├── database/
│   └── zlanpiko.db                   # SQLite (truth), see docs/05
├── config/
│   └── user.json                     # data-local prefs (display, last view)
├── backups/                          # *.zip created by backup flow (docs/11)
├── exports/                          # generated reports (txt/json)
├── reports/                          # reserved: saved report snapshots [D]
├── units/
│   └── unit-001/                     # stable ID; renames never touch this
│       ├── unit.json                 # sidecar cache (regenerated)
│       ├── topics/
│       │   └── topic-001/
│       │       ├── topic.json
│       │       ├── reading/
│       │       ├── research/
│       │       └── summaries/
│       ├── assignments/              # tasks grouped by kind folder
│       │   ├── task-001/             # task.json + materials/ drafts/ submissions/
│       │   └── task-002/
│       └── notes/
└── inbox/                            # unlinked imports land here
```

`topics/<id>/` always gets `reading/`, `research/`, `summaries/`; task folders
get `materials/`, `drafts/`, `submissions/`. Extra user folders are allowed and
left alone. IDs are stable (`docs/04`); display names live in the DB.

## Path resolution

- All in-root paths resolve via `filesystem.Join(root, rel...)` which Cleans,
  rejects absolute segments, `..`, drive letters, UNC prefixes and (on Windows)
  reserved names (`CON`, `NUL`, …); violations are errors, never clamped.
- Stored `rel_path` values use `/` separators; conversion to OS paths happens
  at the boundary only.
- Symlinks/junctions under the root are not followed by scans (listed as-is).

## Safe operations (package `filesystem`)

| Op | Rule |
| -- | ---- |
| create | `MkdirAll` for known skeleton; single files via temp + rename |
| rename/move | same-volume rename; cross-volume = copy + verify + delete source; collision → fail with alternatives, never overwrite |
| delete file | require explicit confirm if linked to a topic/task record |
| delete non-empty folder | require explicit confirm; refuse if it contains a linked record's folder unless the record is being deleted too |
| import | copy to temp name in destination, hash, then rename; record row written first (DB-first), file copied second |
| open | `rundll32 url.dll,FileProtocolHandler <path>` (documented Windows launcher), no execution of content |

Bulk moves/imports build a **plan** (source → dest pairs + collision list)
shown to the user before anything is copied (see `docs/11` for import flow).

## DB ↔ filesystem consistency

- SQLite is truth for structure/status; sidecars (`unit.json`, …) mirror the
  row and are rewritten by the same service call after the DB commit.
- Staleness rule: if sidecar `updated_at` ≠ row `updated_at`, the DB wins;
  sidecar is regenerated on next write or by `maintenance verify --repair`.
- Missing-file scan: walk root, compare with `files` table; set `missing=1`
  for absent paths, report unindexed files (offer import from `inbox/`-style
  flow, never auto-link).
- Crash order (DB commit → file op) is deliberate: a half-finished op leaves a
  DB row with `missing=1`, which verify detects; the reverse (file without row)
  would be invisible. Full rationale in `docs/16`.

## Metadata and search (v1)

- Metadata: size, mod time, extension; `sha256` computed at import/backup,
  lazily otherwise.
- Search v1 = filename substring over the root (case-insensitive) + record
  name search in DB. Content indexing is deferred (`docs/17`).
