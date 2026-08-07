// LLMUsagePage —— LLM 用量统计
// ------------------------------------------------------------
// 布局：
// 1. 顶部：总览卡片（请求数 / Token / 成本 / 今日）
// 2. 中：最近调用日志表（provider / model / caller / token / 状态 / 耗时）
// 3. 下：按日聚合表（日期 / 调用方 / 模型 / 请求数 / token）
//
// 对接后端：llmUsageApi（LLMUsageXxx 方法）

import { useEffect, useMemo } from "react";
import { useLLMUsageStore } from "@/state/llmUsage";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Badge } from "@/components/ui/Badge";
import {
  Activity,
  Cpu,
  Coins,
  CalendarDays,
  Loader2,
  Trash2,
  AlertCircle,
} from "lucide-react";

export function LLMUsagePage() {
  const loadAll = useLLMUsageStore((s) => s.loadAll);
  const loading = useLLMUsageStore((s) => s.loading);
  const error = useLLMUsageStore((s) => s.error);
  const toast = useLLMUsageStore((s) => s.toast);
  const cleanup = useLLMUsageStore((s) => s.cleanup);

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  return (
    <div className="flex h-full w-full min-h-0 flex-col bg-background">
      {/* 头部 */}
      <div className="shrink-0 border-b border-border bg-card px-4 py-3">
        <div className="flex items-center justify-between">
          <h1 className="text-sm font-semibold text-foreground">LLM 用量统计</h1>
          <div className="flex items-center gap-2">
            {toast && <span className="text-[11px] text-emerald-500">{toast}</span>}
            <Button
              size="sm"
              variant="outline"
              className="h-7"
              onClick={() => void cleanup(30)}
            >
              <Trash2 size={12} className="mr-1" />
              清理 30 天前
            </Button>
            <Button size="sm" variant="outline" className="h-7" onClick={() => void loadAll()}>
              <Loader2 size={12} className={loading ? "animate-spin" : ""} />
              刷新
            </Button>
          </div>
        </div>
        {error && (
          <div className="mt-1.5 flex items-center gap-1.5 text-[11px] text-destructive">
            <AlertCircle size={12} />
            {error}
          </div>
        )}
      </div>

      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto scrollbar-dark p-4">
        <SummaryCards />
        <DailyTable />
        <LogTable />
      </div>
    </div>
  );
}

// —— 总览卡片 ——
function SummaryCards() {
  const summary = useLLMUsageStore((s) => s.summary);

  const cards = [
    {
      label: "总请求",
      value: summary ? String(summary.totalRequests) : "-",
      icon: Activity,
    },
    {
      label: "总 Token",
      value: summary ? summary.totalTokens.toLocaleString() : "-",
      icon: Cpu,
    },
    {
      label: "估算成本",
      value: summary ? `$${summary.totalCost.toFixed(4)}` : "-",
      icon: Coins,
    },
    {
      label: "今日 Token",
      value: summary ? summary.todayTokens.toLocaleString() : "-",
      icon: CalendarDays,
    },
  ];

  return (
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
      {cards.map((c) => (
        <div key={c.label} className="rounded-md border border-border bg-card p-3">
          <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            <c.icon size={12} className="text-primary" />
            {c.label}
          </div>
          <div className="mt-1.5 font-mono text-lg font-semibold text-foreground">
            {c.value}
          </div>
        </div>
      ))}
    </div>
  );
}

