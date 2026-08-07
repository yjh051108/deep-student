// HubPage —— 1:1 对齐原版 LearningHubPage（学习资源）
// ------------------------------------------------------------
// 桌面两栏（可调宽，默认左 25%）：
// - 左 LearningHubSidebar：快捷入口分组（桌面/所有文件/最近/收藏 + 类型分组
//   notes/textbook/qbank/essay/translation/mindmap/image/document）+ 文件树
// - 右 TabBar 标签页栏（可固定）+ TabPanelContainer 应用面板（资源列表）
// 顶部面包屑（学习资源 > 类型 > …）

import { useEffect, useState } from "react";
import {
  Desktop,
  FolderSimple,
  ClockCounterClockwise,
  Star,
  FileText,
  BookOpenText,
  ListChecks,
  PenNib,
  Translate,
  ShareNetwork,
  ImageSquare,
  Files,
  Trash,
  MagnifyingGlass,
  Plus,
  X,
  DotsSixVertical,
} from "@phosphor-icons/react";
import { cn } from "@/lib/utils";
import { useHubStore } from "@/state/hub";
import { ResourceList } from "@/components/hub/ResourceList";
import { ResourcePreview } from "@/components/hub/ResourcePreview";
import { ImportDialog } from "@/components/hub/ImportDialog";
import { WorkbenchGrid } from "@/components/hub/WorkbenchGrid";

type View = "workbench" | "hub";

interface HubTab {
  id: string;
  label: string;
  pinned?: boolean;
}

