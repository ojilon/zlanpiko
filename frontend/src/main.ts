import { getDashboard, getMonthOffset, getVersion, getWeekOffset, windowControl } from './api';
import type { WeekDTO } from './types';
import { renderDashboard, statusLine } from './views/dashboard';
import { renderTimeline } from './views/timeline';
import { renderCalendar } from './views/calendar';
import { renderFiles } from './views/files';
import { renderSettings } from './views/settings';
import { renderAnalytics } from './views/analytics';
import { renderUnits } from './views/units';
import { renderTopics } from './views/topics';
import { renderTasks } from './views/tasks';
import { REFRESH_EVENT, refreshOpenDrawer, wireDrawerKeys } from './components/drawer';
import { wireGuideKeys } from './components/guide';
import { wireCommandBar } from './components/commandbar';
import { initTheme } from './theme';
import { initAppearance } from './appearance';

const NAV = [
  'Dashboard',
  'Units',
  'Topics',
  'Tasks',
  'Timeline',
  'Calendar',
  'Files',
  'Analytics',
  'Settings',
] as const;

type View = (typeof NAV)[number];

/** Stroke-only 24×24 glyphs, inlined so nothing is fetched at runtime. */
const ICONS: Record<View, string> = {
  Dashboard: '<path d="M3 3h7v7H3zM14 3h7v7h-7zM14 14h7v7h-7zM3 14h7v7H3z"/>',
  Units: '<path d="M4 5a2 2 0 0 1 2-2h11v16H6a2 2 0 0 0-2 2z"/><path d="M4 19a2 2 0 0 1 2-2h11"/>',
  Topics: '<path d="M8 6h13M8 12h13M8 18h13"/><path d="M3 6h.01M3 12h.01M3 18h.01"/>',
  Tasks: '<path d="M9 11l3 3 8-8"/><path d="M20 12v7a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h9"/>',
  Timeline: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
  Calendar: '<rect x="3" y="5" width="18" height="16" rx="2"/><path d="M16 3v4M8 3v4M3 11h18"/>',
  Files: '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>',
  Analytics: '<path d="M4 20V12M10 20V4M16 20v-7"/><path d="M2 20h20"/>',
  Settings: '<path d="M4 7h16M4 12h16M4 17h16"/><circle cx="9" cy="7" r="2"/><circle cx="15" cy="12" r="2"/><circle cx="8" cy="17" r="2"/>',
};

const CHEVRON_LEFT = '<path d="M15 6l-6 6 6 6"/>';
const CHEVRON_RIGHT = '<path d="M9 6l6 6-6 6"/>';

