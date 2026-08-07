// Chat Store v2 —— 接入 ChatV2 后端（持久化/分组/回收站/工具循环）
// ------------------------------------------------------------
// 数据以 SQLite 为准（重启不丢）；store 维护 UI 状态 + 本地乐观更新。
// 发送走 ChatV2Send（工具循环），工具调用记录折叠展示。

import { create } from "zustand";
import {
  chatV2Api,
  chatLegacyApi,
  type ChatV2Session,
  type ChatV2Group,
  type ChatV2Message,
  type ToolCallRecord,
} from "@/lib/chat";
import { uid } from "@/lib/utils";

/** 前端消息（扩展后端模型：流式/错误/工具记录） */
export interface UIMessage extends ChatV2Message {
  streaming?: boolean;
  error?: string;
  toolRecords?: ToolCallRecord[];
}

export interface UISession {
  id: string;
  groupId: string;
  title: string;
  model: string;
  provider: string;
  messages: UIMessage[];
  tags: string[];
  pinned: boolean;
  isDeleted: boolean;
  createdAt: string;
  updatedAt: string;
  systemHint?: string;
}

export interface UIGroup {
  id: string;
  name: string;
  systemHint: string;
  defaultSkill: string;
  tags: string[];
  isDeleted: boolean;
}

type View = "normal" | "trash";

interface ChatState {
  groups: UIGroup[];
  sessions: UISession[];
  activeSessionId: string | null;
  view: View;
  loading: boolean;
  error: string | null;
  deepThink: boolean;
  refs: string[];

  init: () => Promise<void>;
  reloadAll: () => Promise<void>;
  createSession: (title?: string) => Promise<string | null>;
  createGroup: (name: string) => Promise<UIGroup | null>;
  selectSession: (id: string) => void;
  setView: (v: View) => void;
  sendMessage: (content: string) => Promise<void>;
  renameSession: (id: string, title: string) => Promise<void>;
  togglePin: (id: string) => Promise<void>;
  softDeleteSession: (id: string) => Promise<void>;
  restoreSession: (id: string) => Promise<void>;
  purgeSession: (id: string) => Promise<void>;
  deleteGroup: (id: string) => Promise<void>;
  restoreGroup: (id: string) => Promise<void>;
  toggleDeepThink: () => void;
  addRef: (uri: string) => void;
  removeRef: (uri: string) => void;
  clearRefs: () => void;
}

const DEFAULT_GROUP = "默认分组";

