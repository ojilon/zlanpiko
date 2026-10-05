import {
  confirmImport,
  deleteFile,
  listFiles,
  makeDir,
  openFile,
  previewImport,
  renameFile,
  searchFiles,
} from '../api';
import { esc } from './dashboard';

// File explorer: breadcrumb navigation, list/search/open, mkdir/rename/
// delete, and preview-first import (docs/cs/02: never silently import).

let rel = '';

function crumbs(): string {
  const parts = rel ? rel.split('/') : [];
  let html = `<button data-crumb="">root</button>`;
  let acc = '';
  for (const p of parts) {
    acc = acc ? `${acc}/${p}` : p;
    html += ` / <button data-crumb="${esc(acc)}">${esc(p)}</button>`;
  }
  return `<div class="crumbs">${html}</div>`;
}

function errBox(el: HTMLElement, e: unknown): void {
  const box = el.querySelector('#files-msg');
  if (box) {
    box.className = 'errline';
    box.textContent = e instanceof Error ? e.message : String(e);
  }
}

export async function renderFiles(el: HTMLElement): Promise<void> {
  el.innerHTML = `
    <div class="view-head"><h2>Files</h2>
      <div class="navbtns"><button id="files-up">up</button><button id="files-mkdir">new folder</button></div>
    </div>
    <div class="ftools">
      <input id="files-search" type="text" placeholder="search files…" />
      <input id="files-src" type="text" placeholder="import source path…" />
      <input id="files-dest" type="text" placeholder="dest (empty = inbox)" />
      <button id="files-preview">preview import</button>
    </div>
    ${crumbs()}
    <div id="files-msg" class="dim"></div>
    <div id="files-list"></div>
    <div id="files-plan"></div>`;

  const list = async (): Promise<void> => {
    const box = el.querySelector('#files-list');
    if (!box) return;
    try {
      const rows = await listFiles(rel);
      box.innerHTML =
        rows
          .map(
            (f) => `<div class="frow">
              <button class="fname" data-rel="${esc(f.rel)}" data-dir="${f.is_dir ? '1' : ''}">
                ${f.is_dir ? '📁' : '📄'} ${esc(f.name)}</button>
              <span class="dim">${esc(f.size_text)} · ${esc(f.mod_display)}</span>
              <span class="fops">
                ${f.is_dir ? '' : `<button data-open="${esc(f.rel)}">open</button>`}
                <button data-ren="${esc(f.rel)}">rename</button>
                <button data-del="${esc(f.rel)}">delete</button>
              </span>
            </div>`,
          )
          .join('') || '<div class="dim">empty folder</div>';
      for (const b of box.querySelectorAll<HTMLButtonElement>('.fname')) {
        b.addEventListener('click', () => {
          if (b.dataset.dir === '1') {
            rel = b.dataset.rel ?? '';
            void renderFiles(el);
          } else {
            openFile(b.dataset.rel ?? '').catch((e: unknown) => errBox(el, e));
          }
        });
      }
      for (const b of box.querySelectorAll<HTMLButtonElement>('[data-open]')) {
        b.addEventListener('click', (ev) => {
          ev.stopPropagation();
          openFile(b.dataset.open ?? '').catch((e: unknown) => errBox(el, e));
        });
      }
      for (const b of box.querySelectorAll<HTMLButtonElement>('[data-ren]')) {
        b.addEventListener('click', () => {
          const cur = (b.dataset.ren ?? '').split('/').pop() ?? '';
          const name = window.prompt('New name:', cur);
          if (!name || name === cur) return;
          renameFile(b.dataset.ren ?? '', name)
            .then(() => list())
            .catch((e: unknown) => errBox(el, e));
        });
      }
      for (const b of box.querySelectorAll<HTMLButtonElement>('[data-del]')) {
        b.addEventListener('click', () => {
          if (!window.confirm(`Delete ${b.dataset.del}? There is no trash.`)) return;
          deleteFile(b.dataset.del ?? '', true)
            .then(() => list())
            .catch((e: unknown) => errBox(el, e));
        });
      }
    } catch (e: unknown) {
      errBox(el, e);
    }
  };

  el.querySelector('#files-up')?.addEventListener('click', () => {
    const i = rel.lastIndexOf('/');
    rel = i >= 0 ? rel.slice(0, i) : '';
    void renderFiles(el);
  });
  for (const b of el.querySelectorAll<HTMLButtonElement>('[data-crumb]')) {
    b.addEventListener('click', () => {
      rel = b.dataset.crumb ?? '';
      void renderFiles(el);
    });
  }
  el.querySelector('#files-mkdir')?.addEventListener('click', () => {
    const name = window.prompt('Folder name:');
    if (!name) return;
    makeDir(rel ? `${rel}/${name}` : name)
      .then(() => list())
      .catch((e: unknown) => errBox(el, e));
  });
  const search = el.querySelector('#files-search') as HTMLInputElement | null;
  search?.addEventListener('keydown', (e) => {
    if (e.key !== 'Enter') return;
    const box = el.querySelector('#files-list');
    if (!box || !search.value.trim()) {
      void list();
      return;
    }
    searchFiles(search.value.trim())
      .then((rows) => {
        box.innerHTML =
          rows.map((f) => `<div class="frow"><span>📄 ${esc(f.name)}</span><span class="dim">${esc(f.rel)}</span></div>`).join('') ||
          '<div class="dim">no matches</div>';
      })
      .catch((err: unknown) => errBox(el, err));
  });
  el.querySelector('#files-preview')?.addEventListener('click', () => {
    const src = (el.querySelector('#files-src') as HTMLInputElement | null)?.value.trim() ?? '';
    const dest = (el.querySelector('#files-dest') as HTMLInputElement | null)?.value.trim() ?? '';
    const plan = el.querySelector('#files-plan');
    if (!src || !plan) return;
    previewImport(src, dest, true)
      .then((p) => {
        plan.innerHTML = `<h3>Import preview: ${p.copies} file(s) → ${esc(p.dest_dir)}</h3>
          ${p.plans.map((r) => `<div class="frow"><span>${esc(r.action)} ${esc(r.name)}</span><span class="dim">${esc(r.dest_rel)}${r.reason ? ' · ' + esc(r.reason) : ''}</span></div>`).join('')}
          <button id="files-confirm">confirm import (${p.copies})</button>`;
        plan.querySelector('#files-confirm')?.addEventListener('click', () => {
          confirmImport(p.token)
            .then((rep) => {
              plan.innerHTML = `<div class="dim">imported ${rep.copied}, skipped ${rep.skipped}, failed ${rep.failed}</div>`;
              void list();
            })
            .catch((err: unknown) => errBox(el, err));
        });
      })
      .catch((err: unknown) => errBox(el, err));
  });

  await list();
}
