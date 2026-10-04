import { getDashboard, getMonthOffset, getVersion, getWeekOffset, windowControl } from './api';
import type { WeekDTO } from './types';
import { renderDashboard, statusLine } from './views/dashboard';
import { renderTimeline } from './views/timeline';
import { renderCalendar } from './views/calendar';
import { wireDrawerKeys } from './components/drawer';

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

function renderSidebar(onNav: (v: View) => void): void {
  const bar = document.getElementById('sidebar');
  if (!bar) return;
  bar.innerHTML = '';
  for (const name of NAV) {
    const b = document.createElement('button');
    b.innerHTML = `<span class="lbl">${name}</span>`;
    b.title = name;
    b.addEventListener('click', () => {
      for (const el of bar.querySelectorAll('button')) el.classList.remove('active');
      b.classList.add('active');
      onNav(name);
    });
    if (name === 'Dashboard') b.classList.add('active');
    bar.appendChild(b);
  }
}

let weekOff = 0;
let monthOff = 0;

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
      void renderView('Timeline', version);
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
      void renderView('Calendar', version);
    });
    return;
  }
  el.innerHTML = `<h2>${view}</h2><p>Coming in Phase B–F (see docs/cs/11).</p>`;
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

function wireCommandBar(): void {
  const input = document.getElementById('cmdinput') as HTMLInputElement | null;
  const msg = document.getElementById('cmdmsg');
  if (!input || !msg) return;
  const run = (): void => {
    const v = input.value.trim();
    if (!v) return;
    msg.className = '';
    msg.textContent = `Phase D wires commands — you typed: ${v}`;
  };
  document.getElementById('cmdrun')?.addEventListener('click', run);
  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') run();
    if (e.key === 'Escape') {
      input.value = '';
      msg.textContent = '';
    }
  });
  window.addEventListener('keydown', (e) => {
    if (e.key === '/' && document.activeElement !== input) {
      e.preventDefault();
      input.focus();
    }
  });
}

async function boot(): Promise<void> {
  wireTitlebar();
  wireCommandBar();
  wireDrawerKeys();
  const version = await getVersion();
  const meta = document.getElementById('titlebar-meta');
  if (meta) meta.textContent = `gui shell · ${version}`;
  const status = document.getElementById('statusline');
  if (status) status.textContent = `backend: ${version}`;
  renderSidebar((v) => {
    void renderView(v, version);
  });
  await renderView('Dashboard', version);
}

void boot();
