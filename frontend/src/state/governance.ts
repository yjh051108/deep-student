// Governance Store —— 数据治理状态管理
// ------------------------------------------------------------
// 对接后端 Wails 绑定：
// - GovBackup(target) / GovRestore(source)
// - GovSwitchSlot(to) —— 加密槽位 A/B 切换
// - GovExport(target, types) / GovImport(source)
// - GovAudit(limit) —— 审计日志
// - GovStatus() —— 治理状态 map
// - GovIntegrityCheck() —— 完整性检查（返回缺失文件列表）

import { create } from "zustand";
import { callWails } from "@/lib/wails";

/** 审计日志条目 —— 与 deepstudent.auditDTO 对齐 */
export interface AuditEntry {
  actor: string;
  action: string;
  detail: string;
  ts: number;
}

/** 导出类型选项 */
export interface ExportTypeOption {
  key: string;
  label: string;
}

export const EXPORT_TYPES: ExportTypeOption[] = [
  { key: "note", label: "笔记" },
  { key: "textbook", label: "教材" },
  { key: "qbank", label: "题库" },
  { key: "mindmap", label: "思维导图" },
  { key: "translation", label: "翻译" },
  { key: "flashcard", label: "卡片" },
  { key: "paper", label: "论文" },
  { key: "chat", label: "会话" },
  { key: "todo", label: "待办" },
  { key: "skill", label: "技能" },
];

/** 加密槽位 */
export type GovSlot = "A" | "B";

interface GovernanceState {
  /** GovStatus 返回的 map */
  status: Record<string, unknown>;
  /** 审计日志 */
  audit: AuditEntry[];
  /** 完整性检查结果（缺失文件列表） */
  integrity: string[];
  /** 备份目标路径 */
  backupTarget: string;
  /** 恢复源路径 */
  restoreSource: string;
  /** 导出目标路径 */
  exportTarget: string;
  /** 导入源路径 */
  importSource: string;
  /** 导出类型多选 */
  exportTypes: string[];
  /** 当前槽位 */
  slot: GovSlot;
  /** 审计数量 */
  auditLimit: number;
  /** 各操作的加载标志 */
  loadingStatus: boolean;
  loadingAudit: boolean;
  loadingIntegrity: boolean;
  loadingBackup: boolean;
  loadingRestore: boolean;
  loadingExport: boolean;
  loadingImport: boolean;
  loadingSlot: boolean;
  /** 错误信息 */
  error: string | null;
  /** 操作成功提示 */
  notice: string | null;

  // —— Actions ——
  refreshStatus: () => Promise<void>;
  refreshAudit: () => Promise<void>;
  checkIntegrity: () => Promise<void>;
  setBackupTarget: (s: string) => void;
  setRestoreSource: (s: string) => void;
  setExportTarget: (s: string) => void;
  setImportSource: (s: string) => void;
  toggleExportType: (key: string) => void;
  setSlot: (s: GovSlot) => void;
  setAuditLimit: (n: number) => void;
  backup: () => Promise<void>;
  restore: () => Promise<void>;
  exportData: () => Promise<void>;
  importData: () => Promise<void>;
  switchSlot: () => Promise<void>;
  clearError: () => void;
  clearNotice: () => void;
}

export const useGovernanceStore = create<GovernanceState>((set, get) => ({
  status: {},
  audit: [],
  integrity: [],
  backupTarget: "",
  restoreSource: "",
  exportTarget: "",
  importSource: "",
  exportTypes: [],
  slot: "A",
  auditLimit: 50,
  loadingStatus: false,
  loadingAudit: false,
  loadingIntegrity: false,
  loadingBackup: false,
  loadingRestore: false,
  loadingExport: false,
  loadingImport: false,
  loadingSlot: false,
  error: null,
  notice: null,

  refreshStatus: async () => {
    set({ loadingStatus: true, error: null });
    try {
      const out = await callWails<Record<string, unknown>>("GovStatus");
      set({ status: out ?? {} });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loadingStatus: false });
    }
  },

  refreshAudit: async () => {
    set({ loadingAudit: true, error: null });
    try {
      const out = await callWails<AuditEntry[]>("GovAudit", get().auditLimit);
      set({ audit: out ?? [] });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loadingAudit: false });
    }
  },

  checkIntegrity: async () => {
    set({ loadingIntegrity: true, error: null, integrity: [] });
    try {
      const out = await callWails<string[]>("GovIntegrityCheck");
      set({
        integrity: out ?? [],
        notice:
          out && out.length > 0
            ? `发现 ${out.length} 个问题`
            : "完整性检查通过",
      });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loadingIntegrity: false });
    }
  },

  setBackupTarget: (backupTarget) => set({ backupTarget }),
  setRestoreSource: (restoreSource) => set({ restoreSource }),
  setExportTarget: (exportTarget) => set({ exportTarget }),
  setImportSource: (importSource) => set({ importSource }),

  toggleExportType: (key) =>
    set((s) => ({
      exportTypes: s.exportTypes.includes(key)
        ? s.exportTypes.filter((t) => t !== key)
        : [...s.exportTypes, key],
    })),

  setSlot: (slot) => set({ slot }),
  setAuditLimit: (auditLimit) => set({ auditLimit }),

  backup: async () => {
    const { backupTarget } = get();
    if (!backupTarget.trim()) {
      set({ error: "请输入备份目标路径" });
      return;
    }
    set({ loadingBackup: true, error: null, notice: null });
    try {
      const out = await callWails<string>("GovBackup", backupTarget);
      set({ notice: out ? `备份成功：${out}` : "备份完成" });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loadingBackup: false });
    }
  },

  restore: async () => {
    const { restoreSource } = get();
    if (!restoreSource.trim()) {
      set({ error: "请输入恢复源路径" });
      return;
    }
    set({ loadingRestore: true, error: null, notice: null });
    try {
      await callWails<void>("GovRestore", restoreSource);
      set({ notice: "恢复完成" });
      await get().refreshStatus();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loadingRestore: false });
    }
  },

  exportData: async () => {
    const { exportTarget, exportTypes } = get();
    if (!exportTarget.trim()) {
      set({ error: "请输入导出目标路径" });
      return;
    }
    set({ loadingExport: true, error: null, notice: null });
    try {
      await callWails<void>("GovExport", exportTarget, exportTypes);
      set({ notice: `已导出 ${exportTypes.length} 类资源` });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loadingExport: false });
    }
  },

  importData: async () => {
    const { importSource } = get();
    if (!importSource.trim()) {
      set({ error: "请输入导入源路径" });
      return;
    }
    set({ loadingImport: true, error: null, notice: null });
    try {
      await callWails<void>("GovImport", importSource);
      set({ notice: "导入完成" });
      await get().refreshStatus();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loadingImport: false });
    }
  },

  switchSlot: async () => {
    const { slot } = get();
    set({ loadingSlot: true, error: null, notice: null });
    try {
      await callWails<void>("GovSwitchSlot", slot);
      set({ notice: `已切换到加密槽位 ${slot}` });
      await get().refreshStatus();
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loadingSlot: false });
    }
  },

  clearError: () => set({ error: null }),
  clearNotice: () => set({ notice: null }),
}));
