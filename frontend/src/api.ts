// Typed wrapper over the Wails-injected Go bridge.
// At runtime Wails exposes bound methods as window.go.gui.GuiApi.*;
// in plain-browser preview (vite dev without Wails) the bridge is absent
// and every call falls back to an explicit "preview" value.

import type {
  AnalyticsDTO,
  BackupDTO,
  CommandResultDTO,
  ConfigDTO,
  DashboardDTO,
  DayItemsDTO,
  FileDTO,
  ImportPlanDTO,
  ImportReportDTO,
  MonthDTO,
  ReportDTO,
  TaskDTO,
  TaskDetailDTO,
  TaskFilterDTO,
  TopicDTO,
  UnitCardDTO,
  UnitDetailDTO,
  WeekDTO,
} from './types';

interface WailsBridge {
  go?: {
    gui?: {
      GuiApi?: Record<string, (...args: unknown[]) => Promise<unknown>>;
    };
  };
  runtime?: {
    WindowMinimise?: () => void;
    WindowToggleMaximise?: () => void;
    Quit?: () => void;
  };
}

function bridge(): Record<string, (...args: unknown[]) => Promise<unknown>> | null {
  const w = window as unknown as WailsBridge;
  return w.go?.gui?.GuiApi ?? null;
}

function winCtl(): WailsBridge['runtime'] | null {
  const w = window as unknown as WailsBridge;
  return w.runtime ?? null;
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

export async function getMonthOffset(offset: number): Promise<MonthDTO | null> {
  const api = bridge();
  if (!api?.GetMonthOffset) return null;
  return (await api.GetMonthOffset(offset)) as MonthDTO;
}

export async function getDayItems(dateISO: string): Promise<DayItemsDTO | null> {
  const api = bridge();
  if (!api?.GetDayItems) return null;
  return (await api.GetDayItems(dateISO)) as DayItemsDTO;
}

export async function getTaskDetail(unitID: string, taskID: string): Promise<TaskDetailDTO | null> {
  const api = bridge();
  if (!api?.GetTaskDetail) return null;
  return (await api.GetTaskDetail(unitID, taskID)) as TaskDetailDTO;
}

export async function moveDeadline(unitID: string, taskID: string, due: string): Promise<TaskDetailDTO> {
  const api = bridge();
  if (!api?.MoveDeadline) throw new Error('no backend in browser preview');
  return (await api.MoveDeadline(unitID, taskID, due)) as TaskDetailDTO;
}

export async function setTaskStatus(unitID: string, taskID: string, status: string): Promise<TaskDetailDTO> {
  const api = bridge();
  if (!api?.SetTaskStatus) throw new Error('no backend in browser preview');
  return (await api.SetTaskStatus(unitID, taskID, status)) as TaskDetailDTO;
}

export async function runCommand(input: string): Promise<CommandResultDTO | null> {
  const api = bridge();
  if (!api?.RunCommand) return null;
  return (await api.RunCommand(input)) as CommandResultDTO;
}

function need(name: string): Record<string, (...args: unknown[]) => Promise<unknown>> {
  const api = bridge();
  if (!api?.[name]) throw new Error('no backend in browser preview');
  return api;
}

export async function listFiles(rel: string): Promise<FileDTO[]> {
  return (await need('ListFiles').ListFiles(rel)) as FileDTO[];
}

export async function searchFiles(query: string): Promise<FileDTO[]> {
  return (await need('SearchFiles').SearchFiles(query)) as FileDTO[];
}

export async function openFile(rel: string): Promise<string> {
  return (await need('OpenFile').OpenFile(rel)) as string;
}

export async function makeDir(rel: string): Promise<FileDTO> {
  return (await need('MakeDir').MakeDir(rel)) as FileDTO;
}

export async function renameFile(rel: string, newName: string): Promise<FileDTO> {
  return (await need('RenameFile').RenameFile(rel, newName)) as FileDTO;
}

export async function deleteFile(rel: string, recursive: boolean): Promise<string> {
  return (await need('DeleteFile').DeleteFile(rel, recursive)) as string;
}

export async function previewImport(srcAbs: string, dest: string, recursive: boolean): Promise<ImportPlanDTO> {
  return (await need('PreviewImport').PreviewImport(srcAbs, dest, recursive)) as ImportPlanDTO;
}

export async function confirmImport(token: string): Promise<ImportReportDTO> {
  return (await need('ConfirmImport').ConfirmImport(token)) as ImportReportDTO;
}

export async function getReport(unitID: string): Promise<ReportDTO> {
  return (await need('GetReport').GetReport(unitID)) as ReportDTO;
}

export async function listBackups(): Promise<BackupDTO[]> {
  return (await need('ListBackups').ListBackups()) as BackupDTO[];
}

export async function createBackup(full: boolean): Promise<BackupDTO> {
  return (await need('CreateBackup').CreateBackup(full)) as BackupDTO;
}

export async function verifyBackup(path: string): Promise<BackupDTO> {
  return (await need('VerifyBackup').VerifyBackup(path)) as BackupDTO;
}

export async function restoreBackup(path: string, overwrite: boolean): Promise<string> {
  return (await need('RestoreBackup').RestoreBackup(path, overwrite)) as string;
}

export async function getConfig(): Promise<ConfigDTO> {
  return (await need('GetConfig').GetConfig()) as ConfigDTO;
}

export async function getAnalytics(): Promise<AnalyticsDTO> {
  return (await need('GetAnalytics').GetAnalytics()) as AnalyticsDTO;
}

export async function getUnitCards(): Promise<UnitCardDTO[]> {
  return (await need('GetUnitCards').GetUnitCards()) as UnitCardDTO[];
}

export async function getUnitDetail(unitID: string): Promise<UnitDetailDTO> {
  return (await need('GetUnitDetail').GetUnitDetail(unitID)) as UnitDetailDTO;
}

export async function createUnit(name: string, code: string): Promise<UnitCardDTO> {
  return (await need('CreateUnit').CreateUnit(name, code)) as UnitCardDTO;
}

export async function renameUnit(unitID: string, newName: string): Promise<UnitCardDTO> {
  return (await need('RenameUnit').RenameUnit(unitID, newName)) as UnitCardDTO;
}

export async function setUnitArchived(unitID: string, archived: boolean): Promise<UnitCardDTO> {
  return (await need('SetUnitArchived').SetUnitArchived(unitID, archived)) as UnitCardDTO;
}

export async function deleteUnit(unitID: string, confirm: boolean): Promise<string> {
  return (await need('DeleteUnit').DeleteUnit(unitID, confirm)) as string;
}

export async function listTopics(unitID: string, status: string): Promise<TopicDTO[]> {
  return (await need('ListTopics').ListTopics(unitID, status)) as TopicDTO[];
}

export async function createTopic(unitID: string, name: string, priority: string): Promise<TopicDTO> {
  return (await need('CreateTopic').CreateTopic(unitID, name, priority)) as TopicDTO;
}

export async function setTopicStatus(unitID: string, topicID: string, status: string): Promise<TopicDTO> {
  return (await need('SetTopicStatus').SetTopicStatus(unitID, topicID, status)) as TopicDTO;
}

export async function deleteTopic(unitID: string, topicID: string, confirm: boolean): Promise<string> {
  return (await need('DeleteTopic').DeleteTopic(unitID, topicID, confirm)) as string;
}

export async function listTasksFlat(f: TaskFilterDTO): Promise<TaskDTO[]> {
  return (await need('ListTasksFlat').ListTasksFlat(f)) as TaskDTO[];
}

export async function createTask(
  unitID: string,
  title: string,
  kind: string,
  due: string,
  priority: string,
): Promise<TaskDetailDTO> {
  return (await need('CreateTask').CreateTask(unitID, title, kind, due, priority)) as TaskDetailDTO;
}

export async function deleteTask(unitID: string, taskID: string, confirm: boolean): Promise<string> {
  return (await need('DeleteTask').DeleteTask(unitID, taskID, confirm)) as string;
}

export async function setDataRoot(path: string): Promise<string> {
  return (await need('SetDataRoot').SetDataRoot(path)) as string;
}

export function windowControl(action: 'min' | 'max' | 'close'): void {
  const ctl = winCtl();
  if (!ctl) return; // browser preview: buttons are inert
  if (action === 'min') ctl.WindowMinimise?.();
  else if (action === 'max') ctl.WindowToggleMaximise?.();
  else ctl.Quit?.();
}
