// Typed wrapper over the Wails-injected Go bridge.
// At runtime Wails exposes bound methods as window.go.gui.GuiApi.*;
// in plain-browser preview (vite dev without Wails) the bridge is absent
// and every call falls back to an explicit "preview" value.

import type { DashboardDTO, WeekDTO } from './types';

interface WailsBridge {
  go?: {
    gui?: {
      GuiApi?: Record<string, (...args: unknown[]) => Promise<unknown>>;
    };
    runtime?: {
      WindowMinimise?: () => void;
      WindowToggleMaximise?: () => void;
      Quit?: () => void;
    };
  };
}

function bridge(): Record<string, (...args: unknown[]) => Promise<unknown>> | null {
  const w = window as unknown as WailsBridge;
  return w.go?.gui?.GuiApi ?? null;
}

function winCtl(): NonNullable<WailsBridge['go']>['runtime'] | null {
  const w = window as unknown as WailsBridge;
  return w.go?.runtime ?? null;
}

export async function getVersion(): Promise<string> {
  const api = bridge();
  if (!api?.GetVersion) return '0.1.0-dev (browser preview)';
  return (await api.GetVersion()) as string;
}

export async function getDashboard(): Promise<DashboardDTO | null> {
  const api = bridge();
  if (!api?.GetDashboard) return null; // browser preview: no backend
  return (await api.GetDashboard()) as DashboardDTO;
}

export async function getWeekOffset(offset: number): Promise<WeekDTO | null> {
  const api = bridge();
  if (!api?.GetWeekOffset) return null;
  return (await api.GetWeekOffset(offset)) as WeekDTO;
}

export function windowControl(action: 'min' | 'max' | 'close'): void {
  const ctl = winCtl();
  if (!ctl) return; // browser preview: buttons are inert
  if (action === 'min') ctl.WindowMinimise?.();
  else if (action === 'max') ctl.WindowToggleMaximise?.();
  else ctl.Quit?.();
}
