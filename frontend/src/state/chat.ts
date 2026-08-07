// Chat Store —— 会话与消息状态管理
// ------------------------------------------------------------
// 对接后端 Wails 绑定：
// - ChatCreateGroup / ChatCreateSession / ChatSend
// - 维护客户端会话列表与当前激活会话
// - 模拟流式渲染（后端返回完整字符串后逐字推送）

import { create } from "zustand";
import { callWails } from "@/lib/wails";
import { uid } from "@/lib/utils";

export interface ChatMessage {
  id: string;
  role: "user" | "assistant" | "system";
  content: string;
  reasoning?: string;
  refs?: string[];
  createdAt: number;
  /** 是否正在流式输出 */
  streaming?: boolean;
  /** 错误信息 */
  error?: string;
}

export interface ChatSession {
  id: string;
  groupId: string;
  title: string;
  model: string;
  provider: string;
  messages: ChatMessage[];
  createdAt: number;
  updatedAt: number;
  systemHint?: string;
}

export interface ChatGroup {
  id: string;
  name: string;
  systemHint: string;
  defaultSkill: string;
  tags: string[];
}

interface ChatState {
  groups: ChatGroup[];
  sessions: ChatSession[];
  activeSessionId: string | null;
  loading: boolean;
  error: string | null;
  deepThink: boolean;
  refs: string[];

  // —— Actions ——
  initDefault: () => Promise<void>;
  createSession: (title?: string, provider?: string) => Promise<string | null>;
  selectSession: (id: string) => void;
  removeSession: (id: string) => void;
  sendMessage: (content: string) => Promise<void>;
  toggleDeepThink: () => void;
  addRef: (uri: string) => void;
  removeRef: (uri: string) => void;
  clearRefs: () => void;
  renameSession: (id: string, title: string) => void;
}

const DEFAULT_GROUP_NAME = "默认分组";
const DEFAULT_PROVIDER = "openai";
const DEFAULT_MODEL = "gpt-4o-mini";

