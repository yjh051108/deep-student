// ResearchControlBar —— 调研控制条
// ------------------------------------------------------------
// 顶部：主题输入 + 深度选择 + 格式选择 + 引擎多选
// + "生成计划" / "开始调研" / "保存到 Hub" 按钮

import {
  useResearchStore,
  AVAILABLE_ENGINES,
  DEPTH_META,
  FORMAT_META,
  type ResearchDepth,
  type ResearchFormat,
} from "@/state/research";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { cn } from "@/lib/utils";
import {
  Search,
  Loader2,
  AlertCircle,
  Play,
  Save,
  CheckCircle2,
  Sparkles,
  ListChecks,
} from "lucide-react";

export function ResearchControlBar() {
  const topic = useResearchStore((s) => s.topic);
  const depth = useResearchStore((s) => s.depth);
  const format = useResearchStore((s) => s.format);
  const engines = useResearchStore((s) => s.engines);
  const plan = useResearchStore((s) => s.plan);
  const report = useResearchStore((s) => s.report);
  const planning = useResearchStore((s) => s.planning);
  const running = useResearchStore((s) => s.running);
  const saving = useResearchStore((s) => s.saving);
  const savedUri = useResearchStore((s) => s.savedUri);
  const error = useResearchStore((s) => s.error);
  const setTopic = useResearchStore((s) => s.setTopic);
  const setDepth = useResearchStore((s) => s.setDepth);
  const setFormat = useResearchStore((s) => s.setFormat);
  const toggleEngine = useResearchStore((s) => s.toggleEngine);
  const generatePlan = useResearchStore((s) => s.generatePlan);
  const run = useResearchStore((s) => s.run);
  const save = useResearchStore((s) => s.save);

  return (
    <div className="shrink-0 space-y-2 border-b border-border bg-card px-4 py-3">
      {/* 第一行：主题 + 按钮 */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search
            size={14}
            className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground"
          />
          <Input
            value={topic}
            onChange={(e) => setTopic(e.target.value)}
            placeholder="输入调研主题，如：Transformer 架构演进"
            className="h-8 pl-8 text-[13px]"
          />
        </div>
        <Button
          variant="outline"
          size="sm"
          className="h-8"
          onClick={() => void generatePlan()}
          disabled={!topic.trim() || planning || running}
        >
          {planning ? (
            <Loader2 size={12} className="animate-spin" />
          ) : (
            <ListChecks size={12} />
          )}
          {planning ? "规划中…" : "生成计划"}
        </Button>
        <Button
          size="sm"
          className="h-8"
          onClick={() => void run()}
          disabled={
            !topic.trim() ||
            engines.length === 0 ||
            running
          }
        >
          {running ? (
            <Loader2 size={12} className="animate-spin" />
          ) : (
            <Play size={12} />
          )}
          {running ? "调研中…" : "开始调研"}
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="h-8"
          onClick={() => void save()}
          disabled={!report || saving || !!savedUri}
        >
          {saving ? (
            <Loader2 size={12} className="animate-spin" />
          ) : savedUri ? (
            <CheckCircle2 size={12} />
          ) : (
            <Save size={12} />
          )}
          {saving ? "保存中…" : savedUri ? "已保存" : "保存到 Hub"}
        </Button>
      </div>

      {/* 第二行：深度 + 格式 + 引擎 */}
      <div className="flex flex-wrap items-center gap-3">
        {/* 深度 */}
        <div className="flex items-center gap-1">
          <span className="text-[10px] text-muted-foreground/70">深度</span>
          {(Object.keys(DEPTH_META) as ResearchDepth[]).map((d) => (
            <button
              key={d}
              type="button"
              onClick={() => setDepth(d)}
              className={cn(
                "rounded-full border px-2 py-0.5 text-[10px] font-medium transition-colors",
                depth === d
                  ? "border-primary/40 bg-primary/15 text-primary"
                  : "border-border bg-transparent text-muted-foreground hover:bg-accent hover:text-foreground"
              )}
            >
              {DEPTH_META[d]}
            </button>
          ))}
        </div>

        {/* 格式 */}
        <div className="flex items-center gap-1">
          <span className="text-[10px] text-muted-foreground/70">格式</span>
          {(Object.keys(FORMAT_META) as ResearchFormat[]).map((f) => (
            <button
              key={f}
              type="button"
              onClick={() => setFormat(f)}
              className={cn(
                "rounded-full border px-2 py-0.5 text-[10px] font-medium transition-colors",
                format === f
                  ? "border-primary/40 bg-primary/15 text-primary"
                  : "border-border bg-transparent text-muted-foreground hover:bg-accent hover:text-foreground"
              )}
            >
              {FORMAT_META[f]}
            </button>
          ))}
        </div>

        {/* 引擎 */}
        <div className="flex items-center gap-1">
          <span className="text-[10px] text-muted-foreground/70">引擎</span>
          {AVAILABLE_ENGINES.map((e) => {
            const checked = engines.includes(e);
            return (
              <button
                key={e}
                type="button"
                onClick={() => toggleEngine(e)}
                className={cn(
                  "rounded-full border px-2 py-0.5 text-[10px] font-mono transition-colors",
                  checked
                    ? "border-primary/40 bg-primary/15 text-primary"
                    : "border-border bg-transparent text-muted-foreground hover:bg-accent hover:text-foreground"
                )}
              >
                {e}
              </button>
            );
          })}
        </div>

        {/* 计划状态 */}
        {plan && (
          <div className="flex items-center gap-1 text-[10px] text-success">
            <Sparkles size={10} />
            <span>已生成 {plan.steps.length} 步计划</span>
          </div>
        )}
      </div>

      {/* 错误横幅 */}
      {error && (
        <div className="flex items-center gap-2 border border-destructive/30 bg-destructive/10 px-3 py-1.5 text-[11px] text-destructive">
          <AlertCircle size={11} />
          <span className="truncate">{error}</span>
        </div>
      )}

      {/* 已保存提示 */}
      {savedUri && (
        <div className="truncate text-[10px] text-muted-foreground/70">
          已保存到：{savedUri}
        </div>
      )}
    </div>
  );
}
