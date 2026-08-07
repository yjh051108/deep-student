// ProgressPanel —— 答题进度 + 提交评分 + AI 解析
// ------------------------------------------------------------
// 顶部：掌握度雷达（知识点 -> 分数）
// 中部：答题进度条 + 提交按钮
// 下部：当前题目的 AI 解析

import { useQBankStore } from "@/state/qbank";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { cn } from "@/lib/utils";
import {
  Trophy,
  Sparkles,
  Loader2,
  Send,
  TrendingUp,
  Lightbulb,
  Inbox,
} from "lucide-react";

export function ProgressPanel() {
  const activeSet = useQBankStore((s) => s.activeSet);
  const attempt = useQBankStore((s) => s.attempt);
  const draftAnswers = useQBankStore((s) => s.draftAnswers);
  const mastery = useQBankStore((s) => s.mastery);
  const currentIndex = useQBankStore((s) => s.currentIndex);
  const analysisMap = useQBankStore((s) => s.analysisMap);
  const analyzingQid = useQBankStore((s) => s.analyzingQid);
  const submitting = useQBankStore((s) => s.submitting);
  const submit = useQBankStore((s) => s.submit);
  const analyze = useQBankStore((s) => s.analyze);

  if (!activeSet) {
    return <EmptyState />;
  }

  const questions = activeSet.questions ?? [];
  const current = questions[currentIndex];
  const answeredCount = questions.filter(
    (q) => !!draftAnswers[q.id]?.trim()
  ).length;
  const submitted = Boolean(attempt && attempt.finished_at != null);
  const totalScore = attempt?.total ?? 0;
  const gotScore = attempt?.score ?? 0;

  return (
    <div className="flex h-full w-full flex-col">
      {/* 头部 */}
      <div className="shrink-0 border-b border-border bg-card px-4 py-2.5">
        <div className="flex items-center gap-1.5">
          <TrendingUp size={13} className="text-primary" />
          <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            进度与解析
          </span>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        <div className="space-y-4 px-4 py-3">
          {/* 掌握度 */}
          <Section title="知识点掌握度" icon={TrendingUp}>
            {Object.keys(mastery).length === 0 ? (
              <div className="rounded-md border border-dashed border-border bg-muted/30 px-3 py-2 text-[11px] text-muted-foreground">
                暂无掌握度数据
              </div>
            ) : (
              <div className="space-y-1.5">
                {Object.entries(mastery).map(([k, v]) => (
                  <MasteryRow key={k} knowledge={k} value={v} />
                ))}
              </div>
            )}
          </Section>

          {/* 答题进度 */}
          <Section title="答题进度" icon={Trophy}>
            <div className="space-y-2">
              <div className="flex items-center justify-between text-[11px]">
                <span className="text-muted-foreground">
                  已答 {answeredCount} / {questions.length}
                </span>
                {submitted && (
                  <span className="font-semibold text-primary">
                    得分 {gotScore} / {totalScore}
                  </span>
                )}
              </div>
              <div className="h-2 overflow-hidden rounded-full bg-muted">
                <div
                  className="h-full rounded-full bg-primary transition-[width] duration-300"
                  style={{
                    width: `${questions.length === 0 ? 0 : (answeredCount / questions.length) * 100}%`,
                  }}
                />
              </div>
              {/* 提交按钮 */}
              <Button
                size="sm"
                className="h-8 w-full"
                disabled={!attempt || submitting || submitted}
                onClick={() => void submit()}
              >
                {submitting ? (
                  <Loader2 size={12} className="animate-spin" />
                ) : (
                  <Send size={12} />
                )}
                {submitted
                  ? `已交卷（${gotScore} / ${totalScore}）`
                  : submitting
                    ? "提交中…"
                    : "提交评分"}
              </Button>
            </div>
          </Section>

          {/* 当前题目解析 */}
          {current && (
            <Section title="本题 AI 解析" icon={Lightbulb}>
              <div className="space-y-2">
                <Button
                  variant="outline"
                  size="sm"
                  className="h-8 w-full"
                  disabled={analyzingQid === current.id}
                  onClick={() => void analyze(current.id)}
                >
                  {analyzingQid === current.id ? (
                    <Loader2 size={12} className="animate-spin" />
                  ) : (
                    <Sparkles size={12} />
                  )}
                  {analyzingQid === current.id ? "解析中…" : "生成解析"}
                </Button>
                {analysisMap[current.id] && (
                  <div className="rounded-md border border-primary/30 bg-primary/5 p-3">
                    <div className="mb-1.5 flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wider text-primary">
                      <Sparkles size={10} />
                      解析结果
                    </div>
                    <pre className="whitespace-pre-wrap break-words font-sans text-[12px] leading-relaxed text-foreground/90">
                      {analysisMap[current.id]}
                    </pre>
                  </div>
                )}
              </div>
            </Section>
          )}

          {/* 提交后：参考答案 */}
          {submitted && current && (
            <Section title="参考答案" icon={Trophy}>
              <div className="rounded-md border border-border bg-background px-3 py-2">
                <pre className="whitespace-pre-wrap break-words font-mono text-[12px] leading-relaxed text-foreground/90">
                  {current.answer}
                </pre>
              </div>
            </Section>
          )}
        </div>
      </div>
    </div>
  );
}

/** 小节容器 */
function Section({
  title,
  icon: Icon,
  children,
}: {
  title: string;
  icon: typeof Trophy;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
        <Icon size={11} />
        {title}
      </div>
      {children}
    </div>
  );
}

/** 掌握度行 */
function MasteryRow({ knowledge, value }: { knowledge: string; value: number }) {
  // value 期望 0-100，归一化到 0-1
  const pct = Math.max(0, Math.min(100, value));
  const tone =
    pct >= 80
      ? "text-success"
      : pct >= 50
        ? "text-warning"
        : "text-destructive";
  return (
    <div className="flex items-center gap-2">
      <span className="min-w-0 flex-1 truncate font-mono text-[11px] text-foreground/80">
        {knowledge}
      </span>
      <div className="h-1.5 w-20 overflow-hidden rounded-full bg-muted">
        <div
          className={cn(
            "h-full rounded-full",
            pct >= 80
              ? "bg-success"
              : pct >= 50
                ? "bg-warning"
                : "bg-destructive"
          )}
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className={cn("w-9 text-right text-[11px] font-semibold", tone)}>
        {pct.toFixed(0)}
      </span>
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
        <Inbox size={20} />
      </div>
      <div className="space-y-1">
        <div className="text-sm font-medium text-foreground">未选中题集</div>
        <div className="text-xs text-muted-foreground">
          选中题集后可查看进度与解析
        </div>
      </div>
    </div>
  );
}

// 保留 Badge 引用以备未来扩展
void Badge;
