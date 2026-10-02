# 16 — Security and Data Safety

Principle: **no silent loss, no silent overwrite, fail loudly**. This doc is
the safety checklist every phase must satisfy; violations block the checkpoint.

## Path safety (`filesystem` boundary, tested in Phase 4)

- Reject absolute segments, `..`, drive letters, UNC, reserved Windows names,
  and over-long paths before touching disk; errors name the offending input.
- Never follow symlinks/junctions during scans, imports or deletes.
- All writes stay under the resolved data root; `academic .` never writes
  outside it (CWD files are *read*, then copied in).

## Database safety

- Foreign keys on; related rows change inside one `BEGIN IMMEDIATE` transaction.
- Crash order is **DB commit → filesystem op → sidecar refresh**: a crash
  leaves a detectable `missing=1` row (fixable by verify/retry), never an
  invisible orphan file. Backups use `VACUUM INTO`, not raw copies (`docs/11`).
- Newer-schema DBs are refused, never auto-migrated down; migrations are
  forward-only and tested with seed data (`docs/05`, `docs/13`).

## Destructive operations (all fronts, same rules)

| Operation | Guard |
| --------- | ----- |
| delete unit/topic/task with children | refuse without explicit confirm (`--yes` / dialog showing counts); suggest archive |
| delete non-empty folder | refuse without explicit confirm |
| filename collision | `rename` by default; `overwrite` needs per-file consent |
| existing backup name | never overwrite; new timestamped name instead |
| restore | target must be empty or `--overwrite-data` (which forces a safety backup first) |
| reinstall / update | program files only; data root untouched; pre-update backup |

Non-TTY CLI defaults to refusal (no hanging prompts): missing `--yes` = error.

## Partial-operation recovery

- Copies go to temp names + rename; interrupted ops leave temp files, never
  half-written destinations (startup/verify cleans `*.tmp` after listing them).
- Bulk import executes per-file with a journal (planned/done/failed) printed
  at the end; re-running skips completed entries.
- `maintenance verify [--repair]` reports: integrity check, orphan rows/files,
  stale sidecars (DB wins, regenerate), temp leftovers.

## Malformed input

Bad config JSON (path + line in error), malformed import JSON (per-item errors,
valid items importable on retry), bad dates (accepted formats shown), past
deadlines (allowed but flagged). Logs (`log/slog` to file) keep full detail;
user messages stay short and actionable.

## CLI operating on external folders

`import`/`academic .` treat sources as untrusted: read-only access, skip
executables/hidden/oversize by default (lists skips), never delete or modify
sources. Destination preview precedes every bulk write.

## What v1 does NOT claim

No encryption at rest (OS-level protection assumed; noted as future work in
`docs/17`), no multi-user access control (single-user file ownership), no
network attack surface (no listeners, no auto-update downloads).
