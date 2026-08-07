// session store —— 全局 UI 会话状态
// 包含：provider / slot / user / sidebar 折叠 / 命令面板开关 / 主题调色板 / LLM Key
//       + P0-A：vendor / profile / assignment 状态

import { create } from "zustand";
import type {
  VendorConfig,
  ModelProfile,
  ModelAssignments,
} from "@/lib/llmcfg";

export type ProviderKey = "openai" | "anthropic" | "google" | "custom";

export interface ProviderOption {
  key: ProviderKey;
  label: string;
}

export const PROVIDERS: ProviderOption[] = [
  { key: "openai", label: "OpenAI" },
  { key: "anthropic", label: "Anthropic" },
  { key: "google", label: "Google" },
  { key: "custom", label: "Custom" },
];

export type SlotKey = "A" | "B";

/** 主题调色板 key —— 与 shadcn-variables.css 中 [data-theme-palette="xxx"] 一一对应 */
export type ThemePalette =
  | "default"
  | "purple"
  | "green"
  | "orange"
  | "pink"
  | "teal"
  | "muted"
  | "paper";

export interface PaletteOption {
  key: ThemePalette;
  label: string;
  /** 调色板对应的 primary 色相（用于色块预览） */
  swatch: string;
}

export const PALETTES: PaletteOption[] = [
  { key: "default", label: "静海蓝", swatch: "hsl(214 64% 72%)" },
  { key: "purple", label: "墨藤紫", swatch: "hsl(262 52% 74%)" },
  { key: "green", label: "林影绿", swatch: "hsl(154 42% 68%)" },
  { key: "orange", label: "琥珀橙", swatch: "hsl(32 72% 70%)" },
  { key: "pink", label: "胭脂红", swatch: "hsl(340 58% 72%)" },
  { key: "teal", label: "青岩蓝", swatch: "hsl(184 48% 68%)" },
  { key: "muted", label: "雾灰", swatch: "hsl(220 24% 70%)" },
  { key: "paper", label: "米纸", swatch: "hsl(36 40% 70%)" },
];

interface SessionState {
  provider: ProviderKey;
  slot: SlotKey;
  user: string;
  dataDir: string;
  version: string;
  commandPaletteOpen: boolean;
  sidebarCollapsed: boolean;
  /** 当前主题调色板 */
  themePalette: ThemePalette;
  /** 各 provider 的 API Key（仅前端状态，不持久化到后端） */
  apiKeys: Record<ProviderKey, string>;
  /** 已注册的 LLM provider 列表（来自后端 LLMProviders） */
  registeredProviders: string[];
  /** 隐私模式开关 */
  privacyMode: boolean;
  /** P0-A：所有供应商（来自 LLMCfgGetVendors） */
  vendors: VendorConfig[];
  /** P0-A：所有模型（来自 LLMCfgGetProfiles） */
  profiles: ModelProfile[];
  /** P0-A：模型分配（来自 LLMCfgGetAssignments） */
  assignments: ModelAssignments;
  setProvider: (p: ProviderKey) => void;
  setSlot: (s: SlotKey) => void;
  setUser: (u: string) => void;
  setDataDir: (d: string) => void;
  setCommandPaletteOpen: (v: boolean) => void;
  setSidebarCollapsed: (v: boolean) => void;
  toggleSidebar: () => void;
  setThemePalette: (p: ThemePalette) => void;
  setApiKey: (p: ProviderKey, key: string) => void;
  setRegisteredProviders: (list: string[]) => void;
  setPrivacyMode: (v: boolean) => void;
  setVendors: (v: VendorConfig[]) => void;
  setProfiles: (p: ModelProfile[]) => void;
  setAssignments: (a: ModelAssignments) => void;
}

export const useSessionStore = create<SessionState>((set) => ({
  provider: "openai",
  slot: "A",
  user: "student",
  dataDir: "~/.deepstudent",
  version: "v1.0.0",
  commandPaletteOpen: false,
  sidebarCollapsed: false,
  themePalette: "default",
  apiKeys: {
    openai: "",
    anthropic: "",
    google: "",
    custom: "",
  },
  registeredProviders: [],
  privacyMode: false,
  vendors: [],
  profiles: [],
  assignments: {},
  setProvider: (provider) => set({ provider }),
  setSlot: (slot) => set({ slot }),
  setUser: (user) => set({ user }),
  setDataDir: (dataDir) => set({ dataDir }),
  setCommandPaletteOpen: (commandPaletteOpen) => set({ commandPaletteOpen }),
  setSidebarCollapsed: (sidebarCollapsed) => set({ sidebarCollapsed }),
  toggleSidebar: () =>
    set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
  setThemePalette: (themePalette) => set({ themePalette }),
  setApiKey: (p, key) =>
    set((s) => ({ apiKeys: { ...s.apiKeys, [p]: key } })),
  setRegisteredProviders: (registeredProviders) => set({ registeredProviders }),
  setPrivacyMode: (privacyMode) => set({ privacyMode }),
  setVendors: (vendors) => set({ vendors }),
  setProfiles: (profiles) => set({ profiles }),
  setAssignments: (assignments) => set({ assignments }),
}));
