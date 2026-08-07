// Skills Store —— 技能与 MCP 状态管理
// ------------------------------------------------------------
// 对接后端 Wails 绑定：
// - SkillsList() —— 列出技能
// - SkillsTools() —— 列出工具
// - SkillsCall(name, argsJSON) —— 调用工具
// - SkillsSpawnMCP(name, cmd, args, env) —— 启动 MCP 进程
// - SkillsEnableServer(name, cfgJSON) —— 启用 MCP 服务器
// - SkillsDisableServer(name) —— 禁用 MCP 服务器
// - SkillsListMCPServers() —— 已启用的 MCP 服务器列表
// - SkillsLoadSKILLMD(tier, path) —— 加载 SKILL.md
//
// 设计要点：
// - 左：技能列表 + 加载 SKILL.md 表单
// - 中：工具列表 + 工具调用面板
// - 右：MCP 服务器管理

import { create } from "zustand";
import { callWails } from "@/lib/wails";

/** 与 wailsjs/go/models.ts: skills.Skill 对齐 */
export interface Skill {
  name: string;
  title: string;
  description: string;
  tier: string;
  tools: SkillTool[];
  prompt: string;
  source: string;
}

export interface SkillTool {
  name: string;
  description: string;
  parameters: unknown;
}

/** 与 wailsjs/go/models.ts: deepstudent.skillToolDTO 对齐 */
export interface SkillToolDTO {
  name: string;
  description: string;
}

/** SKILL.md 加载层级 */
export type SkillTier = "built-in" | "global" | "project";

export interface TierOption {
  key: SkillTier;
  label: string;
  description: string;
}

export const TIERS: TierOption[] = [
  { key: "built-in", label: "内置", description: "随应用内置的 SKILL.md" },
  { key: "global", label: "全局", description: "用户全局配置（~/.deepstudent/skills）" },
  { key: "project", label: "项目", description: "当前项目本地 .skills 目录" },
];

/** MCP 服务器配置 */
export interface MCPServerConfig {
  command: string;
  args: string[];
  env: string[];
  url: string;
  enabled: boolean;
}

interface SkillsState {
  /** 技能列表 */
  skills: Skill[];
  /** 当前选中技能名称 */
  activeSkillName: string | null;
  /** 工具列表 */
  tools: SkillToolDTO[];
  /** 已启用的 MCP 服务器列表 */
  mcpServers: string[];
  /** 工具调用结果（JSON 字符串） */
  callResult: string;
  /** 当前调用工具名 */
  callingTool: string | null;
  /** 加载状态 */
  loading: boolean;
  /** MCP 操作加载状态 */
  mcpLoading: boolean;
  /** 错误信息 */
  error: string | null;
  /** 操作成功提示 */
  notice: string | null;

  // —— Actions ——
  refresh: () => Promise<void>;
  refreshTools: () => Promise<void>;
  refreshMCP: () => Promise<void>;
  selectSkill: (name: string | null) => void;
  callTool: (name: string, argsJSON: string) => Promise<void>;
  loadSkillMD: (tier: SkillTier, path: string) => Promise<void>;
  spawnMCP: (name: string, cmd: string, args: string[], env: string[]) => Promise<void>;
  enableServer: (name: string, cfg: MCPServerConfig) => Promise<void>;
  disableServer: (name: string) => Promise<void>;
  clearError: () => void;
  clearNotice: () => void;
}

export const useSkillsStore = create<SkillsState>((set, get) => ({
  skills: [],
  activeSkillName: null,
  tools: [],
  mcpServers: [],
  callResult: "",
  callingTool: null,
  loading: false,
  mcpLoading: false,
  error: null,
  notice: null,

  refresh: async () => {
    set({ loading: true, error: null });
    try {
      const list = await callWails<Skill[]>("SkillsList");
      set({ skills: list ?? [] });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  refreshTools: async () => {
    set({ loading: true, error: null });
    try {
      const list = await callWails<SkillToolDTO[]>("SkillsTools");
      set({ tools: list ?? [] });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  refreshMCP: async () => {
    set({ loading: true, error: null });
    try {
      const list = await callWails<string[]>("SkillsListMCPServers");
      set({ mcpServers: list ?? [] });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  selectSkill: (name) => set({ activeSkillName: name, callResult: "" }),

  callTool: async (name, argsJSON) => {
    set({
      callingTool: name,
      error: null,
      callResult: "",
    });
    try {
      // 后端接受 JSON 字符串参数；空字符串视为 {}
      const argsArg = argsJSON.trim() || "{}";
      // 验证 JSON 合法性
      JSON.parse(argsArg);
      const out = await callWails<unknown>("SkillsCall", name, argsArg);
      set({
        callResult: JSON.stringify(out ?? "[后端未连接] 工具调用不可用", null, 2),
      });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : String(err),
        callResult: "",
      });
    } finally {
      set({ callingTool: null });
    }
  },

  loadSkillMD: async (tier, path) => {
    set({ loading: true, error: null, notice: null });
    try {
      await callWails<void>("SkillsLoadSKILLMD", tier, path);
      set({ notice: `已加载 SKILL.md（${tier}）` });
      await get().refresh();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  spawnMCP: async (name, cmd, args, env) => {
    set({ mcpLoading: true, error: null, notice: null });
    try {
      await callWails<void>("SkillsSpawnMCP", name, cmd, args, env);
      set({ notice: `MCP 进程已启动：${name}` });
      await get().refreshMCP();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ mcpLoading: false });
    }
  },

  enableServer: async (name, cfg) => {
    set({ mcpLoading: true, error: null, notice: null });
    try {
      const cfgJSON = JSON.stringify(cfg);
      await callWails<void>("SkillsEnableServer", name, cfgJSON);
      set({ notice: `MCP 服务器已启用：${name}` });
      await get().refreshMCP();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ mcpLoading: false });
    }
  },

  disableServer: async (name) => {
    set({ mcpLoading: true, error: null, notice: null });
    try {
      await callWails<void>("SkillsDisableServer", name);
      set({ notice: `MCP 服务器已禁用：${name}` });
      await get().refreshMCP();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ mcpLoading: false });
    }
  },

  clearError: () => set({ error: null }),
  clearNotice: () => set({ notice: null }),
}));