export const useChatStore = create<ChatState>((set, get) => ({
  groups: [],
  sessions: [],
  activeSessionId: null,
  loading: false,
  error: null,
  deepThink: false,
  refs: [],

  // —— 初始化：确保默认分组存在 ——
  initDefault: async () => {
    if (get().groups.length > 0) return;
    const group = await callWails<ChatGroup>("ChatCreateGroup", DEFAULT_GROUP_NAME, "", "", []);
    if (!group) {
      // 后端不可用时，创建本地占位分组
      const localGroup: ChatGroup = {
        id: uid("group"),
        name: DEFAULT_GROUP_NAME,
        systemHint: "",
        defaultSkill: "",
        tags: [],
      };
      set({ groups: [localGroup] });
      return;
    }
    set({ groups: [group] });
  },

  // —— 创建会话 ——
  createSession: async (title, provider) => {
    const state = get();
    const groupId = state.groups[0]?.id ?? "";
    if (!groupId) {
      set({ error: "无可用分组" });
      return null;
    }
    const prov = provider ?? DEFAULT_PROVIDER;
    const session = await callWails<ChatSession>(
      "ChatCreateSession",
      groupId,
      title ?? "新会话",
      DEFAULT_MODEL,
      prov
    );
    let newSession: ChatSession;
    if (session) {
      newSession = {
        id: session.id,
        groupId: session.groupId,
        title: session.title,
        model: session.model,
        provider: session.provider,
        messages: [],
        createdAt: Date.now(),
        updatedAt: Date.now(),
        systemHint: session.systemHint,
      };
    } else {
      // 后端不可用时的本地占位会话
      newSession = {
        id: uid("sess"),
        groupId,
        title: title ?? "新会话",
        model: DEFAULT_MODEL,
        provider: prov,
        messages: [],
        createdAt: Date.now(),
        updatedAt: Date.now(),
      };
    }
    set((s) => ({
      sessions: [newSession, ...s.sessions],
      activeSessionId: newSession.id,
      error: null,
    }));
    return newSession.id;
  },

  selectSession: (id) => set({ activeSessionId: id, error: null }),

  removeSession: (id) =>
    set((s) => ({
      sessions: s.sessions.filter((x) => x.id !== id),
      activeSessionId:
        s.activeSessionId === id ? null : s.activeSessionId,
    })),

  // —— 发送消息 ——
  sendMessage: async (content) => {
    const state = get();
    let sessionId = state.activeSessionId;
    if (!sessionId) {
      sessionId = await get().createSession("新会话");
      if (!sessionId) return;
    }
    const session = get().sessions.find((s) => s.id === sessionId);
    if (!session) return;

    // 构造 user 消息
    const userMsg: ChatMessage = {
      id: uid("msg"),
      role: "user",
      content,
      refs: state.refs.slice(),
      createdAt: Date.now(),
    };
    // 构造 assistant 占位消息（流式）
    const asstMsg: ChatMessage = {
      id: uid("msg"),
      role: "assistant",
      content: "",
      createdAt: Date.now(),
      streaming: true,
    };

    set((s) => ({
      sessions: s.sessions.map((sess) =>
        sess.id === sessionId
          ? {
              ...sess,
              messages: [...sess.messages, userMsg, asstMsg],
              updatedAt: Date.now(),
            }
          : sess
      ),
      loading: true,
      error: null,
    }));

    // 调用后端 ChatSend（返回完整字符串）
    try {
      const reply = await callWails<string>(
        "ChatSend",
        sessionId,
        content,
        state.refs.slice(),
        state.deepThink
      );
      const finalContent =
        reply ?? "[后端未连接] 这条消息来自本地占位 —— Wails 绑定不可用时只做 UI 演示。";

      // 模拟流式渲染：逐字推送
      await streamIntoMessage(
        sessionId,
        asstMsg.id,
        finalContent,
        set
      );
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : String(err);
      set((s) => ({
        sessions: s.sessions.map((sess) =>
          sess.id === sessionId
            ? {
                ...sess,
                messages: sess.messages.map((m) =>
                  m.id === asstMsg.id
                    ? { ...m, streaming: false, error: errorMsg }
                    : m
                ),
              }
            : sess
        ),
        error: errorMsg,
      }));
    } finally {
      set({ loading: false });
    }
  },

  toggleDeepThink: () => set((s) => ({ deepThink: !s.deepThink })),

  addRef: (uri) =>
    set((s) => (s.refs.includes(uri) ? s : { refs: [...s.refs, uri] })),

  removeRef: (uri) =>
    set((s) => ({ refs: s.refs.filter((r) => r !== uri) })),

  clearRefs: () => set({ refs: [] }),

  renameSession: (id, title) =>
    set((s) => ({
      sessions: s.sessions.map((sess) =>
        sess.id === id ? { ...sess, title } : sess
      ),
    })),
}));

// —— 模拟流式渲染：把完整字符串按 chunk 推送到指定消息 ——
async function streamIntoMessage(
  sessionId: string,
  messageId: string,
  fullContent: string,
  set: (fn: (s: ChatState) => Partial<ChatState>) => void
) {
  // 字符块大小（每帧 3-6 字符，模拟真实流式体验）
  const chunkSize = 4;
  const intervalMs = 16;
  let pos = 0;

  return new Promise<void>((resolve) => {
    const tick = () => {
      pos += chunkSize;
      const slice = fullContent.slice(0, pos);
      set((s) => ({
        sessions: s.sessions.map((sess) =>
          sess.id === sessionId
            ? {
                ...sess,
                messages: sess.messages.map((m) =>
                  m.id === messageId
                    ? { ...m, content: slice, streaming: pos < fullContent.length }
                    : m
                ),
              }
            : sess
        ),
      }));
      if (pos < fullContent.length) {
        setTimeout(tick, intervalMs);
      } else {
        resolve();
      }
    };
    tick();
  });
}
