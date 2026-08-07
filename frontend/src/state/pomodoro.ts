// Pomodoro Store —— 番茄钟状态管理
// ------------------------------------------------------------
// 对接后端 pomodoroApi（PomodoroXxx 方法）。
// 支持：25/5/15 计时、开始/暂停/完成/中断、今日统计、近 7 天柱状。

import { create } from "zustand";
import {
  pomodoroApi,
  type PomodoroRecord,
  type PomodoroStats,
  type DailyStat,
} from "@/lib/pomodoro";

type TimerStatus = "idle" | "running" | "paused";

interface PomodoroState {
  // 计时器
  status: TimerStatus;
  mode: "work" | "short_break" | "long_break";
  /** 剩余秒数 */
  remaining: number;
  /** 计划秒数 */
  planned: number;
  /** 已专注秒数（本段） */
  elapsed: number;
  todoItemId: string | null;

  // 数据
  records: PomodoroRecord[];
  stats: PomodoroStats | null;
  daily: DailyStat[];
  loading: boolean;
  error: string | null;
  toast: string | null;

  // 动作
  start: (todoItemId?: string) => void;
  pause: () => void;
  resume: () => void;
  reset: () => void;
  /** 每秒 tick（由页面 setInterval 驱动） */
  tick: () => void;
  /** 完成当前段（写记录） */
  complete: () => Promise<void>;
  /** 中断当前段 */
  interrupt: () => Promise<void>;
  switchMode: (mode: PomodoroState["mode"]) => void;
  loadToday: () => Promise<void>;
  loadDaily: (days?: number) => Promise<void>;
  loadAll: () => Promise<void>;
  clearToast: () => void;
}

const DURATIONS: Record<PomodoroState["mode"], number> = {
  work: 25 * 60,
  short_break: 5 * 60,
  long_break: 15 * 60,
};

export const usePomodoroStore = create<PomodoroState>((set, get) => ({
  status: "idle",
  mode: "work",
  remaining: DURATIONS.work,
  planned: DURATIONS.work,
  elapsed: 0,
  todoItemId: null,
  records: [],
  stats: null,
  daily: [],
  loading: false,
  error: null,
  toast: null,

  start: (todoItemId?: string) => {
    if (get().status === "running") return;
    set({
      status: "running",
      todoItemId: todoItemId ?? null,
      remaining: DURATIONS[get().mode],
      planned: DURATIONS[get().mode],
      elapsed: 0,
    });
  },

  pause: () => set({ status: "paused" }),
  resume: () => set({ status: "running" }),
  reset: () =>
    set({
      status: "idle",
      remaining: DURATIONS[get().mode],
      elapsed: 0,
      todoItemId: null,
    }),

  tick: () => {
    const s = get();
    if (s.status !== "running") return;
    if (s.remaining <= 1) {
      void s.complete();
      return;
    }
    set({ remaining: s.remaining - 1, elapsed: s.elapsed + 1 });
  },

  complete: async () => {
    const s = get();
    const duration = s.planned;
    const rec = await pomodoroApi.create({
      todoItemId: s.todoItemId,
      duration,
      actualDuration: s.elapsed,
      type: s.mode,
      status: "completed",
    });
    if (rec) {
      set({
        records: [rec, ...s.records],
        toast:
          s.mode === "work"
            ? `专注完成 ${Math.round(s.elapsed / 60)} 分钟 🍅`
            : "休息结束",
      });
    }
    get().reset();
    await get().loadToday();
  },

  interrupt: async () => {
    const s = get();
    if (s.elapsed > 0) {
      await pomodoroApi.create({
        todoItemId: s.todoItemId,
        duration: s.planned,
        actualDuration: s.elapsed,
        type: s.mode,
        status: "interrupted",
      });
    }
    set({ toast: "已中断" });
    get().reset();
    await get().loadToday();
  },

  switchMode: (mode) => {
    set({
      mode,
      status: "idle",
      remaining: DURATIONS[mode],
      planned: DURATIONS[mode],
      elapsed: 0,
      todoItemId: null,
    });
  },

  loadToday: async () => {
    const [records, stats] = await Promise.all([
      pomodoroApi.listToday(),
      pomodoroApi.todayStats(),
    ]);
    set({ records: records ?? [], stats: stats ?? null });
  },

  loadDaily: async (days = 7) => {
    const daily = await pomodoroApi.dailyStats(days);
    set({ daily: daily ?? [] });
  },

  loadAll: async () => {
    set({ loading: true, error: null });
    try {
      const [records, stats, daily] = await Promise.all([
        pomodoroApi.listToday(),
        pomodoroApi.todayStats(),
        pomodoroApi.dailyStats(7),
      ]);
      set({
        records: records ?? [],
        stats: stats ?? null,
        daily: daily ?? [],
        loading: false,
      });
    } catch (err) {
      set({ error: String(err), loading: false });
    }
  },

  clearToast: () => set({ toast: null }),
}));
