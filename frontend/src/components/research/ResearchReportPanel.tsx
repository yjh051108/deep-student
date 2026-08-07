// ResearchReportPanel —— 调研报告主体
// ------------------------------------------------------------
// 三阶段展示：
// 1. 计划阶段：显示 Plan 步骤列表（左侧）+ 提示确认开始调研
// 2. 调研中：显示进度提示
// 3. 报告阶段：左侧章节列表 + 右侧章节内容 + 底部来源列表

import { useResearchStore } from "@/state/research";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { cn } from "@/lib/utils";
import {
  Loader2,
  Inbox,
  ListTree,
  FileText,
  ExternalLink,
  Search as SearchIcon,
  Play,
  Sparkles,
} from "lucide-react";

export function ResearchReportPanel() {
  const plan = useResearchStore((s) => s.plan);
  const report = useResearchStore((s) => s.report);
  const running = useResearchStore((s) => s.running);
  const planning = useResearchStore((s) => s.planning);
  const run = useResearchStore((s) => s.run);

  // 调研中：显示进度
  if (running) {
    return <LoadingState />;
  }

  // 有报告：显示报告
  if (report) {
    return <ReportView />;
  }

  // 有计划：显示计划步骤
  if (plan) {
    return (
      <div className="flex h-full w-full flex-col">
        <div className="shrink-0 border-b border-border bg-card px-4 py-2.5">
          <div className="flex items-center gap-1.5">
            <ListTree size={13} className="text-primary" />
            <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              调研计划（{plan.steps.length} 步）
            </span>
          </div>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
          <div className="space-y-3 px-6 py-5">
            <ol className="space-y-2">
              {plan.steps.map((step, idx) => (
                <li
                  key={idx}
                  className="flex items-start gap-3 rounded-md border border-border bg-background px-4 py-2.5"
                >
                  <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary/15 text-[11px] font-semibold text-primary">
                    {idx + 1}
                  </span>
                  <span className="flex-1 whitespace-pre-wrap break-words text-[13px] leading-relaxed text-foreground/90">
                    {step}
                  </span>
                </li>
              ))}
            </ol>
            <div className="flex items-center justify-center gap-2 pt-2">
              <Button size="sm" className="h-8" onClick={() => void run()}>
                <Play size={12} />
                确认并开始调研
              </Button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // 空状态
  return (
    <EmptyState planning={planning} />
  );
}

/** 报告视图：章节列表 + 内容 + 来源 */
function ReportView() {
  const report = useResearchStore((s) => s.report);
  const activeSectionIdx = useResearchStore((s) => s.activeSectionIdx);
  const selectSection = useResearchStore((s) => s.selectSection);

  if (!report) return null;

  const sections = report.sections ?? [];
  const sources = report.sources ?? [];
  const active = sections[activeSectionIdx];

  return (
    <div className="flex h-full w-full min-h-0">
      {/* 左：章节列表 */}
      <aside className="w-56 shrink-0 border-r border-border bg-card">
        <div className="shrink-0 border-b border-border px-3 py-2.5">
          <div className="flex items-center gap-1.5">
            <ListTree size={13} className="text-primary" />
            <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              章节（{sections.length}）
            </span>
          </div>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
          {sections.length === 0 ? (
            <div className="px-3 py-4 text-[11px] text-muted-foreground">
              无章节
            </div>
          ) : (
            <ul className="divide-y divide-border/60">
              {sections.map((sec, idx) => (
                <li
                  key={idx}
                  onClick={() => selectSection(idx)}
                  className={cn(
                    "cursor-pointer px-3 py-2 text-[12px] transition-colors",
                    idx === activeSectionIdx
                      ? "bg-primary/10 text-foreground"
                      : "text-foreground/80 hover:bg-accent/50"
                  )}
                >
                  <div className="flex items-center gap-2">
                    <span
                      className={cn(
                        "flex h-5 w-5 shrink-0 items-center justify-center rounded text-[10px] font-medium",
                        idx === activeSectionIdx
                          ? "bg-primary text-primary-foreground"
                          : "bg-muted text-muted-foreground"
                      )}
                    >
                      {idx + 1}
                    </span>
                    <span className="min-w-0 flex-1 truncate">{sec.title}</span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </aside>

      {/* 右：章节内容 + 来源 */}
      <section className="flex min-w-0 flex-1 flex-col">
        {/* 章节内容 */}
        <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
          {active ? (
            <div className="space-y-3 px-6 py-5 animate-fade-in" key={activeSectionIdx}>
              <div className="flex items-center gap-2">
                <FileText size={14} className="text-primary" />
                <h3 className="text-base font-semibold text-foreground">
                  {active.title}
                </h3>
              </div>
              <div className="rounded-md border border-border bg-background px-4 py-3">
                <pre className="whitespace-pre-wrap break-words font-sans text-[13px] leading-relaxed text-foreground/90">
                  {active.content}
                </pre>
              </div>
            </div>
          ) : (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
              无章节内容
            </div>
          )}
        </div>

        {/* 底部：来源列表 */}
        <div className="shrink-0 max-h-56 overflow-y-auto border-t border-border bg-card scrollbar-dark">
          <div className="sticky top-0 bg-card px-4 py-2">
            <div className="flex items-center gap-1.5">
              <SearchIcon size={11} className="text-muted-foreground/70" />
              <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                来源（{sources.length}）
              </span>
            </div>
          </div>
          {sources.length === 0 ? (
            <div className="px-4 py-2 text-[11px] text-muted-foreground">
              无来源
            </div>
          ) : (
            <ul className="divide-y divide-border/60">
              {sources.map((src, idx) => (
                <li
                  key={idx}
                  className="px-4 py-2 text-[11px] hover:bg-accent/40"
                >
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="text-[9px]">
                      {src.engine}
                    </Badge>
                    <a
                      href={src.url}
                      target="_blank"
                      rel="noreferrer"
                      className="flex min-w-0 flex-1 items-center gap-1 truncate text-primary hover:underline"
                      title={src.url}
                    >
                      <span className="truncate font-medium">{src.title}</span>
                      <ExternalLink size={9} className="shrink-0" />
                    </a>
                  </div>
                  {src.snippet && (
                    <div className="mt-0.5 line-clamp-2 text-[10px] leading-relaxed text-muted-foreground/80">
                      {src.snippet}
                    </div>
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
      </section>
    </div>
  );
}

/** 调研中状态 */
function LoadingState() {
  return (
    <div className="flex h-full w-full flex-col items-center justify-center gap-4 px-6 text-center">
      <div className="relative flex h-16 w-16 items-center justify-center">
        <div className="absolute inset-0 rounded-full border-2 border-primary/20" />
        <div className="absolute inset-0 animate-spin rounded-full border-2 border-primary/60 border-t-transparent" />
        <Loader2 size={20} className="text-primary" />
      </div>
      <div className="space-y-1">
        <div className="flex items-center justify-center gap-1.5 text-sm font-medium text-foreground">
          <Sparkles size={13} className="text-primary" />
          正在执行多引擎调研…
        </div>
        <div className="text-xs text-muted-foreground">
          这可能需要数十秒到数分钟，请耐心等待
        </div>
      </div>
    </div>
  );
}

/** 空状态 */
function EmptyState({ planning }: { planning: boolean }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
        {planning ? (
          <Loader2 size={20} className="animate-spin" />
        ) : (
          <Inbox size={20} />
        )}
      </div>
      <div className="space-y-1">
        <div className="text-sm font-medium text-foreground">
          {planning ? "正在生成计划…" : "尚未开始调研"}
        </div>
        <div className="text-xs text-muted-foreground">
          {planning
            ? "正在调用 LLM 规划调研步骤"
            : `输入主题后点击"生成计划"或"开始调研"`}
        </div>
      </div>
    </div>
  );
}
