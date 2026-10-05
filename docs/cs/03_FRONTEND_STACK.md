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
  public/
    backgrounds/        # optional backdrop photos + manifest.json (docs/24)
  src/
    main.ts             # boots shell, frameless controls, router, command bar
    api.ts              # typed wrapper over window.go.zlanpiko.GuiApi.*
    types.ts            # mirrors internal/gui/dto.go (hand-kept, tested)
    theme.ts            # active theme + system (data-theme on <html>)
    themes.ts           # registry: 10 photo-aware presets (add one to plug in)
    themes.test.ts      # registry completeness + safe slider ranges
    appearance.ts       # background + transparency values -> CSS custom props
    backgrounds.ts      # bundled manifest + IndexedDB store for "my image"
    styles.css          # all tokens and rules, 12 numbered sections
    views/              # dashboard.ts units.ts topics.ts tasks.ts timeline.ts
                        # calendar.ts files.ts analytics.ts settings.ts
    components/         # drawer.ts guide.ts commandbar.ts appearance.ts
  assets/               # logo.svg, icons (inline SVG preferred)
```

Note: the styling stays in **one** `styles.css` rather than the `styles/`
split floated in earlier drafts — one file keeps the ~1.4 k lines of tokens and
rules easy to diff in order, and avoids a dozen `@import` round-trips on a slow
WebView. Revisit only if it passes ~2 k lines.

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
  centre; `— ▢ ✕` buttons right calling the Wails runtime bridge
  (`window.runtime.WindowMinimise/WindowToggleMaximise/Quit`, exposed via
  `api.ts` — note: `window.runtime`, NOT `window.go.runtime`).
- Drag region: `#titlebar { --wails-draggable: drag; }` with buttons set to
  `--wails-draggable: no-drag` (the Wails v2 property; `-webkit-app-region`
  does not work — fixed in 5cd4e45).
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

`src/styles.css` is the single source of truth, organised in twelve numbered
sections (tokens → themes → surfaces → backdrop → reset → shell → cards →
parts → views → drawer/guide → appearance controls → responsive).

Ten themes ship, selected by `data-theme` on `<html>` (set in `index.html` so
the first paint is already themed): five dark (Abyss, Forest, Tide, Ember,
Dusk) and five light (Paper, Meadow, Shore, Sunshine, Bloom). Each pairs its
palette with a photo family — Forest's fern green and warm amber for nature
shots, Tide's sea-glass cyan for water, Ember's glow for sunsets, and so on —
and brings a recommended photo look (transparency, blur, dim) that the
Settings gallery applies on selection. Old `dark`/`light` values stored by
earlier versions map onto Abyss/Paper, so updating never resets a user's look.
To plug in an eleventh theme: add a preset to `themes.ts`, a
`[data-theme="<id>"]` block plus a `.theme-sw-<id>` swatch in `styles.css` —
nothing else reads theme names directly.

Each theme defines its palette as **raw
channel triplets** rather than hex, so every surface can be re-tinted at any
alpha:

```css
[data-theme="abyss"] {
  --panel-rgb: 22 30 41;      /* not --panel: #161d27 */
  --ink-rgb: 221 229 240;
  --accent-rgb: 127 182 232;  /* light-blue identity, unchanged */
  --ok-rgb: 111 191 143; --warn-rgb: 224 180 106; --bad-rgb: 224 122 122;
}

/* derived, opacity-aware — resolved lazily, so --card-alpha can change live */
:root, [data-theme] {
  --surface:       rgb(var(--panel-rgb) / var(--card-alpha));
  --surface-chrome: rgb(var(--panel-rgb) / calc(0.4 + 0.6 * var(--card-alpha)));
  --surface-inset:  rgb(var(--inset-rgb) / calc(0.35 + 0.65 * var(--card-alpha)));
}
```

That indirection is what makes the optional background image possible
(`docs/24`): Settings writes `--card-alpha`, `--card-blur-n`, `--bg-blur-n`,
`--bg-dim`, `--grain` and `--bg-image` on `<html>`, and the whole app re-tints
without re-rendering. Chrome and inputs derive *higher* alphas than cards, so
text stays readable while gutters show the photo.

Type uses a fluid scale (`--fs-2xs` … `--fs-2xl`, `clamp()` at the top end) and
**system font stacks only** — `Segoe UI Variable Text` / `Segoe UI` /
`ui-sans-serif`, with `Cascadia Mono` / `Consolas` for the command bar. No
webfonts are ever downloaded.

All cards: `1px solid var(--line)`, `border-radius: 10px`, a theme-tinted drop
shadow plus a lit top hairline (`--edge-hi-*`) — the glass edge that keeps
cards distinct over busy photos without hiding them. Sidebar, terminal dock,
titlebar and inputs derive from the same panel/line tokens at higher alphas,
so every theme dresses the whole chrome, not just cards. The card grid is
`repeat(auto-fill, minmax(clamp(210px, 16vw, 280px), 1fr))`, which reflows
continuously instead of snapping at breakpoints.

Measured contrast, verified per palette with a script (body / secondary /
accent against the panel): darks 12.6–13.6 / 6.5–6.8 / 7.3–8.5, lights
12.6–15.5 / 4.9–5.5 / 5.2–6.0 (WCAG AAA body, AA or better throughout).
