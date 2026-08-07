// Todo 前端类型与 Wails 封装
// ------------------------------------------------------------
// 与后端 internal/todo/types.go 对齐。
// 所有调用统一通过 callWails 走 window.go.deepstudent.App 绑定。

import { callWails } from "@/lib/wails";

/** 待办列表 —— 与后端 todo.List 对齐 */
export interface TodoList {
  id: string;
  name: string;
  color?: string;
  icon?: string;
  isInbox: boolean;
  isFavorite: boolean;
  isDeleted: boolean;
  deletedAt?: string | null;
  sortOrder: number;
  createdAt: string;
  updatedAt: string;
  itemCount?: number;
  pendingCount?: number;
  completedCount?: number;
}

/** 待办条目 —— 与后端 todo.Item 对齐 */
export interface TodoItem {
  id: string;
  listId: string;
  title: string;
  notes?: string;
  dueAt?: string | null;
  completedAt?: string | null;
  priority: number; // 0=无 1=低 2=中 3=高
  tags?: string[];
  parentId?: string | null;
  estPomodoros: number;
  donePomodoros: number;
  repeat?: { frequency?: string; days?: number[]; interval?: number } | null;
  remindAt?: string | null;
  isDeleted: boolean;
  deletedAt?: string | null;
  sortOrder: number;
  createdAt: string;
  updatedAt: string;
  subCount?: number;
}

/** 创建列表参数 */
export interface CreateListParams {
  name: string;
  color?: string;
  icon?: string;
}

/** 更新列表参数 */
export interface UpdateListParams {
  id: string;
  name?: string | null;
  color?: string | null;
  icon?: string | null;
  favorite?: boolean | null;
}

/** 创建条目参数 */
export interface CreateItemParams {
  listId: string;
  title: string;
  notes?: string;
  dueAt?: string | null;
  priority?: number;
  tags?: string[];
  parentId?: string | null;
  estPomodoros?: number;
  repeat?: { frequency?: string; days?: number[]; interval?: number } | null;
  remindAt?: string | null;
}

/** 更新条目参数 */
export interface UpdateItemParams {
  id: string;
  listId?: string | null;
  title?: string | null;
  notes?: string | null;
  dueAt?: string | null;
  priority?: number | null;
  tags?: string[] | null;
  parentId?: string | null;
  estPomodoros?: number | null;
  donePomodoros?: number | null;
  repeat?: { frequency?: string; days?: number[]; interval?: number } | null;
  remindAt?: string | null;
}

/** 活跃待办总览 */
export interface TodoSummary {
  totalPending: number;
  totalCompleted: number;
  overdueCount: number;
  dueTodayCount: number;
  lists: TodoList[];
}

/** Todo API 封装 */
export const todoApi = {
  ensureInbox: () => callWails<TodoList>("TodoEnsureInbox"),
  createList: (p: CreateListParams) => callWails<TodoList>("TodoCreateList", p),
  getList: (id: string) => callWails<TodoList>("TodoGetList", id),
  listLists: (includeDeleted = false) =>
    callWails<TodoList[]>("TodoListLists", includeDeleted),
  updateList: (p: UpdateListParams) => callWails<TodoList>("TodoUpdateList", p),
  deleteList: (id: string) => callWails<void>("TodoDeleteList", id),
  restoreList: (id: string) => callWails<void>("TodoRestoreList", id),
  purgeList: (id: string) => callWails<void>("TodoPurgeList", id),
  purgeDeletedLists: () => callWails<number>("TodoPurgeDeletedLists"),
  listDeletedLists: () => callWails<TodoList[]>("TodoListDeletedLists"),

  createItem: (p: CreateItemParams) => callWails<TodoItem>("TodoCreateItem", p),
  getItem: (id: string) => callWails<TodoItem>("TodoGetItem", id),
  listItems: (listId = "", filter = "all") =>
    callWails<TodoItem[]>("TodoListItems", listId, filter),
  updateItem: (p: UpdateItemParams) => callWails<TodoItem>("TodoUpdateItem", p),
  toggleItem: (id: string) => callWails<TodoItem>("TodoToggleItem", id),
  deleteItem: (id: string) => callWails<void>("TodoDeleteItem", id),
  restoreItem: (id: string) => callWails<void>("TodoRestoreItem", id),
  purgeItem: (id: string) => callWails<void>("TodoPurgeItem", id),
  purgeDeletedItems: () => callWails<number>("TodoPurgeDeletedItems"),
  listDeletedItems: () => callWails<TodoItem[]>("TodoListDeletedItems"),
  reorderItems: (listId: string, ids: string[]) =>
    callWails<void>("TodoReorderItems", listId, ids),

  listToday: () => callWails<TodoItem[]>("TodoListToday"),
  listOverdue: () => callWails<TodoItem[]>("TodoListOverdue"),
  listUpcoming: () => callWails<TodoItem[]>("TodoListUpcoming"),
  listReminders: (limit = 20) => callWails<TodoItem[]>("TodoListReminders", limit),
  search: (keyword: string, limit = 50) =>
    callWails<TodoItem[]>("TodoSearch", keyword, limit),
  summary: () => callWails<TodoSummary>("TodoSummary"),
  aiBreakdown: (title: string, notes = "") =>
    callWails<TodoItem[]>("TodoAIBreakdown", title, notes),
};
