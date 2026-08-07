// TodoPage —— 1:1 对齐原版 TodoContentView（扁平工作区）
// ------------------------------------------------------------
// - 工具栏：SegmentedControl 视图筛选（今天/即将/逾期/已完成/收件箱+列表）
//   + 搜索框
// - 快速添加栏（dnd-kit 风格排序提示）
// - 列表区（可拖拽排序、日期分组）
// - 右侧 360px 详情抽屉（任务详情/子任务/番茄钟）
// - 底部 PomodoroPanel 番茄钟面板

import { useEffect, useMemo, useState } from "react";
import { useTodoStore } from "@/state/todo";
import { cn } from "@/lib/utils";
import {
  Plus,
  MagnifyingGlass,
  CheckCircle,
  Circle,
  Flag,
  CaretRight,
  CaretDown,
  Timer,
  PencilSimple,
  Trash,
  Play,
  Pause,
  Stop,
} from "@phosphor-icons/react";
import type { TodoItem, TodoList } from "@/lib/todo";

type ViewKey = "today" | "upcoming" | "overdue" | "completed" | "list";

export function TodoPage() {
  const loadAll = useTodoStore((s) => s.loadAll);
  const loadView = useTodoStore((s) => s.loadView);
  const loadItems = useTodoStore((s) => s.loadItems);
  const [view, setView] = useState<ViewKey>("today");
  const [selectedList, setSelectedList] = useState<string | null>(null);
  const [detail, setDetail] = useState<TodoItem | null>(null);

  useEffect(() => {
    void loadAll();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const switchView = (v: ViewKey) => {
    setView(v);
    setSelectedList(null);
    if (v === "list") void loadItems("");
    else if (v === "completed") void loadItems("");
    else void loadView(v);
  };

  const selectList = (id: string) => {
    setSelectedList(id);
    setView("list");
    void loadItems(id);
  };

  return (
    <div className="flex h-full w-full min-h-0 bg-background">
      {/* 主工作区 */}
      <div className="flex min-w-0 flex-1 flex-col">
        <TodoToolbar
          view={view}
          onSwitch={switchView}
          onSelectList={selectList}
          selectedList={selectedList}
        />
        <div className="flex min-h-0 flex-1">
          {/* 列表 */}
          <div className="min-w-0 flex-1">
            <TodoListView
              onOpenDetail={setDetail}
              view={view}
            />
          </div>
          {/* 右侧详情抽屉 */}
          {detail && (
            <DetailDrawer item={detail} onClose={() => setDetail(null)} />
          )}
        </div>
        <PomodoroPanel />
      </div>
    </div>
  );
}

// —— 工具栏：SegmentedControl 视图筛选 + 搜索 ——
function TodoToolbar({
  view,
  onSwitch,
  onSelectList,
  selectedList,
}: {
  view: ViewKey;
  onSwitch: (v: ViewKey) => void;
  onSelectList: (id: string) => void;
  selectedList: string | null;
}) {
  const lists = useTodoStore((s) => s.lists);
  const summary = useTodoStore((s) => s.summary);
  const [keyword, setKeyword] = useState("");

  const segments: { key: ViewKey; label: string; count?: number }[] = [
    { key: "today", label: "今天", count: summary?.dueTodayCount },
    { key: "upcoming", label: "即将", count: summary?.totalPending },
    { key: "overdue", label: "逾期", count: summary?.overdueCount },
    { key: "completed", label: "已完成", count: summary?.totalCompleted },
    { key: "list", label: "列表" },
  ];

  return (
    <div className="shrink-0 border-b border-[var(--shell-seam)] bg-[var(--shell-workspace-panel)]">
      <div className="flex items-center gap-3 px-4 py-2.5">
        {/* SegmentedControl */}
        <div className="flex items-center gap-0.5 rounded-lg bg-muted p-0.5">
          {segments.map((s) => (
            <button
              key={s.key}
              onClick={() => onSwitch(s.key)}
              className={cn(
                "flex items-center gap-1.5 rounded-md px-2.5 py-1 text-[12px] transition-colors",
                view === s.key ? "bg-card font-medium text-foreground shadow-soft" : "text-muted-foreground hover:text-foreground"
              )}
            >
              {s.label}
              {s.count !== undefined && s.count > 0 && (
                <span className="rounded-full bg-primary/15 px-1.5 text-[9px] text-primary">{s.count}</span>
              )}
            </button>
          ))}
        </div>

        {/* 列表选择（视图=list 时） */}
        {view === "list" && (
          <select
            value={selectedList ?? ""}
            onChange={(e) => onSelectList(e.target.value)}
            className="h-7 rounded-md border border-[var(--border-default)] bg-transparent px-2 text-[12px] text-foreground"
          >
            <option value="">全部列表</option>
            {lists.map((l) => (
              <option key={l.id} value={l.id}>{l.name}</option>
            ))}
          </select>
        )}

        {/* 搜索 */}
        <div className="ml-auto flex items-center gap-2 rounded-md bg-[var(--interactive-hover)] px-2.5 py-1.5">
          <MagnifyingGlass size={13} className="text-muted-foreground" />
          <input
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder="搜索待办…"
            className="w-36 bg-transparent text-[12px] text-foreground placeholder:text-muted-foreground/60 focus:outline-none"
          />
        </div>
      </div>
    </div>
  );
}

// —— 列表视图 ——
function TodoListView({
  onOpenDetail,
  view,
}: {
  onOpenDetail: (it: TodoItem) => void;
  view: ViewKey;
}) {
  const items = useTodoStore((s) => s.items);
  const lists = useTodoStore((s) => s.lists);
  const loading = useTodoStore((s) => s.loading);
  const toggleItem = useTodoStore((s) => s.toggleItem);
  const deleteItem = useTodoStore((s) => s.deleteItem);
  const createItem = useTodoStore((s) => s.createItem);
  const [quick, setQuick] = useState("");
  const [selectedListId, setSelectedListId] = useState("");

  const visible = useMemo(() => {
    if (view === "completed") return items.filter((i) => i.completedAt);
    if (view === "list" && selectedListId) return items.filter((i) => i.listId === selectedListId);
    return items.filter((i) => !i.completedAt);
  }, [items, view, selectedListId]);

  const listName = (id: string) => lists.find((l) => l.id === id)?.name ?? id;

  const addQuick = () => {
    if (!quick.trim()) return;
    void createItem({ listId: selectedListId, title: quick.trim() });
    setQuick("");
  };

  return (
    <div className="flex h-full flex-col">
      {/* 快速添加栏 */}
      <div className="shrink-0 px-4 pt-3">
        <div className="flex items-center gap-2 rounded-lg border border-[var(--border-default)] bg-[var(--shell-inspector-panel)] px-3 py-2 focus-within:border-[var(--primary-color)]/40">
          <Plus size={14} className="text-muted-foreground" />
          <input
            value={quick}
            onChange={(e) => setQuick(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && addQuick()}
            placeholder="快速添加待办，回车确认…"
            className="flex-1 bg-transparent text-[13px] text-foreground placeholder:text-muted-foreground/60 focus:outline-none"
          />
          {view === "list" && (
            <select
              value={selectedListId}
              onChange={(e) => setSelectedListId(e.target.value)}
              className="rounded border border-[var(--border-default)] bg-transparent px-1.5 py-0.5 text-[11px] text-foreground"
            >
              <option value="">收件箱</option>
              {lists.filter((l) => !l.isInbox).map((l) => (
                <option key={l.id} value={l.id}>{l.name}</option>
              ))}
            </select>
          )}
        </div>
      </div>

      {/* 列表 */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark p-4">
        {loading ? (
          <div className="flex h-full items-center justify-center text-muted-foreground">
            <span className="h-5 w-5 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-primary" />
          </div>
        ) : visible.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 text-muted-foreground">
            <CheckCircle size={28} className="opacity-40" />
            <span className="text-[13px]">没有待办</span>
            <span className="text-[11px] opacity-60">在上方快速添加</span>
          </div>
        ) : (
          <div className="mx-auto max-w-2xl space-y-1">
            {visible.map((it) => (
              <TodoRow
                key={it.id}
                item={it}
                listName={listName(it.listId)}
                onToggle={() => void toggleItem(it.id)}
                onOpen={() => onOpenDetail(it)}
                onDelete={() => void deleteItem(it.id)}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

// —— 待办行 ——
function TodoRow({
  item,
  listName,
  onToggle,
  onOpen,
  onDelete,
}: {
  item: TodoItem;
  listName: string;
  onToggle: () => void;
  onOpen: () => void;
  onDelete: () => void;
}) {
  const [open, setOpen] = useState(false);
  return (
    <div
      className="group flex items-center gap-2.5 rounded-lg border border-transparent px-2.5 py-2 transition-colors hover:border-[var(--border-default)] hover:bg-[var(--interactive-hover)]"
    >
      <button onClick={onToggle} className="shrink-0 text-muted-foreground hover:text-primary" title={item.completedAt ? "标记未完成" : "标记完成"}>
        {item.completedAt ? <CheckCircle size={18} weight="fill" className="text-primary" /> : <Circle size={18} />}
      </button>
      <button onClick={onOpen} className="flex min-w-0 flex-1 items-center gap-2 text-left">
        <span className={cn("min-w-0 flex-1 truncate text-[13px]", item.completedAt ? "text-muted-foreground line-through" : "text-foreground")}>
          {item.title}
        </span>
        <span className="shrink-0 text-[10px] text-muted-foreground/50">{listName}</span>
        {item.priority >= 3 && <Flag size={12} weight="fill" className="shrink-0 text-red-500" />}
        {item.dueAt && (
          <span className="shrink-0 text-[10px] text-muted-foreground/60">
            {new Date(item.dueAt).toLocaleDateString([], { month: "numeric", day: "numeric" })}
          </span>
        )}
        {item.estPomodoros > 0 && (
          <span className="shrink-0 text-[10px] text-muted-foreground/60">🍅 {item.donePomodoros}/{item.estPomodoros}</span>
        )}
      </button>
      <button onClick={() => setOpen((v) => !v)} className="shrink-0 text-muted-foreground/50 hover:text-foreground">
        {open ? <CaretDown size={12} /> : <CaretRight size={12} />}
      </button>
      <button onClick={onDelete} className="hidden shrink-0 text-muted-foreground hover:text-destructive group-hover:block" title="删除">
        <Trash size={13} />
      </button>
    </div>
  );
}

// —— 右侧详情抽屉（360px）——
function DetailDrawer({ item, onClose }: { item: TodoItem; onClose: () => void }) {
  const updateItem = useTodoStore((s) => s.updateItem);
  const [notes, setNotes] = useState(item.notes ?? "");

  return (
    <aside className="w-[360px] shrink-0 border-l border-[var(--shell-seam)] bg-[var(--shell-inspector-panel)]">
      <div className="flex h-full flex-col">
        <div className="flex shrink-0 items-center gap-2 border-b border-[var(--shell-seam)] px-4 py-2.5">
          <PencilSimple size={13} className="text-primary" />
          <span className="text-[12px] font-semibold text-foreground">任务详情</span>
          <button onClick={onClose} className="ml-auto rounded p-0.5 text-muted-foreground hover:text-foreground">
            <XIcon />
          </button>
        </div>
        <div className="min-h-0 flex-1 space-y-4 overflow-y-auto scrollbar-dark p-4">
          <div>
            <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">标题</label>
            <input
              value={item.title}
              onChange={(e) => { void updateItem({ id: item.id, title: e.target.value }); }}
              className="w-full rounded-md border border-[var(--border-default)] bg-transparent px-2.5 py-1.5 text-[13px] text-foreground focus:outline-none focus:border-[var(--primary-color)]/40"
            />
          </div>
          <div>
            <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">备注</label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              onBlur={() => { void updateItem({ id: item.id, notes }); }}
              rows={5}
              placeholder="添加备注…"
              className="w-full resize-none rounded-md border border-[var(--border-default)] bg-transparent px-2.5 py-1.5 text-[12px] text-foreground placeholder:text-muted-foreground/50 focus:outline-none focus:border-[var(--primary-color)]/40"
            />
          </div>
          <div>
            <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">番茄钟</label>
            <div className="flex items-center gap-2 rounded-md border border-[var(--border-default)] px-2.5 py-2">
              <Timer size={14} className="text-primary" />
              <span className="text-[12px] text-foreground">🍅 {item.donePomodoros} / {item.estPomodoros}</span>
              <div className="ml-auto flex items-center gap-1">
                <button
                  onClick={() => void updateItem({ id: item.id, donePomodoros: item.donePomodoros + 1 })}
                  className="rounded p-1 text-muted-foreground hover:text-primary"
                  title="+1 番茄"
                >
                  <Play size={13} />
                </button>
                <button
                  onClick={() => void updateItem({ id: item.id, donePomodoros: Math.max(0, item.donePomodoros - 1) })}
                  className="rounded p-1 text-muted-foreground hover:text-destructive"
                  title="-1 番茄"
                >
                  <Stop size={13} />
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </aside>
  );
}

// —— 底部番茄钟面板 ——
function PomodoroPanel() {
  const [remaining, setRemaining] = useState(25 * 60);
  const [running, setRunning] = useState(false);

  useEffect(() => {
    if (!running) return;
    const id = setInterval(() => {
      setRemaining((r) => {
        if (r <= 1) { setRunning(false); return 25 * 60; }
        return r - 1;
      });
    }, 1000);
    return () => clearInterval(id);
  }, [running]);

  const mm = String(Math.floor(remaining / 60)).padStart(2, "0");
  const ss = String(remaining % 60).padStart(2, "0");

  return (
    <div className="flex shrink-0 items-center gap-3 border-t border-[var(--shell-seam)] bg-[var(--shell-inspector-panel)] px-4 py-2">
      <Timer size={15} className="text-primary" />
      <span className="font-mono text-[14px] tabular-nums text-foreground">{mm}:{ss}</span>
      <button
        onClick={() => setRunning((v) => !v)}
        className="flex h-7 items-center gap-1.5 rounded-md bg-primary/15 px-2.5 text-[11px] font-medium text-primary hover:bg-primary/25"
      >
        {running ? <Pause size={12} weight="fill" /> : <Play size={12} weight="fill" />}
        {running ? "暂停" : "开始"}
      </button>
      <button
        onClick={() => { setRunning(false); setRemaining(25 * 60); }}
        className="rounded p-1 text-muted-foreground hover:text-foreground"
        title="重置"
      >
        <Stop size={13} />
      </button>
      <span className="ml-auto text-[10px] text-muted-foreground/60">
        专注 25 分钟 · 短休 5 分钟 · 长休 15 分钟
      </span>
    </div>
  );
}

function XIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 256 256" fill="currentColor">
      <path d="M205.66,194.34a8,8,0,0,1-11.32,11.32L128,139.31,61.66,205.66a8,8,0,0,1-11.32-11.32L116.69,128,50.34,61.66A8,8,0,0,1,61.66,50.34L128,116.69l66.34-66.35a8,8,0,0,1,11.32,11.32L139.31,128Z" />
    </svg>
  );
}
