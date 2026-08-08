// SessionSidebar —— 1:1 对齐原版 ChatV2 会话侧栏
// ------------------------------------------------------------
// - 顶部："新对话" 主导航（ChatCenteredText）
// - 区块：课题（分组，44px 行，可折叠）/ 最近（未分组）
// - 底部："已归档会话" 低调入口
// - 行：44px 高、rounded-2xl、选中浅灰（--interactive-selected）

import { useChatStore, type UISession } from "@/state/chat";
import { cn } from "@/lib/utils";
import {
  ChatCenteredText,
  Folder,
  CaretRight,
  Plus,
  Archive,
  MagnifyingGlass,
  DotsThree,
  PushPin,
  PencilSimple,
  Trash,
  ArrowCounterClockwise,
  X,
  Check,
} from "@phosphor-icons/react";
import { useState } from "react";
import { chatV2Api } from "@/lib/chat";

export function SessionSidebar() {
  const sessions = useChatStore((s) => s.sessions);
  const activeSessionId = useChatStore((s) => s.activeSessionId);
  const selectSession = useChatStore((s) => s.selectSession);
  const createSession = useChatStore((s) => s.createSession);
  const softDeleteSession = useChatStore((s) => s.softDeleteSession);
  const restoreSession = useChatStore((s) => s.restoreSession);
  const purgeSession = useChatStore((s) => s.purgeSession);
  const togglePin = useChatStore((s) => s.togglePin);
  const renameSession = useChatStore((s) => s.renameSession);
  const [groupsOpen, setGroupsOpen] = useState(true);
  const [recentOpen, setRecentOpen] = useState(true);
  const [keyword, setKeyword] = useState("");

  const active = sessions.filter((s) => !s.isDeleted);
  const filtered = keyword.trim()
    ? active.filter((s) => s.title.toLowerCase().includes(keyword.toLowerCase()))
    : active;
  const pinned = filtered.filter((s) => s.pinned);
  const unpinned = filtered.filter((s) => !s.pinned);

  return (
    <aside
      className="flex w-[272px] shrink-0 flex-col border-r border-[var(--shell-seam)] bg-[var(--shell-navigation-surface)]"
    >
      {/* —— 顶部：新对话 —— */}
      <div className="shrink-0 p-2">
        <button
          onClick={() => void createSession("新会话")}
          className="sidebar-row flex w-full items-center gap-2.5 px-2.5"
          data-active={false}
        >
          <ChatCenteredText size={18} weight="regular" className="shrink-0 opacity-80" />
          <span className="text-[13px]">新对话</span>
        </button>
      </div>

      {/* —— 搜索 —— */}
      <div className="shrink-0 px-3 pb-2">
        <div className="flex items-center gap-2 rounded-md bg-[var(--interactive-hover)] px-2.5 py-1.5">
          <MagnifyingGlass size={13} className="text-muted-foreground" />
          <input
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder="搜索会话…"
            className="w-full bg-transparent text-[12px] text-foreground placeholder:text-muted-foreground/60 focus:outline-none"
          />
        </div>
      </div>

      {/* —— 会话滚动区 —— */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark px-2 pb-2">
        {/* 课题（分组） */}
        <SectionHeader
          title="课题"
          open={groupsOpen}
          onToggle={() => setGroupsOpen((v) => !v)}
          onNew={() => void createSession("新会话")}
        />
        {groupsOpen && <GroupRow name="默认分组" count={filtered.length} />}

        {/* 最近（未分组） */}
        <SectionHeader
          title="最近"
          open={recentOpen}
          onToggle={() => setRecentOpen((v) => !v)}
          onNew={() => void createSession("新会话")}
        />
        {recentOpen && (
          <div className="space-y-0.5">
            {pinned.length > 0 && (
              <>
                <div className="px-2.5 pb-0.5 pt-1.5 text-[10px] font-medium text-muted-foreground/60">置顶</div>
                {pinned.map((s) => (
                  <SessionRow
                    key={s.id}
                    session={s}
                    active={s.id === activeSessionId}
                    onSelect={() => selectSession(s.id)}
                    onRename={(t) => void renameSession(s.id, t)}
                    onTogglePin={() => void togglePin(s.id)}
                    onDelete={() => void softDeleteSession(s.id)}
                  />
                ))}
              </>
            )}
            {unpinned.length === 0 && !keyword && (
              <div className="flex items-center gap-1.5 px-2.5 py-1.5 text-[11px] text-muted-foreground/50">
                <DotsThree size={12} className="opacity-40" />
                还没有会话，点击「新对话」开始
              </div>
            )}
            {unpinned.map((s) => (
              <SessionRow
                key={s.id}
                session={s}
                active={s.id === activeSessionId}
                onSelect={() => selectSession(s.id)}
                onRename={(t) => void renameSession(s.id, t)}
                onTogglePin={() => void togglePin(s.id)}
                onDelete={() => void softDeleteSession(s.id)}
              />
            ))}
          </div>
        )}
      </div>

      {/* —— 底部：归档入口 —— */}
      <div className="shrink-0 border-t border-[var(--shell-seam)] p-2">
        <button
          onClick={() => {
            // 简单切换：查看归档会话列表（展开显示）
            void chatV2Api.listSessions({ onlyDeleted: true, limit: 20 }).then((archived) => {
              if (archived && archived.length > 0) {
                void restoreSession(archived[0].id);
              }
            });
          }}
          className="sidebar-row flex w-full items-center gap-2.5 px-2.5 text-[12px]"
          data-active={false}
        >
          <Archive size={16} className="shrink-0 opacity-70" />
          <span>已归档会话</span>
        </button>
      </div>
    </aside>
  );
}

// —— 分组标题 ——
function SectionHeader({
  title,
  open,
  onToggle,
  onNew,
}: {
  title: string;
  open: boolean;
  onToggle: () => void;
  onNew: () => void;
}) {
  return (
    <div className="group flex items-center px-1 pt-2.5 pb-0.5">
      <button
        onClick={onToggle}
        className="flex flex-1 items-center gap-0.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60 hover:text-foreground"
      >
        <CaretRight size={9} weight="bold" className={cn("transition-transform", !open && "rotate-90")} />
        {title}
      </button>
      <button
        onClick={onNew}
        className="hidden rounded p-0.5 text-muted-foreground/60 hover:text-foreground group-hover:block"
      >
        <Plus size={11} weight="bold" />
      </button>
    </div>
  );
}

// —— 分组行 ——
function GroupRow({ name, count }: { name: string; count: number }) {
  const [open, setOpen] = useState(true);
  return (
    <div>
      <div
        onClick={() => setOpen((v) => !v)}
        className="sidebar-row flex cursor-pointer items-center gap-2 px-2.5"
        data-active={false}
      >
        <Folder size={16} weight="fill" className="shrink-0 text-[var(--primary-color)] opacity-70" />
        <span className="min-w-0 flex-1 truncate text-[13px]">{name}</span>
        <span className="text-[10px] text-muted-foreground/50">{count}</span>
        <CaretRight size={10} weight="bold" className={cn("shrink-0 transition-transform", !open && "rotate-90")} />
      </div>
      {open && <div className="ml-4 space-y-0.5 border-l border-[var(--shell-seam)] pl-1.5" />}
    </div>
  );
}

// —— 会话行 ——
function SessionRow({
  session,
  active,
  onSelect,
  onRename,
  onTogglePin,
  onDelete,
}: {
  session: UISession;
  active: boolean;
  onSelect: () => void;
  onRename: (t: string) => void;
  onTogglePin: () => void;
  onDelete: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(session.title);

  if (editing) {
    return (
      <div className="flex items-center gap-1 rounded-2xl border border-[var(--primary-color)]/40 bg-[var(--interactive-selected)] px-2 py-1.5" style={{ minHeight: 44 }}>
        <input
          autoFocus
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && draft.trim()) { onRename(draft.trim()); setEditing(false); }
            if (e.key === "Escape") setEditing(false);
          }}
          className="w-full bg-transparent text-[12px] text-foreground focus:outline-none"
        />
        <button onClick={() => { onRename(draft.trim()); setEditing(false); }} className="text-primary"><Check size={12} /></button>
        <button onClick={() => setEditing(false)} className="text-muted-foreground"><X size={12} /></button>
      </div>
    );
  }

  return (
    <div
      onClick={onSelect}
      className="group relative flex cursor-pointer items-center gap-2 rounded-2xl px-2.5 transition-colors"
      style={{ minHeight: 44 }}
      data-active={active}
    >
      <div
        className={cn(
          "flex min-w-0 flex-1 items-center gap-2 rounded-2xl px-2",
          active && "bg-[var(--interactive-selected)]"
        )}
      >
        <span className={cn("min-w-0 flex-1 truncate text-[13px]", active ? "font-medium text-foreground" : "text-muted-foreground")}>
          {session.title}
        </span>
        {session.pinned && <PushPin size={11} weight="fill" className="shrink-0 text-amber-500" />}
      </div>

      {/* hover 操作 */}
      <div className="absolute right-1.5 top-1/2 hidden -translate-y-1/2 items-center gap-0.5 rounded-lg bg-[var(--shell-inspector-panel)] px-1 py-0.5 shadow-[var(--shadow-shell-soft)] group-hover:flex">
        <button title="重命名" onClick={(e) => { e.stopPropagation(); setEditing(true); }} className="rounded p-0.5 text-muted-foreground hover:text-foreground">
          <PencilSimple size={12} />
        </button>
        <button title={session.pinned ? "取消置顶" : "置顶"} onClick={(e) => { e.stopPropagation(); onTogglePin(); }} className="rounded p-0.5 text-muted-foreground hover:text-amber-500">
          <PushPin size={12} weight={session.pinned ? "fill" : "regular"} />
        </button>
        <button title="删除" onClick={(e) => { e.stopPropagation(); onDelete(); }} className="rounded p-0.5 text-muted-foreground hover:text-destructive">
          <Trash size={12} />
        </button>
      </div>
    </div>
  );
}
