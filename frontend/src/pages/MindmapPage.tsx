// MindmapPage —— 思维导图
// ------------------------------------------------------------
// 布局：
// 1. 顶部工具条：主题输入 + 生成按钮 + 视图切换（导图/大纲）
// 2. 主体：左侧导图/大纲渲染 + 右侧操作面板（AI 编辑 / 蒙版 / 保存）
//
// 对接后端：MindmapGenerate / MindmapEdit / MindmapToOutline
//           MindmapMask / MindmapSave / MindmapFromOutline

import { useMindmapStore, type MindmapNode } from "@/state/mindmap";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { cn } from "@/lib/utils";
import {
  Brain,
  Sparkles,
  Loader2,
  AlertCircle,
  Save,
  EyeOff,
  ListTree,
  Network,
  FileText,
  Inbox,
  CheckCircle2,
} from "lucide-react";

export function MindmapPage() {
  const topic = useMindmapStore((s) => s.topic);
  const setTopic = useMindmapStore((s) => s.setTopic);
  const map = useMindmapStore((s) => s.map);
  const viewMode = useMindmapStore((s) => s.viewMode);
  const setViewMode = useMindmapStore((s) => s.setViewMode);
  const loading = useMindmapStore((s) => s.loading);
  const saving = useMindmapStore((s) => s.saving);
  const error = useMindmapStore((s) => s.error);
  const savedUri = useMindmapStore((s) => s.savedUri);
  const generate = useMindmapStore((s) => s.generate);
  const save = useMindmapStore((s) => s.save);

  return (
    <div className="flex h-full w-full min-h-0 flex-col bg-background">
      {/* —— 顶部工具条 —— */}
      <div className="shrink-0 space-y-2 border-b border-[var(--shell-seam)] bg-[var(--shell-inspector-panel)] px-4 py-3">
        <div className="flex items-center gap-2">
          <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-primary/15 text-primary">
            <Brain size={15} />
          </div>
          <h1 className="shrink-0 text-sm font-semibold text-foreground">
            思维导图
          </h1>
          <div className="flex flex-1 items-center gap-2">
            <Input
              value={topic}
              onChange={(e) => setTopic(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") void generate();
              }}
              placeholder="输入主题，例如：光合作用、贝叶斯推断…"
              className="h-8 text-[13px]"
            />
            <Button
              size="sm"
              className="h-8"
              disabled={loading || !topic.trim()}
              onClick={() => void generate()}
            >
              {loading ? (
                <Loader2 size={13} className="animate-spin" />
              ) : (
                <Sparkles size={13} />
              )}
              {loading ? "生成中…" : "生成导图"}
            </Button>
          </div>
          {/* 视图切换 */}
          <div className="flex items-center gap-0.5 rounded-md bg-muted p-0.5">
            <ViewToggle
              active={viewMode === "map"}
              onClick={() => setViewMode("map")}
              icon={Network}
              label="导图"
            />
            <ViewToggle
              active={viewMode === "outline"}
              onClick={() => setViewMode("outline")}
              icon={ListTree}
              label="大纲"
            />
          </div>
          <Button
            variant="outline"
            size="sm"
            className="h-8"
            disabled={saving || !map}
            onClick={() => void save()}
          >
            {saving ? (
              <Loader2 size={13} className="animate-spin" />
            ) : (
              <Save size={13} />
            )}
            {saving ? "保存中…" : "保存到 Hub"}
          </Button>
        </div>
      </div>

      {/* —— 错误横幅 —— */}
      {error && (
        <div className="shrink-0 flex items-center gap-2 border-b border-destructive/30 bg-destructive/10 px-4 py-2 text-xs text-destructive">
          <AlertCircle size={12} />
          <span className="truncate">{error}</span>
        </div>
      )}

      {/* —— 保存成功提示 —— */}
      {savedUri && (
        <div className="shrink-0 flex items-center gap-2 border-b border-emerald-500/30 bg-emerald-500/10 px-4 py-2 text-xs text-emerald-500">
          <CheckCircle2 size={12} />
          <span>已保存：{savedUri}</span>
        </div>
      )}

      {/* —— 主体 —— */}
      <div className="flex min-h-0 flex-1">
        {/* 左侧：导图 / 大纲渲染 */}
        <section className="flex min-w-0 flex-1 flex-col">
          {loading && !map ? (
            <LoadingState />
          ) : !map ? (
            <EmptyState />
          ) : viewMode === "map" ? (
            <MapView />
          ) : (
            <OutlineView />
          )}
        </section>

        {/* 右侧：操作面板 */}
        <aside className="w-80 shrink-0 border-l border-[var(--shell-seam)] bg-[var(--shell-inspector-panel)]">
          <OperationsPanel />
        </aside>
      </div>
    </div>
  );
}

