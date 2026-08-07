// PomodoroPage —— 番茄钟
// ------------------------------------------------------------
// 布局：
// 1. 左：计时器（模式切换 / 开始 / 暂停 / 重置 / 中断）
// 2. 右：今日统计 + 近 7 天专注柱状 + 今日记录列表
//
// 对接后端：pomodoroApi（PomodoroXxx 方法）

import { useEffect, useState, type ReactNode } from "react";
import { usePomodoroStore } from "@/state/pomodoro";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { cn } from "@/lib/utils";
import {
  Play,
  Pause,
  RotateCcw,
  X,
  Timer,
  Coffee,
  ListRestart,
  Loader2,
  CheckCircle2,
  AlertCircle,
} from "lucide-react";

export function PomodoroPage() {
  const loadAll = usePomodoroStore((s) => s.loadAll);

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  return (
    <div className="flex h-full w-full min-h-0 bg-background">
      {/* —— 左：计时器 —— */}
      <section className="flex w-[380px] shrink-0 flex-col border-r border-border bg-card">
        <TimerPanel />
      </section>

      {/* —— 右：统计 —— */}
      <section className="flex min-w-0 flex-1 flex-col overflow-y-auto scrollbar-dark">
        <StatsPanel />
      </section>
    </div>
  );
}

