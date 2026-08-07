// NotesPage —— 笔记系统
// ------------------------------------------------------------
// 三栏布局：
// 1. 左 (220px)：文件夹树 + 回收站 + 统计
// 2. 中 (300px)：笔记列表（搜索、排序、置顶、新建）
// 3. 右 (flex-1)：编辑器（标题、标签、Markdown 正文、附件、操作）
//
// 对接后端：NotesXxx 系列 Wails 绑定

import { useEffect, useState } from "react";
import { useNotesStore, type SortBy, type SortDir } from "@/state/notes";
import { formatDateTime, formatSize, type ExportFormat } from "@/lib/notes";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { Badge } from "@/components/ui/Badge";
import { cn } from "@/lib/utils";
import {
  NotebookPen,
  Folder as FolderIcon,
  FolderPlus,
  Trash2,
  RotateCcw,
  Search,
  Plus,
  Save,
  Pin,
  PinOff,
  Download,
  Upload,
  Loader2,
  AlertCircle,
  Inbox,
  Hash,
  CheckCircle2,
  FileText,
  Paperclip,
  X,
  MoreVertical,
  FileDown,
  Files,
  ChevronDown,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuLabel,
} from "@/components/ui/DropdownMenu";

export function NotesPage() {
  const init = useNotesStore((s) => s.init);

  useEffect(() => {
    void init();
  }, [init]);

  return (
    <div className="flex h-full w-full min-h-0 bg-background">
      {/* —— 左：文件夹树 + 回收站 —— */}
      <aside className="w-56 shrink-0 border-r border-border bg-card">
        <FolderPanel />
      </aside>

      {/* —— 中：笔记列表 —— */}
      <section className="w-72 shrink-0 flex min-w-0 flex-col border-r border-border bg-background">
        <NoteList />
      </section>

      {/* —— 右：编辑器 —— */}
      <section className="flex min-w-0 flex-1 flex-col">
        <EditorPanel />
      </section>
    </div>
  );
}

// ===================== 左：文件夹面板 =====================

