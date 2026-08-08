// ReaderPage —— 1:1 对齐原版 PdfReader（中央内容 + 底部单行工具栏）
// ------------------------------------------------------------
// - 顶部：URI 输入 + 打开
// - 中央：文档内容（双向滚动区）
// - 底部工具栏（单行居中）：目录/缩略图开关 | 搜索 | 缩放 | 页码 | 全屏 | 关闭
// - 右侧：操作面板（AI 总结 + 注入聊天）

import { useState } from "react";
import { ReaderDocumentPanel } from "@/components/reader/ReaderDocumentPanel";
import { ReaderActionPanel } from "@/components/reader/ReaderActionPanel";
import { useReaderStore } from "@/state/reader";
import { cn } from "@/lib/utils";
import {
  BookOpen,
  MagnifyingGlass,
  MagnifyingGlassPlus,
  MagnifyingGlassMinus,
  ArrowsOut,
  List,
  GridFour,
  CaretLeft,
  CaretRight,
} from "@phosphor-icons/react";

export function ReaderPage() {
  const open = useReaderStore((s) => s.open);
  const opening = useReaderStore((s) => s.opening);
  const error = useReaderStore((s) => s.error);
  const doc = useReaderStore((s) => s.doc);
  const [uri, setUri] = useState("");
  const [showSidebar, setShowSidebar] = useState(true);
  const [actionPanel, setActionPanel] = useState(true);
  const page = useReaderStore((s) => s.currentPageIdx + 1);
  const goPage = useReaderStore((s) => s.setPage);
  const nextPage = useReaderStore((s) => s.nextPage);
  const prevPage = useReaderStore((s) => s.prevPage);

  const handleOpen = async () => {
    if (!uri.trim() || opening) return;
    await open(uri.trim());
    goPage(0);
  };

  const totalPages = doc?.pages?.length ?? 0;

  return (
    <div className="flex h-full w-full min-h-0 flex-col bg-background">
      {/* —— 顶部：URI 输入 —— */}
      <div className="shrink-0 border-b border-[var(--shell-seam)] bg-[var(--shell-workspace-panel)] px-4 py-2.5">
        <div className="mx-auto flex max-w-3xl items-center gap-2">
          <div className="flex flex-1 items-center gap-2 rounded-md bg-[var(--interactive-hover)] px-2.5 py-1.5">
            <BookOpen size={14} className="shrink-0 text-muted-foreground" />
            <input
              value={uri}
              onChange={(e) => setUri(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && void handleOpen()}
              placeholder="输入文档 URI（vfs://textbook/xxx 或 file:///path）"
              className="w-full bg-transparent text-[12.5px] text-foreground placeholder:text-muted-foreground/60 focus:outline-none"
            />
          </div>
          <button
            onClick={() => void handleOpen()}
            disabled={!uri.trim() || opening}
            className="flex h-7 items-center gap-1 rounded-md bg-black px-3 text-[12px] font-medium text-white hover:bg-black/80 disabled:opacity-40 dark:bg-white dark:text-black"
          >
            {opening ? "打开中…" : "打开"}
          </button>
        </div>
        {error && <div className="mx-auto mt-1.5 max-w-3xl text-[11px] text-destructive">{error}</div>}
      </div>

      {/* —— 主体 —— */}
      <div className="flex min-h-0 flex-1">
        {/* 左：目录/缩略图侧栏（可选） */}
        {showSidebar && (
          <aside className="flex w-44 shrink-0 flex-col border-r border-[var(--shell-seam)] bg-[var(--shell-navigation-surface)]">
            <div className="shrink-0 border-b border-[var(--shell-seam)] px-3 py-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">
              目录
            </div>
            <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark p-2">
              {totalPages > 0 ? (
                <div className="space-y-0.5">
                  {Array.from({ length: Math.min(totalPages, 100) }).map((_, i) => (
                    <button
                      key={i}
                      onClick={() => goPage(i)}
                      className={cn(
                        "flex w-full items-center gap-1.5 rounded px-2 py-1 text-[11px]",
                        page === i + 1
                          ? "bg-[var(--interactive-selected)] font-medium text-foreground"
                          : "text-muted-foreground hover:bg-[var(--interactive-hover)]"
                      )}
                    >
                      <GridFour size={10} className="opacity-50" />
                      第 {i + 1} 页
                    </button>
                  ))}
                </div>
              ) : (
                <p className="px-2 py-3 text-center text-[10px] text-muted-foreground/60">打开文档后显示目录</p>
              )}
            </div>
          </aside>
        )}

        {/* 中：文档内容 */}
        <section className="flex min-w-0 flex-1 flex-col">
          <div className="min-h-0 flex-1">
            <ReaderDocumentPanel />
          </div>

          {/* 底部工具栏（单行居中） */}
          <div className="flex shrink-0 items-center justify-center gap-1 border-t border-[var(--shell-seam)] bg-[var(--shell-inspector-panel)] px-3 py-1.5">
            <ToolBtn title="目录/缩略图" active={showSidebar} onClick={() => setShowSidebar((v) => !v)}>
              {showSidebar ? <List size={14} /> : <GridFour size={14} />}
            </ToolBtn>
            <ToolBtn title="搜索"><MagnifyingGlass size={14} /></ToolBtn>
            <ToolBtn title="上一页" disabled={page <= 1} onClick={prevPage}>
              <CaretLeft size={14} />
            </ToolBtn>
            <span className="min-w-16 text-center font-mono text-[11px] text-muted-foreground">
              {totalPages > 0 ? `${page} / ${totalPages}` : "- / -"}
            </span>
            <ToolBtn title="下一页" disabled={page >= totalPages} onClick={nextPage}>
              <CaretRight size={14} />
            </ToolBtn>
            <div className="mx-1 h-4 w-px bg-[var(--shell-seam)]" />
            <ToolBtn title="缩小"><MagnifyingGlassMinus size={14} /></ToolBtn>
            <ToolBtn title="放大"><MagnifyingGlassPlus size={14} /></ToolBtn>
            <ToolBtn title="全屏"><ArrowsOut size={14} /></ToolBtn>
          </div>
        </section>

        {/* 右：操作面板（AI 总结/注入） */}
        {actionPanel && (
          <aside className="w-80 shrink-0 border-l border-[var(--shell-seam)] bg-[var(--shell-navigation-surface)]">
            <ReaderActionPanel />
          </aside>
        )}
      </div>
    </div>
  );
}

// —— 工具栏按钮 ——
function ToolBtn({
  title,
  children,
  active = false,
  disabled = false,
  onClick,
}: {
  title: string;
  children: React.ReactNode;
  active?: boolean;
  disabled?: boolean;
  onClick?: () => void;
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      title={title}
      className={cn(
        "flex h-7 w-7 items-center justify-center rounded-md transition-colors",
        active
          ? "bg-primary/15 text-primary"
          : "text-muted-foreground hover:bg-[var(--interactive-hover)] hover:text-foreground",
        disabled && "opacity-30"
      )}
    >
      {children}
    </button>
  );
}
