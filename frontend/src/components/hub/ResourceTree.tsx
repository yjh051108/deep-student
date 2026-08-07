// ResourceTree —— 左侧资源类型树
// ------------------------------------------------------------
// 渲染 10 种资源类型 + "全部"，对照原版侧栏分类
// 选中类型后切换中间列表内容

import { RESOURCE_TYPES, useHubStore, type ResourceType } from "@/state/hub";
import { cn } from "@/lib/utils";
import {
  FileText,
  BookOpen,
  ListChecks,
  Brain,
  Languages,
  Layers,
  FileSearch,
  MessageSquare,
  CheckSquare,
  Sparkles,
  LayoutGrid,
} from "lucide-react";
import { useMemo } from "react";

const TYPE_ICON: Record<string, typeof FileText> = {
  note: FileText,
  textbook: BookOpen,
  qbank: ListChecks,
  mindmap: Brain,
  translation: Languages,
  flashcard: Layers,
  paper: FileSearch,
  chat: MessageSquare,
  todo: CheckSquare,
  skill: Sparkles,
};

export function ResourceTree() {
  const activeType = useHubStore((s) => s.activeType);
  const selectType = useHubStore((s) => s.selectType);
  const entries = useHubStore((s) => s.entries);

  // 统计当前列表数量（仅显示选中类型的数量）
  const countByType = useMemo(() => {
    const m: Record<string, number> = {};
    for (const e of entries) {
      m[e.type] = (m[e.type] ?? 0) + 1;
    }
    return m;
  }, [entries]);

  return (
    <div className="flex h-full w-full flex-col">
      <div className="shrink-0 border-b border-border px-3 py-2.5">
        <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
          资源类型
        </div>
      </div>
      <nav className="min-h-0 flex-1 overflow-y-auto px-2 py-2 scrollbar-dark">
        {/* 全部 */}
        <TreeRow
          label="全部资源"
          icon={LayoutGrid}
          active={activeType === ""}
          count={undefined}
          onClick={() => selectType("")}
        />
        <div className="my-1.5 border-t border-border/60" />
        {/* 各类型 */}
        {RESOURCE_TYPES.map((meta) => {
          const Icon = TYPE_ICON[meta.type] ?? FileText;
          return (
            <TreeRow
              key={meta.type}
              label={meta.label}
              description={meta.description}
              icon={Icon}
              active={activeType === meta.type}
              count={activeType === meta.type ? entries.length : undefined}
              onClick={() => selectType(meta.type as ResourceType)}
            />
          );
        })}
      </nav>
    </div>
  );
}

interface TreeRowProps {
  label: string;
  description?: string;
  icon: typeof FileText;
  active: boolean;
  count?: number;
  onClick: () => void;
}

function TreeRow({ label, description, icon: Icon, active, count, onClick }: TreeRowProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      title={description}
      className={cn(
        "group relative flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left text-sm transition-colors",
        active
          ? "bg-primary/12 text-primary font-medium"
          : "text-muted-foreground hover:bg-accent hover:text-foreground"
      )}
    >
      {active && (
        <span className="absolute left-0 top-1/2 h-4 w-0.5 -translate-y-1/2 rounded-r-full bg-primary" />
      )}
      <Icon
        size={15}
        className={cn(
          "shrink-0 transition-colors",
          active ? "text-primary" : "text-muted-foreground group-hover:text-foreground"
        )}
      />
      <span className="min-w-0 flex-1 truncate">{label}</span>
      {count !== undefined && count > 0 && (
        <span
          className={cn(
            "shrink-0 rounded-full px-1.5 py-0.5 text-[10px] font-medium",
            active
              ? "bg-primary/20 text-primary"
              : "bg-muted text-muted-foreground"
          )}
        >
          {count}
        </span>
      )}
    </button>
  );
}
