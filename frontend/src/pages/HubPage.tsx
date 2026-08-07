// HubPage —— 学习中心（资源中枢 + 工作台双视图）
// ------------------------------------------------------------
// 视图切换：工作台（应用网格）/ 资源（三栏资源管理）
// 对照原版学习中心工作台概念。

import { useEffect, useState } from "react";
import { ResourceTree } from "@/components/hub/ResourceTree";
import { ResourceList } from "@/components/hub/ResourceList";
import { ResourcePreview } from "@/components/hub/ResourcePreview";
import { ImportDialog } from "@/components/hub/ImportDialog";
import { WorkbenchGrid } from "@/components/hub/WorkbenchGrid";
import { useHubStore } from "@/state/hub";
import { LayoutGrid, FolderOpen } from "lucide-react";
import { cn } from "@/lib/utils";

type View = "workbench" | "resources";

export function HubPage() {
  const refresh = useHubStore((s) => s.refresh);
  const [importOpen, setImportOpen] = useState(false);
  const [view, setView] = useState<View>(() => {
    const saved = localStorage.getItem("hub.view") as View | null;
    return saved === "resources" ? "resources" : "workbench";
  });

  useEffect(() => {
    localStorage.setItem("hub.view", view);
  }, [view]);

  // 挂载时拉取一次
  useEffect(() => {
    void refresh();
  }, [refresh]);

  if (view === "workbench") {
    return (
      <div className="flex h-full w-full min-h-0 flex-col bg-background">
        {/* 顶部切换栏 */}
        <div className="flex shrink-0 items-center gap-1 border-b border-border bg-card px-3 py-2">
          <ViewSwitch active={view} onChange={setView} />
        </div>
        <div className="min-h-0 flex-1">
          <WorkbenchGrid />
        </div>
      </div>
    );
  }

  return (
    <div className="relative flex h-full w-full min-h-0 bg-background">
      {/* 顶部切换栏（叠加） */}
      <div className="absolute left-56 top-2 z-10">
        <ViewSwitch active={view} onChange={setView} />
      </div>

      {/* —— 左：资源类型树 —— */}
      <aside className="w-56 shrink-0 border-r border-border bg-card">
        <ResourceTree />
      </aside>

      {/* —— 中：资源列表 —— */}
      <section className="flex min-w-0 flex-1 flex-col">
        <ResourceList onOpenImport={() => setImportOpen(true)} />
      </section>

      {/* —— 右：资源预览 —— */}
      <aside className="w-[28rem] shrink-0 border-l border-border bg-background">
        <ResourcePreview />
      </aside>

      {/* 导入对话框 */}
      <ImportDialog open={importOpen} onClose={() => setImportOpen(false)} />
    </div>
  );
}

function ViewSwitch({ active, onChange }: { active: View; onChange: (v: View) => void }) {
  return (
    <div className="flex items-center gap-0.5 rounded-lg bg-muted p-0.5 shadow-soft">
      <button
        onClick={() => onChange("workbench")}
        className={cn(
          "flex items-center gap-1.5 rounded-md px-2.5 py-1 text-[11px] transition-colors",
          active === "workbench"
            ? "bg-card font-medium text-primary"
            : "text-muted-foreground hover:text-foreground"
        )}
      >
        <LayoutGrid size={12} />
        工作台
      </button>
      <button
        onClick={() => onChange("resources")}
        className={cn(
          "flex items-center gap-1.5 rounded-md px-2.5 py-1 text-[11px] transition-colors",
          active === "resources"
            ? "bg-card font-medium text-primary"
            : "text-muted-foreground hover:text-foreground"
        )}
      >
        <FolderOpen size={12} />
        资源
      </button>
    </div>
  );
}
