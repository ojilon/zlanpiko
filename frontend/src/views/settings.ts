import {
  createBackup,
  getConfig,
  getReport,
  listBackups,
  restoreBackup,
  setDataRoot,
  verifyBackup,
} from '../api';
import { esc } from './dashboard';
import { openGuide } from '../components/guide';

// Settings: storage + identity, plain-text/JSON reports, backup lifecycle
// (docs/cs/09: restore needs explicit confirmation, overwrite takes a safety
// backup first).

function errBox(el: HTMLElement, e: unknown): void {
  const box = el.querySelector('#settings-msg');
  if (box) {
    box.className = 'errline';
    box.textContent = e instanceof Error ? e.message : String(e);
  }
}

export async function renderSettings(el: HTMLElement): Promise<void> {
  el.innerHTML = `<h2>Settings</h2><div id="settings-msg" class="dim"></div>
    <div class="navbtns"><button id="guide-open">📖 User guide</button></div>
    <div class="cols"><div>
      <div class="card"><h3>Storage & version</h3><div id="cfg">loading…</div>
        <div class="drawer-edit"><input id="cfg-root" type="text" placeholder="new storage directory…" />
        <button id="cfg-save">set (restart)</button></div>
      </div>
      <div class="card"><h3>Status report</h3>
        <button id="rep-load">load report</button>
        <pre id="rep-text" class="report"></pre>
      </div>
    </div><div>
      <div class="card"><h3>Backups</h3>
        <label><input id="bak-full" type="checkbox" checked /> include academic files</label>
        <button id="bak-create">create backup</button>
        <div id="bak-list">loading…</div>
      </div>
    </div></div>`;

  try {
    const cfg = await getConfig();
    const box = el.querySelector('#cfg');
    if (box) {
      box.innerHTML = `<div class="dim">data root</div><div>${esc(cfg.data_root)}</div>
        <div class="dim">build</div><div>${esc(cfg.version)}</div>
        <div class="dim">schema</div><div>${cfg.schema_version}</div>`;
    }
  } catch (e: unknown) {
    errBox(el, e);
  }

  el.querySelector('#guide-open')?.addEventListener('click', () => openGuide());

  el.querySelector('#cfg-save')?.addEventListener('click', () => {
    const v = (el.querySelector('#cfg-root') as HTMLInputElement | null)?.value.trim() ?? '';
    setDataRoot(v)
      .then((m) => {
        const box = el.querySelector('#settings-msg');
        if (box) {
          box.className = '';
          box.textContent = m;
        }
      })
      .catch((e: unknown) => errBox(el, e));
  });

  el.querySelector('#rep-load')?.addEventListener('click', () => {
    getReport('')
      .then((r) => {
        const pre = el.querySelector('#rep-text');
        if (pre) pre.textContent = r.text;
      })
      .catch((e: unknown) => errBox(el, e));
  });

  const loadBackups = async (): Promise<void> => {
    const box = el.querySelector('#bak-list');
    if (!box) return;
    try {
      const rows = await listBackups();
      box.innerHTML =
        rows
          .map(
            (b) => `<div class="frow"><span>📦 ${esc(b.name)}</span>
              <span class="dim">${esc(b.kind)} · ${esc(b.created)} · ${esc(b.size_text)}</span>
              <span class="fops"><button data-ver="${esc(b.path)}">verify</button>
              <button data-res="${esc(b.path)}">restore</button></span></div>`,
          )
          .join('') || '<div class="dim">no backups yet</div>';
      for (const btn of box.querySelectorAll<HTMLButtonElement>('[data-ver]')) {
        btn.addEventListener('click', () => {
          verifyBackup(btn.dataset.ver ?? '')
            .then(() => {
              const m = el.querySelector('#settings-msg');
              if (m) {
                m.className = '';
                m.textContent = 'backup verified OK';
              }
            })
            .catch((e: unknown) => errBox(el, e));
        });
      }
      for (const btn of box.querySelectorAll<HTMLButtonElement>('[data-res]')) {
        btn.addEventListener('click', () => {
          const ow = window.confirm(
            'Restore OVERWRITES current data (a safety backup is taken first). Continue?',
          );
          if (!window.confirm('Really restore? This is the second and final confirmation.')) return;
          restoreBackup(btn.dataset.res ?? '', ow)
            .then((m) => {
              const mbox = el.querySelector('#settings-msg');
              if (mbox) {
                mbox.className = '';
                mbox.textContent = m;
              }
            })
            .catch((e: unknown) => errBox(el, e));
        });
      }
    } catch (e: unknown) {
      errBox(el, e);
    }
  };

  el.querySelector('#bak-create')?.addEventListener('click', () => {
    const full = (el.querySelector('#bak-full') as HTMLInputElement | null)?.checked ?? true;
    createBackup(full)
      .then(() => loadBackups())
      .catch((e: unknown) => errBox(el, e));
  });

  await loadBackups();
}
