// Pomodoro 前端类型与 Wails 封装
// ------------------------------------------------------------
// 与后端 internal/pomodoro/types.go 对齐。

import { callWails } from "@/lib/wails";

/** 番茄钟记录 —— 与后端 pomodoro.Record 对齐 */
export interface PomodoroRecord {
  id: string;
  todoItemId?: string | null;
  startTime: string;
  endTime?: string | null;
  duration: number; // 计划秒数
  actualDuration: number; // 实际秒数
  type: "work" | "short_break" | "long_break";
  status: "completed" | "interrupted";
  createdAt: string;
}

/** 创建记录参数 */
export interface CreateRecordParams {
  todoItemId?: string | null;
  startTime?: string | null;
  duration?: number;
  actualDuration?: number;
  type?: string;
  status?: string;
}

/** 统计结果 —— 与后端 pomodoro.Stats 对齐 */
export interface PomodoroStats {
  date: string;
  totalSeconds: number;
  plannedSeconds: number;
  completedCount: number;
  interruptedCount: number;
  workCount: number;
  breakCount: number;
}

/** 每日统计 */
export interface DailyStat {
  date: string;
  totalSeconds: number;
  count: number;
}

export const pomodoroApi = {
  create: (p: CreateRecordParams) => callWails<PomodoroRecord>("PomodoroCreate", p),
  get: (id: string) => callWails<PomodoroRecord>("PomodoroGet", id),
  listByTodo: (todoItemId: string) =>
    callWails<PomodoroRecord[]>("PomodoroListByTodo", todoItemId),
  listToday: () => callWails<PomodoroRecord[]>("PomodoroListToday"),
  todayStats: () => callWails<PomodoroStats>("PomodoroTodayStats"),
  dailyStats: (days = 7) => callWails<DailyStat[]>("PomodoroDailyStats", days),
};
