import { describe, expect, it } from 'vitest';
import { esc, statusLine } from './dashboard';
import type { SummaryDTO } from '../types';

// Pure view helpers: no backend, no DOM. Backend parity for the numbers
// themselves lives in internal/gui/*_test.go goldens.

describe('esc', () => {
  it('escapes user data for innerHTML', () => {
    expect(esc('<b>"&"</b>')).toBe('&lt;b&gt;&quot;&amp;&quot;&lt;/b&gt;');
  });
});

describe('statusLine', () => {
  it('mirrors the TUI status format', () => {
    const s: SummaryDTO = {
      units: 11,
      total_topics: 50,
      read: 0,
      pending: 0,
      unread: 50,
      coverage: 0,
      coverage_text: '0%',
      has_topics: true,
      active_tasks: 1,
      completed_tasks: 0,
      overdue_tasks: 0,
    };
    expect(statusLine(s)).toBe('11 units · 50 topics (R0/P0/U50 0%) · 1 active · 0 overdue');
  });
});
