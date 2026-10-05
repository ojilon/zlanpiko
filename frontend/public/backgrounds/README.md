# Background images

Drop image files in this folder, then regenerate the manifest:

```bash
node scripts/gen-background-manifest.mjs
```

The generator writes `manifest.json`, which the Settings → Appearance panel
reads to build its gallery. Files are served from `backgrounds/<file>` by the
Vite dev server, and after `pnpm build` they are copied into `dist-frontend/backgrounds/`,
which Go embeds into the executable.

Full instructions, including how to download images and how much they cost in
binary size, live in [`docs/24_BACKGROUND_IMAGES.md`](../../../docs/24_BACKGROUND_IMAGES.md).

This folder ships empty. `manifest.json` is committed; the images themselves are
not — see the note about `.gitignore` and licensing in the docs.
