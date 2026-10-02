# 11 — Import, Export and Backup

## File import

Single-file: `files import SRC --to DEST` copies one file; bulk:
`files import <dir> --to DEST` or `zlanpiko . [--to DEST]`.

- `DEST` is a unit/topic/task folder or `inbox`; default `inbox/<YYYY-MM-DD>/`.
- Preview-first for bulk: list of files + sizes, destination mapping, and
  collisions. Nothing copies without `--yes` (non-TTY) or interactive confirm.
- Skipped by default (reported, not copied): hidden files, `.exe/.msi/.bat`,
  files > 100 MB (override `--include-large`), reparse points.
- Collisions: `skip | rename (file-2.ext) | overwrite` — overwrite needs
  explicit per-file confirm; default is `rename`.
- Implementation: `importer.Preview()` (pure plan) then `importer.Execute()`
  (DB row → copy via temp+rename → hash → sidecar), cancellable via context.
  See `docs/06` for the safe-copy primitives.

## Text status report (phone-friendly)

`zlanpiko export --format txt [--unit U] [--out FILE]` renders from live DB
rows (never hand-edited text):

```text
ACADEMIC STATUS REPORT
Generated: 2026-10-06 09:30 (local) · zlanpiko 0.1.0

UNITS (9 active)
----------------
Linear Algebra [MATH201] — 20 topics (R8/P5/U7) 40.0% · 2 active · next Oct 09
...

NEEDS ATTENTION
---------------
! Problem set 4 (Linear Algebra) — due 2026-10-09, In Progress
! Eigenvalues (Linear Algebra) — Unread, test in 3 days

UPCOMING (14 days) / OVERDUE / SUMMARY (totals + overall coverage)
```

Per-unit mode narrows to one unit. Wrapping at 72 columns for phone screens.

## JSON export

`--format json`: stable schema `{app, generated_at, units[], topics[],
tasks[], summary}` with the same data as the text report (field names frozen
in v1; additions only). Used by future integrations and by the
export→wipe→restore round-trip test.

## Backup and restore (package `backup`)

- Manifest: `{app, version, schema_version, created_at, kind: full|db-only,
  files: [{path, sha256, size}], db_sha256}`.
- Create: `VACUUM INTO <tmp>` for a consistent DB snapshot (never raw-copy a
  live DB) → zip DB + files + `manifest.json` → `backups/backup-<UTCts>.zip`.
  Refuse to overwrite an existing name; verify by re-reading the zip and
  re-hashing the DB entry.
- `backup list` shows date/kind/size; `verify FILE` re-checks hashes;
  `restore FILE --yes` requires the target data root to be either empty or an
  explicit `--overwrite-data` (which first takes a safety backup). Restore
  re-runs migrations if the backup's schema is older; refuses if newer.
- Available identically in TUI Settings and CLI.

## Formats and portability

- Text report: UTF-8, CRLF on Windows, ≤ 72-col lines, no ANSI escapes.
- Zip layout: `/database/zlanpiko.db`, `/files/…` (root-relative), `/manifest.json`.
- Restore is verified by the round-trip acceptance test
  (export → wipe → restore → identical export), see `docs/13`.
