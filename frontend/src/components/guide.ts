import { esc } from '../views/dashboard';

// In-app user guide (Settings → 📖 User guide). Static content for a friend
// receiving the packaged app: no backend needed, works offline.
// NOTE: the command list here mirrors the backend help in
// internal/gui/commands.go (guiCommandHelp) plus the TUI grammar in
// internal/tui/commands.go — update all three together.

interface Section {
  id: string;
  title: string;
  body: string;
}

const SECTIONS: Section[] = [
  {
    id: 'welcome',
    title: 'Welcome',
    body: `<p><strong>zlanpiko</strong> is a local-first academic manager: your course units,
      topics, assignments, deadlines and files live on <em>your</em> PC — no account, no
      internet needed. The database and folders sit in one data directory you choose at install.</p>
      <p>Three ways to work: this window (daily driver), the terminal UI
      (<code>zlanpiko</code> with no arguments), and scriptable commands
      (<code>zlanpiko units list</code>). All three share the same data.</p>`,
  },
  {
    id: 'around',
    title: 'Getting around',
    body: `<p>The left <strong>sidebar</strong> switches views: Dashboard, Units, Topics, Tasks,
      Timeline, Calendar, Files, Analytics, Settings. The top bar shows the current week and
      app version; window buttons are <code>— ▢ ✕</code> (double-click the bar to maximise).</p>
      <p>Above the command bar, the <strong>status line</strong> always reads like
      <code>11 units · 50 topics (R0/P0/U50 0%) · 1 active · 0 overdue</code>:
      R/P/U are read / pending / unread topics.</p>`,
  },
  {
    id: 'dashboard',
    title: 'Dashboard',
    body: `<p><strong>Academic overview</strong> rolls everything up: units, topics by reading
      status, and overall coverage (<code>read ÷ total × 100</code>). An empty unit shows
      <code>—</code>, never 0% or 100%.</p>
      <p>The <strong>thin rail</strong> is a 3-week strip of days; dots are deadlines
      (amber = soon, red = overdue, green = done). Hover a dot for details.</p>
      <p><strong>Deadline-distance bars</strong> show created → due with a fill for elapsed
      time: blue = comfortable, amber = within ~2 days, red = overdue. Completed work just
      says <code>done</code>.</p>
      <p><strong>Needs attention</strong> cards flag <code>At Risk / Attention / Overdue</code>
      units with the reason (e.g. <code>deadline 2026-10-06 at 20%</code>). Coverage measures
      reading, not mastery.</p>`,
  },
  {
    id: 'units',
    title: 'Units, topics, tasks',
    body: `<p><strong>Units</strong>: click a card for its workspace. Rename freely (folders use
      stable IDs, so nothing breaks). Prefer <strong>archive</strong> over delete to keep
      history — delete needs two confirmations and can be refused when children exist.</p>
      <p><strong>Topics</strong>: pick a unit, filter by status, cycle
      <code>unread → pending → read</code> with one click. Nothing changes status except you.</p>
      <p><strong>Tasks</strong> (assignments, coursework, tests, exams, projects): filter by
      unit, status (incl. derived <code>overdue</code>) and kind. Click any task for its drawer:
      move the deadline, change status, delete. Deadlines accept
      <code>YYYY-MM-DD</code>, <code>YYYY-MM-DDTHH:MM</code> or
      <code>YYYY-MM-DD HH:MM</code> (e.g. <code>2026-10-06 07:00</code>).</p>`,
  },
  {
    id: 'time',
    title: 'Timeline & calendar',
    body: `<p><strong>Timeline</strong> is week-oriented (Monday–Sunday): prev / today / next,
      or jump with <code>/timeline 2026-W41</code>. Overdue items pin to a red anchor lane so
      they never hide in a past day; click any chip for its drawer.</p>
      <p><strong>Calendar</strong> shows the month grid: dot colours are worst-of-day
      (red = overdue-anchored, amber = due within 48h). Click a numbered day for its agenda,
      then a task for its drawer. Edit deadlines right there — every view recomputes.</p>`,
  },
  {
    id: 'files',
    title: 'Files & import',
    body: `<p><strong>Files</strong> browses your real academic folders: navigate, search,
      open (with your normal apps), new folder, rename, delete (no trash — it asks first).</p>
      <p><strong>Import</strong> never happens silently: enter a source path, pick a
      destination (empty = inbox), press <em>preview import</em>, review the copy/skip plan
      with collision notes, then <em>confirm</em>. Name clashes become
      <code>name-2.ext</code>, never overwrites.</p>`,
  },
  {
    id: 'analytics',
    title: 'Analytics',
    body: `<p>Five charts, all computed from your live records — each card states its formula:</p>
      <ul><li>coverage ring + read/pending/unread stack (<code>read ÷ total × 100</code>)</li>
      <li>tasks by status (<code>overdue</code> is derived, never stored)</li>
      <li>deadlines per day, next 14 days</li>
      <li>coverage per unit, worst first</li>
      <li>attention table, most urgent first</li></ul>
      <p>Labels <code>On Track / Attention / At Risk / Overdue / Completed</code> describe
      workload risk from deadlines and reading state — never predictions about grades.</p>`,
  },
  {
    id: 'commands',
    title: 'Command bar',
    body: `<p>Press <code>/</code> or <code>Ctrl+K</code> anywhere, type, <code>Enter</code> to run.
      <code>↑↓</code> history (kept), <code>Tab</code> completes, <code>Esc</code> clears.</p>
      <pre>/help                              this list
/dashboard /units /topics [unit] /tasks [unit]
/timeline [next|prev|today|YYYY-Www]  /calendar
/files /analytics /settings /verify /export /backup /version
/units list                        unit roster
/topics status &lt;unit&gt; &lt;topic&gt; &lt;unread|pending|read&gt;
/tasks deadline &lt;unit&gt; &lt;task&gt; &lt;YYYY-MM-DD[ HH:MM]&gt;
/tasks status &lt;unit&gt; &lt;task&gt; &lt;not_started|in_progress|completed|submitted&gt;
/tasks list · /tasks deadlines</pre>
      <p>Every important action also has a clickable twin — commands are a shortcut, never a
      requirement. Unknown commands suggest the closest match.</p>`,
  },
  {
    id: 'safety',
    title: 'Reports, backups & safety',
    body: `<p><strong>Status report</strong> (Settings): one plain-text summary for copying to
      your phone — units, attention topics, upcoming and overdue deadlines.</p>
      <p><strong>Backups</strong>: one click writes a timestamped zip (tick the box to include
      academic files). <em>Verify</em> re-checks every hash. <em>Restore</em> asks twice and,
      for overwrites, takes a safety backup first.</p>
      <p><strong>Storage</strong>: changing the data directory saves the pointer and asks for
      a restart — the running app keeps its open database, so nothing half-migrates.
      Updates never wipe data: the installer backs up first and keeps previous executables as
      <code>*.prev.exe</code>.</p>`,
  },
];

