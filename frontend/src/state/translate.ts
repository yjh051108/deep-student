// Translate Store —— 翻译状态管理
// ------------------------------------------------------------
// 对接后端 Wails 绑定：
// - TranslateText(text, src, tgt, domain, custom, glossary) —— 文本翻译
// - TranslateDocument(uri, src, tgt, domain) —— 文档翻译
//
// 设计要点：
// - 顶部：源/目标语言、领域预设、自定义 Prompt
// - 主体双栏：左源文本，右译文
// - 底部：术语表（key-value 对，可增删）

import { create } from "zustand";
import { callWails } from "@/lib/wails";
import { uid } from "@/lib/utils";

/** 翻译领域预设 —— 与后端 domain 字符串对齐 */
export type TranslateDomain =
  | "academic"
  | "technical"
  | "literary"
  | "legal"
  | "medical"
  | "business"
  | "general";

export interface DomainOption {
  key: TranslateDomain;
  label: string;
  description: string;
}

export const DOMAINS: DomainOption[] = [
  { key: "academic", label: "学术", description: "学术论文 / 研究文献" },
  { key: "technical", label: "技术", description: "技术文档 / 工程规范" },
  { key: "literary", label: "文学", description: "小说 / 散文 / 诗歌" },
  { key: "legal", label: "法律", description: "合同 / 法条 / 判决书" },
  { key: "medical", label: "医学", description: "病历 / 论著 / 药品说明" },
  { key: "business", label: "商业", description: "财报 / 商务信函 / 报告" },
  { key: "general", label: "通用", description: "通用文本翻译" },
];

export interface LangOption {
  code: string;
  label: string;
}

/** 常用语言列表 —— 后端按字符串代码识别（en/zh/ja/...） */
export const LANGUAGES: LangOption[] = [
  { code: "auto", label: "自动检测" },
  { code: "zh", label: "中文" },
  { code: "en", label: "English" },
  { code: "ja", label: "日本語" },
  { code: "ko", label: "한국어" },
  { code: "fr", label: "Français" },
  { code: "de", label: "Deutsch" },
  { code: "es", label: "Español" },
  { code: "ru", label: "Русский" },
  { code: "it", label: "Italiano" },
  { code: "pt", label: "Português" },
  { code: "ar", label: "العربية" },
];

/** 术语表条目 */
export interface GlossaryEntry {
  id: string;
  key: string;
  value: string;
}

interface TranslateState {
  /** 源语言代码 */
  srcLang: string;
  /** 目标语言代码 */
  tgtLang: string;
  /** 翻译领域 */
  domain: TranslateDomain;
  /** 自定义 Prompt */
  customPrompt: string;
  /** 源文本 */
  sourceText: string;
  /** 译文 */
  targetText: string;
  /** 文档翻译 URI */
  documentUri: string;
  /** 文档翻译输出 */
  documentOutput: string;
  /** 术语表 */
  glossary: GlossaryEntry[];
  /** 文本翻译加载状态 */
  translating: boolean;
  /** 文档翻译加载状态 */
  translatingDoc: boolean;
  /** 错误信息 */
  error: string | null;
  /** 操作成功提示 */
  notice: string | null;

  // —— Actions ——
  setSrcLang: (code: string) => void;
  setTgtLang: (code: string) => void;
  setDomain: (d: TranslateDomain) => void;
  setCustomPrompt: (s: string) => void;
  setSourceText: (s: string) => void;
  setDocumentUri: (s: string) => void;
  addGlossary: () => void;
  updateGlossary: (id: string, field: "key" | "value", val: string) => void;
  removeGlossary: (id: string) => void;
  translateText: () => Promise<void>;
  translateDocument: () => Promise<void>;
  swapLangs: () => void;
  clearError: () => void;
  clearNotice: () => void;
}

export const useTranslateStore = create<TranslateState>((set, get) => ({
  srcLang: "auto",
  tgtLang: "zh",
  domain: "general",
  customPrompt: "",
  sourceText: "",
  targetText: "",
  documentUri: "",
  documentOutput: "",
  glossary: [],
  translating: false,
  translatingDoc: false,
  error: null,
  notice: null,

  setSrcLang: (srcLang) => set({ srcLang }),
  setTgtLang: (tgtLang) => set({ tgtLang }),
  setDomain: (domain) => set({ domain }),
  setCustomPrompt: (customPrompt) => set({ customPrompt }),
  setSourceText: (sourceText) => set({ sourceText }),
  setDocumentUri: (documentUri) => set({ documentUri }),

  addGlossary: () =>
    set((s) => ({
      glossary: [...s.glossary, { id: uid("gloss"), key: "", value: "" }],
    })),

  updateGlossary: (id, field, val) =>
    set((s) => ({
      glossary: s.glossary.map((g) =>
        g.id === id ? { ...g, [field]: val } : g
      ),
    })),

  removeGlossary: (id) =>
    set((s) => ({ glossary: s.glossary.filter((g) => g.id !== id) })),

  swapLangs: () =>
    set((s) => ({
      srcLang: s.tgtLang === "auto" ? s.srcLang : s.tgtLang,
      tgtLang: s.srcLang === "auto" ? s.tgtLang : s.srcLang,
      sourceText: s.targetText,
      targetText: s.sourceText,
    })),

  translateText: async () => {
    const { sourceText, srcLang, tgtLang, domain, customPrompt, glossary } =
      get();
    if (!sourceText.trim()) {
      set({ error: "请输入要翻译的源文本" });
      return;
    }
    set({ translating: true, error: null, notice: null });
    try {
      // 术语表转后端 Record<string,string>[]
      const gloss = glossary
        .filter((g) => g.key.trim() && g.value.trim())
        .map((g) => ({ key: g.key.trim(), value: g.value.trim() }));
      const out = await callWails<string>(
        "TranslateText",
        sourceText,
        srcLang,
        tgtLang,
        domain,
        customPrompt,
        gloss
      );
      set({
        targetText: out ?? "[后端未连接] 翻译服务不可用",
      });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : String(err),
        targetText: "",
      });
    } finally {
      set({ translating: false });
    }
  },

  translateDocument: async () => {
    const { documentUri, srcLang, tgtLang, domain } = get();
    if (!documentUri.trim()) {
      set({ error: "请输入文档的 vfs:// URI" });
      return;
    }
    set({ translatingDoc: true, error: null, notice: null });
    try {
      const out = await callWails<string>(
        "TranslateDocument",
        documentUri,
        srcLang,
        tgtLang,
        domain
      );
      set({
        documentOutput: out ?? "[后端未连接] 文档翻译不可用",
        notice: out ? "文档翻译完成" : null,
      });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : String(err),
        documentOutput: "",
      });
    } finally {
      set({ translatingDoc: false });
    }
  },

  clearError: () => set({ error: null }),
  clearNotice: () => set({ notice: null }),
}));
