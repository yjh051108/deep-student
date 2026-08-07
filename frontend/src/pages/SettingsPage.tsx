// SettingsPage —— 设置
// ------------------------------------------------------------
// 分组卡片布局：
// - 通用：数据目录 / 版本号 / 隐私模式开关
// - LLM 提供商（P0-A）：供应商管理 / 模型管理 / 角色分配 三 Tab
// - 外观：主题调色板选择（8 种）—— 通过 document.documentElement.dataset.themePalette 切换
// - 关于：版本信息 / 数据目录 / providers 列表
//
// 复用 session.ts 状态管理；设置变更时显示 toast/横幅提示。

import { useEffect, useRef, useState, type ReactNode } from "react";
import {
  Robot as PRobot,
  Flask as PFlask,
  SlidersHorizontal as PSliders,
  Palette as PPalette,
  Plug as PPlug,
  Globe as PGlobe,
  ChartBar as PChartBar,
  ShieldCheck as PShield,
  Wrench as PWrench,
  Keyboard as PKeyboard,
  BookOpen as PBook,
  CaretRight as PCaret,
  GearSix as PGear,
} from "@phosphor-icons/react";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Badge } from "@/components/ui/Badge";
import { Switch } from "@/components/ui/Switch";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/Card";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/Tabs";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/Dialog";
import { cn } from "@/lib/utils";
import { useI18n } from "@/lib/i18n";
import { callWails } from "@/lib/wails";
import {
  fetchVendors,
  fetchProfiles,
  fetchAssignments,
  saveVendor,
  deleteVendor,
  saveProfile,
  deleteProfile,
  saveAssignments,
  testConnection,
  reloadBuiltins,
  MODEL_ROLES,
  type VendorConfig,
  type ModelProfile,
  type ModelAssignments,
  type TestConnectionResult,
} from "@/lib/llmcfg";
import {
  fetchIndexStats,
  rebuildAllIndexes,
  indexSearch,
  defaultIndexOptions,
  defaultSearchQuery,
  type IndexStats,
  type IndexOptions,
  type SearchResult,
} from "@/lib/index";
import {
  useSessionStore,
  PROVIDERS,
  PALETTES,
  type ProviderKey,
  type SlotKey,
  type ThemePalette,
} from "@/state/session";
import {
  Settings,
  FolderOpen,
  Info,
  Palette,
  Cpu,
  ShieldCheck,
  HardDrive,
  AlertCircle,
  CheckCircle2,
  Loader2,
  Key,
  Check,
  Plus,
  Pencil,
  Trash2,
  Star,
  Zap,
  X,
  Database,
  Search,
  RefreshCw,
  Activity,
} from "lucide-react";

export type SettingsTab =
  | "apis" | "models" | "general" | "appearance"
  | "mcp" | "search" | "statistics" | "data-governance"
  | "params" | "shortcuts" | "about";

const SETTINGS_NAV: { group: string; items: { key: SettingsTab; label: string; icon: typeof PRobot }[] }[] = [
  {
    group: "模型服务",
    items: [
      { key: "apis", label: "模型服务", icon: PRobot },
      { key: "models", label: "模型分配", icon: PFlask },
    ],
  },
  {
    group: "通用",
    items: [
      { key: "general", label: "通用", icon: PSliders },
      { key: "appearance", label: "外观", icon: PPalette },
    ],
  },
  {
    group: "扩展",
    items: [
      { key: "mcp", label: "MCP 工具", icon: PPlug },
      { key: "search", label: "外部搜索", icon: PGlobe },
    ],
  },
  {
    group: "数据",
    items: [
      { key: "statistics", label: "统计", icon: PChartBar },
      { key: "data-governance", label: "数据治理", icon: PShield },
    ],
  },
  {
    group: "系统",
    items: [
      { key: "params", label: "生成参数", icon: PWrench },
      { key: "shortcuts", label: "快捷键", icon: PKeyboard },
      { key: "about", label: "关于", icon: PBook },
    ],
  },
];

