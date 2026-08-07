// Chat v2 前端类型与 Wails 封装
// ------------------------------------------------------------
// 对接后端 chat_v2 RPC（ChatV2* 方法）+ 旧 Chat* 方法兼容。
// 数据模型与后端 internal/chat 对齐。

import { callWails } from "@/lib/wails";

/** 会话 —— 与后端 chat.Session 对齐 */
export interface ChatV2Session {
  id: string;
  group_id: string;
  title: string;
  branch_of?: string;
  tags: string[];
  model: string;
  provider: string;
  messages: ChatV2Message[];
  created_at: string;
  updated_at: string;
  system_hint?: string;
  is_deleted: boolean;
  deleted_at?: string | null;
  pinned: boolean;
}

/** 消息 —— 与后端 chat.Message 对齐 */
export interface ChatV2Message {
  id: string;
  session_id?: string;
  role: "user" | "assistant" | "system" | "tool";
  content: string;
  reasoning?: string;
  refs?: string[];
  model?: string;
  created_at: string;
}

/** 分组 —— 与后端 chat.Group 对齐 */
export interface ChatV2Group {
  id: string;
  name: string;
  system_hint: string;
  default_skill: string;
  tags: string[];
  is_deleted: boolean;
  deleted_at?: string | null;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

/** 工具调用记录 —— 与后端 chat.ToolCallRecord 对齐 */
export interface ToolCallRecord {
  name: string;
  arguments: string;
  output?: string;
  error?: string;
}

/** 会话过滤 —— 与后端 chat.SessionFilter 对齐 */
export interface SessionFilter {
  groupId?: string;
  keyword?: string;
  tags?: string[];
  includeDeleted?: boolean;
  onlyDeleted?: boolean;
  limit?: number;
}

/** 消息搜索命中 —— 与后端 chat.SearchHit 对齐 */
export interface SearchHit {
  messageId: string;
  sessionId: string;
  sessionTitle: string;
  role: string;
  content: string;
  createdAt: string;
}

export const chatV2Api = {
  // 分组
  listGroups: (includeDeleted = false) =>
    callWails<ChatV2Group[]>("ChatV2ListGroups", includeDeleted),
  updateGroup: (g: ChatV2Group) => callWails<void>("ChatV2UpdateGroup", g),
  deleteGroup: (id: string) => callWails<void>("ChatV2DeleteGroup", id),
  restoreGroup: (id: string) => callWails<void>("ChatV2RestoreGroup", id),
  purgeGroup: (id: string) => callWails<void>("ChatV2PurgeGroup", id),

  // 会话
  listSessions: (filter: SessionFilter) =>
    callWails<ChatV2Session[]>("ChatV2ListSessions", filter),
  getSession: (id: string) => callWails<ChatV2Session>("ChatV2GetSession", id),
  updateTitle: (id: string, title: string) =>
    callWails<void>("ChatV2UpdateTitle", id, title),
  pin: (id: string, pinned: boolean) => callWails<void>("ChatV2Pin", id, pinned),
  softDelete: (id: string) => callWails<void>("ChatV2SoftDelete", id),
  restore: (id: string) => callWails<void>("ChatV2Restore", id),
  purge: (id: string) => callWails<void>("ChatV2Purge", id),
  updateTags: (id: string, tags: string[]) =>
    callWails<void>("ChatV2UpdateTags", id, tags),
  count: () => callWails<number>("ChatV2Count"),
  deleteMessage: (sessionId: string, messageId: string) =>
    callWails<void>("ChatV2DeleteMessage", sessionId, messageId),

  // 发送（工具循环）
  send: (sessionId: string, content: string, refs: string[]) =>
    callWails<[string, ToolCallRecord[]]>("ChatV2Send", sessionId, content, refs),

  // 搜索 / 导出
  search: (keyword: string, limit = 50) =>
    callWails<SearchHit[]>("ChatV2Search", keyword, limit),
  export: () => callWails<Uint8Array | number[]>("ChatV2Export"),
};

// —— 旧版兼容（未接 v2 时的兜底） ——
export const chatLegacyApi = {
  createGroup: (name: string) =>
    callWails<ChatV2Group>("ChatCreateGroup", name, "", "", []),
  createSession: (groupId: string, title: string) =>
    callWails<ChatV2Session>("ChatCreateSession", groupId, title, "gpt-4o-mini", "openai"),
};
