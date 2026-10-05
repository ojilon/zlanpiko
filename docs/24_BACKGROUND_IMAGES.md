# 24 — Background Images (GUI)

How to put a photograph behind the zlanpiko window, where the files live, how
they get into the executable, and how the transparency/blur controls keep text
readable on top of them.

Companion to `docs/cs/03` (frontend stack) and `docs/cs/04` (UI concept).
Scope: the **GUI only**. The CLI/TUI is unaffected.

---

## 1. The three ways to get an image

| Source | Where it lives | Needs a network? | Ships in the exe? | Best for |
|---|---|---|---|---|
| **Bundled** | `frontend/public/backgrounds/` | only to download it once | **yes** (embedded) | the default set everyone gets |
| **My image** | IndexedDB in the user's WebView profile | no (after picking) | no | a personal photo on one machine |
| **URL** | wherever you host it | **yes, every launch** | no | testing, or a LAN/intranet image |

Local-first rule: nothing in the app downloads an image by itself. Fetching
images onto disk is a **build-time step you run on purpose** (section 3).

---

## 2. What the Settings panel does

Settings → **Appearance**:

| Control | Range | Default | Effect |
|---|---|---|---|
| Theme | Dark / Light / System | Dark | palette; System follows the OS app-colour setting |
| Background image | Off / On | Off | master switch for the whole backdrop layer |
| Gallery | bundled images + *my image* | — | pick which image to show |
| Panel transparency | 0–85 % | 18 % | how much of the photo shows through cards |
| Card blur | 0–24 px | 10 px | frosted glass behind cards (0 disables it) |
| Background blur | 0–30 px | 0 px | softens the photo itself |
| Background dim | 0–85 % | 34 % | scrim over the photo — the main readability control |
| Texture | 0–60 % | 12 % | fine grain that stops flat colour from banding |

Everything is stored in `localStorage` (`zlanpiko.appearance`) except *my
image*, which is a `Blob` in IndexedDB — base64 in `localStorage` would blow
the ≈5 MB quota on a single photo.

### How readability is protected

The design keeps **text** solid and lets **gutters** show through, rather than
making everything uniformly translucent:

1. Gutters — the space between cards — have no surface at all, so the photo is
   fully visible there.
2. Cards, rails and rows paint `rgb(panel / alpha)`. The slider drives `alpha`;
   it is clamped to a **minimum of 0.15**, so there is always some plate behind
   text.
3. Inputs and the command bar derive a *higher* alpha than cards
   (`0.35 + 0.65 × card alpha`), so the things you type into stay legible
   first.
4. The chrome (titlebar, sidebar, status line) is likewise more opaque than
   cards (`0.4 + 0.6 × card alpha`).
5. `backdrop-filter` frost (card blur) plus the dim scrim do the rest.

Result: measured contrast on the default dark palette is 13.2:1 for body text
and 6.5:1 for secondary text (WCAG AAA / AA); the light palette is 15.5:1 and
5.5:1. If you push transparency high, raise **Background dim** to compensate.

---

## 3. Bundled images: downloading them

Bundled images live in **`frontend/public/backgrounds/`**. Vite serves that
folder at `/backgrounds/…` in dev and copies it to `dist-frontend/backgrounds/`
on build; `main.go` then embeds it:

```go
//go:embed all:dist-frontend
var frontendFS embed.FS
```

So a bundled image costs you **executable size** — keep them small (section 4).

### 3.1 Pick a licence you can actually ship

Only download images you are allowed to redistribute. Safe starting points:

- <https://unsplash.com> (Unsplash License — commercial use OK)
- <https://www.pexels.com> (Pexels License)
- <https://pixabay.com> (Pixabay Content License)
- Your own photos.

Avoid anything "editorial use only", and record the attribution in
`frontend/public/backgrounds/CREDITS.md` (not committed by default — see 3.4).

### 3.2 Download (pick your shell)

All commands run from the **repository root**.

