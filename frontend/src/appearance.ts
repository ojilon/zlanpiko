// Appearance: theme-adjacent look controls — background image, card
// transparency, blur, dim and surface texture (docs/24).
//
// This module owns the *values*; styles.css owns what they mean. Everything is
// written to CSS custom properties on <html>, so a change re-tints the whole
// app without any view re-rendering.

import { bundledUrl, clearCustomBackground, getCustomBackground, putCustomBackground } from './backgrounds';

export type BackgroundKind = 'none' | 'bundled' | 'custom' | 'remote';

export interface AppearanceSettings {
  /** Master switch: false renders the flat theme with fully opaque surfaces. */
  backgroundEnabled: boolean;
  backgroundKind: BackgroundKind;
  /** Filename (bundled), URL (remote) or 'custom' (IndexedDB). */
  backgroundRef: string;
  /** 0 = solid panels, 100 = maximum see-through. */
  cardTransparency: number;
  /** Frosted-glass blur behind cards, px. 0 disables backdrop-filter. */
  cardBlur: number;
  /** Blur applied to the photo itself, px. */
  backgroundBlur: number;
  /** Scrim strength over the photo, %. Keeps text legible. */
  backgroundDim: number;
  /** Grain/texture amount, %. */
  texture: number;
}

export const DEFAULT_APPEARANCE: AppearanceSettings = {
  backgroundEnabled: false,
  backgroundKind: 'none',
  backgroundRef: '',
  cardTransparency: 18,
  cardBlur: 10,
  backgroundBlur: 0,
  backgroundDim: 34,
  texture: 12,
};

const KEY = 'zlanpiko.appearance';

let settings: AppearanceSettings = { ...DEFAULT_APPEARANCE };
/** Object URL for the IndexedDB blob; revoked when replaced. */
let customUrl: string | null = null;

function clamp(n: number, lo: number, hi: number): number {
  if (!Number.isFinite(n)) return lo;
  return Math.min(hi, Math.max(lo, n));
}

function num(value: unknown, fallback: number, lo: number, hi: number): number {
  return typeof value === 'number' && Number.isFinite(value) ? clamp(value, lo, hi) : fallback;
}

export function coerceAppearance(raw: unknown): AppearanceSettings {
  if (typeof raw !== 'object' || raw === null) return { ...DEFAULT_APPEARANCE };
  const r = raw as Record<string, unknown>;
  const kind = r.backgroundKind;
  const validKind: BackgroundKind =
    kind === 'bundled' || kind === 'custom' || kind === 'remote' ? kind : 'none';
  return {
    backgroundEnabled: r.backgroundEnabled === true,
    backgroundKind: validKind,
    backgroundRef: typeof r.backgroundRef === 'string' ? r.backgroundRef : '',
    cardTransparency: num(r.cardTransparency, DEFAULT_APPEARANCE.cardTransparency, 0, 85),
    cardBlur: num(r.cardBlur, DEFAULT_APPEARANCE.cardBlur, 0, 24),
    backgroundBlur: num(r.backgroundBlur, DEFAULT_APPEARANCE.backgroundBlur, 0, 30),
    backgroundDim: num(r.backgroundDim, DEFAULT_APPEARANCE.backgroundDim, 0, 85),
    texture: num(r.texture, DEFAULT_APPEARANCE.texture, 0, 60),
  };
}

export function getAppearance(): AppearanceSettings {
  return { ...settings };
}

export function saveAppearance(next: Partial<AppearanceSettings>): AppearanceSettings {
  settings = coerceAppearance({ ...settings, ...next });
  try {
    localStorage.setItem(KEY, JSON.stringify(settings));
  } catch {
    /* private mode: settings last for this session only */
  }
  applyAppearance();
  return getAppearance();
}

export function resetAppearance(): AppearanceSettings {
  return saveAppearance({ ...DEFAULT_APPEARANCE });
}

/* -------------------------------------------------------------------------
   CSS plumbing
   ------------------------------------------------------------------------- */

/**
 * Turn a reference into something safe to place inside `url("…")`.
 * Rejects anything that could break out of the string (quotes, backslashes,
 * newlines) and only permits image-ish schemes.
 */
