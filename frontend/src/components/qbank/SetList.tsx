// SetList —— 题集列表 + 抽题入口
// ------------------------------------------------------------
// 顶部：URI 输入 + 标题 + 抽题按钮
// 下方：已保存题集列表（点击切换）

import { useQBankStore, type QBSet } from "@/state/qbank";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { cn } from "@/lib/utils";
import {
  ListChecks,
  Loader2,
  Plus,
  Inbox,
  Sparkles,
  AlertCircle,
} from "lucide-react";
import { useState } from "react";

export function SetList() {
  const sets = useQBankStore((s) => s.sets);
  const activeSetId = useQBankStore((s) => s.activeSetId);
  const extracting = useQBankStore((s) => s.extracting);
  const error = useQBankStore((s) => s.error);
  const extract = useQBankStore((s) => s.extract);
  const selectSet = useQBankStore((s) => s.selectSet);

  const [uri, setUri] = useState("");
  const [title, setTitle] = useState("");

  const handleExtract = async () => {
    if (!uri.trim() || !title.trim() || extracting) return;
    await extract(uri.trim(), title.trim());
    // 抽题成功后清空输入
    setUri("");
    setTitle("");
  };

  return (
    <div className="flex h-full w-full flex-col">
      {/* 头部标题 */}
      <div className="shrink-0 border-b border-border bg-card px-3 py-2.5">
        <div className="flex items-center gap-1.5">
          <ListChecks size={13} className="text-primary" />
          <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            题集
          </span>
        </div>
      </div>

      {/* 抽题入口 */}
      <div className="shrink-0 space-y-2 border-b border-border bg-card px-3 py-3">
        <Input
          value={uri}
          onChange={(e) => setUri(e.target.value)}
          placeholder="源文档 URI，如 vfs://textbook/xxx"
          className="h-8 text-[12px]"
        />
        <Input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="题集标题"
          className="h-8 text-[12px]"
        />
        <Button
          size="sm"
          className="h-8 w-full"
          onClick={handleExtract}
          disabled={!uri.trim() || !title.trim() || extracting}
        >
          {extracting ? (
            <Loader2 size={12} className="animate-spin" />
          ) : (
            <Sparkles size={12} />
          )}
          {extracting ? "抽题中…" : "从文档抽题"}
        </Button>
      </div>

      {/* 错误横幅 */}
      {error && (
        <div className="shrink-0 flex items-center gap-2 border-b border-destructive/30 bg-destructive/10 px-3 py-2 text-[11px] text-destructive">
          <AlertCircle size={11} />
          <span className="truncate">{error}</span>
        </div>
      )}

      {/* 题集列表 */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        {sets.length === 0 ? (
          <EmptyState />
        ) : (
          <ul className="divide-y divide-border/60">
            {sets.map((s) => (
              <SetRow
                key={s.id}
                set={s}
                active={s.id === activeSetId}
                onSelect={() => selectSet(s.id)}
              />
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

interface SetRowProps {
  set: QBSet;
  active: boolean;
  onSelect: () => void;
}

function SetRow({ set, active, onSelect }: SetRowProps) {
  return (
    <li
      onClick={onSelect}
      className={cn(
        "group relative flex cursor-pointer items-start gap-2.5 px-3 py-2.5 transition-colors",
        active ? "bg-primary/10" : "hover:bg-accent/50"
      )}
    >
      {active && (
        <span className="absolute left-0 top-1/2 h-6 w-0.5 -translate-y-1/2 rounded-r-full bg-primary" />
      )}
      <div
        className={cn(
          "flex h-7 w-7 shrink-0 items-center justify-center rounded-md",
          active
            ? "bg-primary/20 text-primary"
            : "bg-muted text-muted-foreground"
        )}
      >
        <ListChecks size={14} />
      </div>
      <div className="min-w-0 flex-1">
        <div
          className={cn(
            "truncate text-[13px] font-medium",
            active ? "text-foreground" : "text-foreground/90"
          )}
          title={set.title}
        >
          {set.title}
        </div>
        <div className="mt-0.5 flex items-center gap-2 text-[10px] text-muted-foreground/70">
          <span>{set.questions.length} 题</span>
          <span>·</span>
          <span className="font-mono truncate">{set.id.slice(0, 12)}</span>
        </div>
      </div>
    </li>
  );
}

function EmptyState() {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
        <Inbox size={20} />
      </div>
      <div className="space-y-1">
        <div className="text-sm font-medium text-foreground">暂无题集</div>
        <div className="text-xs text-muted-foreground">
          在上方输入 URI 与标题后抽题
        </div>
      </div>
    </div>
  );
}

// 保留 Plus 引用以备未来扩展
void Plus;
