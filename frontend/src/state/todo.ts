// Todo Store —— 待办状态管理
// ------------------------------------------------------------
// 对接后端 todoApi（TodoXxx 方法）。
// 支持：列表 CRUD / 条目 CRUD / 子任务 / 完成切换 / 视图 / 回收站 / AI 拆解。

import { create } from "zustand";
import {
  todoApi,
  type TodoList,
  type TodoItem,
  type TodoSummary,
  type CreateListParams,
  type UpdateListParams,
  type CreateItemParams,
  type UpdateItemParams,
} from "@/lib/todo";

type View = "today" | "overdue" | "upcoming" | "all" | "list" | "trash";

interface TodoState {
  // 数据
  lists: TodoList[];
  items: TodoItem[];
  currentListId: string | null;
  view: View;
  summary: TodoSummary | null;
  loading: boolean;
  error: string | null;
  toast: string | null;

  // 动作
  loadAll: () => Promise<void>;
  loadLists: () => Promise<void>;
  loadItems: (listId?: string) => Promise<void>;
  loadView: (view: Exclude<View, "list" | "trash">) => Promise<void>;
  loadTrash: () => Promise<void>;
  createList: (name: string) => Promise<void>;
  updateList: (p: UpdateListParams) => Promise<void>;
  deleteList: (id: string) => Promise<void>;
  restoreList: (id: string) => Promise<void>;
  purgeList: (id: string) => Promise<void>;
  createItem: (p: CreateItemParams) => Promise<void>;
  updateItem: (p: UpdateItemParams) => Promise<void>;
  toggleItem: (id: string) => Promise<void>;
  deleteItem: (id: string) => Promise<void>;
  restoreItem: (id: string) => Promise<void>;
  purgeItem: (id: string) => Promise<void>;
  aiBreakdown: (title: string, notes: string) => Promise<TodoItem[]>;
  refreshSummary: () => Promise<void>;

  // 清提示
  clearToast: () => void;
}

export const useTodoStore = create<TodoState>((set, get) => ({
  lists: [],
  items: [],
  currentListId: null,
  view: "today",
  summary: null,
  loading: false,
  error: null,
  toast: null,

  loadAll: async () => {
    set({ loading: true, error: null });
    try {
      const [lists, summary] = await Promise.all([
        todoApi.listLists(false),
        todoApi.summary(),
      ]);
      set({ lists: lists ?? [], summary: summary ?? null, loading: false });
    } catch (err) {
      set({ error: String(err), loading: false });
    }
  },

  loadLists: async () => {
    const lists = await todoApi.listLists(false);
    if (lists) set({ lists });
  },

  loadItems: async (listId = "") => {
    set({ loading: true, error: null });
    const items = await todoApi.listItems(listId, "all");
    set({ items: items ?? [], loading: false });
  },

  loadView: async (view) => {
    set({ view, loading: true, error: null, currentListId: null });
    let items: TodoItem[] = [];
    if (view === "today") items = (await todoApi.listToday()) ?? [];
    else if (view === "overdue") items = (await todoApi.listOverdue()) ?? [];
    else if (view === "upcoming") items = (await todoApi.listUpcoming()) ?? [];
    else if (view === "all") items = (await todoApi.listItems("", "all")) ?? [];
    set({ items, loading: false });
  },

  loadTrash: async () => {
    set({ view: "trash", loading: true, error: null, currentListId: null });
    const [items, lists] = await Promise.all([
      todoApi.listDeletedItems(),
      todoApi.listDeletedLists(),
    ]);
    set({ items: items ?? [], lists: lists ?? [], loading: false });
  },

  createList: async (name) => {
    if (!name.trim()) return;
    const l = await todoApi.createList({ name: name.trim() });
    if (l) {
      set({ lists: [...get().lists, l], toast: `列表「${l.name}」已创建` });
      get().refreshSummary();
    }
  },

  updateList: async (p) => {
    const l = await todoApi.updateList(p);
    if (l) {
      set({
        lists: get().lists.map((x) => (x.id === l.id ? l : x)),
        toast: "已更新",
      });
    }
  },

  deleteList: async (id) => {
    await todoApi.deleteList(id);
    set({
      lists: get().lists.filter((x) => x.id !== id),
      toast: "已移入回收站",
    });
    get().refreshSummary();
  },

  restoreList: async (id) => {
    await todoApi.restoreList(id);
    await get().loadLists();
    set({ toast: "已恢复" });
  },

  purgeList: async (id) => {
    await todoApi.purgeList(id);
    await get().loadTrash();
    set({ toast: "已彻底删除" });
  },

  createItem: async (p) => {
    const it = await todoApi.createItem(p);
    if (it) {
      set({ items: [...get().items, it], toast: `「${it.title}」已添加` });
      get().refreshSummary();
    }
  },

  updateItem: async (p) => {
    const it = await todoApi.updateItem(p);
    if (it) {
      set({
        items: get().items.map((x) => (x.id === it.id ? it : x)),
        toast: "已更新",
      });
    }
  },

  toggleItem: async (id) => {
    const it = await todoApi.toggleItem(id);
    if (it) {
      set({
        items: get().items.map((x) => (x.id === it.id ? it : x)),
        toast: it.completedAt ? "已完成 🎉" : "已恢复未完成",
      });
      get().refreshSummary();
    }
  },

  deleteItem: async (id) => {
    await todoApi.deleteItem(id);
    set({ items: get().items.filter((x) => x.id !== id), toast: "已移入回收站" });
    get().refreshSummary();
  },

  restoreItem: async (id) => {
    await todoApi.restoreItem(id);
    await get().loadTrash();
    set({ toast: "已恢复" });
  },

  purgeItem: async (id) => {
    await todoApi.purgeItem(id);
    set({ items: get().items.filter((x) => x.id !== id), toast: "已彻底删除" });
  },

  aiBreakdown: async (title, notes) => {
    const items = await todoApi.aiBreakdown(title, notes);
    return items ?? [];
  },

  refreshSummary: async () => {
    const summary = await todoApi.summary();
    if (summary) set({ summary });
  },

  clearToast: () => set({ toast: null }),
}));
