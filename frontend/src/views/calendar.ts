import { getDayItems } from '../api';
import type { MonthDTO } from '../types';
import { esc } from './dashboard';
import { openTask } from '../components/drawer';

// Month calendar grid + day agenda (docs/cs/06). Month arithmetic stays in
// Go (GetMonthOffset); day agendas come from GetDayItems so no date logic
// lives here.

const WD = ['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su'];

function lvlCls(level: string): string {
  switch (level) {
    case 'bad':
      return 'mcell bad';
    case 'warn':
      return 'mcell warn';
    case 'info':
      return 'mcell info';
    case 'ok':
      return 'mcell ok';
    default:
      return 'mcell';
  }
}

export function renderCalendar(
  el: HTMLElement,
  m: MonthDTO,
  offset: number,
  onNav: (next: number) => void,
): void {
  const cells = m.days
    .map(
      (d) => `<button class="${lvlCls(d.level)}${d.in_month ? '' : ' filler'}" data-date="${d.date}"
          ${d.count ? '' : 'disabled'} title="${d.date} · ${d.count} item(s)">
        <span class="mnum">${esc(d.display)}</span>
        ${d.count ? `<span class="mdot">${d.count}</span>` : ''}
      </button>`,
    )
    .join('');
  el.innerHTML = `
    <div class="view-head">
      <h2>${esc(m.title)}</h2>
      <div class="navbtns">
        <button data-nav="-1">◀ prev</button>
        <button data-nav="today">Today</button>
        <button data-nav="1">next ▶</button>
      </div>
    </div>
    ${m.overdue_count ? `<div><span class="pill bad">overdue ${m.overdue_count}</span></div>` : ''}
    <div class="mgrid-head">${WD.map((d) => `<span>${d}</span>`).join('')}</div>
    <div class="mgrid">${cells}</div>
    <div id="agenda"><div class="dim">Select a day with items to see its agenda.</div></div>`;
  for (const b of el.querySelectorAll<HTMLButtonElement>('[data-nav]')) {
    b.addEventListener('click', () => {
      const nav = b.dataset.nav ?? '';
      onNav(nav === 'today' ? 0 : offset + Number(nav));
    });
  }
  for (const b of el.querySelectorAll<HTMLButtonElement>('.mcell[data-date]:not([disabled])')) {
    b.addEventListener('click', () => {
      void showAgenda(b.dataset.date ?? '');
    });
  }
}

async function showAgenda(dateISO: string): Promise<void> {
  const box = document.getElementById('agenda');
  if (!box) return;
  box.innerHTML = `<div class="dim">loading ${esc(dateISO)}…</div>`;
  const d = await getDayItems(dateISO);
  if (!d) {
    box.innerHTML = `<div class="dim">No backend in browser preview.</div>`;
    return;
  }
  const rows = [...d.overdue, ...d.items]
    .map(
      (t) => `<button class="chip${t.overdue ? ' bad' : ''}" data-unit="${esc(t.unit_id)}" data-task="${esc(t.id)}">
        <span class="chip-title">${esc(t.title)}</span>
        <span class="dim">${esc(t.unit_name)} · ${esc(t.due_display)} · ${esc(t.distance)}</span></button>`,
    )
    .join('');
  box.innerHTML = `<h3>${esc(d.display)}</h3>${rows || '<div class="dim">nothing that day</div>'}`;
  for (const b of box.querySelectorAll<HTMLButtonElement>('button.chip')) {
    b.addEventListener('click', () => {
      void openTask(b.dataset.unit ?? '', b.dataset.task ?? '');
    });
  }
}