function FolderPanel() {
  const folders = useNotesStore((s) => s.folders);
  const selectedFolderId = useNotesStore((s) => s.selectedFolderId);
  const viewingTrash = useNotesStore((s) => s.viewingTrash);
  const setFolder = useNotesStore((s) => s.setFolder);
  const viewTrash = useNotesStore((s) => s.viewTrash);
  const createFolder = useNotesStore((s) => s.createFolder);
  const deleteFolder = useNotesStore((s) => s.deleteFolder);
  const stats = useNotesStore((s) => s.stats);
  const toast = useNotesStore((s) => s.toast);
  const error = useNotesStore((s) => s.error);

  const [newFolderName, setNewFolderName] = useState("");
  const [showNewFolder, setShowNewFolder] = useState(false);

  const handleCreateFolder = async () => {
    if (!newFolderName.trim()) return;
    await createFolder(newFolderName.trim());
    setNewFolderName("");
    setShowNewFolder(false);
  };

  return (
    <div className="flex h-full w-full flex-col">
      {/* 头部 */}
      <div className="shrink-0 border-b border-border px-3 py-2.5">
        <div className="flex items-center gap-2">
          <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary/15 text-primary">
            <NotebookPen size={13} />
          </div>
          <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            笔记
          </span>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark px-2 py-2">
        {/* 全部笔记 */}
        <button
          type="button"
          onClick={() => setFolder(null)}
          className={cn(
            "group flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-[12px] transition-colors",
            !viewingTrash && selectedFolderId === null
              ? "bg-primary/12 text-primary font-medium"
              : "text-muted-foreground hover:bg-accent hover:text-foreground"
          )}
        >
          <Files size={13} className="shrink-0" />
          <span className="truncate">全部笔记</span>
          <span className="ml-auto text-[10px] text-muted-foreground/70">
            {stats.total}
          </span>
        </button>

        {/* 文件夹分组 */}
        <div className="mt-3">
          <div className="mb-1 flex items-center justify-between px-2">
            <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              文件夹
            </span>
            <button
              type="button"
              onClick={() => setShowNewFolder((v) => !v)}
              className="rounded p-0.5 text-muted-foreground hover:bg-accent hover:text-foreground"
              title="新建文件夹"
            >
              <FolderPlus size={11} />
            </button>
          </div>

          {showNewFolder && (
            <div className="mb-1.5 px-1">
              <Input
                value={newFolderName}
                onChange={(e) => setNewFolderName(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") void handleCreateFolder();
                  if (e.key === "Escape") {
                    setShowNewFolder(false);
                    setNewFolderName("");
                  }
                }}
                placeholder="文件夹名称…"
                className="h-7 text-[12px]"
                autoFocus
              />
            </div>
          )}

          <div className="space-y-0.5">
            {folders.length === 0 && !showNewFolder ? (
              <div className="px-2 py-1 text-[10px] text-muted-foreground/50">
                无文件夹
              </div>
            ) : (
              folders.map((f) => (
                <div
                  key={f.id}
                  className={cn(
                    "group flex items-center gap-2 rounded-md px-2 py-1.5 text-[12px] transition-colors cursor-pointer",
                    !viewingTrash && selectedFolderId === f.id
                      ? "bg-primary/12 text-primary font-medium"
                      : "text-muted-foreground hover:bg-accent hover:text-foreground"
                  )}
                  onClick={() => setFolder(f.id)}
                >
                  <FolderIcon size={13} className="shrink-0" />
                  <span className="min-w-0 flex-1 truncate">{f.name}</span>
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      if (confirm(`删除文件夹「${f.name}」？文件夹下笔记将移到根目录。`)) {
                        void deleteFolder(f.id);
                      }
                    }}
                    className="shrink-0 rounded p-0.5 opacity-0 transition-opacity group-hover:opacity-100 hover:bg-destructive/15 hover:text-destructive"
                    title="删除文件夹"
                  >
                    <X size={10} />
                  </button>
                </div>
              ))
            )}
          </div>
        </div>

        {/* 回收站 */}
        <div className="mt-3">
          <div className="mb-1 px-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            回收站
          </div>
          <button
            type="button"
            onClick={() => viewTrash(!viewingTrash)}
            className={cn(
              "group flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-[12px] transition-colors",
              viewingTrash
                ? "bg-primary/12 text-primary font-medium"
                : "text-muted-foreground hover:bg-accent hover:text-foreground"
            )}
          >
            <Trash2 size={13} className="shrink-0" />
            <span className="truncate">回收站</span>
            <span className="ml-auto text-[10px] text-muted-foreground/70">
              {stats.trash}
            </span>
          </button>
        </div>
      </div>

      {/* 底部状态 */}
      <div className="shrink-0 border-t border-border px-3 py-2">
        {toast ? (
          <div className="flex items-center gap-1.5 text-[10px] text-emerald-500">
            <CheckCircle2 size={10} />
            <span className="truncate">{toast}</span>
          </div>
        ) : error ? (
          <div className="flex items-center gap-1.5 text-[10px] text-destructive">
            <AlertCircle size={10} />
            <span className="truncate">{error}</span>
          </div>
        ) : (
          <div className="text-[10px] text-muted-foreground/60">
            共 {stats.total} 篇 · 回收站 {stats.trash}
          </div>
        )}
      </div>
    </div>
  );
}

// ===================== 中：笔记列表 =====================

