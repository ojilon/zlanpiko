// Theme registry: every preset is complete, unique and safe to apply.

import { describe, expect, it } from 'vitest';
import {
  DEFAULT_DARK_ID,
  DEFAULT_LIGHT_ID,
  THEMES,
  coerceThemeId,
  suggestedLook,
  themeKind,
} from './themes';

describe('theme registry', () => {
  it('ships ten themes, five dark and five light', () => {
    expect(THEMES).toHaveLength(10);
    expect(THEMES.filter((t) => t.kind === 'dark')).toHaveLength(5);
    expect(THEMES.filter((t) => t.kind === 'light')).toHaveLength(5);
  });

  it('has unique ids, names, blurbs and photo hints', () => {
    const ids = new Set(THEMES.map((t) => t.id));
    expect(ids.size).toBe(THEMES.length);
    for (const t of THEMES) {
      expect(t.name.trim()).not.toBe('');
      expect(t.blurb.trim()).not.toBe('');
      expect(t.photos.trim()).not.toBe('');
    }
  });

  it('maps legacy dark/light names onto their successors', () => {
    expect(coerceThemeId('dark')).toBe(DEFAULT_DARK_ID);
    expect(coerceThemeId('light')).toBe(DEFAULT_LIGHT_ID);
    expect(coerceThemeId('forest')).toBe('forest');
  });

  it('falls back to the dark default for junk input', () => {
    expect(coerceThemeId(null)).toBe(DEFAULT_DARK_ID);
    expect(coerceThemeId('')).toBe(DEFAULT_DARK_ID);
    expect(coerceThemeId('midnight-express')).toBe(DEFAULT_DARK_ID);
    expect(coerceThemeId(42)).toBe(DEFAULT_DARK_ID);
  });

  it('reports each theme kind consistently', () => {
    for (const t of THEMES) expect(themeKind(t.id)).toBe(t.kind);
  });

  it('suggests slider values inside the safe appearance ranges', () => {
    for (const t of THEMES) {
      const look = suggestedLook(t.id);
      expect(look.cardTransparency).toBeGreaterThanOrEqual(0);
      expect(look.cardTransparency).toBeLessThanOrEqual(85);
      expect(look.cardBlur).toBeGreaterThanOrEqual(0);
      expect(look.cardBlur).toBeLessThanOrEqual(24);
      expect(look.backgroundBlur).toBeGreaterThanOrEqual(0);
      expect(look.backgroundBlur).toBeLessThanOrEqual(30);
      expect(look.backgroundDim).toBeGreaterThanOrEqual(0);
      expect(look.backgroundDim).toBeLessThanOrEqual(85);
      expect(look.texture).toBeGreaterThanOrEqual(0);
      expect(look.texture).toBeLessThanOrEqual(60);
    }
  });
});