**PowerShell** (Windows — the project's primary platform):

```powershell
$dir = "frontend\public\backgrounds"
New-Item -ItemType Directory -Force -Path $dir | Out-Null

Invoke-WebRequest -Uri "https://images.unsplash.com/photo-0000000000000?w=1920&q=70" `
  -OutFile "$dir\morning-mist.jpg"
Invoke-WebRequest -Uri "https://images.pexels.com/...?auto=compress&w=1920" `
  -OutFile "$dir\dunes.jpg"
```

**curl** (Git Bash, WSL, macOS, Linux):

```bash
mkdir -p frontend/public/backgrounds
curl -L -o frontend/public/backgrounds/morning-mist.jpg \
  "https://images.unsplash.com/photo-0000000000000?w=1920&q=70"
curl -L -o frontend/public/backgrounds/dunes.jpg \
  "https://images.pexels.com/...?auto=compress&w=1920"
```

**wget**:

```bash
wget -O frontend/public/backgrounds/aurora.jpg \
  "https://images.unsplash.com/photo-0000000000000?w=1920&q=70"
```

Batch-download a list:

```bash
# urls.txt holds one image URL per line
mkdir -p frontend/public/backgrounds
i=1
while read -r url; do
  [ -z "$url" ] && continue
  curl -sL -o "frontend/public/backgrounds/img-$(printf '%02d' $i).jpg" "$url"
  i=$((i+1))
done < urls.txt
```

### 3.3 Regenerate the manifest

The Settings gallery reads `backgrounds/manifest.json`, **not** the directory —
so new files appear only after you regenerate it:

```bash
node scripts/gen-background-manifest.mjs
```

```cmd
:: Windows
scripts\gen-background-manifest.bat
```

Output:

```text
backgrounds: 2 image(s) in frontend/public/backgrounds
  - deep_space.png  "Deep Space"
  - morning-mist.jpg  "Morning Mist"
```

The generator derives a label from the filename (`morning-mist.jpg` →
`Morning Mist`) and **preserves names you have edited by hand** for files that
are still present. Edit `manifest.json` to rename an image, then re-run — your
name survives.

Manifest shape:

```json
{
  "images": [
    { "file": "morning-mist.jpg", "name": "Morning Mist" }
  ]
}
```

Files outside `frontend/public/backgrounds/` are ignored; entries whose file no
longer exists are dropped.

### 3.4 What to commit

The repository ships the folder **empty**. `.gitignore` keeps the photos out and
the manifest in:

```gitignore
# Background images are user-supplied and can be large; the manifest is committed.
frontend/public/backgrounds/*
!frontend/public/backgrounds/manifest.json
!frontend/public/backgrounds/README.md
```

Rationale: images are megabytes of binary that Git handles badly, they are
usually personal, and their licences belong to whoever added them. Committing
`manifest.json` means a fresh clone still builds; the gallery is simply empty
until someone runs section 3.2.

To ship a **default set** in a release, commit the images explicitly:

```bash
git add -f frontend/public/backgrounds/morning-mist.jpg
```

### 3.5 Rebuild

Bundled images only reach the executable through a frontend build:

```cmd
cmd /c "cd frontend & pnpm install & pnpm build"
scripts\build-gui.bat 0.1.4
```

`pnpm dev` alone is enough while iterating — the Vite server serves
`frontend/public/` directly, no rebuild needed.

---

## 4. Recommended image specs

| Property | Recommendation | Why |
|---|---|---|
| Format | **JPEG** for photos, WebP if you prefer | PNG photos are 5–10× larger |
| Resolution | 1920×1200 is plenty | the window is at most ~2× that on a 4K display; `background-size: cover` scales down for free |
| File size | **150–400 KB** each | this is added directly to the exe |
| Total budget | ≤ 2 MB (≈ 4–6 images) | keeps the installer small on low-end PCs |
| Subject | low-contrast, soft gradients, few hard edges | busy photos need more dim, which defeats the point |
| Aspect | landscape | `cover` crops the long edge on portrait windows |

Resizing before you commit (ImageMagick):

```bash
magick input.jpg -resize 1920x1200^ -gravity center -extent 1920x1200 \
  -quality 78 frontend/public/backgrounds/aurora.jpg
```

---

## 5. Security and privacy notes

- Bundled filenames are **flattened to their basename** before use, so a
  hand-edited manifest cannot point outside `backgrounds/`.
- Remote URLs must be `http(s)://` and are rejected if they contain quotes,
  backslashes or newlines — they are interpolated into `url("…")`.
- A remote background **phones home on every launch**. On a metered or
  private network, use bundled or *my image* instead.
- *my image* is stored in the WebView's IndexedDB next to the rest of the
  profile; **Settings → remove custom** deletes it. Backups
  (`docs/26`) do not include it.
- `backdrop-filter` is only enabled while a background is showing and card
  blur > 0, so there is no compositing cost on the default flat theme.

---

## 6. Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| Gallery shows only "none" and "my image" | empty or missing manifest | run the generator (3.3) |
| New file does not appear | manifest not regenerated, or not an image extension | re-run generator; check `.jpg/.png/.webp/.avif/.gif/.bmp` |
| Image looks washed out | dim too low | raise **Background dim** |
| Text is hard to read | transparency too high | lower **Panel transparency** or raise **Card blur** |
| Remote image never loads | URL not a direct image link, or offline | the panel reports `could not load that image`; use a bundled file |
| Blur looks soft/laggy | `backdrop-filter` on a low-end GPU | set **Card blur** to 0 |
| Colour looks flat/banded | texture too low | raise **Texture** to ~15–25 % |

---

## 7. Adding your own bundled set — checklist

1. Download licenced images into `frontend/public/backgrounds/` (3.2).
2. Resize/compress them (section 4).
3. `node scripts/gen-background-manifest.mjs` (3.3).
4. Optionally hand-edit `name` fields in `manifest.json`, then re-run step 3.
5. `cd frontend && pnpm build`, then `scripts\build-gui.bat <version>` (3.5).
6. Verify in Settings → Appearance in both themes.
