// NotesApp —— Hub 内嵌笔记应用（轻量：列表 + 选中预览）
// ------------------------------------------------------------
// 对齐原版：笔记资源在 Hub 中打开时显示笔记列表，
// 点击进入编辑（此处内嵌简化列表 + 内容预览）。

import { useEffect, useState } from "react";
import { useNotesStore } from "@/state/notes";
import { cn } from "@/lib/utils";
import { FileText, MagnifyingGlass, PencilSimple } from "@phosphor-icons/react";

export function NotesApp() {
  const init = useNotesStore((s) => s.init);
  const items = useNotesStore((s) => s.items);
  const refresh = useNotesStore((s) => s.refresh);
  const select = useNotesStore((s) => s.select);
  const current = useNotesStore((s) => s.current);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [content, setContent] = useState("");
  const [keyword, setKeyword] = useState("");

  useEffect(() => {
    void init();
    void refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const visible = keyword.trim()
    ? items.filter((n) => n.title.toLowerCase().includes(keyword.toLowerCase()))
    : items;

  const openNote = async (id: string) => {
    setSelectedId(id);
    await select(id);
    setContent(useNotesStore.getState().current?.contentMd ?? "");
  };

  return (
    <div className="flex h-full w-full min-h-0">
      {/* 左：笔记列表 */}
      <div className="flex w-64 shrink-0 flex-col border-r border-[var(--shell-seam)] bg-[var(--shell-navigation-surface)]">
        <div className="shrink-0 border-b border-[var(--shell-seam)] p-2.5">
          <div className="flex items-center gap-2 rounded-md bg-[var(--interactive-hover)] px-2 py-1.5">
            <MagnifyingGlass size={13} className="text-muted-foreground" />
            <input
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              placeholder="搜索笔记…"
              className="w-full bg-transparent text-[12px] text-foreground placeholder:text-muted-foreground/60 focus:outline-none"
            />
          </div>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark p-1.5">
          {visible.length === 0 ? (
            <p className="px-3 py-4 text-center text-[11px] text-muted-foreground/60">没有笔记</p>
          ) : (
            visible.map((n) => (
              <button
                key={n.id}
                onClick={() => void openNote(n.id)}
                className={cn(
                  "flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left transition-colors",
                  selectedId === n.id ? "bg-[var(--interactive-selected)]" : "hover:bg-[var(--interactive-hover)]"
                )}
              >
                <FileText size={14} className="shrink-0 text-muted-foreground" />
                <span className={cn("min-w-0 flex-1 truncate text-[12.5px]", selectedId === n.id ? "font-medium text-foreground" : "text-muted-foreground")}>
                  {n.title}
                </span>
              </button>
            ))
          )}
        </div>
      </div>

      {/* 右：内容预览 */}
      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex shrink-0 items-center gap-2 border-b border-[var(--shell-seam)] px-4 py-2.5">
          <PencilSimple size={13} className="text-primary" />
          <span className="text-[12px] font-medium text-foreground">
            {items.find((n) => n.id === selectedId)?.title ?? "选择一篇笔记"}
          </span>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark p-4">
          {selectedId ? (
            <div className="mx-auto max-w-2xl whitespace-pre-wrap text-[13px] leading-relaxed text-foreground">
              {content || "（空笔记）"}
            </div>
          ) : (
            <div className="flex h-full items-center justify-center text-muted-foreground">
              <span className="text-[12px]">从左侧选择笔记查看内容</span>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
