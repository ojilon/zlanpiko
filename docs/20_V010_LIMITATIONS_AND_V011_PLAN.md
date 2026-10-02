# 20 — v0.1.0 Known Limitations and v0.1.1 Plan

v0.1.0 (tag `v0.1.0`) is the first usable release: all of `docs/15` phases
0–10 are implemented, tested and rehearsed. It is reliable for its designed
scope, but it has sharp edges. This document lists every one the author is
aware of, then proposes the v0.1.1 fix/feature list. Nothing here is
conjecture — each item traces to actual code behavior (see `docs/21`).

## A. What v0.1.0 does not do (by design, not bugs)

These were deferred deliberately (`docs/17`): no cloud sync, no
calendar sync, no reminders/notifications daemon, no full-text content
search, no encryption at rest, no multi-user support, no portable mode,
no auto-updates, no AI features, no web/mobile front. The attention labels
are workload indicators, never performance predictions.

## B. Known limitations and rough edges

### B1. No way to create an empty folder (CLI and TUI)

`files` has list/import/move/rename/delete/open/search but **no mkdir**, and
the TUI files screen has no create-folder key either. New folders only
appear as a side effect of creating units/topics/tasks or as import
destinations. So "add an empty folder named X" is currently impossible
without smuggling a file in. → v0.1.1 item 1.

### B2. TUI/CLI parity gaps (CLI can, TUI cannot)

- Unit add dialog has no **description** field (`units add --description` only).
- Task edit dialog cannot change **kind** (`tasks edit --kind` only).
- Task edit dialog cannot edit **notes** as a field (form has title/due/
  priority/description; notes only via CLI).
- No `/search` command yet (reserved word, prints "later release"); CLI
  `files search` covers filenames only.
- Export/backup in the TUI run with fixed defaults (text report, full
  backup) — no per-unit or JSON options like the CLI.
- `topics move` target typed blind (no picker); CLI has the same UX, equal.
- No `config set` UI beyond theme cycling (data_root is CLI-only).

### B3. Behaviors that surprise

- `config set data_root` **repoints, it does not move**: existing data stays
  where it was; you must move/copy the folders yourself, then verify.
- `export --out` and `backup create --out` refuse existing files — delete or
  rename the old one deliberately first (no `--force` in v0.1.0).
- Two backups within the same second collide by timestamped name and the
  second is refused.
- `backup create` without `--full` is **database-only** (no files) — the word
  "backup" alone may oversell it; always pass `--full` for real safety.
- Analytics thresholds (3/7/14 days, 50/70%) and the 14-day dashboard window
  are hardcoded; the 100-result search cap and 100-command history cap too.
- Attention labels ignore everything except deadlines + reading flags (by
  design, but easy to over-read — `docs/10` states the inputs plainly).
- Coverage counts topics only: files read and assignments done do not move it.
- Archived units vanish from every default view including analytics; the
  `--archived` flag (CLI) is the only way back, plus unarchive.
- Dateless tasks never appear on the timeline (they live under Tasks).
- Completed tasks stay visible in lists until filtered out — history by design,
  but noisy with `--status` unset.
- The TUI import cannot be cancelled once started (`esc` only closes dialogs).
- First-run setup on a **piped** stdin errors instead of guiding — correct
  (never invent storage), but the message assumes you know about
  `--data-root`/`$ZLANPIKO_DATA`/installer.
- Only one writer at a time: a second TUI/CLI on the same data waits on the
  5 s busy timeout, then errors. Close the other window.
- Logs go to **stderr only** — there is no persistent log file, so diagnosing
  a past crash means re-running with redirection.

### B4. Installer and packaging notes

- The interactive drive/folder chooser was unit-tested and rehearsed for
  defaults/flags/pipes, but a full hands-on run (drive → folder → confirm)
  on a real terminal has not been done by the author yet — do it before
  handing the installer to anyone else.
- PATH/shortcut steps print manual fallback instructions on failure; verify
  them once on a real machine (they were tested with fakes/skips in CI).
- `release\` and `dist\` are git-ignored build outputs; the `--help` of the
  installer documents flags, but there is no graphical installer (console
  only, by design).
- `configs/app.json` still points `repository` at `TBD` and channel `dev`.

### B5. Documentation gaps this release fixes

- Until now there was no user guide: `docs/08` specified the CLI and
  `docs/07` the TUI layout, but neither taught daily use. `docs/18` (CLI)
  and `docs/19` (TUI) are new in this batch.
- `AGENTS.md` still used the old `academic.*` placeholder names; corrected
  in this batch (mechanical identity references only).

## C. Proposed v0.1.1 scope (for the maintainer to confirm/cut)

Prioritised by user pain, smallest safe change first. Each item states its
acceptance check.

1. **`files mkdir PATH` + TUI `m` key** — create empty folders anywhere
   unprotected (B1). Accepts: `mkdir inbox/reader` then `list` shows it;
   TUI key mirrors it; traversal/reserved-name rules shared with existing ops.
2. **TUI form parity** — description on unit add; kind (+notes field) on task
   edit (B2). Accepts: everything `tasks edit`/`units add` accept is settable
   in the TUI.
3. **`/search` that actually searches** — filenames (existing scan) plus
   unit/topic/task names from the DB, one results view, jump-to-item.
   Accepts: `/search eigen` finds the topic, the file, and jumps on select.
   (Content indexing stays deferred.)
4. **`config set data_root` that moves** — `--move` flag copying DB + files
   to the new root, verifying, then repointing; refusal when the target is
   non-empty without `--overwrite-data`. Accepts: old root empty-ish,
   new root verifies clean.
5. **Persistent log file** — `logs/zlanpiko-<date>.log` under the data root
   (or install dir), rotation by date, `--log-level` plumbing. Accepts: crash
   leaves a file with the diagnostic lines currently on stderr.
6. **`export --force`** (overwrite with explicit consent) and **backup name
   auto-suffix** (`-2`, `-3`) instead of same-second refusal.
7. **Installer real-terminal rehearsal** + PATH verification on a real
   machine (B4); record results in `IMPLEMENTATION_STATUS.md`.
8. **Small UX**: `units show` full detail (topics/tasks preview); empty-state
   hints on Tasks/Timeline when there is nothing to show; confirm dialog
   before TUI `q` with unsent input? (cheap, prevents lost typing).

Explicitly **not** in v0.1.1: anything from section A (sync, reminders,
content search, encryption, multi-user, portable flag, auto-update, AI),
configurable analytics thresholds (revisit only on request), graphical
installer, `Makefile`/CI (`.bat` scripts stand until a *nix need appears).

## D. How to use this document

When a limitation is fixed, delete its B-item and check the C-item off in
the release notes (`CHANGELOG.md`), keeping the numbers stable so external
references (`docs/20` item numbers) keep working. Newly found limitations go
in section B with the version found.
