import type { TaskDTO, WeekDTO } from '../types';
import { esc } from './dashboard';
import { openTask } from '../components/drawer';

// Weekly timeline: overdue anchor lane + 7 day lanes (docs/cs/06).
// Navigation state (offset) lives in main.ts; this module only renders.

function chip(t: TaskDTO): string {
  const cls = t.status === 'completed' || t.status === 'submitted' ? 'chip done' : t.overdue ? 'chip bad' : 'chip';
  return `<button class="${cls}" data-unit="${esc(t.unit_id)}" data-task="${esc(t.id)}"
    title="${esc(t.title)} · ${esc(t.due_display)} · ${esc(t.distance)}">
    <span class="chip-time">${esc(t.due_display.slice(11) || '')}</span>
    <span class="chip-title">${esc(t.title)}</span>
    <span class="dim">${esc(t.unit_name)}</span>
  </button>`;
}

function wireChips(root: HTMLElement): void {
  for (const b of root.querySelectorAll<HTMLButtonElement>('button.chip')) {
    b.addEventListener('click', () => {
      void openTask(b.dataset.unit ?? '', b.dataset.task ?? '');
    });
  }
}

export function renderTimeline(
  el: HTMLElement,
  w: WeekDTO,
  offset: number,
  onNav: (next: number) => void,
): void {
  const lanes = w.days
    .map(
      (d) => `<div class="lane"><div class="lane-head">${esc(d.display)}</div>
        ${d.items.length ? d.items.map(chip).join('') : '<div class="dim">—</div>'}</div>`,
    )
    .join('');
  el.innerHTML = `
    <div class="view-head">
      <h2>${esc(w.title)}</h2>
      <div class="navbtns">
        <button data-nav="-1">◀ prev</button>
        <button data-nav="today">Today</button>
        <button data-nav="1">next ▶</button>
      </div>
    </div>
    ${w.overdue.length ? `<h3>Overdue (${w.overdue.length})</h3><div class="lane overdue">${w.overdue.map(chip).join('')}</div>` : '<div class="dim">Overdue: none, good.</div>'}
    <div class="lanes">${lanes}</div>
    <p class="dim">←/→ weeks · T today · Enter opens (keyboard arrives with Phase D command work).</p>`;
  for (const b of el.querySelectorAll<HTMLButtonElement>('[data-nav]')) {
    b.addEventListener('click', () => {
      const nav = b.dataset.nav ?? '';
      onNav(nav === 'today' ? 0 : offset + Number(nav));
    });
  }
  wireChips(el);
}
