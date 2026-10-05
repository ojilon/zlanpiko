// Theme registry — the catalogue of named looks (docs/cs/03 §5, docs/24).
//
// This file is the *only* place a theme is defined in TypeScript. To plug in
// a new theme:
//
//   1. Add one entry to THEMES below (id, name, kind, photo hint, look).
//   2. Add a matching `[data-theme="<id>"]` palette block in styles.css
//      (section 2) plus a `.theme-sw-<id>` swatch gradient (section 11).
//   3. Nothing else changes: theme.ts, appearance.ts and the Settings panel
//      all read from this registry.
//
// Palettes are photo-aware, not just dark/light. Each theme pairs surfaces,
// accents and scrim tint with the *kind* of photograph it sits over (nature,
// sea, sunshine, sunset …), following the standard photo-UI recipe: a tinted
// scrim compresses the photo's range so text keeps a predictable contrast
// floor, while translucent panels + blur + a lit hairline edge keep the card
// readable without hiding the picture. Contrast targets: body text ≥ 7:1
// against its panel, secondary text ≥ 4.5:1 (WCAG AA or better).

import type { AppearanceSettings } from './appearance';

export type ThemeKind = 'dark' | 'light';

/** Every concrete look the app can paint. 'system' is handled by theme.ts. */
export type ThemeId =
  | 'abyss'
  | 'forest'
  | 'tide'
  | 'ember'
  | 'dusk'
  | 'paper'
  | 'meadow'
  | 'shore'
  | 'sun'
  | 'bloom';

/**
 * Recommended photo look for a theme: slider values that make this theme's
 * panels blend with its photo family while keeping text legible. Applied when
 * the user picks the theme in Settings → Appearance; every slider stays
 * adjustable afterwards, and the values are clamped by coerceAppearance.
 */
export interface ThemeLook {
  cardTransparency: number;
  cardBlur: number;
  backgroundBlur: number;
  backgroundDim: number;
  texture: number;
}

export interface ThemePreset {
  id: ThemeId;
  /** Short display name shown in the gallery. */
  name: string;
  kind: ThemeKind;
  /** One-line character, e.g. "Deep slate · calm default". */
  blurb: string;
  /** Photo family this theme was tuned against. */
  photos: string;
  /** Suggested Appearance sliders for that family. */
  look: ThemeLook;
}

export const THEMES: readonly ThemePreset[] = [
  // -- Dark family -------------------------------------------------------
  {
    id: 'abyss',
    name: 'Abyss',
    kind: 'dark',
    blurb: 'Deep slate · the calm default',
    photos: 'Any photo',
    look: { cardTransparency: 18, cardBlur: 10, backgroundBlur: 0, backgroundDim: 34, texture: 12 },
  },
  {
    id: 'forest',
    name: 'Forest',
    kind: 'dark',
    blurb: 'Moss & fern · warm amber accents',
    photos: 'Green nature, forests, gardens',
    look: { cardTransparency: 22, cardBlur: 12, backgroundBlur: 2, backgroundDim: 38, texture: 12 },
  },
  {
    id: 'tide',
    name: 'Tide',
    kind: 'dark',
    blurb: 'Deep ocean · sea-glass cyan',
    photos: 'Sea, lakes, water, rain',
    look: { cardTransparency: 24, cardBlur: 12, backgroundBlur: 0, backgroundDim: 36, texture: 12 },
  },
  {
    id: 'ember',
    name: 'Ember',
    kind: 'dark',
    blurb: 'Charcoal ember · sunset glow',
    photos: 'Sunsets, dusk, warm city lights',
    look: { cardTransparency: 20, cardBlur: 12, backgroundBlur: 2, backgroundDim: 40, texture: 14 },
  },
  {
    id: 'dusk',
    name: 'Dusk',
    kind: 'dark',
    blurb: 'Indigo dusk · soft lavender',
    photos: 'Skies, skylines, night streets',
    look: { cardTransparency: 22, cardBlur: 14, backgroundBlur: 0, backgroundDim: 36, texture: 12 },
  },
  // -- Light family ------------------------------------------------------
  {
    id: 'paper',
    name: 'Paper',
    kind: 'light',
    blurb: 'Clean paper · the bright default',
    photos: 'Any photo',
    look: { cardTransparency: 14, cardBlur: 10, backgroundBlur: 0, backgroundDim: 30, texture: 10 },
  },
  {
    id: 'meadow',
    name: 'Meadow',
    kind: 'light',
    blurb: 'Warm paper · leaf green',
    photos: 'Nature, daylight, parks',
    look: { cardTransparency: 16, cardBlur: 10, backgroundBlur: 0, backgroundDim: 30, texture: 10 },
  },
  {
    id: 'shore',
    name: 'Shore',
    kind: 'light',
    blurb: 'Pale aqua · ocean teal',
    photos: 'Sea, sky, beaches, pools',
    look: { cardTransparency: 16, cardBlur: 10, backgroundBlur: 0, backgroundDim: 30, texture: 10 },
  },
  {
    id: 'sun',
    name: 'Sunshine',
    kind: 'light',
    blurb: 'Sunlit cream · goldenrod & clay',
    photos: 'Sunshine, sand, deserts, cafés',
    look: { cardTransparency: 14, cardBlur: 8, backgroundBlur: 0, backgroundDim: 28, texture: 10 },
  },
  {
    id: 'bloom',
    name: 'Bloom',
    kind: 'light',
    blurb: 'Blush paper · berry accents',
    photos: 'Sunsets, flowers, warm portraits',
    look: { cardTransparency: 14, cardBlur: 10, backgroundBlur: 0, backgroundDim: 30, texture: 10 },
  },
];

export const THEME_MAP: Readonly<Record<ThemeId, ThemePreset>> = Object.fromEntries(
  THEMES.map((t) => [t.id, t]),
) as Readonly<Record<ThemeId, ThemePreset>>;

/** Defaults used for first run, 'system' resolution and unknown values. */
export const DEFAULT_DARK_ID: ThemeId = 'abyss';
export const DEFAULT_LIGHT_ID: ThemeId = 'paper';

/**
 * Accept a stored value, mapping the pre-10-theme names ('dark'/'light')
 * onto their successors so existing users keep their look after updating.
 */
export function coerceThemeId(raw: unknown): ThemeId {
  if (typeof raw !== 'string') return DEFAULT_DARK_ID;
  if (raw === 'dark') return DEFAULT_DARK_ID;
  if (raw === 'light') return DEFAULT_LIGHT_ID;
  const found = THEMES.find((t) => t.id === raw);
  return found ? found.id : DEFAULT_DARK_ID;
}

export function themeKind(id: ThemeId): ThemeKind {
  return THEME_MAP[id].kind;
}

export function themePreset(id: ThemeId): ThemePreset {
  return THEME_MAP[id];
}

/** Suggested sliders for a theme, as a patch for saveAppearance(). */
export function suggestedLook(id: ThemeId): Partial<AppearanceSettings> {
  return { ...THEME_MAP[id].look };
}
