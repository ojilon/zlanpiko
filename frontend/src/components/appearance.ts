// Appearance panel (Settings view): theme + background image + the
// transparency / blur / texture controls.
//
// Every control writes straight through to appearance.ts, which pushes the
// value into a CSS custom property — so dragging a slider previews live.
// Bundled images are listed by frontend/public/backgrounds/manifest.json;
// see docs/24 for how to add them.

import {
  DEFAULT_APPEARANCE,
  getAppearance,
  forgetCustomBackground,
  resetAppearance,
  saveAppearance,
  useCustomBackground,
} from '../appearance';
import type { AppearanceSettings } from '../appearance';
import { bundledUrl, listBundledBackgrounds, probeImage } from '../backgrounds';
import { getTheme, setTheme } from '../theme';
import type { ThemeName } from '../theme';
import { THEMES, suggestedLook, themePreset } from '../themes';
import { esc } from '../views/dashboard';

interface SliderSpec {
  id: string;
  label: string;
  hint: string;
  min: number;
  max: number;
  step: number;
  value: number;
  format: (v: number) => string;
  apply: (v: number) => void;
}

function sliders(s: AppearanceSettings): SliderSpec[] {
  return [
    {
      id: 'ap-transparency',
      label: 'Panel transparency',
      hint: 'How much of the background shows through cards and panels. Text sits on its own layer, so readability holds.',
      min: 0,
      max: 85,
      step: 1,
      value: s.cardTransparency,
      format: (v) => `${v}%`,
      apply: (v) => void saveAppearance({ cardTransparency: v }),
    },
    {
      id: 'ap-cardblur',
      label: 'Card blur',
      hint: 'Frosted-glass blur behind cards. Set to 0 on slower machines.',
      min: 0,
      max: 24,
      step: 1,
      value: s.cardBlur,
      format: (v) => `${v}px`,
      apply: (v) => void saveAppearance({ cardBlur: v }),
    },
    {
      id: 'ap-bgblur',
      label: 'Background blur',
      hint: 'Softens the photo itself — useful behind busy images.',
      min: 0,
      max: 30,
      step: 1,
      value: s.backgroundBlur,
      format: (v) => `${v}px`,
      apply: (v) => void saveAppearance({ backgroundBlur: v }),
    },
    {
      id: 'ap-dim',
      label: 'Background dim',
      hint: 'Darkens the photo so text keeps its contrast.',
      min: 0,
      max: 85,
      step: 1,
      value: s.backgroundDim,
      format: (v) => `${v}%`,
      apply: (v) => void saveAppearance({ backgroundDim: v }),
    },
    {
      id: 'ap-texture',
      label: 'Texture',
      hint: 'A fine grain over flat areas — stops large colour fields from banding.',
      min: 0,
      max: 60,
      step: 1,
      value: s.texture,
      format: (v) => `${v}%`,
      apply: (v) => void saveAppearance({ texture: v }),
    },
  ];
}

function sliderRow(spec: SliderSpec): string {
  return `<div class="opt-row">
    <div class="opt-label"><span>${esc(spec.label)}</span>
      <span class="opt-value" id="${spec.id}-val">${esc(spec.format(spec.value))}</span></div>
    <input type="range" id="${spec.id}" min="${spec.min}" max="${spec.max}"
           step="${spec.step}" value="${spec.value}" />
    <div class="opt-hint">${esc(spec.hint)}</div>
  </div>`;
}

function says(host: HTMLElement, text: string, cls: '' | 'ok' | 'err' = ''): void {
  const box = host.querySelector<HTMLElement>('#ap-status');
  if (!box) return;
  box.className = `bg-status${cls ? ' ' + cls : ''}`;
  box.textContent = text;
}

