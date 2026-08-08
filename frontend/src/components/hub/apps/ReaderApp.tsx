// ReaderApp —— Hub 内嵌阅读器应用（对齐原版 textbook→PDF 阅读器）
// ------------------------------------------------------------
// 显示 textbook 类型的资源列表，选择后进入 ReaderDocumentPanel 阅读。

import { useEffect, useState } from "react";
import { useHubStore } from "@/state/hub";
import { useReaderStore } from "@/state/reader";
import { cn } from "@/lib/utils";
import { BookOpen, CaretLeft, CaretRight, MagnifyingGlass } from "@phosphor-icons/react";

export function ReaderApp() {
  const entries = useHubStore((s) => s.entries);
  const open = useReaderStore((s) => s.open);
  const doc = useReaderStore((s) => s.doc);
  const currentPageIdx = useReaderStore((s) => s.currentPageIdx);
  const nextPage = useReaderStore((s) => s.nextPage);
  const prevPage = useReaderStore((s) => s.prevPage);
  const [keyword, setKeyword] = useState("");
  const [openingUri, setOpeningUri] = useState("");

  const textbooks = entries.filter((e) => e.type === "textbook");
  const visible = keyword.trim()
    ? textbooks.filter((e) => (e.title ?? "").toLowerCase().includes(keyword.toLowerCase()))
    : textbooks;

  const openDoc = async (uri: string) => {
    setOpeningUri(uri);
    await open(uri);
    setOpeningUri("");
  };

  // 已打开文档 → 阅读视图
  if (doc) {
    const pages = doc.pages ?? [];
    return (
      <div className="flex h-full w-full min-h-0 flex-col">
        <div className="flex shrink-0 items-center gap-2 border-b border-[var(--shell-seam)] px-3 py-1.5">
          <button
            onClick={() => { void open(""); }}
            className="rounded px-2 py-0.5 text-[11px] text-muted-foreground hover:bg-[var(--interactive-hover)] hover:text-foreground"
          >
            ← 返回列表
          </button>
          <span className="min-w-0 flex-1 truncate text-[12px] font-medium text-foreground">{doc.title}</span>
          <div className="flex items-center gap-1">
            <button onClick={prevPage} disabled={currentPageIdx <= 0} className="rounded p-1 text-muted-foreground hover:text-foreground disabled:opacity-30">
              <CaretLeft size={13} />
            </button>
            <span className="min-w-10 text-center font-mono text-[11px] text-muted-foreground">
              {currentPageIdx + 1} / {pages.length}
            </span>
            <button onClick={nextPage} disabled={currentPageIdx >= pages.length - 1} className="rounded p-1 text-muted-foreground hover:text-foreground disabled:opacity-30">
              <CaretRight size={13} />
            </button>
          </div>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark p-4">
          <div className="mx-auto max-w-2xl space-y-3">
            {pages[currentPageIdx] && (
              <div className="whitespace-pre-wrap rounded-lg border border-[var(--border-default)] bg-[var(--shell-inspector-panel)] p-5 text-[13px] leading-relaxed text-foreground">
                {pages[currentPageIdx].content}
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }

  // 列表视图
  return (
    <div className="flex h-full w-full min-h-0 flex-col">
      <div className="flex shrink-0 items-center gap-2 border-b border-[var(--shell-seam)] px-3 py-2">
        <div className="flex flex-1 items-center gap-2 rounded-md bg-[var(--interactive-hover)] px-2 py-1.5">
          <MagnifyingGlass size={13} className="text-muted-foreground" />
          <input
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder="搜索教材…"
            className="w-full bg-transparent text-[12px] text-foreground placeholder:text-muted-foreground/60 focus:outline-none"
          />
        </div>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark p-3">
        {visible.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 text-muted-foreground">
            <BookOpen size={26} className="opacity-40" />
            <span className="text-[12px]">没有教材资源，去 Hub 导入</span>
          </div>
        ) : (
          <div className="mx-auto max-w-2xl space-y-1">
            {visible.map((e) => (
              <button
                key={e.uri}
                onClick={() => void openDoc(e.uri)}
                disabled={openingUri === e.uri}
                className={cn(
                  "flex w-full items-center gap-3 rounded-lg border border-[var(--border-default)] bg-[var(--shell-inspector-panel)] px-3.5 py-2.5 text-left transition-colors hover:border-[var(--primary-color)]/40",
                  openingUri === e.uri && "opacity-50"
                )}
              >
                <BookOpen size={16} className="shrink-0 text-primary" />
                <span className="min-w-0 flex-1 truncate text-[13px] text-foreground">{e.title}</span>
                <span className="shrink-0 text-[10px] text-muted-foreground/60">
                  {openingUri === e.uri ? "打开中…" : `${e.size ?? 0} B`}
                </span>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
