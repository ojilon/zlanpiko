# 07 — TUI Design

Stack: Bubble Tea (Elm model), Lipgloss styling, Bubbles components
(`list`, `table`, `textinput`, `viewport`, `spinner`) where they fit;
hand-rolled rendering where they don't (timeline grid, coverage bars).

## Visual identity

- Accent: light-blue (`#7aa2f7`-ish) for selections/headings; grey
  (`#565f89`-ish) for borders/inactive; restrained red/yellow/green only for
  overdue/attention/completed markers. 16-colour fallback theme when the
  terminal reports limited support (`NO_COLOR` respected: no colour at all).
- Subtle single-line borders, one header + one status bar; no animation except
  a spinner for long ops. Density over decoration: tables and compact bars.

## Layout (adapts to size; reference 100×28)

```text
┌ Academic Manager ───────────── [Tue 2026-10-06 · Week 41] ──── 0.1.0 ─┐
│ ▶ Dashboard  Units  Topics  Tasks  Files  Timeline  Analytics  Settings│
├────────────────────────────────────────────────────────────────────────┤
│                                                                        │
│  DYNAMIC CONTENT (viewport; scrolls; per-screen help line at bottom)    │
│                                                                        │
├────────────────────────────────────────────────────────────────────────┤
│ Status: 9 units · 42 topics · 3 due this week · 1 overdue              │
│ > / command or search …                                    [history ↑↓]│
└────────────────────────────────────────────────────────────────────────┘
```

- Header: app name, date/week, version — never duplicated inside screens.
- Tab strip doubles as navigation (`1–8` jump; `←/→`, `h/l`, `tab` cycle).
- Status bar: live counts from the same analytics queries as the dashboard.
- Command input is **persistent**: text, history and validation errors survive
  screen switches (input state lives in the root model, not in screens).

## Screens (one file each in `tui/screens/`)

1. **Dashboard**: counts (units/topics read-pending-unread), per-unit coverage
   bars, next 5 deadlines, overdue list, attention items (top 8). Read-only
   except item selection → jumps to detail.
2. **Units**: table (name, code, topics, coverage, active tasks, next due);
   `a` add, `r` rename, `A` archive/unarchive, `D` delete (confirm), `enter` open.
3. **Topics**: unit picker + topic list; `x` cycles `Unread→Pending→Read`;
   `f` filter by status; sorting `s`; `enter` opens folder actions.
4. **Tasks**: consolidated table with filters (unit, kind, status incl.
   derived overdue, priority); `e` edit deadline inline; `c` complete/submit.
5. **Files**: two-pane tree + detail; `i` import, `m` move, `R` rename,
   `o` open, `Del` safe delete; bulk ops show plan preview first.
6. **Timeline**: week grid (Mon–Sun columns collapse to rows under 90 cols);
   `n/p` next/prev week, `t` today; `enter` detail → edit → jump to unit/folder.
7. **Analytics**: coverage tables + attention breakdown; shows the formula
   used (`read/total`) under each table (reproducibility, `docs/10`).
8. **Settings**: storage path, theme (auto/16-colour/none), backup/export
   shortcuts, DB verify, version, keybinding help.

Minimal wireframe (units screen):

```text
 UNITS (9 active, 1 archived)                    [a]dd [r]ename [A]rchive
 ─────────────────────────────────────────────────────────────
 > MATH201  Linear Algebra      20 topics  ██████░░░░ 40%  2 active  due Oct 09
   PHYS110  Mechanics           14 topics  ███░░░░░░░ 21%  1 active  due Oct 12
   …                                                              f: filter
```

## Keyboard and commands

Global: `1–8` screens, `/` focus input, `Esc` blur, `?` help overlay,
`q` back/quit (with dirty-check), `↑↓/jk` move, `enter` open.
Screen-local keys listed above; every destructive key opens a confirm dialog
showing consequences (child counts). All actions are also reachable as `/`
commands (grammar in `docs/08` §TUI) so keyboard-only and command-only users
both succeed.

## Responsiveness

- `< 80 cols or < 20 rows`: compact mode (hides tab labels → numbers,
  timeline becomes a list, tables drop low-value columns). Minimum supported:
  80×24; below that show a polite "terminal too small" overlay instead of
  corrupt rendering.
- Resize (`WindowSizeMsg`) re-flows the current screen; input text preserved.

## TUI-specific rules

- Never block the event loop: imports/backups/verify run as `tea.Cmd`
  with progress messages and `ctx` cancellation (`Esc` cancels).
- Errors surface as inline status messages + log entries, never panics;
  partial input is kept so the user can fix and retry.