/** Keep every control in sync with the stored settings. */
function sync(host: HTMLElement): void {
  const s = getAppearance();

  for (const btn of host.querySelectorAll<HTMLButtonElement>('.theme-sw')) {
    btn.setAttribute('aria-pressed', String(btn.dataset.theme === getTheme()));
  }
  for (const btn of host.querySelectorAll<HTMLButtonElement>('#ap-theme-system button')) {
    btn.setAttribute('aria-pressed', String('system' === getTheme()));
  }
  for (const btn of host.querySelectorAll<HTMLButtonElement>('#ap-bg-toggle button')) {
    const on = btn.dataset.bg === 'on';
    btn.setAttribute('aria-pressed', String(on === s.backgroundEnabled));
  }
  for (const thumb of host.querySelectorAll<HTMLButtonElement>('#ap-thumbs .bg-thumb')) {
    const kind = thumb.dataset.kind ?? '';
    const ref = thumb.dataset.ref ?? '';
    const selected = s.backgroundEnabled && s.backgroundKind === kind && s.backgroundRef === ref;
    thumb.setAttribute('aria-pressed', String(selected));
  }

  const state = host.querySelector<HTMLElement>('#ap-bg-state');
  if (state) {
    state.textContent = s.backgroundEnabled
      ? `on · ${s.backgroundKind === 'bundled' ? esc(s.backgroundRef) : s.backgroundKind}`
      : 'off';
  }

  for (const spec of sliders(s)) {
    const input = host.querySelector<HTMLInputElement>(`#${spec.id}`);
    const out = host.querySelector<HTMLElement>(`#${spec.id}-val`);
    // Leave a slider alone while the user is dragging it.
    if (input && document.activeElement !== input) input.value = String(spec.value);
    if (out) out.textContent = spec.format(spec.value);
    const disabled = !s.backgroundEnabled;
    if (input) input.disabled = disabled;
    const row = input?.closest<HTMLElement>('.opt-row');
    if (row) row.style.opacity = disabled ? '0.5' : '1';
  }
}

