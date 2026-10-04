import { getVersion, windowControl } from './api';

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

function renderView(view: View, version: string): void {
  const el = document.getElementById('view');
  if (!el) return;
  if (view === 'Dashboard') {
    el.innerHTML = `
      <h2>Academic overview</h2>
      <p>Wails shell is up. Full dashboard (cards, week rail,
      deadline-distance bars) lands in Phase B.</p>
      <div class="cards">
        <div class="card"><h3>Backend</h3>
          <div class="dim">${version}</div>
          <div class="bar"><i style="width:100%"></i></div>
        </div>
      </div>`;
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
  const version = await getVersion();
  const meta = document.getElementById('titlebar-meta');
  if (meta) meta.textContent = `gui shell · ${version}`;
  const status = document.getElementById('statusline');
  if (status) status.textContent = `backend: ${version}`;
  renderSidebar((v) => renderView(v, version));
  renderView('Dashboard', version);
}

void boot();
