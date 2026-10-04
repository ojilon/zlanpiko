import { getTaskDetail } from '../api';
import { esc } from '../views/dashboard';
import type { TaskDetailDTO } from '../types';

// Shared task drawer (docs/cs/06). Read-only in Phase C: details, deadline,
// status, unit link. Editing controls arrive in Phase D.

let current: { unit: string; task: string } | null = null;

function detailHTML(d: TaskDetailDTO): string {
  const t = d.task;
  return `
    <div class="drawer-head">
      <h3>${esc(t.title)}</h3>
      <button id="drawer-close" title="Close">✕</button>
    </div>
    <div class="dim">${esc(t.unit_name)} · ${esc(t.kind)} · ${esc(t.status)}${t.overdue ? ' · <span class="pill bad">overdue</span>' : ''}</div>
    <div class="drawer-row"><span class="dim">Due</span><span>${t.due_display ? esc(t.due_display) + ' · ' + esc(t.distance) : 'no date'}</span></div>
    <div class="drawer-row"><span class="dim">Priority</span><span>${esc(t.priority)}</span></div>
    <div class="drawer-row"><span class="dim">Created</span><span>${esc(d.created_display)}</span></div>
    ${d.completed_display ? `<div class="drawer-row"><span class="dim">Finished</span><span>${esc(d.completed_display)}</span></div>` : ''}
    ${d.description ? `<h4>Description</h4><p>${esc(d.description)}</p>` : ''}
    ${d.notes ? `<h4>Notes</h4><p>${esc(d.notes)}</p>` : ''}
    <p class="dim">Editing (deadline, status) arrives in Phase D.</p>`;
}

function ensureEl(): HTMLElement {
  let el = document.getElementById('drawer');
  if (!el) {
    el = document.createElement('aside');
    el.id = 'drawer';
    el.hidden = true;
    document.getElementById('shell')?.appendChild(el);
  }
  return el;
}

export function closeDrawer(): void {
  current = null;
  const el = document.getElementById('drawer');
  if (el) el.hidden = true;
}

export async function openTask(unitID: string, taskID: string): Promise<void> {
  current = { unit: unitID, task: taskID };
  const el = ensureEl();
  el.hidden = false;
  el.innerHTML = `<div class="dim">loading…</div>`;
  const d = await getTaskDetail(unitID, taskID);
  if (!current || current.unit !== unitID || current.task !== taskID) return; // superseded
  if (!d) {
    el.innerHTML = `<div class="drawer-head"><h3>Task</h3><button id="drawer-close">✕</button></div><p class="dim">No backend in browser preview.</p>`;
  } else {
    el.innerHTML = detailHTML(d);
  }
  document.getElementById('drawer-close')?.addEventListener('click', closeDrawer);
}

export function wireDrawerKeys(): void {
  window.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && current) closeDrawer();
  });
}