function NoteList() {
  const items = useNotesStore((s) => s.items);
  const total = useNotesStore((s) => s.total);
  const loading = useNotesStore((s) => s.loading);
  const selectedId = useNotesStore((s) => s.selectedId);
  const viewingTrash = useNotesStore((s) => s.viewingTrash);
  const keyword = useNotesStore((s) => s.keyword);
  const setKeyword = useNotesStore((s) => s.setKeyword);
  const sortBy = useNotesStore((s) => s.sortBy);
  const sortDir = useNotesStore((s) => s.sortDir);
  const setSort = useNotesStore((s) => s.setSort);
  const select = useNotesStore((s) => s.select);
  const create = useNotesStore((s) => s.create);
  const refresh = useNotesStore((s) => s.refresh);

  // 搜索防抖
  useEffect(() => {
    const t = setTimeout(() => void refresh(), 300);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [keyword]);

  return (
    <div className="flex h-full w-full min-w-0 flex-col">
      {/* 头部 */}
      <div className="shrink-0 border-b border-border bg-card px-3 py-2.5">
        <div className="mb-2 flex items-center justify-between">
          <h1 className="text-sm font-semibold text-foreground">
            {viewingTrash ? "回收站" : "笔记列表"}
          </h1>
          {!viewingTrash && (
            <Button
              size="sm"
              className="h-7"
              onClick={() => void create()}
            >
              <Plus size={12} />
              新建
            </Button>
          )}
        </div>
        {/* 搜索框 */}
        {!viewingTrash && (
          <div className="relative">
            <Search
              size={12}
              className="absolute left-2 top-1/2 -translate-y-1/2 text-muted-foreground"
            />
            <Input
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              placeholder="搜索标题或正文…"
              className="h-8 pl-7 text-[12px]"
            />
          </div>
        )}
        {/* 排序 */}
        {!viewingTrash && (
          <div className="mt-2">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button
                  type="button"
                  className="flex h-6 items-center gap-1 rounded-md border border-border bg-background px-2 text-[10px] text-muted-foreground hover:bg-accent"
                >
                  <ChevronDown size={10} />
                  排序：{sortLabel(sortBy, sortDir)}
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="w-40">
                <DropdownMenuLabel>排序方式</DropdownMenuLabel>
                <DropdownMenuItem onClick={() => setSort("updated", "desc")}>
                  更新时间 · 新→旧
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setSort("updated", "asc")}>
                  更新时间 · 旧→新
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setSort("created", "desc")}>
                  创建时间 · 新→旧
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setSort("created", "asc")}>
                  创建时间 · 旧→新
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setSort("title", "asc")}>
                  标题 · A→Z
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setSort("title", "desc")}>
                  标题 · Z→A
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => setSort("wordCount", "desc")}>
                  字数 · 多→少
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        )}
      </div>

      {/* 列表 */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        {loading ? (
          <div className="flex h-full items-center justify-center">
            <Loader2 size={18} className="animate-spin text-muted-foreground" />
          </div>
        ) : items.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
              <Inbox size={22} />
            </div>
            <div className="space-y-1">
              <div className="text-sm font-medium text-foreground">
                {viewingTrash ? "回收站为空" : "暂无笔记"}
              </div>
              <div className="text-xs text-muted-foreground">
                {viewingTrash
                  ? "删除的笔记会出现在这里"
                  : "点击「新建」创建第一篇笔记"}
              </div>
            </div>
          </div>
        ) : (
          <ul className="space-y-0.5 p-2">
            {items.map((n) => (
              <NoteRow
                key={n.id}
                note={n}
                active={n.id === selectedId}
                onSelect={() => void select(n.id)}
                viewingTrash={viewingTrash}
              />
            ))}
          </ul>
        )}
      </div>

      {/* 底部计数 */}
      <div className="shrink-0 border-t border-border bg-card px-3 py-1.5">
        <span className="text-[10px] text-muted-foreground/70">
          共 {total} 篇
        </span>
      </div>
    </div>
  );
}

function NoteRow({
  note,
  active,
  onSelect,
  viewingTrash,
}: {
  note: { id: string; title: string; isPinned: boolean; wordCount: number; updatedAt: string; tags: string[]; assetCount: number };
  active: boolean;
  onSelect: () => void;
  viewingTrash: boolean;
}) {
  return (
    <li
      onClick={onSelect}
      className={cn(
        "group cursor-pointer rounded-md border px-2.5 py-2 transition-colors",
        active
          ? "border-primary/40 bg-primary/10"
          : "border-transparent bg-background hover:bg-accent/40"
      )}
    >
      <div className="flex items-start gap-1.5">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1">
            {note.isPinned && !viewingTrash && (
              <Pin size={9} className="shrink-0 text-primary" />
            )}
            <p className="truncate text-[12px] font-medium text-foreground">
              {note.title || "未命名"}
            </p>
          </div>
          {/* 元信息 */}
          <div className="mt-0.5 flex items-center gap-2 text-[10px] text-muted-foreground/70">
            <span>{formatDateTime(note.updatedAt)}</span>
            {note.wordCount > 0 && <span>{note.wordCount} 字</span>}
            {note.assetCount > 0 && (
              <span className="flex items-center gap-0.5">
                <Paperclip size={8} />
                {note.assetCount}
              </span>
            )}
          </div>
          {/* 标签 */}
          {note.tags && note.tags.length > 0 && (
            <div className="mt-1 flex flex-wrap gap-0.5">
              {note.tags.slice(0, 3).map((tag) => (
                <span
                  key={tag}
                  className="flex items-center gap-0.5 rounded bg-muted px-1 py-0.5 text-[9px] text-muted-foreground"
                >
                  <Hash size={7} />
                  {tag}
                </span>
              ))}
              {note.tags.length > 3 && (
                <span className="text-[9px] text-muted-foreground/50">
                  +{note.tags.length - 3}
                </span>
              )}
            </div>
          )}
        </div>
      </div>
    </li>
  );
}

