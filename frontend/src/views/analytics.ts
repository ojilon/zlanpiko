import { getAnalytics } from '../api';
import { esc } from './dashboard';
import type { AnalyticsDTO } from '../types';

// Analytics view: five SVG charts rendered from Go-computed numbers plus the
// attention table (docs/cs/07). Every caption states its formula; coverage
// is reading coverage, never a pass/fail prediction.

function ring(pct: number, label: string): string {
  const r = 34;
  const c = 2 * Math.PI * r;
  const off = c * (1 - Math.min(100, Math.max(0, pct)) / 100);
  return `<svg class="ring" viewBox="0 0 84 84" role="img" aria-label="${esc(label)}">
    <circle cx="42" cy="42" r="${r}" class="ring-bg" />
    <circle cx="42" cy="42" r="${r}" class="ring-fg"
      stroke-dasharray="${c.toFixed(1)}" stroke-dashoffset="${off.toFixed(1)}" />
    <text x="42" y="47">${esc(label)}</text>
  </svg>`;
}

function pill(attention: string): string {
  const cls =
    attention === 'Overdue' || attention === 'At Risk'
      ? 'pill bad'
      : attention === 'Attention'
        ? 'pill warn'
        : attention === 'Completed'
          ? 'pill ok'
          : attention === 'On Track'
            ? 'pill info'
            : 'pill';
  return `<span class="${cls}">${esc(attention)}</span>`;
}

const SEV: Record<string, number> = {
  Overdue: 0,
  'At Risk': 1,
  Attention: 2,
  'On Track': 3,
  Completed: 4,
  '—': 5,
};

export async function renderAnalytics(el: HTMLElement): Promise<void> {
  let a: AnalyticsDTO;
  try {
    a = await getAnalytics();
  } catch {
    el.innerHTML = `<h2>Analytics</h2><p class="dim">No backend in browser preview.</p>`;
    return;
  }
  const s = a.summary;
  const total = Math.max(1, s.total_topics);
  const rW = (s.read / total) * 100;
  const pW = (s.pending / total) * 100;
  const st = a.statuses;
  const statusRows = (
    [
      ['not started', st.not_started, ''],
      ['in progress', st.in_progress, ''],
      ['completed', st.completed, 'ok'],
      ['submitted', st.submitted, 'ok'],
      ['overdue (derived)', st.overdue, 'bad'],
    ] as [string, number, string][]
  )
    .map(
      ([label, n, cls]) =>
        `<div class="hrow"><span>${esc(label)}</span><div class="bar grow"><i class="${cls}" style="width:${Math.min(100, n * 20)}%"></i></div><strong>${n}</strong></div>`,
    )
    .join('');
  const maxH = Math.max(1, ...a.histogram.map((h) => h.count));
  const cols = a.histogram
    .map(
      (h) =>
        `<div class="hcol ${h.worst}" title="${h.date} · ${h.count} due"><i style="height:${Math.max(h.count > 0 ? 8 : 2, (h.count / maxH) * 64)}px"></i><span>${esc(h.display.split(' ')[1] ?? '')}</span></div>`,
    )
    .join('');
  const units = [...a.units].sort((x, y) => {
    if (x.has_topics !== y.has_topics) return x.has_topics ? -1 : 1;
    return x.coverage - y.coverage;
  });
  const unitBars = units
    .map(
      (u) => `<div class="hrow"><span class="uname">${esc(u.name)}</span>
        <div class="bar grow">${u.has_topics ? `<i style="width:${u.coverage.toFixed(1)}%"></i>` : ''}</div>
        <strong>${esc(u.coverage_text)}</strong></div>`,
    )
    .join('');
  const attn = [...a.units].sort((x, y) => (SEV[x.attention] ?? 9) - (SEV[y.attention] ?? 9));
  const rows = attn
    .map(
      (u) => `<tr><td>${esc(u.name)}</td><td>${pill(u.attention)}</td>
        <td class="dim">${esc(u.reason || '—')}</td>
        <td>${u.next_due_display ? esc(u.next_due_display) : '—'}</td>
        <td>${esc(u.coverage_text)}</td></tr>`,
    )
    .join('');

  el.innerHTML = `
    <h2>Analytics</h2>
    <div class="cols"><div>
      <div class="card"><h3>Reading coverage</h3>
        <div class="arow">${ring(s.coverage, s.coverage_text)}
        <div><div>R${s.read} / P${s.pending} / U${s.unread} of ${s.total_topics}</div>
        <div class="dim">formula: read ÷ total × 100 · reading coverage, not mastery</div></div></div>
        <div class="stack"><i class="s-ok" style="width:${rW}%"></i><i class="s-warn" style="width:${pW}%"></i><i class="s-rest" style="width:${100 - rW - pW}%"></i></div>
      </div>
      <div class="card"><h3>Deadlines, next 14 days</h3><div class="hcols">${cols}</div>
        <div class="dim">open items per day; red = overdue-anchored, amber = upcoming</div></div>
      <div class="card"><h3>Coverage per unit (worst first)</h3>${unitBars}</div>
    </div><div>
      <div class="card"><h3>Tasks by status</h3>${statusRows}
        <div class="dim">overdue is derived from deadline + state, never stored</div></div>
      <div class="card"><h3>Attention</h3>
        <table class="attn"><thead><tr><th>Unit</th><th>Label</th><th>Reason</th><th>Next due</th><th>Cov.</th></tr></thead>
        <tbody>${rows}</tbody></table></div>
    </div></div>`;
}