export function SettingsPage() {
  const [toast, setToast] = useState<{
    kind: "success" | "error";
    text: string;
  } | null>(null);
  const toastTimer = useRef<number | null>(null);
  const [activeTab, setActiveTab] = useState<SettingsTab>(
    () => (localStorage.getItem("ds.settingsTab") as SettingsTab) || "apis"
  );

  const showToast = (kind: "success" | "error", text: string) => {
    setToast({ kind, text });
    if (toastTimer.current) window.clearTimeout(toastTimer.current);
    toastTimer.current = window.setTimeout(() => setToast(null), 2400);
  };

  useEffect(() => {
    localStorage.setItem("ds.settingsTab", activeTab);
  }, [activeTab]);

  useEffect(() => {
    return () => {
      if (toastTimer.current) window.clearTimeout(toastTimer.current);
    };
  }, []);

  return (
    <div className="flex h-full w-full min-h-0 bg-background">
      {/* —— 左：设置导航侧栏（对齐原版 UnifiedSidebar） —— */}
      <aside
        className="flex w-[240px] shrink-0 flex-col border-r border-[var(--shell-seam)] bg-[var(--shell-navigation-surface)]"
      >
        <div className="shrink-0 border-b border-[var(--shell-seam)] px-4 py-3">
          <div className="flex items-center gap-2">
            <PGear size={16} className="text-primary" />
            <span className="text-[13px] font-semibold text-foreground">设置</span>
          </div>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark p-2">
          {SETTINGS_NAV.map((group) => (
            <div key={group.group} className="mb-3">
              <div className="px-2.5 pb-1 pt-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">
                {group.group}
              </div>
              <div className="space-y-0.5">
                {group.items.map((item) => {
                  const Icon = item.icon;
                  const active = activeTab === item.key;
                  return (
                    <button
                      key={item.key}
                      onClick={() => setActiveTab(item.key)}
                      className={"sidebar-row flex w-full items-center gap-2.5 px-2.5"}
                      data-active={active}
                    >
                      <Icon size={17} className="shrink-0 opacity-80" weight={active ? "fill" : "regular"} />
                      <span className="min-w-0 flex-1 truncate text-[13px]">{item.label}</span>
                      {active && <PCaret size={11} className="shrink-0 text-primary" />}
                    </button>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      </aside>

      {/* —— 右：内容区 —— */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        <div className="mx-auto max-w-[72rem] space-y-4 p-6">
          {toast && (
            <div
              className={cn(
                "flex items-center gap-2 rounded-md border px-4 py-2 text-xs",
                toast.kind === "success"
                  ? "border-emerald-500/30 bg-emerald-500/10 text-emerald-500"
                  : "border-destructive/30 bg-destructive/10 text-destructive"
              )}
            >
              {toast.kind === "success" ? "✓" : "✗"}
              <span className="truncate">{toast.text}</span>
            </div>
          )}

          {activeTab === "apis" && <ProviderSection showToast={showToast} />}
          {activeTab === "models" && <ProviderSection showToast={showToast} initialModels />}
          {activeTab === "general" && <GeneralSection showToast={showToast} />}
          {activeTab === "appearance" && <AppearanceSection showToast={showToast} />}
          {activeTab === "mcp" && <PlaceholderSection title="MCP 工具" desc="管理 MCP 服务器接入（stdio / SSE）" />}
          {activeTab === "search" && <PlaceholderSection title="外部搜索" desc="配置外部搜索供应商（学术 / 网络）" />}
          {activeTab === "statistics" && <PlaceholderSection title="统计" desc="LLM 用量与学习数据统计（见 LLM Usage 页）" />}
          {activeTab === "data-governance" && <PlaceholderSection title="数据治理" desc="备份 / 恢复 / 云同步（见 Sync 页）" />}
          {activeTab === "params" && <PlaceholderSection title="生成参数" desc="默认温度 / TopP / 上下文窗口等" />}
          {activeTab === "shortcuts" && <PlaceholderSection title="快捷键" desc="全局快捷键配置" />}
          {activeTab === "about" && <AboutSection />}
        </div>
      </div>
    </div>
  );
}

// —— 占位区块（导航对齐原版，内容后续填充） ——
function PlaceholderSection({ title, desc }: { title: string; desc: string }) {
  return (
    <div className="rounded-lg border border-border bg-card p-8 text-center">
      <div className="text-[14px] font-semibold text-foreground">{title}</div>
      <div className="mt-1 text-[12px] text-muted-foreground">{desc}</div>
    </div>
  );
}

// ============================================================
// 通用设置
// ============================================================
function GeneralSection({
  showToast,
}: {
  showToast: (kind: "success" | "error", text: string) => void;
}) {
  const lang = useI18n((s) => s.lang);
  const setLang = useI18n((s) => s.setLang);
  const dataDir = useSessionStore((s) => s.dataDir);
  const setDataDir = useSessionStore((s) => s.setDataDir);
  const version = useSessionStore((s) => s.version);
  const privacyMode = useSessionStore((s) => s.privacyMode);
  const setPrivacyMode = useSessionStore((s) => s.setPrivacyMode);
  const [loading, setLoading] = useState(false);

  // 挂载时拉取真实版本号 / 数据目录 / 隐私模式
  useEffect(() => {
    void (async () => {
      const v = await callWails<string>("Version");
      if (v) {
        // 这里仅用于显示，不写回 session（version 已经在 store 中作为默认值）
        // 但我们想展示真实版本，所以直接通过 setDataDir / setVersion 等更新
        // 这里没有 setVersion，因此改成局部状态
        setRealVersion(v);
      }
      const d = await callWails<string>("DataDir");
      if (d) setDataDir(d);
      const p = await callWails<boolean>("IsPrivacyMode");
      if (p !== null) setPrivacyMode(Boolean(p));
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const [realVersion, setRealVersion] = useState<string>("");

  const handlePickDir = async () => {
    setLoading(true);
    try {
      // OpenFileDialog 在后端用于选择文件 —— 这里用作目录选择近似
      const picked = await callWails<string>("OpenFileDialog", "选择数据目录");
      if (picked) {
        setDataDir(picked);
        showToast("success", `已选择数据目录：${picked}`);
      } else {
        showToast("error", "后端未返回路径（可能不可用）");
      }
    } catch (err) {
      showToast(
        "error",
        err instanceof Error ? err.message : String(err)
      );
    } finally {
      setLoading(false);
    }
  };

  const handleTogglePrivacy = async (checked: boolean) => {
    setPrivacyMode(checked);
    try {
      // 后端 MemoryPrivacyMode 接受 bool
      await callWails<void>("MemoryPrivacyMode", checked);
      showToast(
        "success",
        checked ? "已开启隐私模式" : "已关闭隐私模式"
      );
    } catch (err) {
      showToast(
        "error",
        "后端不可用：" + (err instanceof Error ? err.message : String(err))
      );
    }
  };

  return (
    <Card>
      <CardHeader className="py-3">
        <CardTitle className="flex items-center gap-1.5 text-sm">
          <Settings size={14} className="text-primary" />
          通用
        </CardTitle>
        <CardDescription className="text-[11px]">
          数据目录、版本号、隐私模式
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3 pt-0">
        {/* 语言切换（i18n） */}
        <div>
          <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            语言 / Language
          </label>
          <div className="flex items-center gap-2">
            <Button
              size="sm"
              variant={lang === "zh-CN" ? "default" : "outline"}
              className="h-7"
              onClick={() => setLang("zh-CN")}
            >
              简体中文
            </Button>
            <Button
              size="sm"
              variant={lang === "en-US" ? "default" : "outline"}
              className="h-7"
              onClick={() => setLang("en-US")}
            >
              English
            </Button>
          </div>
        </div>
        {/* 数据目录 */}
        <div>
          <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            数据目录
          </label>
          <div className="flex items-center gap-2">
            <div className="flex min-w-0 flex-1 items-center gap-2 rounded-md border border-input bg-transparent px-3 py-2 text-sm text-foreground">
              <HardDrive size={13} className="shrink-0 text-muted-foreground" />
              <span className="min-w-0 flex-1 truncate font-mono text-[12px]">
                {dataDir}
              </span>
            </div>
            <Button
              variant="outline"
              size="sm"
              className="h-9"
              onClick={handlePickDir}
              disabled={loading}
            >
              {loading ? (
                <Loader2 size={12} className="animate-spin" />
              ) : (
                <FolderOpen size={12} />
              )}
              选择
            </Button>
          </div>
          <div className="mt-1 text-[10px] text-muted-foreground/60">
            数据目录用于存储笔记 / 题库 / 卡片等所有用户资源
          </div>
        </div>

        {/* 版本号 */}
        <div className="flex items-center justify-between">
          <label className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            版本号
          </label>
          <Badge variant="outline" className="font-mono text-[11px]">
            {realVersion || version}
          </Badge>
        </div>

        {/* 隐私模式 */}
        <div className="flex items-center justify-between rounded-md border border-border bg-muted/30 px-3 py-2">
          <div className="flex items-center gap-2">
            <ShieldCheck size={14} className="text-primary" />
            <div>
              <div className="text-[13px] font-medium text-foreground">
                隐私模式
              </div>
              <div className="text-[10px] text-muted-foreground/70">
                开启后，记忆与日志不会写入磁盘
              </div>
            </div>
          </div>
          <Switch
            checked={privacyMode}
            onCheckedChange={handleTogglePrivacy}
          />
        </div>
      </CardContent>
    </Card>
  );
}

// ============================================================
// LLM 提供商（P0-A：供应商管理 / 模型管理 / 角色分配 三 Tab）
// ============================================================
function ProviderSection({
  showToast,
  initialModels = false,
}: {
  showToast: (kind: "success" | "error", text: string) => void;
  initialModels?: boolean;
}) {
  const provider = useSessionStore((s) => s.provider);
  const setProvider = useSessionStore((s) => s.setProvider);
  const slot = useSessionStore((s) => s.slot);
  const setSlot = useSessionStore((s) => s.setSlot);
  const registeredProviders = useSessionStore((s) => s.registeredProviders);
  const setRegisteredProviders = useSessionStore(
    (s) => s.setRegisteredProviders
  );
  const vendors = useSessionStore((s) => s.vendors);
  const setVendors = useSessionStore((s) => s.setVendors);
  const profiles = useSessionStore((s) => s.profiles);
  const setProfiles = useSessionStore((s) => s.setProfiles);
  const assignments = useSessionStore((s) => s.assignments);
  const setAssignments = useSessionStore((s) => s.setAssignments);

  const [loading, setLoading] = useState(false);

  // 挂载时拉取已注册 providers + P0-A vendors/profiles/assignments
  const refreshAll = async () => {
    setLoading(true);
    try {
      const [regList, vList, pList, aObj] = await Promise.all([
        callWails<string[]>("LLMProviders"),
        fetchVendors(),
        fetchProfiles(),
        fetchAssignments(),
      ]);
      if (regList) setRegisteredProviders(regList);
      setVendors(vList);
      setProfiles(pList);
      setAssignments(aObj);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void refreshAll();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleProviderChange = (p: ProviderKey) => {
    setProvider(p);
    showToast("success", `已切换 Provider：${p}`);
  };

  const handleSlotChange = (s: SlotKey) => {
    setSlot(s);
    showToast("success", `已切换加密槽位：${s}`);
  };

  const handleReloadBuiltins = async () => {
    setLoading(true);
    try {
      const ok = await reloadBuiltins();
      if (ok) {
        await refreshAll();
        showToast("success", "已重新加载内置厂商和模型");
      } else {
        showToast("error", "重载失败（后端不可用）");
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <CardHeader className="py-3">
        <div className="flex items-center justify-between">
          <div className="flex flex-col gap-1">
            <CardTitle className="flex items-center gap-1.5 text-sm">
              <Cpu size={14} className="text-primary" />
              LLM 提供商
            </CardTitle>
            <CardDescription className="text-[11px]">
              供应商 / 模型 / 角色分配（P0-A）
            </CardDescription>
          </div>
          <div className="flex items-center gap-1.5">
            <Button
              variant="ghost"
              size="sm"
              className="h-7 text-[11px]"
              onClick={handleReloadBuiltins}
              disabled={loading}
              title="重新加载内置厂商和模型"
            >
              {loading ? (
                <Loader2 size={11} className="animate-spin" />
              ) : (
                <Zap size={11} />
              )}
              重载内置
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-3 pt-0">
        {/* 顶部紧凑区：Provider 选择 + 槽位 + 已注册 providers */}
        <div className="space-y-2 rounded-md border border-border bg-muted/20 p-2.5">
          <div className="flex items-center gap-2">
            <span className="shrink-0 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              当前 Provider
            </span>
            <div className="flex flex-wrap gap-1">
              {PROVIDERS.map((p) => {
                const active = provider === p.key;
                return (
                  <button
                    key={p.key}
                    type="button"
                    onClick={() => handleProviderChange(p.key)}
                    className={cn(
                      "rounded border px-2 py-0.5 text-[11px] transition-colors",
                      active
                        ? "border-primary/40 bg-primary/15 text-primary"
                        : "border-border bg-background text-muted-foreground hover:bg-accent hover:text-foreground"
                    )}
                  >
                    {p.label}
                  </button>
                );
              })}
            </div>
            <div className="ml-auto flex items-center gap-1.5">
              <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                槽位
              </span>
              {(["A", "B"] as SlotKey[]).map((s) => {
                const active = slot === s;
                return (
                  <button
                    key={s}
                    type="button"
                    onClick={() => handleSlotChange(s)}
                    className={cn(
                      "rounded border px-2 py-0.5 text-[11px] font-semibold transition-colors",
                      active
                        ? "border-primary/40 bg-primary/15 text-primary"
                        : "border-border bg-background text-muted-foreground hover:bg-accent hover:text-foreground"
                    )}
                  >
                    {s}
                  </button>
                );
              })}
            </div>
          </div>
          {registeredProviders.length > 0 && (
            <div className="flex flex-wrap items-center gap-1">
              <span className="text-[10px] text-muted-foreground/60">
                已注册：
              </span>
              {registeredProviders.map((p) => (
                <Badge
                  key={p}
                  variant="outline"
                  className="font-mono text-[9px] px-1.5 py-0"
                >
                  {p}
                </Badge>
              ))}
            </div>
          )}
        </div>

        {/* 三 Tab 主体 */}
        <Tabs defaultValue="vendors" className="w-full">
          <TabsList className="h-8 w-full">
            <TabsTrigger value="vendors" className="text-[12px] flex-1">
              供应商管理
            </TabsTrigger>
            <TabsTrigger value="models" className="text-[12px] flex-1">
              模型管理
            </TabsTrigger>
            <TabsTrigger value="roles" className="text-[12px] flex-1">
              角色分配
            </TabsTrigger>
          </TabsList>

          <TabsContent value="vendors">
            <VendorsTab
              vendors={vendors}
              profiles={profiles}
              showToast={showToast}
              onChanged={refreshAll}
            />
          </TabsContent>
          <TabsContent value="models">
            <ModelsTab
              vendors={vendors}
              profiles={profiles}
              showToast={showToast}
              onChanged={refreshAll}
            />
          </TabsContent>
          <TabsContent value="roles">
            <RolesTab
              profiles={profiles}
              assignments={assignments}
              showToast={showToast}
              onChanged={refreshAll}
            />
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  );
}

// ---------- 工具函数 ----------

/** 排序供应商：内置在前，然后按 sortOrder，最后按 name */
function sortVendors(list: VendorConfig[]): VendorConfig[] {
  return [...list].sort((a, b) => {
    if (a.isBuiltin !== b.isBuiltin) return a.isBuiltin ? -1 : 1;
    const sa = a.sortOrder ?? 999;
    const sb = b.sortOrder ?? 999;
    if (sa !== sb) return sa - sb;
    return a.name.localeCompare(b.name);
  });
}

/** 排序模型：收藏在前，然后按 label */
function sortProfiles(list: ModelProfile[]): ModelProfile[] {
  return [...list].sort((a, b) => {
    if (a.isFavorite !== b.isFavorite) return a.isFavorite ? -1 : 1;
    return a.label.localeCompare(b.label);
  });
}

// ============================================================
// Tab 1：供应商管理
// ============================================================
function VendorsTab({
  vendors,
  showToast,
  onChanged,
}: {
  vendors: VendorConfig[];
  profiles: ModelProfile[];
  showToast: (kind: "success" | "error", text: string) => void;
  onChanged: () => void | Promise<void>;
}) {
  const sorted = sortVendors(vendors);
  const [editing, setEditing] = useState<VendorConfig | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);

  const handleSaveKey = async (v: VendorConfig, key: string) => {
    const ok = await saveVendor({ ...v, apiKey: key });
    if (ok) {
      showToast("success", `${v.name} API Key 已保存`);
      await onChanged();
    } else {
      showToast("error", `${v.name} API Key 保存失败`);
    }
  };

  const handleDelete = async (v: VendorConfig) => {
    if (v.isBuiltin) {
      showToast("error", "内置供应商不可删除");
      return;
    }
    const ok = await deleteVendor(v.id);
    if (ok) {
      showToast("success", `已删除供应商：${v.name}`);
      await onChanged();
    } else {
      showToast("error", `删除失败：${v.name}`);
    }
  };

  const handleEdit = (v: VendorConfig) => {
    setEditing(v);
    setDialogOpen(true);
  };

  const handleAdd = () => {
    setEditing({
      id: "",
      name: "",
      providerType: "openai",
      baseUrl: "",
      apiKey: "",
      headers: {},
      isBuiltin: false,
      isReadOnly: false,
    });
    setDialogOpen(true);
  };

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
          供应商列表（{sorted.length}）
        </span>
        <Button
          variant="outline"
          size="sm"
          className="h-7 text-[11px]"
          onClick={handleAdd}
        >
          <Plus size={11} />
          添加自定义供应商
        </Button>
      </div>

      <div className="space-y-1.5">
        {sorted.length === 0 && (
          <div className="rounded-md border border-dashed border-border bg-background px-3 py-3 text-center text-[11px] text-muted-foreground">
            暂无供应商 —— 点击「添加自定义供应商」或「重载内置」
          </div>
        )}
        {sorted.map((v) => (
          <VendorRow
            key={v.id}
            vendor={v}
            onSaveKey={handleSaveKey}
            onEdit={handleEdit}
            onDelete={handleDelete}
          />
        ))}
      </div>

      <VendorDialog
        open={dialogOpen}
        vendor={editing}
        onOpenChange={setDialogOpen}
        onSaved={async () => {
          setDialogOpen(false);
          await onChanged();
        }}
        showToast={showToast}
      />
    </div>
  );
}

/** 单个供应商行 */
function VendorRow({
  vendor,
  onSaveKey,
  onEdit,
  onDelete,
}: {
  vendor: VendorConfig;
  onSaveKey: (v: VendorConfig, key: string) => void | Promise<void>;
  onEdit: (v: VendorConfig) => void;
  onDelete: (v: VendorConfig) => void | Promise<void>;
}) {
  const [keyInput, setKeyInput] = useState(vendor.apiKey ?? "");
  const [showKey, setShowKey] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<TestConnectionResult | null>(null);

  useEffect(() => {
    setKeyInput(vendor.apiKey ?? "");
  }, [vendor.apiKey]);

  // 找该供应商下第一个 profile 用于测试连接
  const handleTest = async () => {
    // 测试连接需要 profileID，这里用 vendor 下任意 profile
    // 简化：先保存当前 keyInput，再调用 LLMCfgTestConnection
    if (keyInput !== vendor.apiKey) {
      await onSaveKey(vendor, keyInput);
    }
    setTesting(true);
    setTestResult(null);
    try {
      // 用 ResolveApiConfig 不需要 profileID，但 TestConnection 需要；
      // 这里通过 vendor 找一个 profile 来测；若没有 profile 则提示
      // 直接使用顶部已静态导入的 callWails 拿一个 profile
      const list = await callWails<ModelProfile[]>(
        "LLMCfgGetProfilesByVendor",
        vendor.id
      );
      if (!list || list.length === 0) {
        setTestResult({
          ok: false,
          message: "该供应商下没有模型，无法测试",
          latencyMs: 0,
        });
        return;
      }
      const res = await testConnection(list[0].id);
      setTestResult(res);
    } finally {
      setTesting(false);
    }
  };

  const keyChanged = keyInput !== (vendor.apiKey ?? "");

  return (
    <div className="rounded-md border border-border bg-background px-2.5 py-2">
      <div className="flex items-center gap-2">
        {/* 名称 + 类型 */}
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <span className="truncate text-[12px] font-medium text-foreground">
              {vendor.name}
            </span>
            {vendor.isBuiltin && (
              <Badge variant="info" className="px-1.5 py-0 text-[9px]">
                内置
              </Badge>
            )}
          </div>
          <div className="mt-0.5 flex items-center gap-1.5 text-[10px] text-muted-foreground/70">
            <span className="font-mono">{vendor.providerType}</span>
            <span>·</span>
            <span className="truncate font-mono">{vendor.baseUrl}</span>
          </div>
        </div>

        {/* 操作按钮 */}
        <div className="flex shrink-0 items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            className="h-7 px-2 text-[11px]"
            onClick={handleTest}
            disabled={testing}
            title="测试连接"
          >
            {testing ? (
              <Loader2 size={11} className="animate-spin" />
            ) : (
              <Zap size={11} />
            )}
            测试
          </Button>
          {!vendor.isReadOnly && (
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              onClick={() => onEdit(vendor)}
              title="编辑"
            >
              <Pencil size={11} />
            </Button>
          )}
          {!vendor.isBuiltin && (
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-destructive hover:text-destructive"
              onClick={() => onDelete(vendor)}
              title="删除"
            >
              <Trash2 size={11} />
            </Button>
          )}
        </div>
      </div>

      {/* API Key 输入行 */}
      <div className="mt-1.5 flex items-center gap-1.5">
        <Key size={10} className="shrink-0 text-muted-foreground" />
        <Input
          type={showKey ? "text" : "password"}
          value={keyInput}
          onChange={(e) => setKeyInput(e.target.value)}
          onBlur={() => keyChanged && onSaveKey(vendor, keyInput)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && keyChanged) {
              void onSaveKey(vendor, keyInput);
            }
          }}
          placeholder={`${vendor.name} API Key…`}
          className="h-7 font-mono text-[11px]"
        />
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 shrink-0"
          onClick={() => setShowKey((s) => !s)}
          title={showKey ? "隐藏" : "显示"}
        >
          {showKey ? <X size={11} /> : <Check size={11} />}
        </Button>
      </div>

      {/* 测试结果 */}
      {testResult && (
        <div
          className={cn(
            "mt-1.5 flex items-center gap-1.5 rounded px-2 py-1 text-[10px]",
            testResult.ok
              ? "bg-emerald-500/10 text-emerald-500"
              : "bg-destructive/10 text-destructive"
          )}
        >
          {testResult.ok ? (
            <CheckCircle2 size={10} />
          ) : (
            <AlertCircle size={10} />
          )}
          <span className="truncate">
            {testResult.ok
              ? `连接成功 · ${testResult.latencyMs}ms · ${testResult.model}`
              : `失败 · ${testResult.message}`}
          </span>
        </div>
      )}

      {/* 备注 */}
      {vendor.notes && (
        <div className="mt-1 text-[10px] text-muted-foreground/50">
          {vendor.notes}
        </div>
      )}
    </div>
  );
}

/** 供应商编辑/新建 Dialog */
function VendorDialog({
  open,
  vendor,
  onOpenChange,
  onSaved,
  showToast,
}: {
  open: boolean;
  vendor: VendorConfig | null;
  onOpenChange: (v: boolean) => void;
  onSaved: () => void | Promise<void>;
  showToast: (kind: "success" | "error", text: string) => void;
}) {
  const [form, setForm] = useState<VendorConfig | null>(vendor);

  useEffect(() => {
    setForm(vendor);
  }, [vendor, open]);

  if (!form) return null;

  const handleField = <K extends keyof VendorConfig>(
    k: K,
    v: VendorConfig[K]
  ) => setForm((f) => (f ? { ...f, [k]: v } : f));

  const handleSave = async () => {
    if (!form.name || !form.baseUrl) {
      showToast("error", "名称和 Base URL 必填");
      return;
    }
    const ok = await saveVendor(form);
    if (ok) {
      showToast("success", form.isBuiltin ? "供应商已更新" : "供应商已保存");
      await onSaved();
    } else {
      showToast("error", "保存失败");
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="text-sm">
            {form.isBuiltin ? `编辑：${form.name}` : "自定义供应商"}
          </DialogTitle>
          <DialogDescription className="text-[11px]">
            填写供应商信息后保存
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <DialogField label="名称">
            <Input
              value={form.name}
              onChange={(e) => handleField("name", e.target.value)}
              className="h-8 text-[12px]"
              placeholder="例如：OpenAI"
              disabled={form.isReadOnly}
            />
          </DialogField>
          <DialogField label="Provider Type">
            <Input
              value={form.providerType}
              onChange={(e) => handleField("providerType", e.target.value)}
              className="h-8 font-mono text-[12px]"
              placeholder="openai / deepseek / ..."
              disabled={form.isReadOnly}
            />
          </DialogField>
          <DialogField label="Base URL">
            <Input
              value={form.baseUrl}
              onChange={(e) => handleField("baseUrl", e.target.value)}
              className="h-8 font-mono text-[12px]"
              placeholder="https://api.example.com/v1"
              disabled={form.isReadOnly}
            />
          </DialogField>
          <DialogField label="API Key">
            <Input
              type="password"
              value={form.apiKey}
              onChange={(e) => handleField("apiKey", e.target.value)}
              className="h-8 font-mono text-[12px]"
              placeholder="sk-..."
            />
          </DialogField>
          <DialogField label="备注（可选）">
            <Input
              value={form.notes ?? ""}
              onChange={(e) => handleField("notes", e.target.value)}
              className="h-8 text-[12px]"
              placeholder="备注信息"
              disabled={form.isReadOnly}
            />
          </DialogField>
        </div>
        <DialogFooter>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onOpenChange(false)}
            className="text-[12px]"
          >
            取消
          </Button>
          <Button size="sm" onClick={handleSave} className="text-[12px]">
            保存
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ============================================================
// Tab 2：模型管理
// ============================================================
function ModelsTab({
  vendors,
  profiles,
  showToast,
  onChanged,
}: {
  vendors: VendorConfig[];
  profiles: ModelProfile[];
  showToast: (kind: "success" | "error", text: string) => void;
  onChanged: () => void | Promise<void>;
}) {
  const sortedVendors = sortVendors(vendors);
  const [selectedVendor, setSelectedVendor] = useState<string>(
    sortedVendors[0]?.id ?? ""
  );
  const [editing, setEditing] = useState<ModelProfile | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);

  useEffect(() => {
    if (!selectedVendor && sortedVendors.length > 0) {
      setSelectedVendor(sortedVendors[0].id);
    }
  }, [sortedVendors, selectedVendor]);

  const vendorProfiles = sortProfiles(
    profiles.filter((p) => p.vendorId === selectedVendor)
  );

  const handleToggleEnabled = async (p: ModelProfile, enabled: boolean) => {
    const ok = await saveProfile({ ...p, enabled });
    if (ok) {
      await onChanged();
    } else {
      showToast("error", `${p.label} 启用切换失败`);
    }
  };

  const handleToggleFavorite = async (p: ModelProfile) => {
    const ok = await saveProfile({ ...p, isFavorite: !p.isFavorite });
    if (ok) {
      await onChanged();
    }
  };

  const handleDelete = async (p: ModelProfile) => {
    const ok = await deleteProfile(p.id);
    if (ok) {
      showToast("success", `已删除模型：${p.label}`);
      await onChanged();
    } else {
      showToast("error", `删除失败：${p.label}`);
    }
  };

  const handleEdit = (p: ModelProfile) => {
    setEditing(p);
    setDialogOpen(true);
  };

  const handleAdd = () => {
    setEditing({
      id: "",
      vendorId: selectedVendor,
      label: "",
      model: "",
      modelAdapter: "general",
      isMultimodal: false,
      isReasoning: false,
      isEmbedding: false,
      isReranker: false,
      isImageGeneration: false,
      supportsTools: false,
      supportsReasoning: false,
      status: "enabled",
      enabled: true,
      maxOutputTokens: 4096,
      temperature: 0.7,
      thinkingEnabled: false,
      includeThoughts: false,
      isBuiltin: false,
      isFavorite: false,
    });
    setDialogOpen(true);
  };

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
          模型列表（{profiles.length}）
        </span>
        <Button
          variant="outline"
          size="sm"
          className="h-7 text-[11px]"
          onClick={handleAdd}
          disabled={!selectedVendor}
        >
          <Plus size={11} />
          添加自定义模型
        </Button>
      </div>

      <div className="grid grid-cols-12 gap-2">
        {/* 左侧 vendor 列表 */}
        <div className="col-span-4 space-y-0.5">
          <div className="max-h-[360px] overflow-y-auto scrollbar-dark rounded-md border border-border bg-background p-1">
            {sortedVendors.map((v) => {
              const active = selectedVendor === v.id;
              const count = profiles.filter((p) => p.vendorId === v.id).length;
              return (
                <button
                  key={v.id}
                  type="button"
                  onClick={() => setSelectedVendor(v.id)}
                  className={cn(
                    "flex w-full items-center justify-between rounded px-2 py-1.5 text-left text-[12px] transition-colors",
                    active
                      ? "bg-primary/15 text-primary"
                      : "text-foreground hover:bg-accent"
                  )}
                >
                  <span className="truncate">{v.name}</span>
                  <span
                    className={cn(
                      "ml-1 shrink-0 font-mono text-[10px]",
                      active ? "text-primary/70" : "text-muted-foreground/60"
                    )}
                  >
                    {count}
                  </span>
                </button>
              );
            })}
          </div>
        </div>

        {/* 右侧 models 列表 */}
        <div className="col-span-8 space-y-1.5">
          <div className="max-h-[360px] overflow-y-auto scrollbar-dark space-y-1.5">
            {vendorProfiles.length === 0 && (
              <div className="rounded-md border border-dashed border-border bg-background px-3 py-3 text-center text-[11px] text-muted-foreground">
                该供应商下暂无模型
              </div>
            )}
            {vendorProfiles.map((p) => (
              <ModelRow
                key={p.id}
                profile={p}
                onToggleEnabled={handleToggleEnabled}
                onToggleFavorite={handleToggleFavorite}
                onEdit={handleEdit}
                onDelete={handleDelete}
              />
            ))}
          </div>
        </div>
      </div>

      <ModelDialog
        open={dialogOpen}
        profile={editing}
        vendors={sortedVendors}
        onOpenChange={setDialogOpen}
        onSaved={async () => {
          setDialogOpen(false);
          await onChanged();
        }}
        showToast={showToast}
      />
    </div>
  );
}

/** 能力 Badge 颜色映射 */
function CapabilityBadges({ p }: { p: ModelProfile }) {
  return (
    <div className="flex flex-wrap gap-0.5">
      {p.isMultimodal && (
        <Badge className="border-blue-500/30 bg-blue-500/15 px-1 py-0 text-[9px] text-blue-400">
          多模态
        </Badge>
      )}
      {p.isReasoning && (
        <Badge className="border-purple-500/30 bg-purple-500/15 px-1 py-0 text-[9px] text-purple-400">
          推理
        </Badge>
      )}
      {p.supportsTools && (
        <Badge className="border-emerald-500/30 bg-emerald-500/15 px-1 py-0 text-[9px] text-emerald-400">
          工具
        </Badge>
      )}
      {p.isEmbedding && (
        <Badge className="border-cyan-500/30 bg-cyan-500/15 px-1 py-0 text-[9px] text-cyan-400">
          嵌入
        </Badge>
      )}
      {p.isReranker && (
        <Badge className="border-orange-500/30 bg-orange-500/15 px-1 py-0 text-[9px] text-orange-400">
          重排序
        </Badge>
      )}
      {p.isImageGeneration && (
        <Badge className="border-pink-500/30 bg-pink-500/15 px-1 py-0 text-[9px] text-pink-400">
          生图
        </Badge>
      )}
    </div>
  );
}

/** 单个模型行 */
function ModelRow({
  profile,
  onToggleEnabled,
  onToggleFavorite,
  onEdit,
  onDelete,
}: {
  profile: ModelProfile;
  onToggleEnabled: (p: ModelProfile, enabled: boolean) => void | Promise<void>;
  onToggleFavorite: (p: ModelProfile) => void | Promise<void>;
  onEdit: (p: ModelProfile) => void;
  onDelete: (p: ModelProfile) => void | Promise<void>;
}) {
  return (
    <div className="rounded-md border border-border bg-background px-2.5 py-1.5">
      <div className="flex items-start gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <button
              type="button"
              onClick={() => onToggleFavorite(profile)}
              className="shrink-0"
              title={profile.isFavorite ? "取消收藏" : "收藏"}
            >
              <Star
                size={11}
                className={cn(
                  profile.isFavorite
                    ? "fill-yellow-400 text-yellow-400"
                    : "text-muted-foreground/40 hover:text-muted-foreground"
                )}
              />
            </button>
            <span className="truncate text-[12px] font-medium text-foreground">
              {profile.label}
            </span>
            {profile.isBuiltin && (
              <Badge variant="info" className="px-1 py-0 text-[9px]">
                内置
              </Badge>
            )}
          </div>
          <div className="mt-0.5 flex items-center gap-1.5 text-[10px] text-muted-foreground/70">
            <span className="truncate font-mono">{profile.model}</span>
            <span>·</span>
            <span>temp {profile.temperature}</span>
            <span>·</span>
            <span>{profile.maxOutputTokens} tok</span>
            {profile.contextWindow && (
              <>
                <span>·</span>
                <span>{(profile.contextWindow / 1000).toFixed(0)}K ctx</span>
              </>
            )}
          </div>
          <div className="mt-1">
            <CapabilityBadges p={profile} />
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-1">
          <Switch
            checked={profile.enabled}
            onCheckedChange={(v) => onToggleEnabled(profile, v)}
            title={profile.enabled ? "已启用" : "已禁用"}
          />
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={() => onEdit(profile)}
            title="编辑"
          >
            <Pencil size={10} />
          </Button>
          {!profile.isBuiltin && (
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6 text-destructive hover:text-destructive"
              onClick={() => onDelete(profile)}
              title="删除"
            >
              <Trash2 size={10} />
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}

/** 模型编辑/新建 Dialog */
function ModelDialog({
  open,
  profile,
  vendors,
  onOpenChange,
  onSaved,
  showToast,
}: {
  open: boolean;
  profile: ModelProfile | null;
  vendors: VendorConfig[];
  onOpenChange: (v: boolean) => void;
  onSaved: () => void | Promise<void>;
  showToast: (kind: "success" | "error", text: string) => void;
}) {
  const [form, setForm] = useState<ModelProfile | null>(profile);

  useEffect(() => {
    setForm(profile);
  }, [profile, open]);

  if (!form) return null;

  const handleField = <K extends keyof ModelProfile>(
    k: K,
    v: ModelProfile[K]
  ) => setForm((f) => (f ? { ...f, [k]: v } : f));

  const handleSave = async () => {
    if (!form.label || !form.model || !form.vendorId) {
      showToast("error", "名称、Model ID、供应商必填");
      return;
    }
    const ok = await saveProfile(form);
    if (ok) {
      showToast("success", form.isBuiltin ? "模型已更新" : "模型已保存");
      await onSaved();
    } else {
      showToast("error", "保存失败");
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="text-sm">
            {form.isBuiltin ? `编辑：${form.label}` : "自定义模型"}
          </DialogTitle>
          <DialogDescription className="text-[11px]">
            填写模型信息后保存
          </DialogDescription>
        </DialogHeader>
        <div className="max-h-[60vh] space-y-2 overflow-y-auto scrollbar-dark pr-1">
          <DialogField label="供应商">
            <select
              value={form.vendorId}
              onChange={(e) => handleField("vendorId", e.target.value)}
              className="h-8 w-full rounded-md border border-input bg-transparent px-2 text-[12px] text-foreground"
              disabled={form.isBuiltin}
            >
              {vendors.map((v) => (
                <option key={v.id} value={v.id}>
                  {v.name}
                </option>
              ))}
            </select>
          </DialogField>
          <DialogField label="标签（显示名）">
            <Input
              value={form.label}
              onChange={(e) => handleField("label", e.target.value)}
              className="h-8 text-[12px]"
              placeholder="例如：GPT-5.5"
            />
          </DialogField>
          <DialogField label="Model ID">
            <Input
              value={form.model}
              onChange={(e) => handleField("model", e.target.value)}
              className="h-8 font-mono text-[12px]"
              placeholder="gpt-5.5"
            />
          </DialogField>
          <DialogField label="Model Adapter">
            <select
              value={form.modelAdapter}
              onChange={(e) => handleField("modelAdapter", e.target.value)}
              className="h-8 w-full rounded-md border border-input bg-transparent px-2 text-[12px] text-foreground"
            >
              <option value="general">general</option>
              <option value="google">google</option>
              <option value="deepseek">deepseek</option>
              <option value="mimo">mimo</option>
            </select>
          </DialogField>
          <div className="grid grid-cols-2 gap-2">
            <DialogField label="温度">
              <Input
                type="number"
                step="0.1"
                value={form.temperature}
                onChange={(e) =>
                  handleField("temperature", parseFloat(e.target.value) || 0)
                }
                className="h-8 text-[12px]"
              />
            </DialogField>
            <DialogField label="最大输出 Tokens">
              <Input
                type="number"
                value={form.maxOutputTokens}
                onChange={(e) =>
                  handleField(
                    "maxOutputTokens",
                    parseInt(e.target.value, 10) || 0
                  )
                }
                className="h-8 text-[12px]"
              />
            </DialogField>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <DialogField label="上下文窗口（可选）">
              <Input
                type="number"
                value={form.contextWindow ?? 0}
                onChange={(e) =>
                  handleField(
                    "contextWindow",
                    e.target.value
                      ? parseInt(e.target.value, 10)
                      : undefined
                  )
                }
                className="h-8 text-[12px]"
                placeholder="如 128000"
              />
            </DialogField>
            <DialogField label="Max Tokens 限制（可选）">
              <Input
                type="number"
                value={form.maxTokensLimit ?? 0}
                onChange={(e) =>
                  handleField(
                    "maxTokensLimit",
                    e.target.value
                      ? parseInt(e.target.value, 10)
                      : undefined
                  )
                }
                className="h-8 text-[12px]"
                placeholder="如 65536"
              />
            </DialogField>
          </div>
          <div className="grid grid-cols-3 gap-2">
            <DialogCheckbox
              label="多模态"
              checked={form.isMultimodal}
              onChange={(v) => handleField("isMultimodal", v)}
            />
            <DialogCheckbox
              label="推理"
              checked={form.isReasoning}
              onChange={(v) => handleField("isReasoning", v)}
            />
            <DialogCheckbox
              label="工具"
              checked={form.supportsTools}
              onChange={(v) => handleField("supportsTools", v)}
            />
            <DialogCheckbox
              label="嵌入"
              checked={form.isEmbedding}
              onChange={(v) => handleField("isEmbedding", v)}
            />
            <DialogCheckbox
              label="重排序"
              checked={form.isReranker}
              onChange={(v) => handleField("isReranker", v)}
            />
            <DialogCheckbox
              label="生图"
              checked={form.isImageGeneration}
              onChange={(v) => handleField("isImageGeneration", v)}
            />
          </div>
        </div>
        <DialogFooter>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onOpenChange(false)}
            className="text-[12px]"
          >
            取消
          </Button>
          <Button size="sm" onClick={handleSave} className="text-[12px]">
            保存
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ============================================================
// Tab 3：角色分配
// ============================================================
function RolesTab({
  profiles,
  assignments,
  showToast,
  onChanged,
}: {
  profiles: ModelProfile[];
  assignments: ModelAssignments;
  showToast: (kind: "success" | "error", text: string) => void;
  onChanged: () => void | Promise<void>;
}) {
  const enabledProfiles = profiles.filter((p) => p.enabled);
  // 收藏在前，然后按 label
  const sorted = sortProfiles(enabledProfiles);

  const handleChange = async (
    key: keyof ModelAssignments,
    value: string
  ) => {
    const next: ModelAssignments = {
      ...assignments,
      [key]: value || undefined,
    };
    setAssignments(next);
    const ok = await saveAssignments(next);
    if (ok) {
      const role = MODEL_ROLES.find((r) => r.key === key);
      showToast(
        "success",
        `已设置「${role?.label ?? key}」→ ${
          sorted.find((p) => p.id === value)?.label ?? "（清空）"
        }`
      );
      await onChanged();
    } else {
      showToast("error", "角色分配保存失败");
    }
  };

  const setAssignments = useSessionStore((s) => s.setAssignments);

  return (
    <div className="space-y-1.5">
      <div className="text-[10px] text-muted-foreground/60">
        为每个角色指定一个已启用的模型；收藏的模型置顶。切换即保存。
      </div>
      <div className="grid grid-cols-1 gap-1.5 sm:grid-cols-2">
        {MODEL_ROLES.map((role) => {
          const value = (assignments[role.key] as string | undefined) ?? "";
          return (
            <div
              key={role.key}
              className="flex items-center gap-2 rounded-md border border-border bg-background px-2.5 py-1.5"
            >
              <div className="min-w-0 flex-1">
                <div className="text-[12px] font-medium text-foreground">
                  {role.label}
                </div>
                <div className="text-[10px] text-muted-foreground/60">
                  {role.hint}
                </div>
              </div>
              <select
                value={value}
                onChange={(e) => handleChange(role.key, e.target.value)}
                className="h-7 max-w-[180px] truncate rounded-md border border-input bg-transparent px-1.5 text-[11px] text-foreground"
              >
                <option value="">（未指定）</option>
                {sorted.map((p) => {
                  const vendorName =
                    profiles.find((v) => v.id === p.vendorId)?.label ??
                    p.vendorId;
                  return (
                    <option key={p.id} value={p.id}>
                      {p.isFavorite ? "★ " : ""}
                      {p.label} ({vendorName})
                    </option>
                  );
                })}
              </select>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ============================================================
// Dialog 复用小组件
// ============================================================
function DialogField({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
        {label}
      </label>
      {children}
    </div>
  );
}

function DialogCheckbox({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label className="flex cursor-pointer items-center gap-1.5 rounded-md border border-border bg-background px-2 py-1.5 text-[11px] text-foreground">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="h-3 w-3 accent-primary"
      />
      {label}
    </label>
  );
}

// ============================================================
// 外观：主题调色板
// ============================================================
function AppearanceSection({
  showToast,
}: {
  showToast: (kind: "success" | "error", text: string) => void;
}) {
  const themePalette = useSessionStore((s) => s.themePalette);
  const setThemePalette = useSessionStore((s) => s.setThemePalette);

  // 挂载时同步 DOM dataset 到当前 store 值
  useEffect(() => {
    document.documentElement.dataset.themePalette = themePalette;
  }, [themePalette]);

  const handlePick = (p: ThemePalette) => {
    setThemePalette(p);
    document.documentElement.dataset.themePalette = p;
    const opt = PALETTES.find((x) => x.key === p);
    showToast("success", `已切换主题：${opt?.label ?? p}`);
  };

  return (
    <Card>
      <CardHeader className="py-3">
        <CardTitle className="flex items-center gap-1.5 text-sm">
          <Palette size={14} className="text-primary" />
          外观
        </CardTitle>
        <CardDescription className="text-[11px]">
          主题调色板 —— 通过 data-theme-palette 切换 primary / ring
        </CardDescription>
      </CardHeader>
      <CardContent className="pt-0">
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
          {PALETTES.map((p) => {
            const active = themePalette === p.key;
            return (
              <button
                key={p.key}
                type="button"
                onClick={() => handlePick(p.key)}
                className={cn(
                  "group flex flex-col items-center gap-2 rounded-md border px-3 py-3 transition-colors",
                  active
                    ? "border-primary/50 bg-primary/10"
                    : "border-border bg-background hover:bg-accent"
                )}
              >
                <span
                  className="h-8 w-8 rounded-full border-2 border-white/20 shadow-soft"
                  style={{ backgroundColor: p.swatch }}
                  aria-hidden
                />
                <span
                  className={cn(
                    "text-[11px] font-medium",
                    active ? "text-primary" : "text-foreground"
                  )}
                >
                  {p.label}
                </span>
                {active && (
                  <span className="flex items-center gap-0.5 text-[9px] text-primary">
                    <Check size={9} />
                    当前
                  </span>
                )}
              </button>
            );
          })}
        </div>
        <div className="mt-2 text-[10px] text-muted-foreground/60">
          调色板只影响 accent（primary / ring），中性灰不变。
        </div>
      </CardContent>
    </Card>
  );
}

// ============================================================
// 索引管理（P0-B）
// ============================================================
function IndexSection({
  showToast,
}: {
  showToast: (kind: "success" | "error", text: string) => void;
}) {
  const [stats, setStats] = useState<IndexStats | null>(null);
  const [opts, setOpts] = useState<IndexOptions>(defaultIndexOptions());
  const [rebuilding, setRebuilding] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [searching, setSearching] = useState(false);
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [searchMode, setSearchMode] = useState<"fts" | "vector" | "hybrid">(
    "hybrid"
  );

  // 挂载时拉取索引统计
  useEffect(() => {
    void (async () => {
      const s = await fetchIndexStats();
      if (s) setStats(s);
    })();
  }, []);

  const refreshStats = async () => {
    const s = await fetchIndexStats();
    if (s) setStats(s);
  };

  const handleRebuild = async () => {
    setRebuilding(true);
    try {
      const res = await rebuildAllIndexes(opts);
      if (res) {
        showToast(
          "success",
          `重建完成：成功 ${res.success} / 失败 ${res.failed} / 共 ${res.total}`
        );
        await refreshStats();
      } else {
        showToast("error", "重建失败：后端不可用");
      }
    } catch (err) {
      showToast(
        "error",
        "重建异常：" + (err instanceof Error ? err.message : String(err))
      );
    } finally {
      setRebuilding(false);
    }
  };

  const handleSearch = async () => {
    if (!searchQuery.trim()) {
      showToast("error", "请输入搜索关键词");
      return;
    }
    setSearching(true);
    try {
      const q = defaultSearchQuery(searchQuery);
      q.useFts = searchMode === "fts" || searchMode === "hybrid";
      q.useVector = searchMode === "vector" || searchMode === "hybrid";
      q.useRerank = searchMode === "vector";
      const res = await indexSearch(q);
      if (res) {
        setSearchResults(res);
        if (res.length === 0) {
          showToast("success", "未找到匹配结果");
        } else {
          showToast("success", `找到 ${res.length} 条结果`);
        }
      } else {
        showToast("error", "搜索失败：后端不可用");
      }
    } finally {
      setSearching(false);
    }
  };

  return (
    <Card>
      <CardHeader className="py-3">
        <CardTitle className="flex items-center gap-1.5 text-sm">
          <Database size={14} className="text-primary" />
          索引管理
        </CardTitle>
        <CardDescription className="text-[11px]">
          FTS5 全文索引 + 向量嵌入 + RAG 检索 —— 重建 / 统计 / 搜索测试
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4 pt-0">
        {/* 索引统计 */}
        <div className="grid grid-cols-3 gap-2 sm:grid-cols-6">
          <StatBox
            label="资源总数"
            value={stats?.totalResources ?? 0}
            icon={<Database size={11} />}
          />
          <StatBox
            label="已索引"
            value={stats?.indexedResources ?? 0}
            icon={<Check size={11} />}
          />
          <StatBox
            label="切片数"
            value={stats?.totalChunks ?? 0}
            icon={<Activity size={11} />}
          />
          <StatBox
            label="已嵌入"
            value={stats?.embeddedChunks ?? 0}
            icon={<Zap size={11} />}
          />
          <StatBox
            label="FTS 行"
            value={stats?.ftsRows ?? 0}
            icon={<Search size={11} />}
          />
          <StatBox
            label="均 token"
            value={stats?.avgChunkTokens ?? 0}
            icon={<Activity size={11} />}
          />
        </div>

        {/* 重建选项 */}
        <div className="rounded-md border border-border bg-muted/30 px-3 py-2.5 space-y-2.5">
          <div className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            重建选项
          </div>
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
            <NumField
              label="切片大小"
              value={opts.chunkSize}
              onChange={(v) => setOpts({ ...opts, chunkSize: v })}
            />
            <NumField
              label="切片重叠"
              value={opts.chunkOverlap}
              onChange={(v) => setOpts({ ...opts, chunkOverlap: v })}
            />
            <NumField
              label="最小切片"
              value={opts.minChunkSize}
              onChange={(v) => setOpts({ ...opts, minChunkSize: v })}
            />
            <div className="flex flex-col gap-1">
              <label className="text-[10px] text-muted-foreground/70">
                向量嵌入
              </label>
              <button
                type="button"
                onClick={() => setOpts({ ...opts, embed: !opts.embed })}
                className={cn(
                  "h-8 rounded-md border px-2 text-[11px] font-medium transition-colors",
                  opts.embed
                    ? "border-primary/40 bg-primary/15 text-primary"
                    : "border-border bg-background text-muted-foreground"
                )}
              >
                {opts.embed ? "已启用" : "已关闭"}
              </button>
            </div>
          </div>
          {opts.embed && (
            <div className="flex items-center gap-2">
              <label className="w-20 shrink-0 text-[10px] font-medium text-muted-foreground/70">
                嵌入模型
              </label>
              <Input
                value={opts.embedModel}
                onChange={(e) =>
                  setOpts({ ...opts, embedModel: e.target.value })
                }
                placeholder="text-embedding-3-small"
                className="h-8 font-mono text-[11px]"
              />
            </div>
          )}
          <div className="flex items-center gap-2">
            <Button
              variant="default"
              size="sm"
              className="h-8"
              onClick={handleRebuild}
              disabled={rebuilding}
            >
              {rebuilding ? (
                <Loader2 size={12} className="animate-spin" />
              ) : (
                <RefreshCw size={12} />
              )}
              重建全部索引
            </Button>
            <span className="text-[10px] text-muted-foreground/60">
              重建会清空旧索引并重新切片（可能耗时较长）
            </span>
          </div>
        </div>

        {/* 搜索测试 */}
        <div className="rounded-md border border-border bg-muted/30 px-3 py-2.5 space-y-2">
          <div className="flex items-center justify-between">
            <div className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              搜索测试
            </div>
            <div className="flex items-center gap-1">
              {(["fts", "vector", "hybrid"] as const).map((m) => (
                <button
                  key={m}
                  type="button"
                  onClick={() => setSearchMode(m)}
                  className={cn(
                    "rounded px-2 py-0.5 text-[10px] font-medium transition-colors",
                    searchMode === m
                      ? "bg-primary/15 text-primary"
                      : "text-muted-foreground hover:bg-accent"
                  )}
                >
                  {m === "fts" ? "FTS" : m === "vector" ? "向量" : "混合"}
                </button>
              ))}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <div className="relative min-w-0 flex-1">
              <Search
                size={11}
                className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground"
              />
              <Input
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") void handleSearch();
                }}
                placeholder="输入关键词测试索引效果…"
                className="h-8 pl-8 text-[11px]"
              />
            </div>
            <Button
              variant="outline"
              size="sm"
              className="h-8"
              onClick={handleSearch}
              disabled={searching}
            >
              {searching ? (
                <Loader2 size={12} className="animate-spin" />
              ) : (
                <Search size={12} />
              )}
              搜索
            </Button>
          </div>
          {searchResults.length > 0 && (
            <div className="max-h-48 space-y-1 overflow-y-auto scrollbar-dark">
              {searchResults.map((r, i) => (
                <div
                  key={`${r.uri}-${i}`}
                  className="rounded border border-border bg-background px-2 py-1.5 text-[11px]"
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="min-w-0 flex-1 truncate font-medium text-foreground">
                      {r.title || r.uri}
                    </span>
                    <Badge variant="outline" className="shrink-0 font-mono text-[9px]">
                      {r.score.toFixed(3)}
                    </Badge>
                  </div>
                  <div className="mt-0.5 line-clamp-2 text-[10px] text-muted-foreground/70">
                    {r.snippet}
                  </div>
                  <div className="mt-0.5 flex items-center gap-2 text-[9px] text-muted-foreground/50">
                    <span className="font-mono">{r.uri}</span>
                    {r.ftsScore !== undefined && r.ftsScore > 0 && (
                      <span>FTS:{r.ftsScore.toFixed(2)}</span>
                    )}
                    {r.vecScore !== undefined && r.vecScore > 0 && (
                      <span>VEC:{r.vecScore.toFixed(2)}</span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

function StatBox({
  label,
  value,
  icon,
}: {
  label: string;
  value: number;
  icon: ReactNode;
}) {
  return (
    <div className="rounded-md border border-border bg-background px-2 py-1.5">
      <div className="flex items-center gap-1 text-[9px] text-muted-foreground/60">
        {icon}
        {label}
      </div>
      <div className="mt-0.5 font-mono text-[14px] font-semibold text-foreground">
        {value}
      </div>
    </div>
  );
}

function NumField({
  label,
  value,
  onChange,
}: {
  label: string;
  value: number;
  onChange: (v: number) => void;
}) {
  return (
    <div className="flex flex-col gap-1">
      <label className="text-[10px] text-muted-foreground/70">{label}</label>
      <Input
        type="number"
        value={value}
        onChange={(e) => onChange(Number(e.target.value) || 0)}
        className="h-8 font-mono text-[11px]"
      />
    </div>
  );
}

// ============================================================
// 关于
// ============================================================
function AboutSection() {
  const version = useSessionStore((s) => s.version);
  const dataDir = useSessionStore((s) => s.dataDir);
  const registeredProviders = useSessionStore((s) => s.registeredProviders);

  return (
    <Card>
      <CardHeader className="py-3">
        <CardTitle className="flex items-center gap-1.5 text-sm">
          <Info size={14} className="text-primary" />
          关于
        </CardTitle>
        <CardDescription className="text-[11px]">
          DeepStudent Go —— 学习中心 / 翻译 / 技能 / 治理
        </CardDescription>
      </CardHeader>
      <CardContent className="pt-0">
        <dl className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <AboutItem label="版本" value={version} />
          <AboutItem label="数据目录" value={dataDir} mono />
          <div className="sm:col-span-2">
            <dt className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              已注册 Providers
            </dt>
            <dd className="mt-1">
              {registeredProviders.length === 0 ? (
                <span className="text-[11px] text-muted-foreground/60">
                  （暂无 —— 后端可能未连接）
                </span>
              ) : (
                <div className="flex flex-wrap gap-1">
                  {registeredProviders.map((p) => (
                    <Badge
                      key={p}
                      variant="outline"
                      className="font-mono text-[10px]"
                    >
                      {p}
                    </Badge>
                  ))}
                </div>
              )}
            </dd>
          </div>
        </dl>
      </CardContent>
    </Card>
  );
}

function AboutItem({
  label,
  value,
  mono,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div className="flex items-center gap-2 rounded-md border border-border bg-muted/30 px-2.5 py-1.5">
      <dt className="shrink-0 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
        {label}
      </dt>
      <dd
        className={cn(
          "min-w-0 flex-1 truncate text-[12px] text-foreground",
          mono && "font-mono"
        )}
        title={value}
      >
        {value}
      </dd>
    </div>
  );
}