// —— 递归树形渲染 ——
function MapView() {
  const map = useMindmapStore((s) => s.map);
  if (!map?.root) {
    return <EmptyState />;
  }
  return (
    <div className="min-h-0 flex-1 overflow-auto scrollbar-dark animate-fade-in">
      <div className="px-6 py-4">
        {/* 标题 */}
        <div className="mb-3 flex items-center gap-2">
          <h2 className="text-base font-semibold text-foreground">
            {map.title}
          </h2>
          <span className="rounded bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground">
            {map.id.slice(0, 8)}
          </span>
        </div>
        {/* 树形渲染 */}
        <ul className="space-y-0.5">
          <NodeRow node={map.root} depth={0} isLast={true} />
        </ul>
      </div>
    </div>
  );
}

function NodeRow({
  node,
  depth,
  isLast,
}: {
  node: MindmapNode;
  depth: number;
  isLast: boolean;
}) {
  const children = node.children ?? [];
  return (
    <li className="relative">
      <div
        className="flex items-center gap-1.5 py-0.5"
        style={{ paddingLeft: `${depth * 20}px` }}
      >
        {/* 连接线 */}
        {depth > 0 && (
          <span
            className={cn(
              "absolute top-0 h-full w-px bg-border",
              isLast ? "h-3" : "h-full"
            )}
            style={{ left: `${(depth - 1) * 20 + 8}px` }}
          />
        )}
        {/* 节点圆点 */}
        <span
          className={cn(
            "h-1.5 w-1.5 shrink-0 rounded-full",
            depth === 0 ? "bg-primary" : "bg-muted-foreground/50"
          )}
        />
        {/* 节点内容 */}
        <span
          className={cn(
            "rounded px-1.5 py-0.5 text-[13px] transition-colors",
            depth === 0
              ? "font-semibold text-foreground"
              : "text-foreground/85"
          )}
        >
          {node.masked ? (
            <span className="font-mono text-muted-foreground/60">???</span>
          ) : (
            node.topic
          )}
        </span>
        {node.masked && (
          <EyeOff size={11} className="text-muted-foreground/50" />
        )}
        {node.notes && !node.masked && (
          <span
            className="ml-1 text-[10px] text-muted-foreground/60"
            title={node.notes}
          >
            📝
          </span>
        )}
      </div>
      {/* 子节点 */}
      {children.length > 0 && (
        <ul className="space-y-0.5">
          {children.map((child, i) => (
            <NodeRow
              key={child.id || i}
              node={child}
              depth={depth + 1}
              isLast={i === children.length - 1}
            />
          ))}
        </ul>
      )}
    </li>
  );
}

// —— 大纲视图 ——
function OutlineView() {
  const outline = useMindmapStore((s) => s.outline);
  const map = useMindmapStore((s) => s.map);
  return (
    <div className="min-h-0 flex-1 overflow-auto scrollbar-dark animate-fade-in">
      <div className="px-6 py-4">
        <div className="mb-3 flex items-center gap-2">
          <ListTree size={14} className="text-primary" />
          <h2 className="text-sm font-semibold text-foreground">
            {map?.title ?? "大纲"} · 纯文本
          </h2>
        </div>
        {outline ? (
          <pre className="whitespace-pre-wrap break-words rounded-md border border-[var(--shell-seam)] bg-background p-4 font-mono text-[12px] leading-relaxed text-foreground/90">
            {outline}
          </pre>
        ) : (
          <div className="py-8 text-center text-xs text-muted-foreground">
            大纲为空
          </div>
        )}
      </div>
    </div>
  );
}

