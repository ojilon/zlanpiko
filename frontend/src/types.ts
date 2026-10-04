// Mirrors internal/gui/dto.go. Hand-kept; parity is enforced by
// internal/gui/dto_test.go golden files (see docs/cs/02, docs/cs/10).

export interface GuiError {
  code: string;
  message: string;
  hint?: string;
}

export type GuiErrorCode =
  | 'VALIDATION'
  | 'NOT_FOUND'
  | 'CONFLICT'
  | 'STORAGE'
  | 'DB'
  | 'BACKUP'
  | 'CANCELLED';

// Phase A placeholder — full dashboard DTO lands in Phase B.
export interface DashboardDTO {
  units: number;
  topics: number;
  activeTasks: number;
  overdue: number;
  version: string;
}