// —— 按日聚合表 ——
function DailyTable() {
  const daily = useLLMUsageStore((s) => s.daily);

  return (
    <div className="rounded-md border border-border bg-card">
      <div className="border-b border-border px-3 py-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
        按日聚合
      </div>
      {daily.length === 0 ? (
        <div className="py-6 text-center text-[11px] text-muted-foreground">
          暂无数据
        </div>
      ) : (
        <div className="max-h-56 overflow-y-auto scrollbar-dark">
          <table className="w-full text-left text-[11px]">
            <thead className="sticky top-0 bg-card">
              <tr className="text-muted-foreground/70">
                <th className="px-3 py-1.5 font-medium">日期</th>
                <th className="px-2 py-1.5 font-medium">调用方</th>
                <th className="px-2 py-1.5 font-medium">模型</th>
                <th className="px-2 py-1.5 text-right font-medium">请求</th>
                <th className="px-2 py-1.5 text-right font-medium">成功</th>
                <th className="px-2 py-1.5 text-right font-medium">Token</th>
              </tr>
            </thead>
            <tbody>
              {daily.slice(0, 50).map((d) => (
                <tr key={`${d.date}-${d.callerType}-${d.model}`} className="border-t border-border/40">
                  <td className="px-3 py-1.5 font-mono text-muted-foreground">{d.date}</td>
                  <td className="px-2 py-1.5">
                    <Badge variant="secondary">{d.callerType}</Badge>
                  </td>
                  <td className="px-2 py-1.5 font-mono">{d.model}</td>
                  <td className="px-2 py-1.5 text-right font-mono">{d.requestCount}</td>
                  <td className="px-2 py-1.5 text-right font-mono text-emerald-500">
                    {d.successCount}
                  </td>
                  <td className="px-2 py-1.5 text-right font-mono">
                    {d.totalTokens.toLocaleString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// —— 最近日志表 ——
function LogTable() {
  const logs = useLLMUsageStore((s) => s.logs);
  const filter = useLLMUsageStore((s) => s.filter);
  const setFilter = useLLMUsageStore((s) => s.setFilter);
  const loadLogs = useLLMUsageStore((s) => s.loadLogs);

  const callerTypes = useMemo(
    () => Array.from(new Set(logs.map((l) => l.callerType))).sort(),
    [logs]
  );

  return (
    <div className="rounded-md border border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-3 py-2">
        <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
          最近调用
        </span>
        <div className="flex items-center gap-2">
          <Input
            value={filter.callerType ?? ""}
            onChange={(e) => setFilter({ callerType: e.target.value || undefined })}
            placeholder="按调用方筛选…"
            className="h-6 w-32 text-[10px]"
          />
          <Button
            size="sm"
            variant="ghost"
            className="h-6"
            onClick={() => void loadLogs()}
          >
            应用
          </Button>
        </div>
      </div>
      {callerTypes.length > 0 && (
        <div className="flex flex-wrap gap-1 border-b border-border/40 px-3 py-1.5">
          {callerTypes.map((c) => (
            <button
              key={c}
              onClick={() => setFilter({ callerType: c })}
              className="rounded bg-muted px-1.5 py-0.5 text-[9px] text-muted-foreground hover:bg-primary/15 hover:text-primary"
            >
              {c}
            </button>
          ))}
        </div>
      )}
      {logs.length === 0 ? (
        <div className="py-6 text-center text-[11px] text-muted-foreground">
          暂无调用记录
        </div>
      ) : (
        <div className="max-h-80 overflow-y-auto scrollbar-dark">
          <table className="w-full text-left text-[11px]">
            <thead className="sticky top-0 bg-card">
              <tr className="text-muted-foreground/70">
                <th className="px-3 py-1.5 font-medium">时间</th>
                <th className="px-2 py-1.5 font-medium">Provider</th>
                <th className="px-2 py-1.5 font-medium">模型</th>
                <th className="px-2 py-1.5 font-medium">调用方</th>
                <th className="px-2 py-1.5 text-right font-medium">Prompt</th>
                <th className="px-2 py-1.5 text-right font-medium">完成</th>
                <th className="px-2 py-1.5 text-right font-medium">Total</th>
                <th className="px-2 py-1.5 font-medium">状态</th>
                <th className="px-2 py-1.5 text-right font-medium">耗时</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((l) => (
                <tr key={l.id} className="border-t border-border/40">
                  <td className="whitespace-nowrap px-3 py-1.5 font-mono text-muted-foreground">
                    {new Date(l.timestamp).toLocaleTimeString([], {
                      hour: "2-digit",
                      minute: "2-digit",
                      second: "2-digit",
                    })}
                  </td>
                  <td className="px-2 py-1.5">{l.provider}</td>
                  <td className="px-2 py-1.5 font-mono">{l.model}</td>
                  <td className="px-2 py-1.5">
                    <Badge variant="secondary">{l.callerType}</Badge>
                  </td>
                  <td className="px-2 py-1.5 text-right font-mono">{l.promptTokens}</td>
                  <td className="px-2 py-1.5 text-right font-mono">{l.completionTokens}</td>
                  <td className="px-2 py-1.5 text-right font-mono">{l.totalTokens}</td>
                  <td className="px-2 py-1.5">
                    <Badge
                      variant={l.status === "success" ? "default" : "destructive"}
                    >
                      {l.status}
                    </Badge>
                  </td>
                  <td className="px-2 py-1.5 text-right font-mono text-muted-foreground">
                    {l.durationMs != null ? `${l.durationMs}ms` : "-"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