function ensureOverlay(): HTMLElement {
  let ov = document.getElementById('guide-overlay');
  if (!ov) {
    ov = document.createElement('div');
    ov.id = 'guide-overlay';
    ov.hidden = true;
    document.body.appendChild(ov);
  }
  return ov;
}

export function closeGuide(): void {
  const ov = document.getElementById('guide-overlay');
  if (ov) ov.hidden = true;
}

export function openGuide(section?: string): void {
  const ov = ensureOverlay();
  const nav = SECTIONS.map((s) => `<button data-sec="${s.id}">${esc(s.title)}</button>`).join('');
  ov.innerHTML = `<div class="guide-box">
    <div class="guide-head"><h3>📖 User guide</h3><button id="guide-close" title="Close">✕</button></div>
    <div class="guide-split"><nav>${nav}</nav><article id="guide-body"></article></div>
  </div>`;
  ov.hidden = false;
  const show = (id: string): void => {
    const s = SECTIONS.find((x) => x.id === id) ?? SECTIONS[0];
    if (!s) return;
    const body = ov.querySelector('#guide-body');
    if (body) body.innerHTML = `<h4>${esc(s.title)}</h4>${s.body}`;
    for (const b of ov.querySelectorAll<HTMLButtonElement>('[data-sec]')) {
      b.classList.toggle('active', b.dataset.sec === s.id);
    }
  };
  for (const b of ov.querySelectorAll<HTMLButtonElement>('[data-sec]')) {
    b.addEventListener('click', () => show(b.dataset.sec ?? ''));
  }
  ov.querySelector('#guide-close')?.addEventListener('click', closeGuide);
  ov.addEventListener('mousedown', (e) => {
    if (e.target === ov) closeGuide();
  });
  show(section ?? 'welcome');
}

export function wireGuideKeys(): void {
  window.addEventListener('keydown', (e) => {
    if (document.activeElement?.id === 'cmdinput') return; // input owns Esc
    if (e.key === 'Escape') closeGuide();
    if (e.key === 'F1') {
      e.preventDefault();
      openGuide();
    }
  });
}