export const useChatStore = create<ChatState>((set, get) => ({
  groups: [],
  sessions: [],
  activeSessionId: null,
  view: "normal",
  loading: false,
  error: null,
  deepThink: false,
  refs: [],

  init: async () => {
    const { groups } = get();
    if (groups.length > 0) return;
    // 从后端加载分组
    const remote = await chatV2Api.listGroups(false);
    if (remote && remote.length > 0) {
      set({
        groups: remote.map(toUIGroup),
      });
      await get().reloadAll();
      return;
    }
    // 无分组 → 创建默认
    const created = await chatLegacyApi.createGroup(DEFAULT_GROUP);
    if (created) {
      set({ groups: [toUIGroup(created)] });
    } else {
      set({
        groups: [
          { id: uid("group"), name: DEFAULT_GROUP, systemHint: "", defaultSkill: "", tags: [], isDeleted: false },
        ],
      });
    }
    await get().reloadAll();
  },

  reloadAll: async () => {
    const [sessions, groups] = await Promise.all([
      chatV2Api.listSessions({ includeDeleted: true, limit: 500 }),
      chatV2Api.listGroups(true),
    ]);
    if (sessions) {
      set({
        sessions: sessions.map(toUISession),
      });
    }
    if (groups && groups.length > 0) {
      set({ groups: groups.map(toUIGroup) });
    }
  },

  createSession: async (title) => {
    const state = get();
    const groupId = state.groups[0]?.id ?? "";
    if (!groupId) {
      set({ error: "无可用分组" });
      return null;
    }
    // 优先后端（持久化）
    const remote = await chatLegacyApi.createSession(groupId, title ?? "新会话");
    const session: UISession = remote
      ? toUISession(remote)
      : {
          id: uid("sess"),
          groupId,
          title: title ?? "新会话",
          model: "gpt-4o-mini",
          provider: "openai",
          messages: [],
          tags: [],
          pinned: false,
          isDeleted: false,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        };
    set((s) => ({
      sessions: [session, ...s.sessions],
      activeSessionId: session.id,
      error: null,
      view: "normal",
    }));
    return session.id;
  },

  createGroup: async (name) => {
    if (!name.trim()) return null;
    const remote = await chatV2Api.updateGroup({
      id: uid("group"),
      name: name.trim(),
      system_hint: "",
      default_skill: "",
      tags: [],
      is_deleted: false,
      sort_order: 0,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    } as never as ChatV2Group);
    const g: UIGroup = {
      id: uid("group"),
      name: name.trim(),
      systemHint: "",
      defaultSkill: "",
      tags: [],
      isDeleted: false,
    };
    // 后端不可用则本地创建
    set((s) => ({ groups: [...s.groups, g] }));
    void remote;
    return g;
  },

  selectSession: (id) => set({ activeSessionId: id, error: null }),

  setView: (v) => set({ view: v }),

  sendMessage: async (content) => {
    const state = get();
    let sessionId = state.activeSessionId;
    if (!sessionId) {
      sessionId = await get().createSession("新会话");
      if (!sessionId) return;
    }
    const session = get().sessions.find((s) => s.id === sessionId);
    if (!session) return;

    const userMsg: UIMessage = {
      id: uid("msg"),
      role: "user",
      content,
      refs: state.refs.slice(),
      created_at: new Date().toISOString(),
    };
    const asstMsg: UIMessage = {
      id: uid("msg"),
      role: "assistant",
      content: "",
      created_at: new Date().toISOString(),
      streaming: true,
    };

    set((s) => ({
      sessions: s.sessions.map((sess) =>
        sess.id === sessionId
          ? {
              ...sess,
              messages: [...sess.messages, userMsg, asstMsg],
              updatedAt: new Date().toISOString(),
            }
          : sess
      ),
      loading: true,
      error: null,
    }));

    try {
      // ChatV2Send：工具循环 + 记录
      const result = await chatV2Api.send(sessionId, content, state.refs.slice());
      let finalContent = "";
      let toolRecords: ToolCallRecord[] = [];
      if (result) {
        finalContent = result[0];
        toolRecords = result[1] ?? [];
      }
      if (!finalContent) {
        finalContent = "[后端未返回内容]";
      }
      set((s) => ({
        sessions: s.sessions.map((sess) =>
          sess.id === sessionId
            ? {
                ...sess,
                messages: sess.messages.map((m) =>
                  m.id === asstMsg.id
                    ? {
                        ...m,
                        content: finalContent,
                        streaming: false,
                        toolRecords: toolRecords.length > 0 ? toolRecords : undefined,
                      }
                    : m
                ),
                updatedAt: new Date().toISOString(),
              }
            : sess
        ),
      }));
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      set((s) => ({
        sessions: s.sessions.map((sess) =>
          sess.id === sessionId
            ? {
                ...sess,
                messages: sess.messages.map((m) =>
                  m.id === asstMsg.id
                    ? { ...m, streaming: false, error: msg }
                    : m
                ),
              }
            : sess
        ),
        error: msg,
      }));
    } finally {
      set({ loading: false });
    }
  },

  renameSession: async (id, title) => {
    set((s) => ({
      sessions: s.sessions.map((x) => (x.id === id ? { ...x, title } : x)),
    }));
    await chatV2Api.updateTitle(id, title);
  },

  togglePin: async (id) => {
    const s = get().sessions.find((x) => x.id === id);
    if (!s) return;
    const pinned = !s.pinned;
    set((st) => ({
      sessions: st.sessions.map((x) => (x.id === id ? { ...x, pinned } : x)),
    }));
    await chatV2Api.pin(id, pinned);
  },

  softDeleteSession: async (id) => {
    set((s) => ({
      sessions: s.sessions.map((x) => (x.id === id ? { ...x, isDeleted: true } : x)),
    }));
    await chatV2Api.softDelete(id);
  },

  restoreSession: async (id) => {
    set((s) => ({
      sessions: s.sessions.map((x) => (x.id === id ? { ...x, isDeleted: false } : x)),
    }));
    await chatV2Api.restore(id);
  },

  purgeSession: async (id) => {
    set((s) => ({ sessions: s.sessions.filter((x) => x.id !== id) }));
    await chatV2Api.purge(id);
  },

  deleteGroup: async (id) => {
    set((s) => ({
      groups: s.groups.map((g) => (g.id === id ? { ...g, isDeleted: true } : g)),
    }));
    await chatV2Api.deleteGroup(id);
  },

  restoreGroup: async (id) => {
    set((s) => ({
      groups: s.groups.map((g) => (g.id === id ? { ...g, isDeleted: false } : g)),
    }));
    await chatV2Api.restoreGroup(id);
  },

  toggleDeepThink: () => set((s) => ({ deepThink: !s.deepThink })),

  addRef: (uri) =>
    set((s) => (s.refs.includes(uri) ? s : { refs: [...s.refs, uri] })),
  removeRef: (uri) => set((s) => ({ refs: s.refs.filter((r) => r !== uri) })),
  clearRefs: () => set({ refs: [] }),
}));

// —— 类型转换 ——
function toUIGroup(g: ChatV2Group): UIGroup {
  return {
    id: g.id,
    name: g.name,
    systemHint: g.system_hint,
    defaultSkill: g.default_skill,
    tags: g.tags,
    isDeleted: g.is_deleted,
  };
}

function toUISession(s: ChatV2Session): UISession {
  return {
    id: s.id,
    groupId: s.group_id,
    title: s.title,
    model: s.model,
    provider: s.provider,
    messages: (s.messages ?? []).map((m) => ({ ...m })),
    tags: s.tags ?? [],
    pinned: s.pinned,
    isDeleted: s.is_deleted,
    createdAt: s.created_at,
    updatedAt: s.updated_at,
    systemHint: s.system_hint,
  };
}
