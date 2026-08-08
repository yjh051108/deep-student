// MindmapApp —— Hub 内嵌思维导图应用
// ------------------------------------------------------------
// 显示 mindmap 类型资源列表 + 选中后渲染导图内容（树形缩进视图，
// 完整画布由独立 MindmapPage 承担）。

import { useEffect, useState } from "react";
import { useHubStore } from "@/state/hub";
import { useMindmapStore } from "@/state/mindmap";
import { cn } from "@/lib/utils";
import { ShareNetwork, Plus, CaretRight, CaretDown, MagnifyingGlass } from "@phosphor-icons/react";

export function MindmapApp() {
  const entries = useHubStore((s) => s.entries);
  const load = useMindmapStore((s) => s.load);
  const map = useMindmapStore((s) => s.map);
  const [selectedUri, setSelectedUri] = useState<string | null>(null);
  const [keyword, setKeyword] = useState("");
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});

  useEffect(() => {
    void load("");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const maps = entries.filter((e) => e.type === "mindmap");
  const visible = keyword.trim()
    ? maps.filter((e) => (e.title ?? "").toLowerCase().includes(keyword.toLowerCase()))
    : maps;

  const openMap = async (uri: string) => {
    setSelectedUri(uri);
    await load(uri);
    const m = useMindmapStore.getState().map;
    if (m?.root) {
      const ex: Record<string, boolean> = {};
      const walk = (n: { id: string; children?: unknown[] }) => {
        ex[n.id] = true;
        (n.children ?? []).forEach((c) => walk(c as { id: string; children?: unknown[] }));
      };
      walk(m.root);
      setExpanded(ex);
    }
  };

  const root = map?.root;

  return (
    <div className="flex h-full w-full min-h-0">
      {/* 左：导图列表 */}
      <div className="flex w-60 shrink-0 flex-col border-r border-[var(--shell-seam)] bg-[var(--shell-navigation-surface)]">
        <div className="shrink-0 border-b border-[var(--shell-seam)] p-2.5">
          <div className="flex items-center gap-2 rounded-md bg-[var(--interactive-hover)] px-2 py-1.5">
            <MagnifyingGlass size={13} className="text-muted-foreground" />
            <input
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              placeholder="搜索导图…"
              className="w-full bg-transparent text-[12px] text-foreground placeholder:text-muted-foreground/60 focus:outline-none"
            />
          </div>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark p-1.5">
          {visible.length === 0 ? (
            <p className="px-3 py-4 text-center text-[11px] text-muted-foreground/60">没有导图</p>
          ) : (
            visible.map((e) => (
              <button
                key={e.uri}
                onClick={() => void openMap(e.uri)}
                className={cn(
                  "flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left transition-colors",
                  selectedUri === e.uri ? "bg-[var(--interactive-selected)]" : "hover:bg-[var(--interactive-hover)]"
                )}
              >
                <ShareNetwork size={14} className="shrink-0 text-primary" />
                <span className={cn("min-w-0 flex-1 truncate text-[12.5px]", selectedUri === e.uri ? "font-medium text-foreground" : "text-muted-foreground")}>
                  {e.title}
                </span>
              </button>
            ))
          )}
        </div>
      </div>

      {/* 右：树形渲染 */}
      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex shrink-0 items-center gap-2 border-b border-[var(--shell-seam)] px-4 py-2.5">
          <ShareNetwork size={13} className="text-primary" />
          <span className="text-[12px] font-medium text-foreground">
            {maps.find((m) => m.uri === selectedUri)?.title ?? "选择一张导图"}
          </span>
          <button
            onClick={() => void openMap(selectedUri ?? "")}
            className="ml-auto rounded px-2 py-0.5 text-[11px] text-muted-foreground hover:bg-[var(--interactive-hover)] hover:text-foreground"
          >
            刷新
          </button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark p-4">
          {root ? (
            <div className="mx-auto max-w-2xl">
              <TreeNode
                node={root}
                depth={0}
                expanded={expanded}
                onToggle={(id) => setExpanded((ex) => ({ ...ex, [id]: !ex[id] }))}
              />
            </div>
          ) : (
            <div className="flex h-full flex-col items-center justify-center gap-2 text-muted-foreground">
              <ShareNetwork size={26} className="opacity-40" />
              <span className="text-[12px]">选择导图查看结构</span>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// —— 树节点 ——
function TreeNode({
  node,
  depth,
  expanded,
  onToggle,
}: {
  node: { id: string; topic: string; children?: unknown[] };
  depth: number;
  expanded: Record<string, boolean>;
  onToggle: (id: string) => void;
}) {
  const children = (node.children ?? []) as { id: string; topic: string; children?: unknown[] }[];
  const hasChildren = children.length > 0;
  const open = expanded[node.id] ?? false;

  return (
    <div>
      <div
        className="flex items-center gap-1.5 rounded-md px-2 py-1 hover:bg-[var(--interactive-hover)]"
        style={{ marginLeft: depth * 18 }}
      >
        {hasChildren ? (
          <button onClick={() => onToggle(node.id)} className="text-muted-foreground">
            {open ? <CaretDown size={11} /> : <CaretRight size={11} />}
          </button>
        ) : (
          <span className="w-[11px]" />
        )}
        <span className="truncate text-[13px] text-foreground">{node.topic}</span>
      </div>
      {open &&
        children.map((c) => (
          <TreeNode key={c.id} node={c} depth={depth + 1} expanded={expanded} onToggle={onToggle} />
        ))}
    </div>
  );
}

// 保留图标
void Plus;