export function cssImageUrl(raw: string): string | null {
  const value = raw.trim();
  if (!value) return null;
  if (/["'\\\n\r]/.test(value)) return null;
  if (/^(https?:|blob:|file:)/i.test(value)) return value;
  if (/^data:image\//i.test(value)) return value;
  if (/^backgrounds\//i.test(value)) return value;
  return null; // relative path inside the bundle
}

function currentImageUrl(s: AppearanceSettings): string | null {
  if (!s.backgroundEnabled) return null;
  switch (s.backgroundKind) {
    case 'bundled':
      return cssImageUrl(bundledUrl(s.backgroundRef));
    case 'custom':
      return customUrl ? cssImageUrl(customUrl) : null;
    case 'remote':
      return cssImageUrl(s.backgroundRef);
    case 'none':
      return null;
  }
}

/**
 * Panel opacity from the 0–100 "transparency" slider.
 * Clamped to 0.15 so text always has something to sit on.
 */
export function panelAlpha(cardTransparency: number): number {
  return clamp(1 - cardTransparency / 100, 0.15, 1);
}

/** Push the current settings into CSS custom properties. */
export function applyAppearance(): void {
  const s = settings;
  const root = document.documentElement;
  const url = currentImageUrl(s);
  const active = url !== null;

  // When the background is off the panels must be solid, otherwise the (now
  // invisible) translucency would just wash the theme out.
  const alpha = active ? panelAlpha(s.cardTransparency) : 1;

  root.style.setProperty('--card-alpha', String(alpha));
  root.style.setProperty('--card-blur-n', String(active ? s.cardBlur : 0));
  root.style.setProperty('--bg-blur-n', String(active ? s.backgroundBlur : 0));
  root.style.setProperty('--bg-dim', String(clamp(s.backgroundDim / 100, 0, 0.85)));
  root.style.setProperty('--grain', String(clamp(s.texture / 100, 0, 0.6)));
  root.style.setProperty('--bg-image', url ? `url("${url}")` : 'none');

  document.body.classList.toggle('glass', active && s.cardBlur > 0);
  document.body.dataset.background = active ? 'on' : 'off';
}

/* -------------------------------------------------------------------------
   Boot + custom image lifecycle
   ------------------------------------------------------------------------- */

/**
 * Load persisted appearance and restore any custom image from IndexedDB.
 * Called once during boot; safe to call again (it re-applies everything).
 */
export async function initAppearance(): Promise<AppearanceSettings> {
  try {
    const raw = localStorage.getItem(KEY);
    settings = coerceAppearance(raw ? (JSON.parse(raw) as unknown) : null);
  } catch {
    settings = { ...DEFAULT_APPEARANCE };
  }
  if (settings.backgroundKind === 'custom') {
    const blob = await getCustomBackground();
    if (blob) {
      if (customUrl) URL.revokeObjectURL(customUrl);
      customUrl = URL.createObjectURL(blob);
    } else {
      settings.backgroundKind = 'none'; // blob missing (profile cleared)
      settings.backgroundRef = '';
    }
  }
  applyAppearance();
  return getAppearance();
}

/** Store a user-picked file as the background and activate it. */
export async function useCustomBackground(file: File): Promise<AppearanceSettings> {
  await putCustomBackground(file);
  const blob = await getCustomBackground();
  if (customUrl) URL.revokeObjectURL(customUrl);
  customUrl = blob ? URL.createObjectURL(blob) : null;
  if (!customUrl) throw new Error('could not store that image');
  return saveAppearance({
    backgroundEnabled: true,
    backgroundKind: 'custom',
    backgroundRef: 'custom',
  });
}

/** Drop the stored custom image; falls back to no background. */
export async function forgetCustomBackground(): Promise<AppearanceSettings> {
  await clearCustomBackground();
  if (customUrl) {
    URL.revokeObjectURL(customUrl);
    customUrl = null;
  }
  if (settings.backgroundKind === 'custom') {
    return saveAppearance({ backgroundKind: 'none', backgroundRef: '', backgroundEnabled: false });
  }
  return getAppearance();
}