// —— 计时器 ——
function TimerPanel() {
  const status = usePomodoroStore((s) => s.status);
  const mode = usePomodoroStore((s) => s.mode);
  const remaining = usePomodoroStore((s) => s.remaining);
  const planned = usePomodoroStore((s) => s.planned);
  const elapsed = usePomodoroStore((s) => s.elapsed);
  const toast = usePomodoroStore((s) => s.toast);
  const start = usePomodoroStore((s) => s.start);
  const pause = usePomodoroStore((s) => s.pause);
  const resume = usePomodoroStore((s) => s.resume);
  const reset = usePomodoroStore((s) => s.reset);
  const interrupt = usePomodoroStore((s) => s.interrupt);
  const switchMode = usePomodoroStore((s) => s.switchMode);

  // 每秒 tick
  useEffect(() => {
    if (status !== "running") return;
    const id = setInterval(() => {
      usePomodoroStore.getState().tick();
    }, 1000);
    return () => clearInterval(id);
  }, [status]);

  // 格式化 mm:ss
  const mm = String(Math.floor(remaining / 60)).padStart(2, "0");
  const ss = String(remaining % 60).padStart(2, "0");
  const progress = planned > 0 ? Math.min(1, elapsed / planned) : 0;

  const modes = [
    { key: "work" as const, label: "专注", icon: Timer },
    { key: "short_break" as const, label: "短休", icon: Coffee },
    { key: "long_break" as const, label: "长休", icon: ListRestart },
  ];

  return (
    <div className="flex h-full w-full flex-col">
      <div className="shrink-0 border-b border-border px-4 py-3">
        <div className="flex items-center gap-2">
          <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary/15 text-primary">
            <Timer size={13} />
          </div>
          <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            番茄钟
          </span>
          {toast && (
            <span className="ml-auto flex items-center gap-1 text-[10px] text-emerald-500">
              <CheckCircle2 size={10} />
              {toast}
            </span>
          )}
        </div>
      </div>

      <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-6 px-6">
        {/* 模式切换 */}
        <div className="flex gap-1 rounded-lg bg-muted p-1">
          {modes.map((m) => (
            <button
              key={m.key}
              onClick={() => switchMode(m.key)}
              className={cn(
                "flex items-center gap-1.5 rounded-md px-3 py-1.5 text-[11px] transition-colors",
                mode === m.key
                  ? "bg-card font-medium text-primary shadow-soft"
                  : "text-muted-foreground hover:text-foreground"
              )}
            >
              <m.icon size={12} />
              {m.label}
            </button>
          ))}
        </div>

        {/* 环形进度 */}
        <div className="relative flex h-52 w-52 items-center justify-center">
          <svg viewBox="0 0 200 200" className="h-full w-full -rotate-90">
            <circle
              cx="100"
              cy="100"
              r="88"
              fill="none"
              stroke="var(--muted)"
              strokeWidth="10"
            />
            <circle
              cx="100"
              cy="100"
              r="88"
              fill="none"
              stroke="var(--primary)"
              strokeWidth="10"
              strokeLinecap="round"
              strokeDasharray={2 * Math.PI * 88}
              strokeDashoffset={2 * Math.PI * 88 * (1 - progress)}
              className="transition-all duration-1000 ease-linear"
            />
          </svg>
          <div className="absolute flex flex-col items-center">
            <span className="font-mono text-5xl font-semibold tabular-nums text-foreground">
              {mm}:{ss}
            </span>
            <span className="mt-1 text-[11px] text-muted-foreground">
              {status === "running"
                ? "专注中…"
                : status === "paused"
                  ? "已暂停"
                  : "准备就绪"}
            </span>
          </div>
        </div>

        {/* 控制按钮 */}
        <div className="flex items-center gap-2">
          {status === "running" ? (
            <>
              <Button size="sm" variant="outline" className="h-9" onClick={pause}>
                <Pause size={14} className="mr-1" />
                暂停
              </Button>
              <Button size="sm" variant="destructive" className="h-9" onClick={() => void interrupt()}>
                <X size={14} className="mr-1" />
                中断
              </Button>
            </>
          ) : (
            <>
              <Button size="sm" className="h-9" onClick={() => start()}>
                <Play size={14} className="mr-1" />
                {status === "paused" ? "继续" : "开始"}
              </Button>
              <Button size="sm" variant="outline" className="h-9" onClick={reset}>
                <RotateCcw size={13} />
              </Button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

// —— 统计 ——
function StatsPanel() {
  const stats = usePomodoroStore((s) => s.stats);
  const daily = usePomodoroStore((s) => s.daily);
  const records = usePomodoroStore((s) => s.records);
  const loading = usePomodoroStore((s) => s.loading);
  const error = usePomodoroStore((s) => s.error);

  const maxDaily = Math.max(1, ...daily.map((d) => d.totalSeconds));

  return (
    <div className="flex h-full w-full flex-col">
      <div className="shrink-0 border-b border-border bg-card px-4 py-3">
        <h1 className="text-sm font-semibold text-foreground">今日统计</h1>
      </div>

      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto scrollbar-dark p-4">
        {error && (
          <div className="flex items-center gap-1.5 text-[11px] text-destructive">
            <AlertCircle size={12} />
            {error}
          </div>
        )}

        {/* 统计卡片 */}
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
          <StatCard
            label="专注时长"
            value={stats ? formatDuration(stats.totalSeconds) : "-"}
            icon={<Timer size={14} />}
          />
          <StatCard
            label="完成"
            value={stats ? String(stats.completedCount) : "-"}
            icon={<CheckCircle2 size={14} />}
          />
          <StatCard
            label="中断"
            value={stats ? String(stats.interruptedCount) : "-"}
            icon={<X size={14} />}
          />
          <StatCard
            label="番茄数"
            value={stats ? String(stats.workCount) : "-"}
            icon={<Timer size={14} />}
          />
        </div>

        {/* 近 7 天 */}
        <div className="rounded-md border border-border bg-card p-3">
          <div className="mb-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            近 7 天专注（分钟）
          </div>
          {daily.length === 0 ? (
            <div className="flex h-20 items-center justify-center text-[11px] text-muted-foreground">
              暂无数据
            </div>
          ) : (
            <div className="flex h-28 items-end gap-1.5">
              {daily.map((d) => (
                <div key={d.date} className="flex flex-1 flex-col items-center gap-1">
                  <span className="text-[9px] text-muted-foreground">
                    {Math.round(d.totalSeconds / 60)}
                  </span>
                  <div
                    className="w-full rounded-t bg-primary/70 transition-all"
                    style={{
                      height: `${Math.max(3, (d.totalSeconds / maxDaily) * 90)}px`,
                    }}
                    title={`${d.date}: ${formatDuration(d.totalSeconds)}`}
                  />
                  <span className="text-[9px] text-muted-foreground">
                    {d.date.slice(5)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* 今日记录 */}
        <div className="rounded-md border border-border bg-card p-3">
          <div className="mb-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            今日记录
          </div>
          {loading ? (
            <div className="flex h-16 items-center justify-center">
              <Loader2 size={16} className="animate-spin text-muted-foreground" />
            </div>
          ) : records.length === 0 ? (
            <div className="py-4 text-center text-[11px] text-muted-foreground">
              今天还没有番茄记录，开始第一个吧 🍅
            </div>
          ) : (
            <ul className="space-y-1">
              {records.map((r) => (
                <li
                  key={r.id}
                  className="flex items-center gap-2 rounded-md border border-border/60 px-2.5 py-1.5"
                >
                  <Badge variant={r.status === "completed" ? "default" : "secondary"}>
                    {r.status === "completed" ? "完成" : "中断"}
                  </Badge>
                  <span className="text-[11px] text-foreground">
                    {r.type === "work" ? "专注" : r.type === "short_break" ? "短休" : "长休"}
                  </span>
                  <span className="ml-auto text-[11px] font-mono text-muted-foreground">
                    {formatDuration(r.actualDuration)}
                  </span>
                  <span className="text-[10px] text-muted-foreground">
                    {new Date(r.startTime).toLocaleTimeString([], {
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}

// —— 统计卡片 ——
function StatCard({
  label,
  value,
  icon,
}: {
  label: string;
  value: string;
  icon: ReactNode;
}) {
  return (
    <div className="rounded-md border border-border bg-card p-3">
      <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
        <span className="text-primary">{icon}</span>
        {label}
      </div>
      <div className="mt-1.5 font-mono text-xl font-semibold text-foreground">
        {value}
      </div>
    </div>
  );
}

/** 秒 → "1h 23m" 或 "45m" */
function formatDuration(sec: number): string {
  const m = Math.round(sec / 60);
  if (m >= 60) {
    return `${Math.floor(m / 60)}h ${m % 60}m`;
  }
  return `${m}m`;
}
