// Theme manager — which registered look is active, and persistence.
//
// The catalogue lives in themes.ts (add a preset + a CSS palette block to
// plug in a new theme); this module only decides which one is painted and
// remembers the choice. It never loads a webfont or a remote stylesheet:
// the app is offline-first (docs/cs/03 §5).
//
// Stored values from before the ten-theme set ('dark'/'light') are mapped
// onto their successors (abyss/paper) by coerceThemeId, so updating never
// resets a user's look.

import {
  DEFAULT_DARK_ID,
  DEFAULT_LIGHT_ID,
  THEMES,
  coerceThemeId,
  type ThemeId,
} from './themes';

/** A concrete look, or 'system' to follow the OS app-colour setting. */
export type ThemeName = ThemeId | 'system';

const THEME_KEY = 'zlanpiko.theme';

let current: ThemeName = DEFAULT_DARK_ID;
let mediaQuery: MediaQueryList | null = null;

function lightPreferred(): boolean {
  return mediaQuery?.matches ?? false;
}

/** Resolve 'system' to the concrete theme the OS is currently asking for. */
export function resolvedTheme(theme: ThemeName = current): ThemeId {
  if (theme === 'system') return lightPreferred() ? DEFAULT_LIGHT_ID : DEFAULT_DARK_ID;
  return theme;
}

function paint(theme: ThemeName): void {
  document.documentElement.dataset.theme = resolvedTheme(theme);
}

function readStored(): ThemeName {
  try {
    const raw = localStorage.getItem(THEME_KEY);
    if (raw === 'system') return 'system';
    if (typeof raw === 'string') return coerceThemeId(raw);
  } catch {
    /* private mode: fall back to the default */
  }
  return DEFAULT_DARK_ID;
}

export function getTheme(): ThemeName {
  return current;
}

export function setTheme(theme: ThemeName): ThemeName {
  current = theme;
  paint(theme);
  try {
    localStorage.setItem(THEME_KEY, theme);
  } catch {
    /* private mode: choice lasts for this session only */
  }
  return theme;
}

/**
 * Apply the stored theme and keep 'system' reactive to OS changes.
 * Safe to call once at boot; index.html ships data-theme="abyss" so there is
 * no flash of an unstyled (transparent) window before this runs.
 */
export function initTheme(): ThemeName {
  if (typeof window.matchMedia === 'function') {
    mediaQuery = window.matchMedia('(prefers-color-scheme: light)');
    // addEventListener is missing on very old WebViews; addListener is the
    // fallback and is still present in WebView2.
    if (typeof mediaQuery.addEventListener === 'function') {
      mediaQuery.addEventListener('change', () => {
        if (current === 'system') paint('system');
      });
    }
  }
  current = readStored();
  paint(current);
  return current;
}

/** Step through every registered theme, then 'system'. */
export function cycleTheme(): ThemeName {
  const order: ThemeName[] = [...THEMES.map((t) => t.id), 'system'];
  const next = order[(order.indexOf(current) + 1) % order.length] ?? DEFAULT_DARK_ID;
  return setTheme(next);
}
