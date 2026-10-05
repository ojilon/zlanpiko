// Background image sources (docs/24).
//
// Three ways an image can reach the backdrop layer, all offline-safe:
//
//   1. bundled  — files dropped into frontend/public/backgrounds/ and listed in
//                 backgrounds/manifest.json. They are served by the Vite dev
//                 server and, after `pnpm build`, copied into dist-frontend/
//                 where Go embeds them (//go:embed all:dist-frontend).
//   2. custom   — a file the user picks once; stored as a Blob in IndexedDB and
//                 revived as an object URL. No base64 in localStorage (that
//                 would blow the ~5 MB quota on a single photo).
//   3. remote   — a URL the user pastes. The only option that needs a network.
//
// Nothing here downloads anything on its own: "download images to the server"
// is a build-time step documented in docs/24.

export interface BundledBackground {
  /** Filename inside frontend/public/backgrounds/. */
  file: string;
  /** Human label shown in the Settings gallery. */
  name: string;
}

interface ManifestFile {
  images?: unknown;
}

/** Where bundled backgrounds live, relative to index.html. */
export const BUNDLED_DIR = 'backgrounds';

const DB_NAME = 'zlanpiko';
const DB_VERSION = 1;
const STORE = 'backgrounds';
const CUSTOM_KEY = 'custom';

/* -------------------------------------------------------------------------
   Bundled gallery
   ------------------------------------------------------------------------- */

function asBundled(value: unknown): BundledBackground | null {
  if (typeof value !== 'object' || value === null) return null;
  const rec = value as Record<string, unknown>;
  if (typeof rec.file !== 'string' || !rec.file) return null;
  const name = typeof rec.name === 'string' && rec.name ? rec.name : rec.file;
  return { file: rec.file, name };
}

/**
 * Read backgrounds/manifest.json. A missing or malformed manifest is normal
 * (the folder ships empty), so it resolves to [] instead of throwing.
 */
export async function listBundledBackgrounds(): Promise<BundledBackground[]> {
  try {
    const res = await fetch(`${BUNDLED_DIR}/manifest.json`, { cache: 'no-store' });
    if (!res.ok) return [];
    const data = (await res.json()) as ManifestFile;
    if (!Array.isArray(data.images)) return [];
    return data.images
      .map(asBundled)
      .filter((b): b is BundledBackground => b !== null);
  } catch {
    return []; // file:// or preview without the folder: no bundled images
  }
}

/**
 * Build the URL for a bundled image.
 * Names come from manifest.json, but they are still untrusted input as far as
 * the renderer is concerned: anything with a separator is flattened to its
 * basename so a crafted entry cannot escape the backgrounds folder.
 */
export function bundledUrl(file: string): string {
  const base = file.split(/[\\/]/).pop() ?? '';
  return `${BUNDLED_DIR}/${base}`;
}

/* -------------------------------------------------------------------------
   Custom image (IndexedDB blob store)
   ------------------------------------------------------------------------- */

function openDb(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    if (typeof indexedDB === 'undefined') {
      reject(new Error('IndexedDB unavailable'));
      return;
    }
    const req = indexedDB.open(DB_NAME, DB_VERSION);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(STORE)) db.createObjectStore(STORE);
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error ?? new Error('cannot open IndexedDB'));
  });
}

function tx<T>(mode: IDBTransactionMode, run: (store: IDBObjectStore) => IDBRequest<T>): Promise<T> {
  return openDb().then(
    (db) =>
      new Promise<T>((resolve, reject) => {
        const t = db.transaction(STORE, mode);
        const req = run(t.objectStore(STORE));
        req.onsuccess = () => resolve(req.result);
        req.onerror = () => reject(req.error ?? new Error('IndexedDB request failed'));
        t.oncomplete = () => db.close();
      }),
  );
}

/** Persist a user-chosen image. Replaces any previous custom image. */
export async function putCustomBackground(blob: Blob): Promise<void> {
  await tx('readwrite', (store) => store.put(blob, CUSTOM_KEY) as IDBRequest<IDBValidKey>);
}

export async function getCustomBackground(): Promise<Blob | null> {
  try {
    const value = await tx<Blob | undefined>('readonly', (store) => store.get(CUSTOM_KEY));
    return value instanceof Blob ? value : null;
  } catch {
    return null;
  }
}

export async function clearCustomBackground(): Promise<void> {
  try {
    await tx('readwrite', (store) => store.delete(CUSTOM_KEY) as unknown as IDBRequest<undefined>);
  } catch {
    /* nothing stored, or IndexedDB blocked: nothing to clean up */
  }
}

/* -------------------------------------------------------------------------
   Helpers
   ------------------------------------------------------------------------- */

/** Resolve to true only if the browser can actually decode the image. */
export function probeImage(url: string, timeoutMs = 8000): Promise<boolean> {
  return new Promise((resolve) => {
    const img = new Image();
    let settled = false;
    const done = (ok: boolean): void => {
      if (settled) return;
      settled = true;
      window.clearTimeout(timer);
      resolve(ok);
    };
    const timer = window.setTimeout(() => done(false), timeoutMs);
    img.onload = () => done(true);
    img.onerror = () => done(false);
    img.src = url;
  });
}
