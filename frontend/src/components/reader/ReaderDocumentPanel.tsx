// ReaderDocumentPanel —— 文档内容显示区
// ------------------------------------------------------------
// 顶部：分页导航（上一页 / 下一页 / 页码选择）
// 主体：当前页内容
// 底部：页码统计

import { useReaderStore } from "@/state/reader";
import { Button } from "@/components/ui/Button";
import { cn } from "@/lib/utils";
import {
  ChevronLeft,
  ChevronRight,
  BookOpen,
  Loader2,
  Inbox,
} from "lucide-react";

export function ReaderDocumentPanel() {
  const doc = useReaderStore((s) => s.doc);
  const currentPageIdx = useReaderStore((s) => s.currentPageIdx);
  const opening = useReaderStore((s) => s.opening);
  const setPage = useReaderStore((s) => s.setPage);
  const nextPage = useReaderStore((s) => s.nextPage);
  const prevPage = useReaderStore((s) => s.prevPage);

  if (!doc) {
    return <EmptyState opening={opening} />;
  }

  const pages = doc.pages ?? [];
  const current = pages[currentPageIdx];

  return (
    <div className="flex h-full w-full min-w-0 flex-col">
      {/* 头部：文档信息 + 分页导航 */}
      <div className="shrink-0 space-y-2 border-b border-border bg-card px-4 py-3">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <h2
              className="truncate text-sm font-semibold text-foreground"
              title={doc.title}
            >
              {doc.title}
            </h2>
            <div className="mt-0.5 flex items-center gap-2 text-[10px] text-muted-foreground/70">
              <span>共 {pages.length} 页</span>
              <span>·</span>
              <span className="truncate font-mono">{doc.uri}</span>
            </div>
          </div>
        </div>
        {/* 分页控制 */}
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            className="h-7"
            onClick={prevPage}
            disabled={currentPageIdx === 0}
          >
            <ChevronLeft size={12} />
            上一页
          </Button>
          <div className="flex items-center gap-1">
            <input
              type="number"
              min={1}
              max={pages.length}
              value={currentPageIdx + 1}
              onChange={(e) => {
                const v = Number(e.target.value);
                if (!Number.isNaN(v)) setPage(v - 1);
              }}
              className="h-7 w-14 rounded border border-input bg-transparent px-2 text-center text-[12px] text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            />
            <span className="text-[11px] text-muted-foreground">
              / {pages.length}
            </span>
          </div>
          <Button
            variant="ghost"
            size="sm"
            className="h-7"
            onClick={nextPage}
            disabled={currentPageIdx >= pages.length - 1}
          >
            下一页
            <ChevronRight size={12} />
          </Button>
          {/* 缩略页码条 */}
          <div className="ml-2 flex max-w-md flex-1 items-center gap-0.5 overflow-x-auto scrollbar-dark">
            {pages.map((_, idx) => (
              <button
                key={idx}
                type="button"
                onClick={() => setPage(idx)}
                className={cn(
                  "h-5 w-5 shrink-0 rounded text-[9px] font-medium transition-colors",
                  idx === currentPageIdx
                    ? "bg-primary text-primary-foreground"
                    : "bg-muted text-muted-foreground hover:bg-accent"
                )}
                title={`第 ${idx + 1} 页`}
              >
                {idx + 1}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* 内容 */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        {current ? (
          <div className="px-8 py-6 animate-fade-in" key={currentPageIdx}>
            <div className="mx-auto max-w-3xl">
              <div className="mb-4 flex items-center gap-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                <BookOpen size={11} />
                第 {currentPageIdx + 1} 页
              </div>
              <article className="prose prose-sm max-w-none">
                <pre className="whitespace-pre-wrap break-words font-sans text-[14px] leading-relaxed text-foreground/90">
                  {current.content}
                </pre>
              </article>
            </div>
          </div>
        ) : (
          <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
            无内容
          </div>
        )}
      </div>
    </div>
  );
}

function EmptyState({ opening }: { opening: boolean }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
        {opening ? (
          <Loader2 size={20} className="animate-spin" />
        ) : (
          <Inbox size={20} />
        )}
      </div>
      <div className="space-y-1">
        <div className="text-sm font-medium text-foreground">
          {opening ? "正在打开文档…" : "未打开文档"}
        </div>
        <div className="text-xs text-muted-foreground">
          {opening
            ? "解析 PDF / EPUB / Markdown 中"
            : `在上方输入 URI 后点击"打开"`}
        </div>
      </div>
    </div>
  );
}
