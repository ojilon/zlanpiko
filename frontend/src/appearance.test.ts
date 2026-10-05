// Appearance: the pure value logic. DOM application stays out of scope here
// (no jsdom dependency); what matters is that persisted settings can never
// put the UI into an unreadable or unsafe state.

import { describe, expect, it } from 'vitest';
import {
  DEFAULT_APPEARANCE,
  coerceAppearance,
  cssImageUrl,
  panelAlpha,
} from './appearance';
import { bundledUrl } from './backgrounds';

describe('coerceAppearance', () => {
  it('falls back to defaults for junk input', () => {
    expect(coerceAppearance(null)).toEqual(DEFAULT_APPEARANCE);
    expect(coerceAppearance('nope')).toEqual(DEFAULT_APPEARANCE);
    expect(coerceAppearance(42)).toEqual(DEFAULT_APPEARANCE);
  });

  it('keeps valid values and repairs invalid ones', () => {
    const out = coerceAppearance({
      backgroundEnabled: true,
      backgroundKind: 'bundled',
      backgroundRef: 'forest.jpg',
      cardTransparency: 40,
      noise: 'ignored',
    });
    expect(out.backgroundEnabled).toBe(true);
    expect(out.backgroundKind).toBe('bundled');
    expect(out.backgroundRef).toBe('forest.jpg');
    expect(out.cardTransparency).toBe(40);
    // untouched fields keep the documented defaults
    expect(out.texture).toBe(DEFAULT_APPEARANCE.texture);
  });

  it('rejects an unknown background kind', () => {
    expect(coerceAppearance({ backgroundKind: 'evil' }).backgroundKind).toBe('none');
  });

  it('clamps every slider into its safe range', () => {
    const tooFar = coerceAppearance({
      cardTransparency: 9999,
      cardBlur: -50,
      backgroundBlur: 1e9,
      backgroundDim: -1,
      texture: 500,
    });
    expect(tooFar.cardTransparency).toBe(85);
    expect(tooFar.cardBlur).toBe(0);
    expect(tooFar.backgroundBlur).toBe(30);
    expect(tooFar.backgroundDim).toBe(0);
    expect(tooFar.texture).toBe(60);
  });

  it('ignores NaN and non-numbers', () => {
    const out = coerceAppearance({ cardTransparency: Number.NaN, texture: '40' });
    expect(out.cardTransparency).toBe(DEFAULT_APPEARANCE.cardTransparency);
    expect(out.texture).toBe(DEFAULT_APPEARANCE.texture);
  });
});

describe('panelAlpha', () => {
  it('is inverse to transparency', () => {
    expect(panelAlpha(0)).toBe(1);
    expect(panelAlpha(50)).toBeCloseTo(0.5);
  });

  it('never goes fully transparent, so text stays readable', () => {
    expect(panelAlpha(100)).toBeGreaterThanOrEqual(0.15);
    expect(panelAlpha(5000)).toBeGreaterThanOrEqual(0.15);
  });

  it('clamps negative transparency to opaque', () => {
    expect(panelAlpha(-20)).toBe(1);
  });
});

describe('cssImageUrl', () => {
  it('accepts bundled, http(s), blob and inline image data', () => {
    expect(cssImageUrl('backgrounds/forest.jpg')).toBe('backgrounds/forest.jpg');
    expect(cssImageUrl('https://example.com/a.jpg')).toBe('https://example.com/a.jpg');
    expect(cssImageUrl('http://example.com/a.jpg')).toBe('http://example.com/a.jpg');
    expect(cssImageUrl('blob:null/abc-123')).toBe('blob:null/abc-123');
    expect(cssImageUrl('data:image/png;base64,AAAA')).toBe('data:image/png;base64,AAAA');
  });

  it('rejects strings that could break out of url("…")', () => {
    expect(cssImageUrl('a.jpg" onload="alert(1)')).toBeNull();
    expect(cssImageUrl("a.jpg'")).toBeNull();
    expect(cssImageUrl('a\\b.jpg')).toBeNull();
    expect(cssImageUrl('a\n.jpg')).toBeNull();
  });

  it('rejects empty and unknown schemes', () => {
    expect(cssImageUrl('')).toBeNull();
    expect(cssImageUrl('   ')).toBeNull();
    expect(cssImageUrl('javascript:alert(1)')).toBeNull();
    expect(cssImageUrl('data:text/html,<script>')).toBeNull();
    expect(cssImageUrl('some/relative/path.jpg')).toBeNull();
  });
});

describe('bundledUrl', () => {
  it('keeps plain filenames', () => {
    expect(bundledUrl('forest.jpg')).toBe('backgrounds/forest.jpg');
  });

  it('flattens traversal attempts to a basename', () => {
    expect(bundledUrl('../../../etc/passwd')).toBe('backgrounds/passwd');
    expect(bundledUrl('sub\\dir\\a.jpg')).toBe('backgrounds/a.jpg');
  });
});
