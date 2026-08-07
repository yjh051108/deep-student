// EssayResultPanel —— 作文批改结果展示
// ------------------------------------------------------------
// 顶部：总分（大字号 + 圆环）
// 中部：维度雷达/列表
// 下部：润色对比（原文 vs 润色）
// 底部：建议 + 亮点 + 保存按钮

import { useEssayStore, type Dimension, type EssayResult } from "@/state/essay";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { cn } from "@/lib/utils";
import {
  Award,
  Save,
  Loader2,
  Inbox,
  Sparkles,
  Lightbulb,
  Star,
  Copy,
  CheckCircle2,
  FileText,
} from "lucide-react";

export function EssayResultPanel() {
  const result = useEssayStore((s) => s.result);
  const history = useEssayStore((s) => s.history);
  const saving = useEssayStore((s) => s.saving);
  const savedUri = useEssayStore((s) => s.savedUri);
  const save = useEssayStore((s) => s.save);

  if (!result) {
    return <EmptyState />;
  }

  return (
    <div className="flex h-full w-full flex-col">
      {/* 头部 */}
      <div className="shrink-0 border-b border-border bg-card px-4 py-2.5">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1.5">
            <Award size={13} className="text-primary" />
            <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              批改结果
            </span>
          </div>
          <Button
            variant="outline"
            size="sm"
            className="h-7"
            onClick={() => void save()}
            disabled={saving || !!savedUri}
          >
            {saving ? (
              <Loader2 size={11} className="animate-spin" />
            ) : savedUri ? (
              <CheckCircle2 size={11} />
            ) : (
              <Save size={11} />
            )}
            {saving ? "保存中…" : savedUri ? "已保存" : "保存到 Hub"}
          </Button>
        </div>
        {savedUri && (
          <div className="mt-1 truncate text-[10px] text-muted-foreground/70">
            URI：{savedUri}
          </div>
        )}
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        <div className="space-y-4 px-4 py-3">
          {/* 总分圆环 */}
          <TotalScore result={result} />

          {/* 维度列表 */}
          <Section title="维度评分" icon={Sparkles}>
            {result.dimensions.length === 0 ? (
              <EmptyHint text="无维度数据" />
            ) : (
              <div className="space-y-2">
                {result.dimensions.map((d) => (
                  <DimensionRow key={d.name} dim={d} />
                ))}
              </div>
            )}
          </Section>

          {/* 润色对比 */}
          <Section title="润色对比" icon={FileText}>
            <div className="grid grid-cols-2 gap-2">
              <div className="rounded-md border border-border bg-background p-2">
                <div className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                  原文
                </div>
                <pre className="max-h-48 overflow-y-auto whitespace-pre-wrap break-words font-sans text-[11px] leading-relaxed text-foreground/80 scrollbar-dark">
                  {result.original || "（无）"}
                </pre>
              </div>
              <div className="rounded-md border border-primary/30 bg-primary/5 p-2">
                <div className="mb-1 flex items-center justify-between">
                  <div className="text-[10px] font-semibold uppercase tracking-wider text-primary">
                    润色
                  </div>
                  <button
                    type="button"
                    onClick={() =>
                      void navigator.clipboard.writeText(result.polished)
                    }
                    className="rounded p-0.5 text-muted-foreground hover:bg-accent hover:text-foreground"
                    title="复制润色"
                  >
                    <Copy size={10} />
                  </button>
                </div>
                <pre className="max-h-48 overflow-y-auto whitespace-pre-wrap break-words font-sans text-[11px] leading-relaxed text-foreground/90 scrollbar-dark">
                  {result.polished || "（无）"}
                </pre>
              </div>
            </div>
          </Section>

          {/* 建议 */}
          <Section title="改进建议" icon={Lightbulb}>
            {result.suggestions.length === 0 ? (
              <EmptyHint text="无建议" />
            ) : (
              <ul className="space-y-1.5">
                {result.suggestions.map((s, idx) => (
                  <li
                    key={idx}
                    className="flex items-start gap-2 text-[12px] leading-relaxed text-foreground/90"
                  >
                    <span className="mt-1 h-1 w-1 shrink-0 rounded-full bg-primary" />
                    <span className="flex-1 whitespace-pre-wrap break-words">
                      {s}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </Section>

          {/* 亮点 */}
          <Section title="亮点" icon={Star}>
            {result.highlights.length === 0 ? (
              <EmptyHint text="无亮点" />
            ) : (
              <ul className="space-y-1.5">
                {result.highlights.map((h, idx) => (
                  <li
                    key={idx}
                    className="flex items-start gap-2 text-[12px] leading-relaxed text-foreground/90"
                  >
                    <Star
                      size={10}
                      className="mt-1 shrink-0 text-warning"
                    />
                    <span className="flex-1 whitespace-pre-wrap break-words">
                      {h}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </Section>

          {/* 历史结果 */}
          {history.length > 1 && (
            <Section title="历史记录" icon={Inbox}>
              <div className="space-y-1.5">
                {history.map((h, idx) => (
                  <HistoryCard key={h.id + idx} result={h} active={h.id === result.id} />
                ))}
              </div>
            </Section>
          )}
        </div>
      </div>
    </div>
  );
}

/** 总分圆环 */
function TotalScore({ result }: { result: EssayResult }) {
  // 假设 total 满分 100，按比例展示
  const pct = Math.max(0, Math.min(100, result.total));
  const tone =
    pct >= 80
      ? "text-success"
      : pct >= 60
        ? "text-warning"
        : "text-destructive";
  const r = 36;
  const circ = 2 * Math.PI * r;
  const offset = circ * (1 - pct / 100);
  return (
    <div className="flex items-center gap-4 rounded-md border border-border bg-background px-4 py-3">
      <div className="relative h-20 w-20 shrink-0">
        <svg className="h-full w-full -rotate-90" viewBox="0 0 96 96">
          <circle
            cx="48"
            cy="48"
            r={r}
            fill="none"
            stroke="hsl(var(--muted))"
            strokeWidth="6"
          />
          <circle
            cx="48"
            cy="48"
            r={r}
            fill="none"
            stroke="hsl(var(--primary))"
            strokeWidth="6"
            strokeDasharray={circ}
            strokeDashoffset={offset}
            strokeLinecap="round"
            className="transition-[stroke-dashoffset] duration-700 ease-out"
          />
        </svg>
        <div className="absolute inset-0 flex items-center justify-center">
          <span className={cn("text-xl font-bold", tone)}>
            {pct.toFixed(0)}
          </span>
        </div>
      </div>
      <div className="min-w-0 flex-1">
        <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
          总分
        </div>
        <div className="mt-0.5 text-sm font-semibold text-foreground">
          {result.total.toFixed(1)} / 100
        </div>
        <Badge variant="outline" className="mt-1 text-[10px]">
          {result.scenario}
        </Badge>
      </div>
    </div>
  );
}

/** 维度行 */
function DimensionRow({ dim }: { dim: Dimension }) {
  const pct = Math.max(0, Math.min(100, dim.score));
  const tone =
    pct >= 80
      ? "text-success"
      : pct >= 60
        ? "text-warning"
        : "text-destructive";
  return (
    <div className="rounded-md border border-border bg-background px-3 py-2">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-1.5">
          <span className="text-[12px] font-medium text-foreground">
            {dim.name}
          </span>
          <span className="rounded bg-muted px-1 py-0.5 text-[9px] font-mono text-muted-foreground">
            权重 {(dim.weight * 100).toFixed(0)}%
          </span>
        </div>
        <span className={cn("text-[12px] font-semibold", tone)}>
          {dim.score.toFixed(1)}
        </span>
      </div>
      <div className="mt-1.5 h-1.5 overflow-hidden rounded-full bg-muted">
        <div
          className={cn(
            "h-full rounded-full transition-[width] duration-500",
            pct >= 80
              ? "bg-success"
              : pct >= 60
                ? "bg-warning"
                : "bg-destructive"
          )}
          style={{ width: `${pct}%` }}
        />
      </div>
      {dim.note && (
        <div className="mt-1.5 text-[11px] leading-relaxed text-muted-foreground">
          {dim.note}
        </div>
      )}
    </div>
  );
}

/** 历史卡片 */
function HistoryCard({ result, active }: { result: EssayResult; active: boolean }) {
  return (
    <div
      className={cn(
        "rounded-md border px-2.5 py-1.5 text-[11px] transition-colors",
        active
          ? "border-primary/40 bg-primary/5"
          : "border-border bg-background"
      )}
    >
      <div className="flex items-center justify-between">
        <span className="font-mono text-muted-foreground">{result.id.slice(0, 12)}</span>
        <span className="font-semibold text-foreground">
          {result.total.toFixed(1)}
        </span>
      </div>
      <div className="mt-0.5 truncate text-muted-foreground/70">
        {result.scenario}
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
  icon: typeof Award;
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

function EmptyHint({ text }: { text: string }) {
  return (
    <div className="rounded-md border border-dashed border-border bg-muted/30 px-3 py-2 text-[11px] text-muted-foreground">
      {text}
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
        <div className="text-sm font-medium text-foreground">尚未批改</div>
        <div className="text-xs text-muted-foreground">
          在左侧输入作文并点击"批改"
        </div>
      </div>
    </div>
  );
}
