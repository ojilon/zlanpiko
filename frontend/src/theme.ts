// Theme manager — dark / light / follow-system.
//
// The palette itself lives in styles.css under [data-theme="..."]; this module
// only decides which one is active and remembers the choice. It never loads a
// webfont or a remote stylesheet: the app is offline-first (docs/cs/03 §5).

export type ThemeName = 'dark' | 'light' | 'system';

const THEME_KEY = 'zlanpiko.theme';

let current: ThemeName = 'dark';
let mediaQuery: MediaQueryList | null = null;

function lightPreferred(): boolean {
  return mediaQuery?.matches ?? false;
}

/** Resolve 'system' to the concrete theme the OS is currently asking for. */
export function resolvedTheme(theme: ThemeName = current): 'dark' | 'light' {
  if (theme === 'system') return lightPreferred() ? 'light' : 'dark';
  return theme;
}

function paint(theme: ThemeName): void {
  document.documentElement.dataset.theme = resolvedTheme(theme);
}

function readStored(): ThemeName {
  try {
    const raw = localStorage.getItem(THEME_KEY);
    if (raw === 'dark' || raw === 'light' || raw === 'system') return raw;
  } catch {
    /* private mode: fall back to the default */
  }
  return 'dark';
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
 * Safe to call once at boot; index.html ships data-theme="dark" so there is
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

export function cycleTheme(): ThemeName {
  const order: ThemeName[] = ['dark', 'light', 'system'];
  const next = order[(order.indexOf(current) + 1) % order.length] ?? 'dark';
  return setTheme(next);
}
