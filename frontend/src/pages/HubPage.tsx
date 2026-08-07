// HubPage —— 学习中心资源中枢
// ------------------------------------------------------------
// 三栏布局：
// 1. 左：资源类型树（10 种类型 + 全部）
// 2. 中：资源列表（搜索 / 标签过滤 / 导入 / 删除）
// 3. 右：资源预览（元数据 + 内容 + AI 续写）
//
// 对照原版学习中心：统一管理 note / textbook / qbank / mindmap /
// translation / flashcard / paper / chat / todo / skill 所有资源

import { useEffect, useState } from "react";
import { ResourceTree } from "@/components/hub/ResourceTree";
import { ResourceList } from "@/components/hub/ResourceList";
import { ResourcePreview } from "@/components/hub/ResourcePreview";
import { ImportDialog } from "@/components/hub/ImportDialog";
import { useHubStore } from "@/state/hub";

export function HubPage() {
  const refresh = useHubStore((s) => s.refresh);
  const [importOpen, setImportOpen] = useState(false);

  // 挂载时拉取一次
  useEffect(() => {
    void refresh();
  }, [refresh]);

  return (
    <div className="flex h-full w-full min-h-0 bg-background">
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
