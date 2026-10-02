# 00 — Project Overview

## Purpose

A local-first academic management application for Windows. One student uses it to
organise course units, topics and reading progress, assignments and deadlines,
study files, and a weekly overview — from a terminal (TUI) and from a
scriptable command line (CLI), with no internet connection required.

Working placeholder identity (see `docs/12_CONFIGURATION_AND_INSTALLER.md`):

| Field              | Value                                        |
| ------------------ | -------------------------------------------- |
| Application name   | zlanpiko                                       |
| Executable         | `zlanpiko.exe`                                 |
| Installer          | `zlanpiko-installer.exe`                       |
| Application ID     | `com.local.zlanpiko`                           |
| Version            | `0.1.0` (pre-release, Phase 0)               |
| DB schema version  | `1`                                          |
| Default storage dir| `AcademicData`                               |
| Repository         | placeholder (no remote selected yet)         |
| Release channel    | `dev`                                        |

The name is intentionally not final. It appears only in `configs/app.json` and
`configs/build.json`; source code must read it from there (see `docs/03`).

## Target user and environment

- A student with **9+ course units**, each with topics, readings, assignments,
  tests and examinations.
- Machine: Windows 10/11, possibly low-end (limited RAM/disk). No admin rights
  assumed. No Go toolchain on the user's machine — they receive two `.exe` files.
- Terminal: Windows Terminal, conhost, or any ANSI-capable console, possibly
  small (80×24) with limited colours. The TUI must degrade gracefully.
- Phone-carrying use case: the student may leave the PC at home, so a
  plain-text status report exportable to a phone is a first-class feature
  (see `docs/11`).

## Scope of the first usable version (MVP)

In scope:

1. Course units (create, rename, archive, safe delete), unlimited count.
2. Topics per unit with `Unread / Pending / Read` status.
3. Assignments, coursework, tests, examinations with deadlines and statuses.
4. Real folder tree on disk per unit/topic/task, with import, move, rename,
   open and safe delete.
5. TUI: dashboard, units, topics, tasks, files, week timeline, analytics,
   settings, persistent command input.
6. CLI mirroring the same operations, usable from any directory.
7. Plain-text + JSON status reports, backup and restore.
8. Separate installer executable with first-run setup and PATH integration.
9. Reproducible Windows builds and release packaging.

Explicitly out of scope for v1 (see `docs/17`): cloud sync, calendar sync,
rich/customisable dashboards, AI workflows, automatic updates, portable mode
beyond what falls out naturally.

## How the pieces relate

```text
                ┌─────────┐   ┌─────────┐
                │   TUI   │   │   CLI   │
                │(Bubble │   │(stdlib  │
                │  Tea)   │   │ flag)   │
                └────┬────┘   └────┬────┘
                     │             │
                     ▼             ▼
              ┌─────────────────────────┐
              │  application services   │  units, topics, tasks,
              │  (internal/services +   │  timeline, analytics,
              │   domain packages)      │  importer/exporter/backup
              └────────┬────────────────┘
                       │
            ┌──────────┴──────────┐
            ▼                     ▼
     ┌─────────────┐      ┌──────────────┐
     │   SQLite    │      │  filesystem  │
     │ (source of  │◄────►│ (real folders│
     │  truth)     │sync  │  + files)    │
     └─────────────┘      └──────────────┘
```

- **TUI and CLI are thin fronts.** Both call the same service functions in
  `internal/services` (and focused packages: `tasks`, `timeline`, `analytics`,
  `importer`, `exporter`, `backup`, `filesystem`). Neither front talks to SQL
  or the filesystem directly.
- **SQLite is the source of truth** for relationships, statuses, deadlines and
  analytics. The filesystem holds real user files plus JSON sidecars that are a
  recoverable cache, never an independent truth (see `docs/04`, `docs/06`).
- **Config is separate from data**: install/config paths vs. the academic data
  root are distinct locations (see `docs/12`).

## Documents map

| Doc   | Subject |
| ----- | ------- |
| 01 | Requirements + acceptance criteria |
| 02 | System architecture + module diagram |
| 03 | Project structure + package rules |
| 04 | Data model (what lives in DB vs. files) |
| 05 | SQLite schema, migrations, recovery |
| 06 | Filesystem layout + safe operations |
| 07 | TUI design, screens, shortcuts |
| 08 | CLI specification |
| 09 | Tasks and deadlines semantics |
| 10 | Analytics and coverage formulas |
| 11 | Import, export, backup |
| 12 | Configuration and installer |
| 13 | Testing strategy |
| 14 | Build and release |
| 15 | Implementation roadmap (phases 0–10) |
| 16 | Security and data safety |
| 17 | Future extensions (deferred) |

`IMPLEMENTATION_STATUS.md` tracks current phase; `DECISIONS.md` records
architecture decisions with rationale.
