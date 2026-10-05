import { createTopic, deleteTopic, getUnitCards, listTopics, setTopicStatus } from '../api';
import { esc } from './dashboard';

// Topics view: unit picker, status filter, inline status cycling, create
// and delete. Status changes are explicit user actions only (docs/04).

const NEXT: Record<string, string> = { unread: 'pending', pending: 'read', read: 'unread' };

export async function renderTopics(el: HTMLElement, presetUnit?: string): Promise<void> {
  let units: { id: string; name: string }[] = [];
  try {
    units = (await getUnitCards()).map((u) => ({ id: u.unit_id, name: u.name }));
  } catch (e: unknown) {
    el.innerHTML = `<h2>Topics</h2><p class="dim">${esc(e instanceof Error ? e.message : String(e))}</p>`;
    return;
  }
  if (!units.length) {
    el.innerHTML = `<h2>Topics</h2><p class="dim">No units yet — create one in Units first.</p>`;
    return;
  }
  let unitID = (presetUnit && units.some((u) => u.id === presetUnit) ? presetUnit : units[0]?.id) ?? '';
  let filter = '';

  el.innerHTML = `<div class="view-head"><h2>Topics</h2></div>
    <div class="ftools">
      <select id="topics-unit">${units.map((u) => `<option value="${esc(u.id)}"${u.id === unitID ? ' selected' : ''}>${esc(u.name)}</option>`).join('')}</select>
      <select id="topics-filter"><option value="">all</option><option value="unread">unread</option><option value="pending">pending</option><option value="read">read</option></select>
      <input id="topics-name" type="text" placeholder="new topic name…" />
      <select id="topics-prio"><option value="normal">normal</option><option value="low">low</option><option value="high">high</option></select>
      <button id="topics-add">add</button>
    </div>
    <div id="topics-msg" class="dim"></div>
    <div id="topics-list"></div>`;

  const say = (t: string): void => {
    const m = el.querySelector('#topics-msg');
    if (m) m.textContent = t;
  };

  const list = async (): Promise<void> => {
    const box = el.querySelector('#topics-list');
    if (!box) return;
    try {
      const rows = await listTopics(unitID, filter);
      box.innerHTML =
        rows
          .map(
            (t) => `<div class="frow"><span>${esc(t.name)}</span>
              <span class="dim">${esc(t.priority)}</span>
              <span class="fops"><button data-cyc="${esc(t.id)}" data-st="${esc(t.status)}">${esc(t.status)} → ${NEXT[t.status] ?? ''}</button>
              <button data-del="${esc(t.id)}">delete</button></span></div>`,
          )
          .join('') || '<div class="dim">no topics match</div>';
      for (const b of box.querySelectorAll<HTMLButtonElement>('[data-cyc]')) {
        b.addEventListener('click', () => {
          setTopicStatus(unitID, b.dataset.cyc ?? '', NEXT[b.dataset.st ?? ''] ?? 'unread')
            .then(() => list())
            .catch((e: unknown) => say(e instanceof Error ? e.message : String(e)));
        });
      }
      for (const b of box.querySelectorAll<HTMLButtonElement>('[data-del]')) {
        b.addEventListener('click', () => {
          if (!window.confirm('Delete this topic?')) return;
          deleteTopic(unitID, b.dataset.del ?? '', true)
            .then(() => list())
            .catch((e: unknown) => say(e instanceof Error ? e.message : String(e)));
        });
      }
    } catch (e: unknown) {
      say(e instanceof Error ? e.message : String(e));
    }
  };

  (el.querySelector('#topics-unit') as HTMLSelectElement | null)?.addEventListener('change', (e) => {
    unitID = (e.target as HTMLSelectElement).value;
    void list();
  });
  (el.querySelector('#topics-filter') as HTMLSelectElement | null)?.addEventListener('change', (e) => {
    filter = (e.target as HTMLSelectElement).value;
    void list();
  });
  el.querySelector('#topics-add')?.addEventListener('click', () => {
    const name = (el.querySelector('#topics-name') as HTMLInputElement | null)?.value.trim() ?? '';
    const prio = (el.querySelector('#topics-prio') as HTMLSelectElement | null)?.value ?? 'normal';
    if (!name) {
      say('name is required');
      return;
    }
    createTopic(unitID, name, prio)
      .then(() => list())
      .catch((e: unknown) => say(e instanceof Error ? e.message : String(e)));
  });

  await list();
}
