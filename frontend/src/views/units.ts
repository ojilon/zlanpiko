import {
  createUnit,
  deleteUnit,
  getUnitCards,
  getUnitDetail,
  renameUnit,
  setUnitArchived,
} from '../api';
import { esc } from './dashboard';
import { openTask } from '../components/drawer';

// Units view: card grid plus a detail workspace (topics, tasks, folder-less
// management actions). Archiving preserves history; deletion needs two
// confirmations and may be refused when children exist.

export async function renderUnits(el: HTMLElement): Promise<void> {
  el.innerHTML = `<div class="view-head"><h2>Units</h2></div>
    <div class="ftools"><input id="units-name" type="text" placeholder="new unit name…" />
    <input id="units-code" type="text" placeholder="code (optional)" />
    <button id="units-add">add unit</button></div>
    <div id="units-msg" class="dim"></div>
    <div id="units-grid" class="cards"></div>
    <div id="units-detail"></div>`;

  const say = (t: string): void => {
    const m = el.querySelector('#units-msg');
    if (m) m.textContent = t;
  };

  const showDetail = async (unitID: string): Promise<void> => {
    const box = el.querySelector('#units-detail');
    if (!box) return;
    try {
      const d = await getUnitDetail(unitID);
      const c = d.card;
      box.innerHTML = `<div class="card"><h3>${esc(c.name)}${c.code ? ` [${esc(c.code)}]` : ''}</h3>
        <div class="dim">R${c.read}/P${c.pending}/U${c.unread} · ${esc(c.coverage_text)} · ${c.active_tasks} active</div>
        <div class="navbtns">
          <button id="u-rename">rename</button>
          <button id="u-arch">${c.attention === '—' && !c.has_topics ? 'archive' : 'archive/unarchive'}</button>
          <button id="u-del">delete</button>
        </div>
        <h4>Topics (${d.topics.length})</h4>
        ${d.topics.map((t) => `<div class="frow"><span>${esc(t.name)}</span><span class="dim">${esc(t.status)} · ${esc(t.priority)}</span></div>`).join('') || '<div class="dim">no topics</div>'}
        <h4>Tasks (${d.tasks.length})</h4>
        ${d.tasks.map((t) => `<button class="chip" data-unit="${esc(t.unit_id)}" data-task="${esc(t.id)}"><span class="chip-title">${esc(t.title)}</span><span class="dim">${esc(t.status)} · ${esc(t.due_display)} · ${esc(t.distance)}</span></button>`).join('') || '<div class="dim">no tasks</div>'}
      </div>`;
      for (const b of box.querySelectorAll<HTMLButtonElement>('button.chip')) {
        b.addEventListener('click', () => {
          void openTask(b.dataset.unit ?? '', b.dataset.task ?? '');
        });
      }
      box.querySelector('#u-rename')?.addEventListener('click', () => {
        const name = window.prompt('New name:', c.name);
        if (!name || name === c.name) return;
        renameUnit(unitID, name)
          .then(() => renderUnits(el))
          .catch((e: unknown) => say(e instanceof Error ? e.message : String(e)));
      });
      box.querySelector('#u-arch')?.addEventListener('click', () => {
        setUnitArchived(unitID, true)
          .then(() => renderUnits(el))
          .catch((e: unknown) => say(e instanceof Error ? e.message : String(e)));
      });
      box.querySelector('#u-del')?.addEventListener('click', () => {
        if (!window.confirm(`Delete ${c.name}? Prefer archiving to keep history.`)) return;
        if (!window.confirm('Final confirmation: delete this unit?')) return;
        deleteUnit(unitID, true)
          .then(() => renderUnits(el))
          .catch((e: unknown) => say(e instanceof Error ? e.message : String(e)));
      });
    } catch (e: unknown) {
      say(e instanceof Error ? e.message : String(e));
    }
  };

  try {
    const cards = await getUnitCards();
    const grid = el.querySelector('#units-grid');
    if (grid) {
      grid.innerHTML = cards
        .map(
          (u) => `<button class="card ucard" data-unit="${esc(u.unit_id)}">
            <h3>${esc(u.name)}${u.code ? ` <span class="dim">[${esc(u.code)}]</span>` : ''}</h3>
            <div class="dim">R${u.read}/P${u.pending}/U${u.unread} · ${esc(u.coverage_text)} · ${u.active_tasks} active</div>
            ${u.has_topics ? `<div class="bar"><i style="width:${u.coverage.toFixed(1)}%"></i></div>` : ''}
          </button>`,
        )
        .join('');
      for (const b of grid.querySelectorAll<HTMLButtonElement>('.ucard')) {
        b.addEventListener('click', () => {
          void showDetail(b.dataset.unit ?? '');
        });
      }
    }
  } catch (e: unknown) {
    say(e instanceof Error ? e.message : String(e));
    return;
  }

  el.querySelector('#units-add')?.addEventListener('click', () => {
    const name = (el.querySelector('#units-name') as HTMLInputElement | null)?.value.trim() ?? '';
    const code = (el.querySelector('#units-code') as HTMLInputElement | null)?.value.trim() ?? '';
    if (!name) {
      say('name is required');
      return;
    }
    createUnit(name, code)
      .then(() => renderUnits(el))
      .catch((e: unknown) => say(e instanceof Error ? e.message : String(e)));
  });
}