export async function renderAppearancePanel(host: HTMLElement): Promise<void> {
  // Manifest entries are developer-supplied but still rendered into a style
  // attribute, so drop anything that is not a plain filename.
  const bundled = (await listBundledBackgrounds()).filter((b) => /^[\w.\- ]+$/.test(b.file));
  const s = getAppearance();

  const thumbs = [
    `<button class="bg-thumb is-none" data-kind="none" data-ref="" type="button"
       title="No background image">none</button>`,
    ...bundled.map(
      (b) => `<button class="bg-thumb" data-kind="bundled" data-ref="${esc(b.file)}" type="button"
        style="background-image:url('${esc(bundledUrl(b.file))}')"
        title="${esc(b.name)}"><span class="bg-thumb-name">${esc(b.name)}</span></button>`,
    ),
    `<button class="bg-thumb is-none" data-kind="custom" data-ref="custom" type="button"
       title="Your own image (stored locally)">my image</button>`,
  ].join('');

  host.innerHTML = `<div class="card">
    <h3>Appearance</h3>
    <div class="opt-grid">

      <div class="opt-row">
        <div class="opt-label"><span>Theme</span></div>
        <div class="theme-groups">
          <div>
            <div class="theme-group-label">Dark · for evenings &amp; rich photos</div>
            <div class="theme-grid">
              ${THEMES.filter((t) => t.kind === 'dark')
                .map(
                  (t) => `<button type="button" class="theme-sw theme-sw-${t.id}" data-theme="${t.id}" aria-pressed="false" title="${esc(t.blurb)} — ${esc(t.photos)}">
                      <span class="sw-chip" aria-hidden="true"></span>
                      <span class="sw-name">${esc(t.name)}</span>
                      <span class="sw-photos">${esc(t.photos)}</span>
                    </button>`,
                )
                .join('')}
            </div>
          </div>
          <div>
            <div class="theme-group-label">Light · for daytime &amp; bright photos</div>
            <div class="theme-grid">
              ${THEMES.filter((t) => t.kind === 'light')
                .map(
                  (t) => `<button type="button" class="theme-sw theme-sw-${t.id}" data-theme="${t.id}" aria-pressed="false" title="${esc(t.blurb)} — ${esc(t.photos)}">
                      <span class="sw-chip" aria-hidden="true"></span>
                      <span class="sw-name">${esc(t.name)}</span>
                      <span class="sw-photos">${esc(t.photos)}</span>
                    </button>`,
                )
                .join('')}
            </div>
          </div>
          <div class="seg" id="ap-theme-system">
            <button type="button" data-theme="system" aria-pressed="false">System (follow OS)</button>
          </div>
        </div>
        <div class="opt-hint">Picking a theme also applies its recommended photo look (transparency, blur, dim) — every slider below stays adjustable. System follows the operating-system app colour setting.</div>
      </div>

      <div class="opt-row">
        <div class="opt-label"><span>Background image</span>
          <span class="opt-value" id="ap-bg-state">off</span></div>
        <div class="seg" id="ap-bg-toggle">
          <button type="button" data-bg="off" aria-pressed="true">Off</button>
          <button type="button" data-bg="on" aria-pressed="false">On</button>
        </div>
        <div class="opt-hint">
          Bundled images live in <code>frontend/public/backgrounds/</code> and are listed in
          <code>backgrounds/manifest.json</code>. See <code>docs/24_BACKGROUND_IMAGES.md</code>
          for how to download them.
        </div>
        <div class="bg-thumbs" id="ap-thumbs">${thumbs}</div>
        <div class="bg-actions">
          <input type="text" id="ap-url" placeholder="https://example.com/photo.jpg" />
          <button type="button" id="ap-url-use">use URL</button>
          <button type="button" id="ap-file">choose file…</button>
          <button type="button" id="ap-forget">remove custom</button>
        </div>
        <input type="file" id="ap-file-input" accept="image/*" hidden />
        <div class="bg-status" id="ap-status"></div>
      </div>

      ${sliders(s).map(sliderRow).join('')}

      <div class="opt-row">
        <div class="bg-actions">
          <button type="button" id="ap-reset">reset appearance</button>
        </div>
        <div class="opt-hint">
          Defaults: no background, ${DEFAULT_APPEARANCE.cardTransparency}% transparency,
          ${DEFAULT_APPEARANCE.cardBlur}px card blur, ${DEFAULT_APPEARANCE.backgroundDim}% dim,
          ${DEFAULT_APPEARANCE.texture}% texture.
        </div>
      </div>

    </div>
  </div>`;

  /* --- theme ---------------------------------------------------------- */
  // Picking a theme brings its recommended photo look with it (documented in
  // themes.ts); the sliders below stay live, so the pairing is a starting
  // point, never a lock-in.
  const pickTheme = (t: ThemeName): void => {
    setTheme(t);
    if (t !== 'system') {
      const preset = themePreset(t);
      saveAppearance({ ...suggestedLook(t), backgroundEnabled: getAppearance().backgroundEnabled });
      says(host, `${preset.name}: ${preset.photos.toLowerCase()} look applied`, 'ok');
    }
    sync(host);
  };
  for (const btn of host.querySelectorAll<HTMLButtonElement>('button.theme-sw')) {
    btn.addEventListener('click', () => {
      const t = btn.dataset.theme;
      if (t === 'system' || (typeof t === 'string' && THEMES.some((p) => p.id === t))) {
        pickTheme(t as ThemeName);
      }
    });
  }
  for (const btn of host.querySelectorAll<HTMLButtonElement>('#ap-theme-system button')) {
    btn.addEventListener('click', () => pickTheme('system'));
  }

  /* --- background on / off -------------------------------------------- */
  for (const btn of host.querySelectorAll<HTMLButtonElement>('#ap-bg-toggle button')) {
    btn.addEventListener('click', () => {
      const on = btn.dataset.bg === 'on';
      // Switching on with nothing selected should do something visible: take
      // the first bundled image if the folder has one, otherwise explain why
      // the backdrop is still flat.
      if (on && getAppearance().backgroundKind === 'none') {
        const first = bundled[0];
        if (first) {
          saveAppearance({
            backgroundEnabled: true,
            backgroundKind: 'bundled',
            backgroundRef: first.file,
          });
          says(host, `using ${first.name}`, 'ok');
        } else {
          saveAppearance({ backgroundEnabled: true });
          says(
            host,
            'no bundled images yet — paste a URL, choose a file, or see docs/24',
            '',
          );
        }
      } else {
        saveAppearance({ backgroundEnabled: on });
      }
      sync(host);
    });
  }

  /* --- gallery --------------------------------------------------------- */
  for (const thumb of host.querySelectorAll<HTMLButtonElement>('#ap-thumbs .bg-thumb')) {
    thumb.addEventListener('click', () => {
      const kind = thumb.dataset.kind ?? 'none';
      if (kind === 'custom') {
        host.querySelector<HTMLInputElement>('#ap-file-input')?.click();
        return;
      }
      if (kind === 'none') {
        saveAppearance({ backgroundEnabled: false, backgroundKind: 'none', backgroundRef: '' });
        says(host, '', '');
      } else {
        saveAppearance({
          backgroundEnabled: true,
          backgroundKind: 'bundled',
          backgroundRef: thumb.dataset.ref ?? '',
        });
        says(host, 'background set', 'ok');
      }
      sync(host);
    });
  }

  /* --- custom file ------------------------------------------------------ */
  const fileInput = host.querySelector<HTMLInputElement>('#ap-file-input');
  host.querySelector('#ap-file')?.addEventListener('click', () => fileInput?.click());
  fileInput?.addEventListener('change', () => {
    const file = fileInput.files?.[0];
    if (!file) return;
    if (!file.type.startsWith('image/')) {
      says(host, 'that file is not an image', 'err');
      return;
    }
    says(host, `storing ${file.name}…`, '');
    useCustomBackground(file)
      .then(() => {
        void renderAppearancePanel(host).then(() => says(host, 'using your image', 'ok'));
      })
      .catch((e: unknown) => says(host, e instanceof Error ? e.message : String(e), 'err'));
  });

  host.querySelector('#ap-forget')?.addEventListener('click', () => {
    void forgetCustomBackground().then(() => {
      void renderAppearancePanel(host).then(() => says(host, 'custom image removed', ''));
    });
  });

  /* --- remote URL -------------------------------------------------------- */
  const useUrl = (): void => {
    const input = host.querySelector<HTMLInputElement>('#ap-url');
    const raw = input?.value.trim() ?? '';
    if (!raw) return;
    if (!/^https?:\/\//i.test(raw)) {
      says(host, 'URL must start with http:// or https://', 'err');
      return;
    }
    says(host, 'loading…', '');
    void probeImage(raw).then((ok) => {
      if (!ok) {
        says(host, 'could not load that image (check the URL or your connection)', 'err');
        return;
      }
      saveAppearance({ backgroundEnabled: true, backgroundKind: 'remote', backgroundRef: raw });
      says(host, 'using remote image', 'ok');
      sync(host);
    });
  };
  host.querySelector('#ap-url-use')?.addEventListener('click', useUrl);
  host.querySelector<HTMLInputElement>('#ap-url')?.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') useUrl();
  });

  /* --- sliders ----------------------------------------------------------- */
  for (const spec of sliders(s)) {
    const input = host.querySelector<HTMLInputElement>(`#${spec.id}`);
    const out = host.querySelector<HTMLElement>(`#${spec.id}-val`);
    input?.addEventListener('input', () => {
      const v = Number(input.value);
      if (out) out.textContent = spec.format(v);
      spec.apply(v);
    });
  }

  /* --- reset -------------------------------------------------------------- */
  host.querySelector('#ap-reset')?.addEventListener('click', () => {
    resetAppearance();
    void renderAppearancePanel(host).then(() => says(host, 'appearance reset', ''));
  });

  sync(host);
}
