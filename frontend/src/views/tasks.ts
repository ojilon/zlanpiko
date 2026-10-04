import { createTask, deleteTask, getUnitCards, listTasksFlat } from '../api';
import { esc } from './dashboard';
import { openTask } from '../components/drawer';

// Tasks view: consolidated assignment/coursework/test/exam list with unit,
// status and kind filters, inline create, drawer on open.

export async function renderTasks(el: HTMLElement, presetUnit?: string): Promise<void> {
  let units: { id: string; name: string }[] = [];
  try {
    units = (await getUnitCards()).map((u) => ({ id: u.unit_id, name: u.name }));
  } catch (e: unknown) {
    el.innerHTML = `<h2>Tasks</h2><p class="dim">${esc(e instanceof Error ? e.message : String(e))}</p>`;
    return;
  }
  let fUnit = presetUnit && units.some((u) => u.id === presetUnit) ? presetUnit : '';
  let fStatus = '';
  let fKind = '';

  el.innerHTML = `<div class="view-head"><h2>Tasks</h2></div>
    <div class="ftools">
      <select id="tasks-unit"><option value="">all units</option>${units.map((u) => `<option value="${esc(u.id)}"${u.id === fUnit ? ' selected' : ''}>${esc(u.name)}</option>`).join('')}</select>
      <select id="tasks-status"><option value="">any status</option><option value="not_started">not started</option><option value="in_progress">in progress</option><option value="completed">completed</option><option value="submitted">submitted</option><option value="overdue">overdue</option></select>
      <select id="tasks-kind"><option value="">any kind</option><option value="assignment">assignment</option><option value="coursework">coursework</option><option value="test">test</option><option value="examination">examination</option><option value="project">project</option><option value="other">other</option></select>
    </div>
    <div class="ftools">
      <select id="tasks-new-unit">${units.map((u) => `<option value="${esc(u.id)}">${esc(u.name)}</option>`).join('')}</select>
      <input id="tasks-title" type="text" placeholder="new task title…" />
      <select id="tasks-new-kind"><option value="assignment">assignment</option><option value="coursework">coursework</option><option value="test">test</option><option value="examination">examination</option><option value="project">project</option><option value="other">other</option></select>
      <input id="tasks-due" type="text" placeholder="due YYYY-MM-DD" />
      <button id="tasks-add">add</button>
    </div>
    <div id="tasks-msg" class="dim"></div>
    <div id="tasks-list"></div>`;

  const say = (t: string): void => {
    const m = el.querySelector('#tasks-msg');
    if (m) m.textContent = t;
  };

  const list = async (): Promise<void> => {
    const box = el.querySelector('#tasks-list');
    if (!box) return;
    try {
      const rows = await listTasksFlat({ unit_id: fUnit, status: fStatus, kind: fKind });
      box.innerHTML =
        rows
          .map(
            (t) => `<div class="frow"><button class="fname" data-unit="${esc(t.unit_id)}" data-task="${esc(t.id)}">${esc(t.title)}</button>
              <span class="dim">${esc(t.unit_name)} · ${esc(t.kind)} · ${esc(t.status)} · ${esc(t.due_display || 'no date')} · ${esc(t.distance)}</span>
              <span class="fops"><button data-del-unit="${esc(t.unit_id)}" data-del="${esc(t.id)}">delete</button></span></div>`,
          )
          .join('') || '<div class="dim">no tasks match</div>';
      for (const b of box.querySelectorAll<HTMLButtonElement>('.fname')) {
        b.addEventListener('click', () => {
          void openTask(b.dataset.unit ?? '', b.dataset.task ?? '');
        });
      }
      for (const b of box.querySelectorAll<HTMLButtonElement>('[data-del]')) {
        b.addEventListener('click', () => {
          if (!window.confirm('Delete this task?')) return;
          deleteTask(b.dataset.delUnit ?? '', b.dataset.del ?? '', true)
            .then(() => list())
            .catch((e: unknown) => say(e instanceof Error ? e.message : String(e)));
        });
      }
    } catch (e: unknown) {
      say(e instanceof Error ? e.message : String(e));
    }
  };

  const refilter = (id: string, set: (v: string) => void): void => {
    (el.querySelector(id) as HTMLSelectElement | null)?.addEventListener('change', (e) => {
      set((e.target as HTMLSelectElement).value);
      void list();
    });
  };
  refilter('#tasks-unit', (v) => {
    fUnit = v;
  });
  refilter('#tasks-status', (v) => {
    fStatus = v;
  });
  refilter('#tasks-kind', (v) => {
    fKind = v;
  });
  el.querySelector('#tasks-add')?.addEventListener('click', () => {
    const unit = (el.querySelector('#tasks-new-unit') as HTMLSelectElement | null)?.value ?? '';
    const title = (el.querySelector('#tasks-title') as HTMLInputElement | null)?.value.trim() ?? '';
    const kind = (el.querySelector('#tasks-new-kind') as HTMLSelectElement | null)?.value ?? 'assignment';
    const due = (el.querySelector('#tasks-due') as HTMLInputElement | null)?.value.trim() ?? '';
    if (!unit || !title) {
      say('unit and title are required');
      return;
    }
    createTask(unit, title, kind, due, 'normal')
      .then(() => list())
      .catch((e: unknown) => say(e instanceof Error ? e.message : String(e)));
  });

  await list();
}
