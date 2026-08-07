// Anki Store —— Anki 制卡状态管理
// ------------------------------------------------------------
// 对接后端 Wails 绑定：
// - AnkiTemplates() / AnkiAddTemplate(t)
// - AnkiGenerate(deck, text, tplID, batch) → Job
// - AnkiSave(job) → uri / AnkiExport(job) → []byte
//
// 设计要点：
// - 左侧模板列表 + 添加模板
// - 中间制卡表单（deck / 模板 / 文本 / 批量）
// - 生成后展示 Job 状态与卡片列表

import { create } from "zustand";
import { callWails } from "@/lib/wails";

/** Anki 卡片 —— 与后端 anki.Card 对齐 */
export interface AnkiCard {
  id: string;
  deck: string;
  front: string;
  back: string;
  tags?: string[];
  source?: string;
  template?: string;
}

/** Anki 模板 —— 与后端 anki.Template 对齐 */
export interface AnkiTemplate {
  id: string;
  name: string;
  front: string;
  back: string;
  style: string;
  css: string;
}

/** Anki 制卡任务 —— 与后端 anki.Job 对齐 */
export interface AnkiJob {
  id: string;
  deck: string;
  source_uri: string;
  total: number;
  done: number;
  cards: AnkiCard[];
  status: string; // pending | running | done | failed
  started_at: string;
  finished_at?: string;
}

interface AnkiState {
  /** 模板列表 */
  templates: AnkiTemplate[];
  /** 当前选中模板 ID */
  selectedTemplateId: string;
  /** deck 名 */
  deck: string;
  /** 源文本 */
  text: string;
  /** 批量大小 */
  batch: number;
  /** 当前 Job */
  job: AnkiJob | null;
  /** 模板加载状态 */
  loadingTemplates: boolean;
  /** 生成状态 */
  generating: boolean;
  /** 保存状态 */
  saving: boolean;
  /** 错误信息 */
  error: string | null;
  /** 保存成功后返回的 URI */
  savedUri: string | null;
  /** 添加模板弹窗 */
  addTemplateOpen: boolean;

  // —— Actions ——
  setDeck: (s: string) => void;
  setText: (s: string) => void;
  setBatch: (n: number) => void;
  setSelectedTemplate: (id: string) => void;
  setAddTemplateOpen: (open: boolean) => void;
  loadTemplates: () => Promise<void>;
  addTemplate: (t: AnkiTemplate) => Promise<void>;
  generate: () => Promise<void>;
  save: () => Promise<void>;
  exportApkg: () => Promise<void>;
}

export const useAnkiStore = create<AnkiState>((set, get) => ({
  templates: [],
  selectedTemplateId: "",
  deck: "",
  text: "",
  batch: 5,
  job: null,
  loadingTemplates: false,
  generating: false,
  saving: false,
  error: null,
  savedUri: null,
  addTemplateOpen: false,

  setDeck: (s) => set({ deck: s }),
  setText: (s) => set({ text: s }),
  setBatch: (n) => set({ batch: n }),
  setSelectedTemplate: (id) => set({ selectedTemplateId: id }),
  setAddTemplateOpen: (open) => set({ addTemplateOpen: open }),

  // —— 加载模板列表 ——
  loadTemplates: async () => {
    set({ loadingTemplates: true, error: null });
    try {
      const list = await callWails<AnkiTemplate[]>("AnkiTemplates");
      const templates = list ?? [];
      set({ templates });
      // 默认选中第一个
      if (templates.length > 0 && !get().selectedTemplateId) {
        set({ selectedTemplateId: templates[0].id });
      }
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loadingTemplates: false });
    }
  },

  // —— 添加模板 ——
  addTemplate: async (t) => {
    set({ error: null });
    try {
      await callWails<void>("AnkiAddTemplate", t);
      // 添加后刷新列表
      await get().loadTemplates();
      set({ addTemplateOpen: false });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    }
  },

  // —— 生成卡片 ——
  generate: async () => {
    const { deck, text, selectedTemplateId, batch } = get();
    if (!deck.trim()) {
      set({ error: "请输入 deck 名称" });
      return;
    }
    if (!text.trim()) {
      set({ error: "请粘贴源文本" });
      return;
    }
    set({ generating: true, error: null, savedUri: null });
    try {
      const job = await callWails<AnkiJob>(
        "AnkiGenerate",
        deck.trim(),
        text,
        selectedTemplateId,
        batch
      );
      if (!job) {
        set({ error: "[后端未连接] 生成不可用" });
        return;
      }
      set({ job });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ generating: false });
    }
  },

  // —— 保存到 Hub ——
  save: async () => {
    const { job } = get();
    if (!job) {
      set({ error: "无可保存的卡片任务" });
      return;
    }
    set({ saving: true, error: null, savedUri: null });
    try {
      const uri = await callWails<string>("AnkiSave", job);
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

  // —— 导出 .apkg ——
  exportApkg: async () => {
    const { job, deck } = get();
    if (!job) {
      set({ error: "无可导出的卡片任务" });
      return;
    }
    set({ error: null });
    try {
      // AnkiExport 返回 []byte，Wails 转 number[]
      const bytes = await callWails<number[]>("AnkiExport", job);
      if (!bytes) {
        set({ error: "[后端未连接] 导出不可用" });
        return;
      }
      // 转 Blob 下载
      const blob = new Blob([new Uint8Array(bytes)], {
        type: "application/octet-stream",
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${deck || "anki"}.apkg`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    }
  },
}));
