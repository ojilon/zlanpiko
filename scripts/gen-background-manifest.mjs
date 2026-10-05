#!/usr/bin/env node
// Regenerate frontend/public/backgrounds/manifest.json from the files that are
// actually present in that folder.
//
// Usage:
//   node scripts/gen-background-manifest.mjs
//
// Run it after adding, renaming or deleting background images — the Settings
// gallery reads the manifest, not the directory (see docs/24).

import { readdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const DIR = join(ROOT, 'frontend', 'public', 'backgrounds');
const MANIFEST = join(DIR, 'manifest.json');

const IMAGE_EXT = new Set(['.jpg', '.jpeg', '.png', '.webp', '.avif', '.gif', '.bmp']);

/** "morning-mist.jpg" -> "Morning Mist" */
function label(file) {
  return file
    .replace(/\.[^.]+$/, '')
    .replace(/[-_]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

const entries = await readdir(DIR, { withFileTypes: true });
const images = entries
  .filter((e) => e.isFile() && IMAGE_EXT.has(e.name.slice(e.name.lastIndexOf('.')).toLowerCase()))
  .filter((e) => e.name !== 'manifest.json' && e.name !== 'README.md')
  .map((e) => ({ file: e.name, name: label(e.name) }))
  .sort((a, b) => a.name.localeCompare(b.name));

// Preserve hand-written names for images that are still on disk.
try {
  const previous = JSON.parse(await readFile(MANIFEST, 'utf8'));
  const known = new Map(
    Array.isArray(previous?.images)
      ? previous.images.filter((i) => i && typeof i.file === 'string').map((i) => [i.file, i.name])
      : [],
  );
  for (const img of images) {
    const custom = known.get(img.file);
    if (typeof custom === 'string' && custom) img.name = custom;
  }
} catch {
  // No previous manifest, or it was malformed: generated names are fine.
}

await writeFile(MANIFEST, `${JSON.stringify({ images }, null, 2)}\n`, 'utf8');

console.log(`backgrounds: ${images.length} image(s) in frontend/public/backgrounds`);
for (const img of images) console.log(`  - ${img.file}  "${img.name}"`);
