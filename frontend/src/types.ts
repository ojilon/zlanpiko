// Mirrors internal/gui/dto.go. Hand-kept; parity is enforced by
// internal/gui/testdata/*.json goldens (see docs/cs/02, docs/cs/10).
// Times arrive as RFC 3339 local strings; display strings are pre-formatted
// by Go — never re-derive deadlines here.

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

export interface TaskDTO {
  id: string;
  unit_id: string;
  unit_name: string;
  title: string;
  kind: string;
  status: string;
  overdue: boolean;
  priority: string;
  due_iso?: string;
  due_display: string;
  distance: string;
  days_left?: number;
  progress_pct?: number;
}

export interface UnitCardDTO {
  unit_id: string;
  name: string;
  code: string;
  total: number;
  read: number;
  pending: number;
  unread: number;
  coverage: number;
  coverage_text: string;
  has_topics: boolean;
  active_tasks: number;
  completed_tasks: number;
  next_due_display: string;
  attention: string;
  reason?: string;
}

export interface DayDTO {
  date: string;
  display: string;
  items: TaskDTO[];
}

export interface WeekDTO {
  year: number;
  week: number;
  title: string;
  monday: string;
  days: DayDTO[];
  overdue: TaskDTO[];
}

export interface SummaryDTO {
  units: number;
  total_topics: number;
  read: number;
  pending: number;
  unread: number;
  coverage: number;
  coverage_text: string;
  has_topics: boolean;
  active_tasks: number;
  completed_tasks: number;
  overdue_tasks: number;
}

export interface DashboardDTO {
  version: string;
  summary: SummaryDTO;
  units: UnitCardDTO[];
  upcoming: TaskDTO[];
  overdue: TaskDTO[];
  week: WeekDTO;
}

export interface MonthDayDTO {
  date: string;
  display: string;
  count: number;
  level: 'none' | 'info' | 'warn' | 'bad' | 'ok';
  in_month: boolean;
}

export interface MonthDTO {
  year: number;
  month: number;
  title: string;
  days: MonthDayDTO[];
  overdue_count: number;
}

export interface DayItemsDTO {
  date: string;
  display: string;
  items: TaskDTO[];
  overdue: TaskDTO[];
}

export interface TaskDetailDTO {
  task: TaskDTO;
  description: string;
  notes: string;
  created_display: string;
  completed_display: string;
}