function svg(paths: string): string {
  return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${paths}</svg>`;
}

function isView(s: string): s is View {
  return (NAV as readonly string[]).includes(s);
}

let current: View = 'Dashboard';
let appVersion = '';
let weekOff = 0;
let monthOff = 0;
let unitPreset = '';

/* -------------------------------------------------------------------------
   Sidebar: collapsible (auto / open / collapsed — persisted)
   ------------------------------------------------------------------------- */

const SIDEBAR_KEY = 'zlanpiko.sidebar';
type SidebarState = 'auto' | 'open' | 'collapsed';
let sidebarState: SidebarState = 'auto';

function readSidebarState(): SidebarState {
  try {
    const raw = localStorage.getItem(SIDEBAR_KEY);
    if (raw === 'auto' || raw === 'open' || raw === 'collapsed') return raw;
  } catch {
    /* private mode: stay on auto */
  }
  return 'auto';
}

function applySidebar(): void {
  const bar = document.getElementById('sidebar');
  if (!bar) return;
  bar.classList.toggle('auto', sidebarState === 'auto');
  bar.classList.toggle('collapsed', sidebarState === 'collapsed');
  const btn = document.getElementById('sidebar-toggle');
  if (!btn) return;
  const collapsed = sidebarState === 'collapsed';
  btn.innerHTML = svg(collapsed ? CHEVRON_RIGHT : CHEVRON_LEFT);
  btn.title = collapsed ? 'Expand sidebar (Ctrl+B)' : 'Collapse sidebar (Ctrl+B)';
  btn.setAttribute('aria-label', btn.title);
  btn.setAttribute('aria-expanded', String(!collapsed));
}

function toggleSidebar(): void {
  // Collapse if currently wide; expand if currently narrow. 'auto' is treated
  // as wide at desktop widths, so the first click always collapses.
  const bar = document.getElementById('sidebar');
  const wide = bar ? bar.getBoundingClientRect().width > 100 : true;
  sidebarState = wide ? 'collapsed' : 'open';
  try {
    localStorage.setItem(SIDEBAR_KEY, sidebarState);
  } catch {
    /* private mode: preference lasts for this session only */
  }
  applySidebar();
}

function markActive(view: View): void {
  const nav = document.getElementById('sidebar-nav');
  if (!nav) return;
  const btns = [...nav.querySelectorAll('button')];
  btns.forEach((b, i) => b.classList.toggle('active', NAV[i] === view));
}

function renderSidebar(): void {
  const nav = document.getElementById('sidebar-nav');
  if (!nav) return;
  nav.innerHTML = '';
  for (const name of NAV) {
    const b = document.createElement('button');
    b.className = 'nav';
    b.innerHTML = `<span class="nav-icon">${svg(ICONS[name])}</span><span class="nav-label">${name}</span>`;
    b.title = name;
    b.addEventListener('click', () => {
      void navigate(name);
    });
    if (name === current) b.classList.add('active');
    nav.appendChild(b);
  }
}

/* -------------------------------------------------------------------------
   Routing
   ------------------------------------------------------------------------- */

async function renderCurrent(): Promise<void> {
  await renderView(current, appVersion);
  await refreshOpenDrawer();
}

async function navigate(view: string, opts?: { weekOffset?: number; unit?: string }): Promise<void> {
  if (!isView(view)) return;
  if (opts?.weekOffset !== undefined) weekOff = opts.weekOffset;
  unitPreset = opts?.unit ?? '';
  current = view;
  markActive(view);
  await renderView(current, appVersion);
}

async function renderView(view: View, version: string): Promise<void> {
  const el = document.getElementById('view');
  if (!el) return;
  if (view === 'Dashboard') {
    const d = await getDashboard();
    if (!d) {
      el.innerHTML = `<h2>Academic overview</h2>
        <p>Browser preview: start the Wails window for live data
        (see docs/cs/10). Dashboard mock lands here.</p>`;
      return;
    }
    const rail: WeekDTO[] = [];
    for (const off of [-1, 0, 1]) {
      const w = await getWeekOffset(off);
      if (w) rail.push(w);
    }
    renderDashboard(el, d, rail);
    const status = document.getElementById('statusline');
    if (status) status.textContent = `Status: ${statusLine(d.summary)}`;
    const meta = document.getElementById('titlebar-meta');
    if (meta) meta.textContent = `${d.week.title} · ${version}`;
    return;
  }
  if (view === 'Timeline') {
    const w = await getWeekOffset(weekOff);
    if (!w) {
      el.innerHTML = `<h2>Timeline</h2><p class="dim">No backend in browser preview.</p>`;
      return;
    }
    renderTimeline(el, w, weekOff, (next) => {
      weekOff = next;
      void renderCurrent();
    });
    return;
  }
  if (view === 'Calendar') {
    const m = await getMonthOffset(monthOff);
    if (!m) {
      el.innerHTML = `<h2>Calendar</h2><p class="dim">No backend in browser preview.</p>`;
      return;
    }
    renderCalendar(el, m, monthOff, (next) => {
      monthOff = next;
      void renderCurrent();
    });
    return;
  }
  if (view === 'Files') {
    await renderFiles(el);
    return;
  }
  if (view === 'Units') {
    await renderUnits(el);
    return;
  }
  if (view === 'Topics') {
    await renderTopics(el, unitPreset);
    return;
  }
  if (view === 'Tasks') {
    await renderTasks(el, unitPreset);
    return;
  }
  if (view === 'Analytics') {
    await renderAnalytics(el);
    return;
  }
  if (view === 'Settings') {
    await renderSettings(el);
    return;
  }
  el.innerHTML = `<h2>${view}</h2><p>Coming in Phase E–F (see docs/cs/11).</p>`;
}

/* -------------------------------------------------------------------------
   Chrome
   ------------------------------------------------------------------------- */

function wireTitlebar(): void {
  for (const btn of document.querySelectorAll<HTMLButtonElement>('#titlebar [data-win]')) {
    const action = btn.getAttribute('data-win') as 'min' | 'max' | 'close';
    btn.addEventListener('click', () => windowControl(action));
  }
  document.getElementById('titlebar')?.addEventListener('dblclick', (e) => {
    if ((e.target as HTMLElement).closest('button')) return;
    windowControl('max');
  });
}

function wireSidebarKeys(): void {
  document.getElementById('sidebar-toggle')?.addEventListener('click', toggleSidebar);
  window.addEventListener('keydown', (e) => {
    if (e.key === 'b' && e.ctrlKey) {
      e.preventDefault();
      toggleSidebar();
    }
  });
}

async function boot(): Promise<void> {
  // Paint the stored theme and look before the first data fetch so the window
  // never shows an unstyled frame.
  initTheme();
  await initAppearance();

  wireTitlebar();
  wireSidebarKeys();
  wireDrawerKeys();
  wireGuideKeys();

  sidebarState = readSidebarState();
  renderSidebar();
  applySidebar();

  appVersion = await getVersion();
  const meta = document.getElementById('titlebar-meta');
  if (meta) meta.textContent = `gui shell · ${appVersion}`;
  const status = document.getElementById('statusline');
  if (status) status.textContent = `backend: ${appVersion}`;

  wireCommandBar({
    navigate: (v, opts) => {
      void navigate(v, opts);
    },
    refresh: () => {
      void renderCurrent();
    },
  });
  window.addEventListener(REFRESH_EVENT, () => {
    void renderCurrent();
  });
  await renderCurrent();
}

void boot();
