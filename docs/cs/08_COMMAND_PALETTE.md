# CS-08 — Command Bar: Keeping Every `/command` in a GUI

The TUI/CLI command set is a feature, not legacy. The GUI keeps it verbatim and
makes it discoverable.

## 1. Grammar (superset, never subset)

All TUI commands keep working through `GuiApi.RunCommand`:

```text
/help /dashboard /units /units add /units list /units open <id> /units rename …
/topics add /topics list /topics status <topic-id> <status>
/tasks add /tasks list /tasks deadlines
/timeline /timeline next /timeline previous
/files /files import <path> /files move <src> <dst>
/search <query> /analytics /export /backup /settings /quit(fullscreen→dashboard)
```

Plus GUI-only conveniences: `/calendar`, `/today`, `/open <file-id>`,
`/goto <unit-id>`. CLI `zlanpiko <command> --flags` parity is verified by
reusing the same service calls — the strings differ, the effects are identical.

## 2. Interaction design

- Focus: `/` or `Ctrl+K` focuses; `Esc` blurs/clears; `↑↓` history (persisted
  per data-root, max 200); `Tab` accepts autocomplete; `Enter` runs.
- Autocomplete: fuzzy over commands + live IDs (unit/topic/task names). Typing
  `/topics status eco` suggests `Ecology · topic-004 · unread`.
- Parsing: same tokenizer as TUI `commands.go` (moved to a shared
  `internal/cmdparse` package so TUI + GUI parse identically — not two regex
  piles). Unknown command → `did you mean …?` + `/help` link.
- Output: result renders **in the current view** (e.g. `/units list` filters the
  Units view; `/tasks deadlines` jumps to Timeline) plus a one-line toast.
  Errors render inline red with `hint` (CS-02 §5), never a modal.
- Non-interactive parity: destructive commands (`delete`, `restore`) require
  the same explicit confirmation the CLI demands (`--yes` equiv. checkbox).

## 3. Implementation notes

- `internal/gui/commands.go` dispatches on the parsed verb to `GuiApi` methods;
  add a `parity_test.go` that runs every documented `/command` and its CLI twin
  against the same fixture and diffs the resulting DB rows.
- Frontend `commandbar.ts`: input + dropdown + history; zero business logic —
  it sends the raw string and renders `CommandResultDTO {view, toast, highlight}`.
- `Ctrl+K` global search reuses `/search` ranking; results group by
  units/topics/tasks/files with keyboard navigation.