// —— 操作面板 ——
function OperationsPanel() {
  const map = useMindmapStore((s) => s.map);
  const editInstruction = useMindmapStore((s) => s.editInstruction);
  const setEditInstruction = useMindmapStore((s) => s.setEditInstruction);
  const maskRate = useMindmapStore((s) => s.maskRate);
  const setMaskRate = useMindmapStore((s) => s.setMaskRate);
  const loading = useMindmapStore((s) => s.loading);
  const edit = useMindmapStore((s) => s.edit);
  const applyMask = useMindmapStore((s) => s.applyMask);

  if (!map) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted text-muted-foreground">
          <Sparkles size={18} />
        </div>
        <div className="space-y-1">
          <div className="text-xs font-medium text-foreground">无可用操作</div>
          <div className="text-[11px] text-muted-foreground">
            请先生成导图
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <div className="shrink-0 border-b border-[var(--shell-seam)] px-3 py-2.5">
        <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
          操作面板
        </div>
      </div>

      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto scrollbar-dark px-3 py-3">
        {/* AI 编辑 */}
        <div className="space-y-2">
          <div className="flex items-center gap-1.5 text-[11px] font-semibold text-foreground">
            <Sparkles size={11} className="text-primary" />
            AI 编辑
          </div>
          <Textarea
            value={editInstruction}
            onChange={(e) => setEditInstruction(e.target.value)}
            placeholder="告诉 AI 如何修改导图…  例如：增加光合作用暗反应的子节点"
            rows={3}
            className="resize-none text-[12px]"
          />
          <Button
            size="sm"
            className="h-7 w-full"
            disabled={loading || !editInstruction.trim()}
            onClick={() => void edit()}
          >
            {loading ? (
              <Loader2 size={12} className="animate-spin" />
            ) : (
              <Sparkles size={12} />
            )}
            {loading ? "处理中…" : "AI 编辑"}
          </Button>
        </div>

        {/* 分隔线 */}
        <div className="border-t border-[var(--shell-seam)]/60" />

        {/* 蒙版 */}
        <div className="space-y-2">
          <div className="flex items-center gap-1.5 text-[11px] font-semibold text-foreground">
            <EyeOff size={11} className="text-primary" />
            节点蒙版（背书模式）
          </div>
          <div className="flex items-center gap-2">
            <input
              type="range"
              min={0}
              max={100}
              value={maskRate}
              onChange={(e) => setMaskRate(Number(e.target.value))}
              className="flex-1 accent-primary"
            />
            <span className="w-10 text-right font-mono text-[11px] text-muted-foreground">
              {maskRate}%
            </span>
          </div>
          <Button
            variant="outline"
            size="sm"
            className="h-7 w-full"
            disabled={loading}
            onClick={() => void applyMask()}
          >
            {loading ? (
              <Loader2 size={12} className="animate-spin" />
            ) : (
              <EyeOff size={12} />
            )}
            应用蒙版
          </Button>
        </div>

        {/* 分隔线 */}
        <div className="border-t border-[var(--shell-seam)]/60" />

        {/* 导图信息 */}
        <div className="space-y-1.5">
          <div className="text-[11px] font-semibold text-foreground">
            导图信息
          </div>
          <dl className="space-y-1 text-[10px]">
            <div className="flex items-center gap-1.5">
              <dt className="text-muted-foreground/60">标题：</dt>
              <dd className="truncate text-foreground/80">{map.title}</dd>
            </div>
            <div className="flex items-center gap-1.5">
              <dt className="text-muted-foreground/60">ID：</dt>
              <dd className="truncate font-mono text-foreground/80">
                {map.id}
              </dd>
            </div>
            <div className="flex items-center gap-1.5">
              <dt className="text-muted-foreground/60">节点数：</dt>
              <dd className="font-mono text-foreground/80">
                {countNodes(map.root)}
              </dd>
            </div>
            <div className="flex items-center gap-1.5">
              <dt className="text-muted-foreground/60">蒙版数：</dt>
              <dd className="font-mono text-foreground/80">
                {countMasked(map.root)}
              </dd>
            </div>
          </dl>
        </div>
      </div>
    </div>
  );
}

// —— 视图切换按钮 ——
function ViewToggle({
  active,
  onClick,
  icon: Icon,
  label,
}: {
  active: boolean;
  onClick: () => void;
  icon: typeof Network;
  label: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex items-center gap-1 rounded px-2 py-1 text-[11px] font-medium transition-colors",
        active
          ? "bg-background text-foreground shadow-sm"
          : "text-muted-foreground hover:text-foreground"
      )}
    >
      <Icon size={11} />
      {label}
    </button>
  );
}

// —— 加载状态 ——
function LoadingState() {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10 text-primary">
        <Loader2 size={22} className="animate-spin" />
      </div>
      <div className="space-y-1">
        <div className="text-sm font-medium text-foreground">生成中…</div>
        <div className="text-xs text-muted-foreground">
          AI 正在构建知识结构
        </div>
      </div>
    </div>
  );
}

// —— 空状态 ——
function EmptyState() {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
        <Brain size={22} />
      </div>
      <div className="space-y-1">
        <div className="text-sm font-medium text-foreground">未生成导图</div>
        <div className="text-xs text-muted-foreground">
          在顶部输入主题，点击"生成导图"开始
        </div>
      </div>
    </div>
  );
}

// —— 工具函数 ——
/** 递归计算节点总数 */
function countNodes(node?: MindmapNode): number {
  if (!node) return 0;
  let count = 1;
  for (const c of node.children ?? []) {
    count += countNodes(c);
  }
  return count;
}

/** 递归计算蒙版节点数 */
function countMasked(node?: MindmapNode): number {
  if (!node) return 0;
  let count = node.masked ? 1 : 0;
  for (const c of node.children ?? []) {
    count += countMasked(c);
  }
  return count;
}

// 保留未使用图标引用
void FileText;
void Inbox;