// ===================== 右：编辑器 =====================

function EditorPanel() {
  const current = useNotesStore((s) => s.current);
  const selectedId = useNotesStore((s) => s.selectedId);
  const viewingTrash = useNotesStore((s) => s.viewingTrash);

  if (!current || !selectedId) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center gap-3 text-center">
        <div className="flex h-14 w-14 items-center justify-center rounded-full bg-muted text-muted-foreground">
          <FileText size={26} />
        </div>
        <div className="space-y-1">
          <div className="text-sm font-medium text-foreground">
            {viewingTrash ? "选择回收站中的笔记" : "选择一篇笔记"}
          </div>
          <div className="text-xs text-muted-foreground">
            {viewingTrash
              ? "可恢复或永久删除"
              : "或点击「新建」创建新笔记"}
          </div>
        </div>
      </div>
    );
  }

  if (viewingTrash) {
    return <TrashDetail noteId={selectedId} title={current.title} />;
  }

  return <NoteEditor key={selectedId} />;
}

// —— 回收站详情 ——
function TrashDetail({ noteId, title }: { noteId: string; title: string }) {
  const restore = useNotesStore((s) => s.restore);
  const hardDelete = useNotesStore((s) => s.hardDelete);
  const emptyTrash = useNotesStore((s) => s.emptyTrash);

  return (
    <div className="flex h-full w-full flex-col items-center justify-center gap-4 text-center">
      <div className="flex h-14 w-14 items-center justify-center rounded-full bg-destructive/10 text-destructive">
        <Trash2 size={26} />
      </div>
      <div className="space-y-1">
        <div className="text-sm font-medium text-foreground">{title}</div>
        <div className="text-xs text-muted-foreground">
          此笔记在回收站中，可恢复或永久删除
        </div>
      </div>
      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm" onClick={() => void restore(noteId)}>
          <RotateCcw size={12} />
          恢复
        </Button>
        <Button
          variant="destructive"
          size="sm"
          onClick={() => {
            if (confirm(`永久删除「${title}」？此操作不可撤销。`)) {
              void hardDelete(noteId);
            }
          }}
        >
          <Trash2 size={12} />
          永久删除
        </Button>
      </div>
      <button
        type="button"
        onClick={() => {
          if (confirm("清空整个回收站？此操作不可撤销。")) {
            void emptyTrash();
          }
        }}
        className="mt-2 text-[10px] text-muted-foreground hover:text-destructive"
      >
        清空回收站
      </button>
    </div>
  );
}

