// ReaderPage —— 阅读器
// ------------------------------------------------------------
// 顶部：URI 输入框 + 打开按钮
// 主体：
//   左 (flex-1) - 文档内容（ReaderDocumentPanel）
//   右 (320px)  - 操作面板（ReaderActionPanel：总结 + 注入）

import { useState } from "react";
import { ReaderDocumentPanel } from "@/components/reader/ReaderDocumentPanel";
import { ReaderActionPanel } from "@/components/reader/ReaderActionPanel";
import { useReaderStore } from "@/state/reader";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { FolderOpen, Loader2, BookOpen, AlertCircle } from "lucide-react";

export function ReaderPage() {
  const open = useReaderStore((s) => s.open);
  const opening = useReaderStore((s) => s.opening);
  const error = useReaderStore((s) => s.error);
  const doc = useReaderStore((s) => s.doc);
  const [uri, setUri] = useState("");

  const handleOpen = async () => {
    if (!uri.trim() || opening) return;
    await open(uri.trim());
  };

  return (
    <div className="flex h-full w-full min-h-0 flex-col bg-background">
      {/* —— 顶部：URI 输入 + 打开按钮 —— */}
      <div className="shrink-0 space-y-2 border-b border-border bg-card px-4 py-3">
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <BookOpen
              size={14}
              className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground"
            />
            <Input
              value={uri}
              onChange={(e) => setUri(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") void handleOpen();
              }}
              placeholder="输入文档 URI，如 vfs://textbook/xxx 或 file:///path/to.pdf"
              className="h-8 pl-8 text-[13px]"
            />
          </div>
          <Button
            size="sm"
            className="h-8"
            onClick={handleOpen}
            disabled={!uri.trim() || opening}
          >
            {opening ? (
              <Loader2 size={12} className="animate-spin" />
            ) : (
              <FolderOpen size={12} />
            )}
            {opening ? "打开中…" : "打开"}
          </Button>
        </div>
        {/* 错误横幅 */}
        {error && (
          <div className="flex items-center gap-2 border border-destructive/30 bg-destructive/10 px-3 py-1.5 text-[11px] text-destructive">
            <AlertCircle size={11} />
            <span className="truncate">{error}</span>
          </div>
        )}
        {doc && (
          <div className="truncate text-[10px] text-muted-foreground/70">
            当前文档：{doc.title} · {doc.pages.length} 页
          </div>
        )}
      </div>

      {/* —— 主体：文档 + 操作面板 —— */}
      <div className="flex min-h-0 flex-1">
        {/* 左：文档内容 */}
        <section className="flex min-w-0 flex-1 flex-col">
          <ReaderDocumentPanel />
        </section>

        {/* 右：操作面板 */}
        <aside className="w-80 shrink-0 border-l border-border bg-background">
          <ReaderActionPanel />
        </aside>
      </div>
    </div>
  );
}
