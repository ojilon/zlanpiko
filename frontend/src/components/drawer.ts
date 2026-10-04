import { getTaskDetail, moveDeadline, setTaskStatus } from '../api';
import { esc } from '../views/dashboard';
import type { TaskDetailDTO } from '../types';

// Shared task drawer (docs/cs/06): details, deadline editor, status
// buttons. Mutations dispatch a window event so main.ts refreshes the
// current view without this module knowing view state.

export const REFRESH_EVENT = 'zlanpiko:refresh';

let current: { unit: string; task: string } | null = null;

function notify(): void {
  window.dispatchEvent(new CustomEvent(REFRESH_EVENT));
}

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
    <h4>Edit</h4>
    <div class="drawer-row"><span class="dim">Deadline</span></div>
    <div class="drawer-edit">
      <input id="drawer-due" type="text" value="${esc(t.due_display)}" placeholder="YYYY-MM-DD[THH:MM]" />
      <button id="drawer-move">move</button>
    </div>
    <div class="drawer-row"><span class="dim">Status</span></div>
    <div class="drawer-edit">
      <select id="drawer-status">
        ${['not_started', 'in_progress', 'completed', 'submitted']
          .map((s) => `<option value="${s}"${s === t.status ? ' selected' : ''}>${s}</option>`)
          .join('')}
      </select>
      <button id="drawer-set">set</button>
    </div>
    <div id="drawer-msg" class="dim"></div>`;
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
  wireEdits();
}

function wireEdits(): void {
  const msg = document.getElementById('drawer-msg');
  const say = (t: string): void => {
    if (msg) msg.textContent = t;
  };
  document.getElementById('drawer-move')?.addEventListener('click', () => {
    if (!current) return;
    const due = (document.getElementById('drawer-due') as HTMLInputElement | null)?.value.trim() ?? '';
    moveDeadline(current.unit, current.task, due)
      .then(() => {
        notify();
        void openTask(current?.unit ?? '', current?.task ?? '');
      })
      .catch((e: unknown) => say(e instanceof Error ? e.message : String(e)));
  });
  document.getElementById('drawer-set')?.addEventListener('click', () => {
    if (!current) return;
    const st = (document.getElementById('drawer-status') as HTMLSelectElement | null)?.value ?? '';
    setTaskStatus(current.unit, current.task, st)
      .then(() => {
        notify();
        void openTask(current?.unit ?? '', current?.task ?? '');
      })
      .catch((e: unknown) => say(e instanceof Error ? e.message : String(e)));
  });
}

export async function refreshOpenDrawer(): Promise<void> {
  if (!current) return;
  await openTask(current.unit, current.task);
}

export function wireDrawerKeys(): void {
  window.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && current) closeDrawer();
  });
}
