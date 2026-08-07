// EssayInput —— 作文输入区
// ------------------------------------------------------------
// 大文本框 + 场景选择 + 维度复选 + 批改按钮

import {
  useEssayStore,
  SCENARIO_META,
  DIM_META,
  type EssayScenario,
  type EssayDim,
} from "@/state/essay";
import { Button } from "@/components/ui/Button";
import { Textarea } from "@/components/ui/Textarea";
import { Checkbox } from "@/components/ui/Checkbox";
import { cn } from "@/lib/utils";
import {
  PenLine,
  Loader2,
  AlertCircle,
  Eraser,
  CheckSquare,
} from "lucide-react";

export function EssayInput() {
  const text = useEssayStore((s) => s.text);
  const scenario = useEssayStore((s) => s.scenario);
  const dims = useEssayStore((s) => s.dims);
  const grading = useEssayStore((s) => s.grading);
  const error = useEssayStore((s) => s.error);
  const setText = useEssayStore((s) => s.setText);
  const setScenario = useEssayStore((s) => s.setScenario);
  const toggleDim = useEssayStore((s) => s.toggleDim);
  const grade = useEssayStore((s) => s.grade);
  const clear = useEssayStore((s) => s.clear);

  const wordCount = text.trim() ? text.trim().split(/\s+/).length : 0;
  const charCount = text.length;

  return (
    <div className="flex h-full w-full flex-col">
      {/* 头部 */}
      <div className="shrink-0 border-b border-border bg-card px-4 py-2.5">
        <div className="flex items-center gap-1.5">
          <PenLine size={13} className="text-primary" />
          <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            作文输入
          </span>
        </div>
      </div>

      {/* 错误横幅 */}
      {error && (
        <div className="shrink-0 flex items-center gap-2 border-b border-destructive/30 bg-destructive/10 px-4 py-2 text-[11px] text-destructive">
          <AlertCircle size={11} />
          <span className="truncate">{error}</span>
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        <div className="space-y-3 px-4 py-3">
          {/* 场景选择 */}
          <div>
            <label className="mb-1.5 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              场景
            </label>
            <div className="flex flex-wrap gap-1">
              {(Object.keys(SCENARIO_META) as EssayScenario[]).map((sc) => (
                <button
                  key={sc}
                  type="button"
                  onClick={() => setScenario(sc)}
                  className={cn(
                    "rounded-full border px-2.5 py-0.5 text-[11px] font-medium transition-colors",
                    scenario === sc
                      ? "border-primary/40 bg-primary/15 text-primary"
                      : "border-border bg-transparent text-muted-foreground hover:bg-accent hover:text-foreground"
                  )}
                >
                  {SCENARIO_META[sc]}
                </button>
              ))}
            </div>
          </div>

          {/* 评分维度 */}
          <div>
            <div className="mb-1.5 flex items-center gap-1.5">
              <label className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                评分维度
              </label>
              <CheckSquare size={10} className="text-muted-foreground/60" />
            </div>
            <div className="flex flex-wrap gap-2">
              {(Object.keys(DIM_META) as EssayDim[]).map((d) => {
                const checked = dims.includes(d);
                return (
                  <label
                    key={d}
                    className={cn(
                      "flex cursor-pointer items-center gap-1.5 rounded-md border px-2.5 py-1 text-[11px] transition-colors",
                      checked
                        ? "border-primary/40 bg-primary/10 text-foreground"
                        : "border-border bg-transparent text-muted-foreground hover:bg-accent"
                    )}
                  >
                    <Checkbox
                      checked={checked}
                      onCheckedChange={() => toggleDim(d)}
                    />
                    {DIM_META[d]}
                  </label>
                );
              })}
            </div>
          </div>

          {/* 文本输入 */}
          <div>
            <div className="mb-1.5 flex items-center justify-between">
              <label className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                作文内容
              </label>
              <span className="text-[10px] text-muted-foreground/60">
                {wordCount} 词 · {charCount} 字符
              </span>
            </div>
            <Textarea
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder="在此粘贴或输入你的作文…"
              rows={14}
              className="resize-none text-[13px] leading-relaxed"
            />
          </div>

          {/* 操作按钮 */}
          <div className="flex items-center gap-2">
            <Button
              size="sm"
              className="h-8 flex-1"
              onClick={() => void grade()}
              disabled={!text.trim() || dims.length === 0 || grading}
            >
              {grading ? (
                <Loader2 size={12} className="animate-spin" />
              ) : (
                <PenLine size={12} />
              )}
              {grading ? "批改中…" : "批改"}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="h-8"
              onClick={clear}
              disabled={grading || (!text && !useEssayStore.getState().result)}
              title="清空"
            >
              <Eraser size={12} />
              清空
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
