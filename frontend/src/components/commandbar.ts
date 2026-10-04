import { runCommand } from '../api';

// Terminal-bordered command bar (docs/cs/08): history, prefix suggestions,
// inline results. Navigation/refresh are delegated to main.ts hooks so this
// module owns no view state.

export interface CmdHooks {
  navigate: (view: string, opts?: { weekOffset?: number; unit?: string }) => void;
  refresh: () => void;
}

const COMMANDS = [
  '/help',
  '/dashboard',
  '/units',
  '/units list',
  '/topics',
  '/topics status ',
  '/tasks',
  '/tasks list',
  '/tasks deadlines',
  '/tasks deadline ',
  '/tasks status ',
  '/timeline',
  '/timeline next',
  '/timeline previous',
  '/calendar',
  '/files',
  '/analytics',
  '/export',
  '/backup',
  '/settings',
  '/verify',
  '/version',
];

const HIST_KEY = 'zlanpiko.cmdhistory';

function loadHist(): string[] {
  try {
    const raw = localStorage.getItem(HIST_KEY);
    const arr = raw ? (JSON.parse(raw) as unknown) : [];
    return Array.isArray(arr) ? arr.filter((s): s is string => typeof s === 'string') : [];
  } catch {
    return [];
  }
}

export function wireCommandBar(hooks: CmdHooks): void {
  const input = document.getElementById('cmdinput') as HTMLInputElement | null;
  const msg = document.getElementById('cmdmsg');
  if (!input || !msg) return;
  let hist = loadHist();
  let hpos = hist.length;

  const say = (text: string, cls: '' | 'ok' | 'err'): void => {
    msg.className = cls;
    msg.textContent = text;
  };

  const complete = (): void => {
    const v = input.value;
    if (!v.startsWith('/')) return;
    const hits = COMMANDS.filter((c) => c.startsWith(v) && c !== v);
    if (hits.length === 1 && hits[0]) {
      input.value = hits[0];
    } else if (hits.length > 1) {
      say(hits.slice(0, 8).join('   '), '');
    }
  };

  const run = async (): Promise<void> => {
    const v = input.value.trim();
    if (!v) return;
    hist = [...hist, v].slice(-200);
    hpos = hist.length;
    try {
      localStorage.setItem(HIST_KEY, JSON.stringify(hist));
    } catch {
      /* private mode: history stays in memory */
    }
    const r = await runCommand(v);
    if (!r) {
      say('browser preview: start the Wails window for commands', 'err');
      return;
    }
    switch (r.kind) {
      case 'navigate':
        if (r.view) hooks.navigate(r.view, { weekOffset: r.week_offset, unit: r.unit });
        if (r.message) say(r.message, '');
        else say('', '');
        break;
      case 'refresh':
        hooks.refresh();
        say(r.message || 'done', 'ok');
        break;
      case 'message':
        if (r.view) hooks.navigate(r.view);
        say(r.message || '', '');
        break;
      case 'error':
        say(r.hint ? `${r.message} ${r.hint}` : (r.message ?? 'error'), 'err');
        break;
    }
  };

  document.getElementById('cmdrun')?.addEventListener('click', () => {
    void run();
  });
  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      void run();
    } else if (e.key === 'Escape') {
      input.value = '';
      say('', '');
    } else if (e.key === 'Tab') {
      e.preventDefault();
      complete();
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (hpos > 0) {
        hpos -= 1;
        input.value = hist[hpos] ?? '';
      }
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (hpos < hist.length) {
        hpos += 1;
        input.value = hist[hpos] ?? '';
      }
    }
  });
  window.addEventListener('keydown', (e) => {
    if (e.key === '/' && document.activeElement !== input) {
      e.preventDefault();
      input.focus();
    } else if (e.key === 'k' && e.ctrlKey) {
      e.preventDefault();
      input.focus();
    }
  });
}
