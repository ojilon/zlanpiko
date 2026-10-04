import { getDashboard, getMonthOffset, getVersion, getWeekOffset, windowControl } from './api';
import type { WeekDTO } from './types';
import { renderDashboard, statusLine } from './views/dashboard';
import { renderTimeline } from './views/timeline';
import { renderCalendar } from './views/calendar';
import { renderFiles } from './views/files';
import { renderSettings } from './views/settings';
import { renderAnalytics } from './views/analytics';
import { REFRESH_EVENT, refreshOpenDrawer, wireDrawerKeys } from './components/drawer';
import { wireCommandBar } from './components/commandbar';

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

function isView(s: string): s is View {
  return (NAV as readonly string[]).includes(s);
}

let current: View = 'Dashboard';
let appVersion = '';
let weekOff = 0;
let monthOff = 0;

function markActive(view: View): void {
  const bar = document.getElementById('sidebar');
  if (!bar) return;
  const btns = [...bar.querySelectorAll('button')];
  btns.forEach((b, i) => b.classList.toggle('active', NAV[i] === view));
}

async function renderCurrent(): Promise<void> {
  await renderView(current, appVersion);
  await refreshOpenDrawer();
}

async function navigate(view: string, opts?: { weekOffset?: number; unit?: string }): Promise<void> {
  if (!isView(view)) return;
  if (opts?.weekOffset !== undefined) weekOff = opts.weekOffset;
  if (opts?.unit) {
    const msg = document.getElementById('cmdmsg');
    if (msg) {
      msg.className = '';
      msg.textContent = `filtered to ${opts.unit} (per-unit views land in Phase E)`;
    }
  }
  current = view;
  markActive(view);
  await renderView(current, appVersion);
}

function renderSidebar(): void {
  const bar = document.getElementById('sidebar');
  if (!bar) return;
  bar.innerHTML = '';
  for (const name of NAV) {
    const b = document.createElement('button');
    b.innerHTML = `<span class="lbl">${name}</span>`;
    b.title = name;
    b.addEventListener('click', () => {
      void navigate(name);
    });
    if (name === current) b.classList.add('active');
    bar.appendChild(b);
  }
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

async function boot(): Promise<void> {
  wireTitlebar();
  wireDrawerKeys();
  appVersion = await getVersion();
  const meta = document.getElementById('titlebar-meta');
  if (meta) meta.textContent = `gui shell · ${appVersion}`;
  const status = document.getElementById('statusline');
  if (status) status.textContent = `backend: ${appVersion}`;
  renderSidebar();
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
