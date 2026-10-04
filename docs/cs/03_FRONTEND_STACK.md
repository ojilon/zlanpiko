# CS-03 — Frontend Stack: pnpm + Vite + TypeScript (Frameless)

## 1. Decisions

- **Vite + TypeScript + hand CSS, no component framework.** No React/Vue/Svelte.
  Rationale: smallest bundle for low-end Windows PCs, zero framework lock-in,
  easiest Wails embedding, and the UI is mostly cards/lists/SVG — framework
  overhead buys nothing. Revisit only if view complexity forces it.
- **pnpm** as the Node package manager (`pnpm@11.25.0` verified). `node_modules/`
  lives under `frontend/` only and is git-ignored.
- **Charts hand-rolled as SVG** (CS-07). No chart.js/echarts dependency in v0.2.
- **No Tailwind.** A ~400-line design-token stylesheet keeps the light-blue/grey
  identity reproducible and the binary small.

## 2. Target tree

```text
frontend/
  package.json          # type: module, scripts: dev/build/preview/test
  pnpm-lock.yaml        # committed
  vite.config.ts        # outDir ../dist-frontend (embedded by Wails, git-ignored)
  tsconfig.json         # strict: true, noUnusedLocals
  index.html            # shell: #titlebar, #sidebar, #view, #commandbar
  src/
    main.ts             # boots shell, frameless controls, router, command bar
    api.ts              # typed wrapper over window.go.zlanpiko.GuiApi.*
    types.ts            # mirrors internal/gui/dto.go (hand-kept, tested)
    views/              # dashboard.ts units.ts topics.ts tasks.ts timeline.ts
                        # calendar.ts files.ts analytics.ts settings.ts
    components/         # titlebar.ts cards.ts timelineStrip.ts deadlineBar.ts
                        # charts.ts commandbar.ts toasts.ts modal.ts
    styles/             # tokens.css base.css cards.css titlebar.css views.css
  assets/               # logo.svg, icons (inline SVG preferred)
```

Go embeds the built output, not the source:

```go
// cmd/gui/main.go
//go:embed all:../../dist-frontend
var distFS embed.FS
```

`frontend/` source stays out of the exe; only the Vite build ships.

## 3. Frameless window — hard requirement

Wails config (`wails.json`, created by `wails init` then edited):

```json
{
  "frameless": true,
  "fullscreen": false,
  "resizable": true,
  "minWidth": 960,
  "minHeight": 640,
  "windows": { "theme": "system", "backdropType": "Mica" }
}
```

Frontend contract (`src/components/titlebar.ts` + `styles/titlebar.css`):

- Custom `#titlebar` (32–40px): app dot + `zlanpiko` wordmark left; week/version
  centre; `— ▢ ✕` buttons right calling Wails runtime
  (`WindowMinimise/ToggleMaximise/Quit` via the runtime bridge `api.ts` exposes).
- Drag region: `body { --wails-draggable: drag }` equivalent —
  `#titlebar { -webkit-app-region: drag; }` with buttons set to
  `-webkit-app-region: no-drag` (Wails v2 maps this to drag handling on Windows).
- Double-click titlebar toggles maximise; right-click shows system menu fallback.
- Keyboard: `Alt+Space` still opens the window menu (do not swallow it);
  window controls must be reachable by Tab with visible focus rings.
- Fallback: `--native-frame` dev flag and `GetConfig().nativeFrame` escape hatch
  that rebuilds with `frameless:false` for remote-desktop / screen-reader users.
- The **installer stays native-framed** (standard OS dialogs) — frameless applies
  to `zlanpiko-gui.exe` only.

Test: drag by titlebar moves window; buttons minimise/maximise/close; resize from
all edges; maximise restores custom maximized padding; taskbar icon + Alt-Tab OK.

## 4. Setup steps (exact)

```cmd
:: PowerShell 5.1 blocks pnpm ps1 — use cmd.exe for everything below
cmd /c "cd frontend & pnpm add -D typescript vite vitest"
cmd /c "cd frontend & pnpm exec tsc --init"
cmd /c "wails init -n zlanpiko-gui -t vanilla"  :: then MOVE output into frontend/
```

`package.json` scripts:

```json
{
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview",
    "typecheck": "tsc --noEmit",
    "test": "vitest run"
  }
}
```

Type generation: `wails generate module` is optional; this plan hand-keeps
`src/types.ts` against `internal/gui/dto.go` and enforces parity with a Go test
that golden-files one JSON sample per DTO (CS-10). No silent drift.

## 5. Styling tokens (single source)

```css
:root {
  --bg: #0f141b; --panel: #161d27; --line: #2a3442;   /* thin card borders */
  --ink: #dbe4f0; --dim: #93a1b5;
  --accent: #7fb6e8; --accent-soft: #22374d;          /* light-blue identity */
  --ok: #6fbf8f; --warn: #e0b46a; --bad: #e07a7a;
  --radius: 10px; --border: 1px solid var(--line);
}
```

Light mode mirrors these tokens (OS theme aware, manual toggle in Settings).
All cards: `1px solid var(--line)`, `border-radius: 10px`, no heavy shadows —
thin demarcations as requested.