// —— 笔记编辑器 ——
function NoteEditor() {
  const current = useNotesStore((s) => s.current);
  const draftTitle = useNotesStore((s) => s.draftTitle);
  const draftContent = useNotesStore((s) => s.draftContent);
  const draftTags = useNotesStore((s) => s.draftTags);
  const dirty = useNotesStore((s) => s.dirty);
  const saving = useNotesStore((s) => s.saving);
  const setDraftTitle = useNotesStore((s) => s.setDraftTitle);
  const setDraftContent = useNotesStore((s) => s.setDraftContent);
  const setDraftTags = useNotesStore((s) => s.setDraftTags);
  const save = useNotesStore((s) => s.save);
  const moveToTrash = useNotesStore((s) => s.moveToTrash);
  const togglePinned = useNotesStore((s) => s.togglePinned);
  const exportNote = useNotesStore((s) => s.exportNote);
  const importMarkdown = useNotesStore((s) => s.importMarkdown);
  const [tagInput, setTagInput] = useState("");
  const [showImport, setShowImport] = useState(false);

  // Ctrl+S 保存
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "s") {
        e.preventDefault();
        if (dirty) void save();
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [dirty, save]);

  if (!current) return null;

  const addTag = () => {
    const t = tagInput.trim();
    if (!t || draftTags.includes(t)) return;
    setDraftTags([...draftTags, t]);
    setTagInput("");
  };

  const removeTag = (tag: string) => {
    setDraftTags(draftTags.filter((t) => t !== tag));
  };

  const handleExport = async (format: ExportFormat) => {
    const content = await exportNote(current.id, format);
    if (!content) return;
    const ext = format === "markdown" ? "md" : format === "html" ? "html" : "json";
    const blob = new Blob([content], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${current.title || "note"}.${ext}`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="flex h-full w-full min-w-0 flex-col">
      {/* 工具栏 */}
      <div className="shrink-0 border-b border-border bg-card px-4 py-2">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground/70">
            <span>{formatDateTime(current.updatedAt)}</span>
            <span>·</span>
            <span>{current.wordCount} 字</span>
            <span>·</span>
            <span>{current.charCount} 字符</span>
            {dirty && (
              <Badge variant="warning" className="ml-2 px-1.5 py-0 text-[9px]">
                未保存
              </Badge>
            )}
          </div>
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              onClick={() => void togglePinned(current.id, !current.isPinned)}
              title={current.isPinned ? "取消置顶" : "置顶"}
            >
              {current.isPinned ? (
                <PinOff size={13} className="text-primary" />
              ) : (
                <Pin size={13} />
              )}
            </Button>

            {/* 导出菜单 */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7"
                  title="导出"
                >
                  <FileDown size={13} />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-36">
                <DropdownMenuLabel>导出格式</DropdownMenuLabel>
                <DropdownMenuItem onClick={() => void handleExport("markdown")}>
                  <Download size={11} />
                  Markdown (.md)
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => void handleExport("html")}>
                  <Download size={11} />
                  HTML (.html)
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => void handleExport("json")}>
                  <Download size={11} />
                  JSON (.json)
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            {/* 更多操作 */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7"
                  title="更多"
                >
                  <MoreVertical size={13} />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-36">
                <DropdownMenuItem onClick={() => setShowImport(true)}>
                  <Upload size={11} />
                  导入 Markdown
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => {
                    if (confirm(`将「${current.title}」移入回收站？`)) {
                      void moveToTrash(current.id);
                    }
                  }}
                  className="text-destructive focus:text-destructive"
                >
                  <Trash2 size={11} />
                  移入回收站
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            <Button
              size="sm"
              className="h-7"
              disabled={!dirty || saving}
              onClick={() => void save()}
            >
              {saving ? (
                <Loader2 size={12} className="animate-spin" />
              ) : (
                <Save size={12} />
              )}
              保存
            </Button>
          </div>
        </div>
      </div>

      {/* 编辑区 */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        <div className="mx-auto max-w-3xl px-6 py-5">
          {/* 标题 */}
          <input
            type="text"
            value={draftTitle}
            onChange={(e) => setDraftTitle(e.target.value)}
            placeholder="笔记标题…"
            className="w-full bg-transparent text-xl font-semibold text-foreground outline-none placeholder:text-muted-foreground/40"
          />

          {/* 标签 */}
          <div className="mt-2 flex flex-wrap items-center gap-1.5">
            {draftTags.map((tag) => (
              <span
                key={tag}
                className="flex items-center gap-0.5 rounded bg-primary/10 px-1.5 py-0.5 text-[10px] text-primary"
              >
                <Hash size={8} />
                {tag}
                <button
                  type="button"
                  onClick={() => removeTag(tag)}
                  className="ml-0.5 rounded p-0.5 hover:bg-primary/20"
                >
                  <X size={8} />
                </button>
              </span>
            ))}
            <input
              type="text"
              value={tagInput}
              onChange={(e) => setTagInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === ",") {
                  e.preventDefault();
                  addTag();
                }
              }}
              onBlur={() => tagInput.trim() && addTag()}
              placeholder="添加标签…"
              className="w-20 bg-transparent text-[10px] text-foreground outline-none placeholder:text-muted-foreground/40"
            />
          </div>

          {/* 正文 */}
          <Textarea
            value={draftContent}
            onChange={(e) => setDraftContent(e.target.value)}
            placeholder="开始书写 Markdown…"
            className="mt-4 min-h-[400px] resize-none border-0 bg-transparent p-0 font-mono text-[13px] leading-relaxed text-foreground/90 focus-visible:ring-0"
          />
        </div>
      </div>

      {/* 底部：资产 */}
      <AssetBar noteId={current.id} />

      {/* 导入对话框 */}
      {showImport && (
        <ImportDialog
          onClose={() => setShowImport(false)}
          onImport={async (filename, content) => {
            await importMarkdown(filename, content);
            setShowImport(false);
          }}
        />
      )}
    </div>
  );
}

// —— 资产栏 ——
function AssetBar({ noteId }: { noteId: string }) {
  const assets = useNotesStore((s) => s.assets);
  const addAsset = useNotesStore((s) => s.addAsset);
  const deleteAsset = useNotesStore((s) => s.deleteAsset);
  const [showFileInput, setShowFileInput] = useState(false);

  const handleFile = async (file: File) => {
    const buf = await file.arrayBuffer();
    const data = Array.from(new Uint8Array(buf));
    await addAsset(noteId, file.name, data, file.type || "application/octet-stream");
  };

  return (
    <div className="shrink-0 border-t border-border bg-card px-4 py-2">
      <div className="flex items-center gap-2">
        <div className="flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
          <Paperclip size={10} />
          附件 ({assets.length})
        </div>
        <button
          type="button"
          onClick={() => setShowFileInput(true)}
          className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
          title="添加附件"
        >
          <Plus size={11} />
        </button>
        {showFileInput && (
          <input
            type="file"
            autoFocus
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (f) void handleFile(f);
              setShowFileInput(false);
              e.target.value = "";
            }}
            onBlur={() => setShowFileInput(false)}
            className="hidden"
          />
        )}
        <div className="ml-auto flex flex-wrap items-center gap-1.5">
          {assets.length === 0 ? (
            <span className="text-[10px] text-muted-foreground/50">无附件</span>
          ) : (
            assets.map((a) => (
              <div
                key={a.id}
                className="group flex items-center gap-1 rounded border border-border bg-background px-1.5 py-0.5 text-[10px] text-muted-foreground"
              >
                <FileText size={9} />
                <span className="max-w-32 truncate">{a.filename}</span>
                <span className="text-muted-foreground/60">
                  {formatSize(a.size)}
                </span>
                <button
                  type="button"
                  onClick={() => void deleteAsset(a.id)}
                  className="ml-0.5 rounded p-0.5 opacity-0 transition-opacity group-hover:opacity-100 hover:bg-destructive/15 hover:text-destructive"
                >
                  <X size={8} />
                </button>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}

// —— 导入 Markdown 对话框 ——
function ImportDialog({
  onClose,
  onImport,
}: {
  onClose: () => void;
  onImport: (filename: string, content: string) => Promise<void>;
}) {
  const [filename, setFilename] = useState("imported.md");
  const [content, setContent] = useState("");
  const [busy, setBusy] = useState(false);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
      <div className="w-full max-w-xl rounded-dialog border border-border bg-card p-6 shadow-floating">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-foreground">
            导入 Markdown
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
          >
            <X size={14} />
          </button>
        </div>
        <div className="space-y-3">
          <div>
            <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              文件名
            </label>
            <Input
              value={filename}
              onChange={(e) => setFilename(e.target.value)}
              className="h-8 text-[12px]"
            />
          </div>
          <div>
            <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              Markdown 内容
            </label>
            <Textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              rows={8}
              placeholder="# 标题&#10;&#10;正文内容…"
              className="resize-none font-mono text-[12px]"
            />
          </div>
        </div>
        <div className="mt-4 flex items-center justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>
            取消
          </Button>
          <Button
            size="sm"
            disabled={!content.trim() || busy}
            onClick={async () => {
              setBusy(true);
              try {
                await onImport(filename || "imported.md", content);
              } finally {
                setBusy(false);
              }
            }}
          >
            {busy ? (
              <Loader2 size={12} className="animate-spin" />
            ) : (
              <Upload size={12} />
            )}
            导入
          </Button>
        </div>
      </div>
    </div>
  );
}

// —— 工具 ——
function sortLabel(by: SortBy, dir: SortDir): string {
  const dirLabel = dir === "asc" ? "升序" : "降序";
  const byLabel: Record<SortBy, string> = {
    updated: "更新时间",
    created: "创建时间",
    title: "标题",
    wordCount: "字数",
  };
  return `${byLabel[by]} · ${dirLabel}`;
}