export function HubPage() {
  const refresh = useHubStore((s) => s.refresh);
  const [view, setView] = useState<View>(() => {
    const saved = localStorage.getItem("hub.view") as View | null;
    return saved === "hub" ? "hub" : "workbench";
  });
  const [importOpen, setImportOpen] = useState(false);
  const [tabs, setTabs] = useState<HubTab[]>([{ id: "all", label: "所有文件", pinned: true }]);
  const [activeTab, setActiveTab] = useState("all");
  const [selectedType, setSelectedType] = useState("");

  useEffect(() => {
    localStorage.setItem("hub.view", view);
  }, [view]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const openTab = (id: string, label: string) => {
    setTabs((ts) => {
      if (ts.some((t) => t.id === id)) return ts;
      return [...ts, { id, label }];
    });
    setActiveTab(id);
  };

  const closeTab = (id: string) => {
    setTabs((ts) => ts.filter((t) => t.id !== id));
    if (activeTab === id) {
      const rest = tabs.filter((t) => t.id !== id);
      setActiveTab(rest.length > 0 ? rest[rest.length - 1].id : "all");
    }
  };

  // 工作台视图
  if (view === "workbench") {
    return (
      <div className="flex h-full w-full min-h-0 flex-col bg-background">
        <div className="flex shrink-0 items-center gap-1 border-b border-[var(--shell-seam)] bg-[var(--titlebar-background)] px-3 py-2">
          <ViewSwitch active={view} onChange={setView} />
        </div>
        <div className="min-h-0 flex-1">
          <WorkbenchGrid />
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full w-full min-h-0 bg-background">
      {/* —— 左：资源访达侧栏（默认 25%） —— */}
      <aside
        className="flex shrink-0 flex-col border-r border-[var(--shell-seam)] bg-[var(--shell-navigation-surface)]"
        style={{ width: "25%", minWidth: 220, maxWidth: 340 }}
      >
        <HubSidebar
          selectedType={selectedType}
          onSelectType={(t) => { setSelectedType(t); openTab(t, typeLabel(t)); }}
          onNew={() => setImportOpen(true)}
        />
      </aside>

      {/* 分隔条 */}
      <div className="panel-resize-handle shrink-0" />

      {/* —— 右：TabBar + 应用面板 —— */}
      <section className="flex min-w-0 flex-1 flex-col">
        {/* TabBar */}
        <div className="flex shrink-0 items-center gap-0.5 overflow-x-auto border-b border-[var(--shell-seam)] bg-[var(--shell-workspace-panel)] px-1.5 pt-1 scrollbar-dark">
          {tabs.map((t) => (
            <div
              key={t.id}
              onClick={() => setActiveTab(t.id)}
              className={cn(
                "group flex max-w-44 cursor-pointer items-center gap-1.5 rounded-t-md border border-b-0 px-3 py-1.5 text-[12px] transition-colors",
                activeTab === t.id
                  ? "border-[var(--shell-seam)] bg-[var(--shell-workspace-panel)] text-foreground"
                  : "border-transparent text-muted-foreground hover:bg-[var(--interactive-hover)]"
              )}
            >
              <span className="truncate">{t.label}</span>
              {!t.pinned && (
                <button
                  onClick={(e) => { e.stopPropagation(); closeTab(t.id); }}
                  className="hidden rounded p-0.5 text-muted-foreground hover:text-destructive group-hover:block"
                >
                  <X size={9} weight="bold" />
                </button>
              )}
            </div>
          ))}
          <div className="ml-auto flex items-center gap-0.5 px-1">
            <button onClick={() => setView("workbench")} className="flex h-6 w-6 items-center justify-center rounded text-muted-foreground hover:bg-[var(--interactive-hover)] hover:text-foreground" title="切换到工作台">
              <DotsSixVertical size={12} />
            </button>
          </div>
        </div>

        {/* 应用面板 */}
        <div className="flex min-h-0 flex-1">
          <div className="flex min-w-0 flex-1 flex-col">
            <ResourceList
              key={activeTab + selectedType}
              typeFilter={activeTab === "all" ? selectedType : activeTab}
              onOpenImport={() => setImportOpen(true)}
            />
          </div>
          {/* 右预览 */}
          <aside className="hidden w-[300px] shrink-0 border-l border-[var(--shell-seam)] xl:block">
            <ResourcePreview />
          </aside>
        </div>
      </section>

      <ImportDialog open={importOpen} onClose={() => setImportOpen(false)} />
    </div>
  );
}

// —— 视图切换 ——
function ViewSwitch({ active, onChange }: { active: View; onChange: (v: View) => void }) {
  return (
    <div className="flex items-center gap-0.5 rounded-lg bg-muted p-0.5">
      <button
        onClick={() => onChange("workbench")}
        className={cn(
          "flex items-center gap-1.5 rounded-md px-2.5 py-1 text-[11px] transition-colors",
          active === "workbench" ? "bg-card font-medium text-primary shadow-soft" : "text-muted-foreground hover:text-foreground"
        )}
      >
        <Desktop size={12} />
        工作台
      </button>
      <button
        onClick={() => onChange("hub")}
        className={cn(
          "flex items-center gap-1.5 rounded-md px-2.5 py-1 text-[11px] transition-colors",
          active === "hub" ? "bg-card font-medium text-primary shadow-soft" : "text-muted-foreground hover:text-foreground"
        )}
      >
        <FolderSimple size={12} />
        学习资源
      </button>
    </div>
  );
}

// —— 左侧栏 ——
const QUICK_GROUPS: { title: string; items: { key: string; label: string; icon: typeof Desktop }[] }[] = [
  {
    title: "",
    items: [
      { key: "desktop", label: "桌面", icon: Desktop },
      { key: "all", label: "所有文件", icon: FolderSimple },
      { key: "recent", label: "最近", icon: ClockCounterClockwise },
      { key: "favorites", label: "收藏", icon: Star },
    ],
  },
  {
    title: "学习资源",
    items: [
      { key: "note", label: "笔记", icon: FileText },
      { key: "textbook", label: "教材", icon: BookOpenText },
      { key: "qbank", label: "题库", icon: ListChecks },
      { key: "essay", label: "作文", icon: PenNib },
      { key: "translation", label: "翻译", icon: Translate },
      { key: "mindmap", label: "导图", icon: ShareNetwork },
      { key: "flashcard", label: "卡片", icon: Star },
      { key: "paper", label: "论文", icon: Files },
    ],
  },
  {
    title: "系统",
    items: [
      { key: "trash", label: "回收站", icon: Trash },
    ],
  },
];

function typeLabel(t: string): string {
  const found = QUICK_GROUPS.flatMap((g) => g.items).find((i) => i.key === t);
  return found?.label ?? t;
}

function HubSidebar({
  selectedType,
  onSelectType,
  onNew,
}: {
  selectedType: string;
  onSelectType: (t: string) => void;
  onNew: () => void;
}) {
  const [query, setQuery] = useState("");
  return (
    <div className="flex h-full flex-col">
      {/* 搜索 + 新建 */}
      <div className="shrink-0 border-b border-[var(--shell-seam)] p-3">
        <div className="flex items-center gap-2 rounded-md bg-[var(--interactive-hover)] px-2.5 py-1.5">
          <MagnifyingGlass size={13} className="text-muted-foreground" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索资源…"
            className="w-full bg-transparent text-[12px] text-foreground placeholder:text-muted-foreground/60 focus:outline-none"
          />
        </div>
        <button
          onClick={onNew}
          className="sidebar-row mt-2 flex w-full items-center gap-2 px-2.5"
          data-active={false}
        >
          <Plus size={15} className="shrink-0 opacity-70" />
          <span className="text-[12px]">导入资源</span>
        </button>
      </div>

      {/* 快捷入口 */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark p-2">
        {QUICK_GROUPS.map((group) => (
          <div key={group.title || "root"} className="mb-3">
            {group.title && (
              <div className="px-2.5 pb-1 pt-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">
                {group.title}
              </div>
            )}
            <div className="space-y-0.5">
              {group.items
                .filter((i) => !query || i.label.includes(query))
                .map((item) => {
                  const Icon = item.icon;
                  const active = selectedType === item.key;
                  return (
                    <button
                      key={item.key}
                      onClick={() => onSelectType(item.key)}
                      className="sidebar-row flex w-full items-center gap-2.5 px-2.5"
                      data-active={active}
                    >
                      <Icon size={16} className="shrink-0 opacity-80" weight={active ? "fill" : "regular"} />
                      <span className="min-w-0 flex-1 truncate text-[12.5px]">{item.label}</span>
                    </button>
                  );
                })}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
