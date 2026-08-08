// HubApp —— 1:1 对齐原版"资源类型路由到原生应用"
// ------------------------------------------------------------
// 在 Hub 右侧 TabBar 打开某类型 tab 时，按类型渲染对应应用：
//   note → 笔记列表（轻量）/ textbook → 阅读器 / mindmap → 导图 / qbank → 题库
// 非核心类型回退到资源列表视图。

import { NotesApp } from "./apps/NotesApp";
import { ReaderApp } from "./apps/ReaderApp";
import { MindmapApp } from "./apps/MindmapApp";
import { QBankApp } from "./apps/QBankApp";
import { ResourceList } from "@/components/hub/ResourceList";

export function HubApp({ type, onOpenImport }: { type: string; onOpenImport: () => void }) {
  switch (type) {
    case "note":
      return <NotesApp />;
    case "textbook":
    case "pdf":
      return <ReaderApp />;
    case "mindmap":
      return <MindmapApp />;
    case "qbank":
      return <QBankApp />;
    default:
      return (
        <ResourceList
          typeFilter={type === "all" ? "" : type}
          onOpenImport={onOpenImport}
        />
      );
  }
}
