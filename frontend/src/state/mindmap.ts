// Mindmap Store —— 思维导图状态管理
// ------------------------------------------------------------
// 对接后端 Wails 绑定：
// - MindmapGenerate(topic) / MindmapSave(m) / MindmapLoad(uri)
// - MindmapEdit(m, instruction) / MindmapToOutline(m)
// - MindmapFromOutline(title, text) / MindmapMask(m, rate)
//
// 设计要点：
// - 主题输入 → 生成导图 → 导图/大纲视图切换
// - 支持 AI 编辑指令与节点蒙版（背书模式）
// - 可保存到 Hub（VFS）

import { create } from "zustand";
import { callWails } from "@/lib/wails";

/** 思维导图节点 —— 与后端 mindmap.Node 对齐 */
export interface MindmapNode {
  id: string;
  topic: string;
  children?: MindmapNode[];
  masked?: boolean;
  notes?: string;
}

/** 完整思维导图 —— 与后端 mindmap.Map 对齐 */
export interface MindmapMap {
  id: string;
  title: string;
  root?: MindmapNode;
}

interface MindmapState {
  /** 当前导图 */
  map: MindmapMap | null;
  /** 大纲文本（调用 MindmapToOutline 后填充） */
  outline: string;
  /** 视图模式：导图 / 大纲 */
  viewMode: "map" | "outline";
  /** 主题输入 */
  topic: string;
  /** AI 编辑指令 */
  editInstruction: string;
  /** 蒙版率（0-100） */
  maskRate: number;
  /** 加载状态（生成 / 编辑中） */
  loading: boolean;
  /** 保存状态 */
  saving: boolean;
  /** 错误信息 */
  error: string | null;
  /** 保存成功后返回的 URI */
  savedUri: string | null;

  // —— Actions ——
  setTopic: (t: string) => void;
  setEditInstruction: (t: string) => void;
  setMaskRate: (n: number) => void;
  setViewMode: (m: "map" | "outline") => void;
  generate: () => Promise<void>;
  edit: () => Promise<void>;
  applyMask: () => Promise<void>;
  save: () => Promise<void>;
  load: (uri: string) => Promise<void>;
  fromOutline: (title: string, text: string) => Promise<void>;
  refreshOutline: () => Promise<void>;
}

export const useMindmapStore = create<MindmapState>((set, get) => ({
  map: null,
  outline: "",
  viewMode: "map",
  topic: "",
  editInstruction: "",
  maskRate: 30,
  loading: false,
  saving: false,
  error: null,
  savedUri: null,

  setTopic: (t) => set({ topic: t }),
  setEditInstruction: (t) => set({ editInstruction: t }),
  setMaskRate: (n) => set({ maskRate: n }),
  setViewMode: (m) => set({ viewMode: m }),

  // —— 生成导图 ——
  generate: async () => {
    const { topic } = get();
    if (!topic.trim()) {
      set({ error: "请输入主题" });
      return;
    }
    set({ loading: true, error: null, savedUri: null });
    try {
      const m = await callWails<MindmapMap>("MindmapGenerate", topic.trim());
      if (!m) {
        set({ error: "[后端未连接] 无法生成导图" });
        return;
      }
      set({ map: m });
      // 同步生成大纲
      void get().refreshOutline();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  // —— AI 编辑导图 ——
  edit: async () => {
    const { map, editInstruction } = get();
    if (!map) {
      set({ error: "请先生成导图" });
      return;
    }
    if (!editInstruction.trim()) {
      set({ error: "请输入编辑指令" });
      return;
    }
    set({ loading: true, error: null });
    try {
      const updated = await callWails<MindmapMap>(
        "MindmapEdit",
        map,
        editInstruction.trim()
      );
      if (!updated) {
        set({ error: "[后端未连接] 编辑不可用" });
        return;
      }
      set({ map: updated, editInstruction: "" });
      void get().refreshOutline();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  // —— 应用蒙版（背书模式） ——
  applyMask: async () => {
    const { map, maskRate } = get();
    if (!map) {
      set({ error: "请先生成导图" });
      return;
    }
    set({ loading: true, error: null });
    try {
      // MindmapMask 返回 void，直接修改传入的 map
      await callWails<void>("MindmapMask", map, maskRate / 100);
      // 触发重新渲染（浅拷贝 map）
      set({ map: { ...map } });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  // —— 保存到 Hub ——
  save: async () => {
    const { map } = get();
    if (!map) {
      set({ error: "请先生成导图" });
      return;
    }
    set({ saving: true, error: null, savedUri: null });
    try {
      const uri = await callWails<string>("MindmapSave", map);
      if (!uri) {
        set({ error: "[后端未连接] 保存不可用" });
        return;
      }
      set({ savedUri: uri });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ saving: false });
    }
  },

  // —— 从 Hub 加载 ——
  load: async (uri) => {
    set({ loading: true, error: null });
    try {
      const m = await callWails<MindmapMap>("MindmapLoad", uri);
      if (!m) {
        set({ error: "[后端未连接] 加载不可用" });
        return;
      }
      set({ map: m });
      void get().refreshOutline();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  // —— 从大纲还原导图 ——
  fromOutline: async (title, text) => {
    set({ loading: true, error: null });
    try {
      const m = await callWails<MindmapMap>("MindmapFromOutline", title, text);
      if (!m) {
        set({ error: "[后端未连接] 大纲转导图不可用" });
        return;
      }
      set({ map: m });
      void get().refreshOutline();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  // —— 刷新大纲文本 ——
  refreshOutline: async () => {
    const { map } = get();
    if (!map) return;
    try {
      const text = await callWails<string>("MindmapToOutline", map);
      set({ outline: text ?? "" });
    } catch {
      // 大纲生成失败不阻塞主流程
    }
  },
}));
