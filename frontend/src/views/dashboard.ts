import type { DashboardDTO, TaskDTO, UnitCardDTO, WeekDTO } from '../types';

// Dashboard home view: header, fortnight rail, attention cards, unit cards,
// deadline-distance bars, overdue + upcoming (see docs/cs/05).
// Display-only: all dates, distances and percentages arrive pre-computed
// from Go. The single exception is labelling "today" on the rail via the
// client's clock (formatting only, never deadline arithmetic).

export function esc(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function pillCls(attention: string): string {
  switch (attention) {
    case 'Overdue':
    case 'At Risk':
      return 'pill bad';
    case 'Attention':
      return 'pill warn';
    case 'Completed':
      return 'pill ok';
    case 'On Track':
      return 'pill info';
    default:
      return 'pill';
  }
}

function dotCls(t: TaskDTO): string {
  if (t.status === 'completed' || t.status === 'submitted') return 'dot ok';
  if (t.overdue) return 'dot bad';
  if (t.days_left !== undefined && t.days_left <= 2) return 'dot warn';
  return 'dot info';
}

function barCls(t: TaskDTO): string {
  if (t.overdue) return 'bad';
  if (t.days_left !== undefined && t.days_left <= 2) return 'warn';
  return '';
}

function todayISO(): string {
  const d = new Date();
  const p = (n: number): string => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

function railCell(date: string, display: string, items: TaskDTO[]): string {
  const [wd, num] = display.split(' ');
  const dots = items
    .map(
      (t) =>
        `<span class="${dotCls(t)}" title="${esc(t.title)} · ${esc(t.unit_name)} · ${esc(t.due_display)} · ${esc(t.distance)}"></span>`,
    )
    .join('');
  const today = date === todayISO() ? ' today' : '';
  return `<div class="rail-cell${today}"><div class="rail-wd">${esc(wd ?? '')}</div><div class="rail-num">${esc(num ?? '')}</div><div class="rail-dots">${dots}</div></div>`;
}

function railHTML(weeks: WeekDTO[]): string {
  const cells = weeks
    .filter((w) => w && w.days)
    .flatMap((w) => w.days.map((d) => railCell(d.date, d.display, d.items)))
    .join('');
  return `<div class="rail">${cells}</div>`;
}

function unitCard(u: UnitCardDTO): string {
  const bar = u.has_topics
    ? `<div class="bar"><i style="width:${u.coverage.toFixed(1)}%"></i></div>`
    : `<div class="dim">no topics yet</div>`;
  return `<div class="card">
    <h3>${esc(u.name)}${u.code ? ` <span class="dim">[${esc(u.code)}]</span>` : ''}</h3>
    <div><span class="${pillCls(u.attention)}">${esc(u.attention)}</span>
    ${u.reason ? `<span class="dim"> ${esc(u.reason)}</span>` : ''}</div>
    <div class="dim">R${u.read}/P${u.pending}/U${u.unread} · ${esc(u.coverage_text)} · ${u.active_tasks} active</div>
    ${bar}
    ${u.next_due_display ? `<div class="dim">next due ${esc(u.next_due_display)}</div>` : ''}
  </div>`;
}

function taskBar(t: TaskDTO): string {
  const pct = t.progress_pct !== undefined ? Math.round(t.progress_pct) : null;
  const bar =
    pct !== null
      ? `<div class="dbar"><i class="${barCls(t)}" style="width:${pct}%"></i></div>`
      : `<div class="dim">${t.due_display ? 'due ' + esc(t.due_display) : 'no date — set one'}</div>`;
  return `<div class="trow">
    <div class="trow-head"><strong>${esc(t.title)}</strong>
      <span class="dim">${esc(t.unit_name)} · ${esc(t.due_display)} · ${esc(t.distance)}</span></div>
    ${bar}
  </div>`;
}

export function statusLine(s: DashboardDTO['summary']): string {
  return `${s.units} units · ${s.total_topics} topics (R${s.read}/P${s.pending}/U${s.unread} ${s.coverage_text}) · ${s.active_tasks} active · ${s.overdue_tasks} overdue`;
}

export function renderDashboard(el: HTMLElement, d: DashboardDTO, rail: WeekDTO[]): void {
  const hot = d.units.filter(
    (u) => u.attention === 'Overdue' || u.attention === 'At Risk' || u.attention === 'Attention',
  );
  el.innerHTML = `
    <h2>Academic overview <span class="dim">${d.summary.units} units · ${d.summary.total_topics} topics · ${d.summary.active_tasks} active tasks</span></h2>
    <div class="dim">Topics read ${d.summary.read} · pending ${d.summary.pending} · unread ${d.summary.unread} &nbsp; coverage ${esc(d.summary.coverage_text)}</div>
    ${railHTML(rail)}
    ${hot.length ? `<h3>Needs attention</h3><div class="cards">${hot.map(unitCard).join('')}</div>` : ''}
    <div class="cols">
      <div>
        <h3>Units</h3>
        <div class="cards">${d.units.map(unitCard).join('')}</div>
      </div>
      <div>
        <h3>Deadline distance</h3>
        ${d.upcoming.length ? d.upcoming.map(taskBar).join('') : '<div class="dim">no open deadlines in 14 days</div>'}
        <h3>Overdue ${d.overdue.length ? `(${d.overdue.length})` : ''}</h3>
        ${d.overdue.length ? d.overdue.map(taskBar).join('') : '<div class="dim">none</div>'}
      </div>
    </div>`;
}
