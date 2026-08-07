// GovernancePage —— 数据治理
// ------------------------------------------------------------
// 顶部：状态卡片（GovStatus map → key-value 网格）
// 主体分块：
// - 备份恢复
// - 导入导出（类型多选）
// - 槽位切换 A/B
// - 完整性检查（缺失文件列表）
// - 审计日志（时间线）

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Badge } from "@/components/ui/Badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/Card";
import { cn, formatTime, formatDate, relativeTime } from "@/lib/utils";
import {
  useGovernanceStore,
  EXPORT_TYPES,
  type GovSlot,
} from "@/state/governance";
import {
  Shield,
  Database,
  Archive,
  Download,
  Upload,
  ArrowLeftRight,
  Activity,
  CheckCircle2,
  AlertCircle,
  Loader2,
  Inbox,
  History,
  RefreshCw,
  HardDrive,
} from "lucide-react";

export function GovernancePage() {
  const refreshStatus = useGovernanceStore((s) => s.refreshStatus);
  const refreshAudit = useGovernanceStore((s) => s.refreshAudit);
  const error = useGovernanceStore((s) => s.error);
  const notice = useGovernanceStore((s) => s.notice);
  const clearError = useGovernanceStore((s) => s.clearError);
  const clearNotice = useGovernanceStore((s) => s.clearNotice);

  useEffect(() => {
    void refreshStatus();
    void refreshAudit();
    // 完整性检查需用户主动触发，避免页面加载就跑
  }, [refreshStatus, refreshAudit]);

  return (
    <div className="flex h-full w-full min-h-0 flex-col bg-background">
      {/* —— 顶部标题栏 —— */}
      <header className="shrink-0 flex items-center justify-between border-b border-border bg-card px-4 py-3">
        <div className="flex items-center gap-2 text-sm">
          <Shield size={16} className="text-primary" />
          <h1 className="font-semibold text-foreground">数据治理</h1>
          <span className="text-[11px] text-muted-foreground/60">
            备份 / 恢复 / 槽位 / 导入导出 / 完整性 / 审计
          </span>
        </div>
        <Button
          variant="ghost"
          size="sm"
          className="h-7"
          onClick={() => {
            void refreshStatus();
            void refreshAudit();
          }}
        >
          <RefreshCw size={12} />
          刷新
        </Button>
      </header>

      {/* —— 错误 / 通知横幅 —— */}
      {error && (
        <div className="shrink-0 flex items-center justify-between gap-2 border-b border-destructive/30 bg-destructive/10 px-4 py-2 text-xs text-destructive">
          <div className="flex min-w-0 items-center gap-2">
            <AlertCircle size={12} className="shrink-0" />
            <span className="truncate">{error}</span>
          </div>
          <button
            type="button"
            onClick={clearError}
            className="shrink-0 text-destructive/70 hover:text-destructive"
          >
            ×
          </button>
        </div>
      )}
      {notice && (
        <div className="shrink-0 flex items-center justify-between gap-2 border-b border-emerald-500/30 bg-emerald-500/10 px-4 py-2 text-xs text-emerald-500">
          <div className="flex min-w-0 items-center gap-2">
            <CheckCircle2 size={12} className="shrink-0" />
            <span className="truncate">{notice}</span>
          </div>
          <button
            type="button"
            onClick={clearNotice}
            className="shrink-0 text-emerald-500/70 hover:text-emerald-500"
          >
            ×
          </button>
        </div>
      )}

      {/* —— 主体滚动区 —— */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        <div className="space-y-4 p-4">
          {/* 状态卡片 */}
          <StatusCard />

          {/* 双栏：备份恢复 + 导入导出 */}
          <div className="grid gap-4 lg:grid-cols-2">
            <BackupRestoreCard />
            <ImportExportCard />
          </div>

          {/* 双栏：槽位切换 + 完整性检查 */}
          <div className="grid gap-4 lg:grid-cols-2">
            <SlotSwitchCard />
            <IntegrityCard />
          </div>

          {/* 审计日志 */}
          <AuditCard />
        </div>
      </div>
    </div>
  );
}

// ============================================================
// 状态卡片
// ============================================================
function StatusCard() {
  const status = useGovernanceStore((s) => s.status);
  const loading = useGovernanceStore((s) => s.loadingStatus);

  const entries = Object.entries(status);

  return (
    <Card>
      <CardHeader className="py-3">
        <CardTitle className="flex items-center gap-1.5 text-sm">
          <Database size={14} className="text-primary" />
          治理状态
          {loading && (
            <Loader2 size={11} className="animate-spin text-muted-foreground" />
          )}
        </CardTitle>
        <CardDescription className="text-[11px]">
          后端 GovStatus() 返回的运行时状态
        </CardDescription>
      </CardHeader>
      <CardContent className="pt-0">
        {entries.length === 0 ? (
          <EmptyMini
            loading={loading}
            icon={Database}
            title="暂无状态数据"
            hint="点击右上角刷新或确认后端可用"
          />
        ) : (
          <dl className="grid grid-cols-2 gap-x-4 gap-y-2 sm:grid-cols-3 md:grid-cols-4">
            {entries.map(([k, v]) => (
              <div
                key={k}
                className="rounded-md border border-border bg-muted/30 px-2.5 py-1.5"
              >
                <dt className="truncate font-mono text-[10px] text-muted-foreground/70">
                  {k}
                </dt>
                <dd className="mt-0.5 truncate text-[12px] font-medium text-foreground">
                  {formatStatusValue(v)}
                </dd>
              </div>
            ))}
          </dl>
        )}
      </CardContent>
    </Card>
  );
}

/** 把后端返回的 status value 格式化为字符串 */
function formatStatusValue(v: unknown): string {
  if (v === null || v === undefined) return "—";
  if (typeof v === "string") return v;
  if (typeof v === "number" || typeof v === "boolean") return String(v);
  try {
    return JSON.stringify(v);
  } catch {
    return String(v);
  }
}

// ============================================================
// 备份恢复卡片
// ============================================================
function BackupRestoreCard() {
  const backupTarget = useGovernanceStore((s) => s.backupTarget);
  const restoreSource = useGovernanceStore((s) => s.restoreSource);
  const loadingBackup = useGovernanceStore((s) => s.loadingBackup);
  const loadingRestore = useGovernanceStore((s) => s.loadingRestore);
  const setBackupTarget = useGovernanceStore((s) => s.setBackupTarget);
  const setRestoreSource = useGovernanceStore((s) => s.setRestoreSource);
  const backup = useGovernanceStore((s) => s.backup);
  const restore = useGovernanceStore((s) => s.restore);

  return (
    <Card>
      <CardHeader className="py-3">
        <CardTitle className="flex items-center gap-1.5 text-sm">
          <Archive size={14} className="text-primary" />
          备份 / 恢复
        </CardTitle>
        <CardDescription className="text-[11px]">
          GovBackup / GovRestore
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3 pt-0">
        <div>
          <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            备份目标路径
          </label>
          <div className="flex items-center gap-2">
            <Input
              value={backupTarget}
              onChange={(e) => setBackupTarget(e.target.value)}
              placeholder="/path/to/backup.tar.gz"
              className="h-8 font-mono text-[11px]"
            />
            <Button
              size="sm"
              className="h-8"
              onClick={() => void backup()}
              disabled={loadingBackup || !backupTarget.trim()}
            >
              {loadingBackup ? (
                <Loader2 size={12} className="animate-spin" />
              ) : (
                <Archive size={12} />
              )}
              备份
            </Button>
          </div>
        </div>
        <div>
          <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            恢复源路径
          </label>
          <div className="flex items-center gap-2">
            <Input
              value={restoreSource}
              onChange={(e) => setRestoreSource(e.target.value)}
              placeholder="/path/to/backup.tar.gz"
              className="h-8 font-mono text-[11px]"
            />
            <Button
              size="sm"
              variant="outline"
              className="h-8"
              onClick={() => void restore()}
              disabled={loadingRestore || !restoreSource.trim()}
            >
              {loadingRestore ? (
                <Loader2 size={12} className="animate-spin" />
              ) : (
                <Upload size={12} />
              )}
              恢复
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// ============================================================
// 导入导出卡片
// ============================================================
function ImportExportCard() {
  const exportTarget = useGovernanceStore((s) => s.exportTarget);
  const importSource = useGovernanceStore((s) => s.importSource);
  const exportTypes = useGovernanceStore((s) => s.exportTypes);
  const loadingExport = useGovernanceStore((s) => s.loadingExport);
  const loadingImport = useGovernanceStore((s) => s.loadingImport);
  const setExportTarget = useGovernanceStore((s) => s.setExportTarget);
  const setImportSource = useGovernanceStore((s) => s.setImportSource);
  const toggleExportType = useGovernanceStore((s) => s.toggleExportType);
  const exportData = useGovernanceStore((s) => s.exportData);
  const importData = useGovernanceStore((s) => s.importData);

  return (
    <Card>
      <CardHeader className="py-3">
        <CardTitle className="flex items-center gap-1.5 text-sm">
          <Download size={14} className="text-primary" />
          导入 / 导出
        </CardTitle>
        <CardDescription className="text-[11px]">
          GovExport / GovImport
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3 pt-0">
        {/* 导出类型多选 */}
        <div>
          <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            导出类型（多选）
          </label>
          <div className="flex flex-wrap gap-1">
            {EXPORT_TYPES.map((t) => {
              const active = exportTypes.includes(t.key);
              return (
                <button
                  key={t.key}
                  type="button"
                  onClick={() => toggleExportType(t.key)}
                  className={cn(
                    "rounded-full border px-2 py-0.5 text-[10px] font-medium transition-colors",
                    active
                      ? "border-primary/40 bg-primary/15 text-primary"
                      : "border-border bg-transparent text-muted-foreground hover:bg-accent hover:text-foreground"
                  )}
                >
                  {t.label}
                </button>
              );
            })}
          </div>
        </div>

        {/* 导出 */}
        <div>
          <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            导出目标路径
          </label>
          <div className="flex items-center gap-2">
            <Input
              value={exportTarget}
              onChange={(e) => setExportTarget(e.target.value)}
              placeholder="/path/to/export"
              className="h-8 font-mono text-[11px]"
            />
            <Button
              size="sm"
              className="h-8"
              onClick={() => void exportData()}
              disabled={
                loadingExport ||
                !exportTarget.trim() ||
                exportTypes.length === 0
              }
            >
              {loadingExport ? (
                <Loader2 size={12} className="animate-spin" />
              ) : (
                <Download size={12} />
              )}
              导出
            </Button>
          </div>
          {exportTypes.length > 0 && (
            <div className="mt-1 text-[10px] text-muted-foreground/60">
              已选 {exportTypes.length} 类
            </div>
          )}
        </div>

        {/* 导入 */}
        <div>
          <label className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            导入源路径
          </label>
          <div className="flex items-center gap-2">
            <Input
              value={importSource}
              onChange={(e) => setImportSource(e.target.value)}
              placeholder="/path/to/import"
              className="h-8 font-mono text-[11px]"
            />
            <Button
              size="sm"
              variant="outline"
              className="h-8"
              onClick={() => void importData()}
              disabled={loadingImport || !importSource.trim()}
            >
              {loadingImport ? (
                <Loader2 size={12} className="animate-spin" />
              ) : (
                <Upload size={12} />
              )}
              导入
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// ============================================================
// 槽位切换卡片
// ============================================================
function SlotSwitchCard() {
  const slot = useGovernanceStore((s) => s.slot);
  const setSlot = useGovernanceStore((s) => s.setSlot);
  const loadingSlot = useGovernanceStore((s) => s.loadingSlot);
  const switchSlot = useGovernanceStore((s) => s.switchSlot);

  return (
    <Card>
      <CardHeader className="py-3">
        <CardTitle className="flex items-center gap-1.5 text-sm">
          <ArrowLeftRight size={14} className="text-primary" />
          加密槽位切换
        </CardTitle>
        <CardDescription className="text-[11px]">
          GovSwitchSlot —— 在 A / B 加密槽位之间切换
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3 pt-0">
        <div className="flex items-center gap-2">
          {(["A", "B"] as GovSlot[]).map((s) => {
            const active = slot === s;
            return (
              <button
                key={s}
                type="button"
                onClick={() => setSlot(s)}
                className={cn(
                  "flex flex-1 items-center justify-center gap-2 rounded-md border px-3 py-2 transition-colors",
                  active
                    ? "border-primary/40 bg-primary/15 text-primary"
                    : "border-border bg-background text-muted-foreground hover:bg-accent hover:text-foreground"
                )}
              >
                <HardDrive
                  size={14}
                  className={active ? "text-primary" : "text-muted-foreground"}
                />
                <span className="text-sm font-semibold">槽位 {s}</span>
              </button>
            );
          })}
        </div>
        <Button
          size="sm"
          className="h-8 w-full"
          onClick={() => void switchSlot()}
          disabled={loadingSlot}
        >
          {loadingSlot ? (
            <Loader2 size={12} className="animate-spin" />
          ) : (
            <ArrowLeftRight size={12} />
          )}
          切换到槽位 {slot}
        </Button>
      </CardContent>
    </Card>
  );
}

// ============================================================
// 完整性检查卡片
// ============================================================
function IntegrityCard() {
  const integrity = useGovernanceStore((s) => s.integrity);
  const loadingIntegrity = useGovernanceStore((s) => s.loadingIntegrity);
  const checkIntegrity = useGovernanceStore((s) => s.checkIntegrity);
  const [hasChecked, setHasChecked] = useState(false);

  const handleCheck = async () => {
    setHasChecked(true);
    await checkIntegrity();
  };

  return (
    <Card>
      <CardHeader className="py-3">
        <CardTitle className="flex items-center gap-1.5 text-sm">
          <CheckCircle2 size={14} className="text-primary" />
          完整性检查
        </CardTitle>
        <CardDescription className="text-[11px]">
          GovIntegrityCheck —— 检查缺失文件 / 损坏条目
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3 pt-0">
        <Button
          size="sm"
          variant="outline"
          className="h-8"
          onClick={handleCheck}
          disabled={loadingIntegrity}
        >
          {loadingIntegrity ? (
            <Loader2 size={12} className="animate-spin" />
          ) : (
            <CheckCircle2 size={12} />
          )}
          检查完整性
        </Button>

        {hasChecked && !loadingIntegrity && integrity.length === 0 ? (
          <div className="flex items-center gap-2 rounded-md border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-[11px] text-emerald-500">
            <CheckCircle2 size={12} />
            完整性检查通过 —— 未发现缺失或损坏
          </div>
        ) : integrity.length > 0 ? (
          <div className="rounded-md border border-destructive/30 bg-destructive/5 p-2">
            <div className="mb-1 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-destructive">
              <AlertCircle size={10} />
              发现 {integrity.length} 个问题
            </div>
            <ul className="max-h-40 space-y-0.5 overflow-y-auto scrollbar-dark">
              {integrity.map((p, idx) => (
                <li
                  key={idx}
                  className="truncate font-mono text-[11px] text-foreground/80"
                  title={p}
                >
                  · {p}
                </li>
              ))}
            </ul>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

// ============================================================
// 审计日志卡片
// ============================================================
function AuditCard() {
  const audit = useGovernanceStore((s) => s.audit);
  const auditLimit = useGovernanceStore((s) => s.auditLimit);
  const loadingAudit = useGovernanceStore((s) => s.loadingAudit);
  const setAuditLimit = useGovernanceStore((s) => s.setAuditLimit);
  const refreshAudit = useGovernanceStore((s) => s.refreshAudit);

  return (
    <Card>
      <CardHeader className="py-3">
        <CardTitle className="flex items-center gap-1.5 text-sm">
          <History size={14} className="text-primary" />
          审计日志
        </CardTitle>
        <CardDescription className="text-[11px]">
          GovAudit —— 最近 N 条操作记录
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3 pt-0">
        {/* 数量输入 + 查询按钮 */}
        <div className="flex items-end gap-2">
          <label className="block">
            <span className="mb-1 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              数量
            </span>
            <Input
              type="number"
              min={1}
              max={500}
              value={auditLimit}
              onChange={(e) =>
                setAuditLimit(
                  Math.max(1, Math.min(500, Number(e.target.value) || 50))
                )
              }
              className="h-8 w-24 text-[12px]"
            />
          </label>
          <Button
            size="sm"
            className="h-8"
            onClick={() => void refreshAudit()}
            disabled={loadingAudit}
          >
            {loadingAudit ? (
              <Loader2 size={12} className="animate-spin" />
            ) : (
              <Activity size={12} />
            )}
            查询审计
          </Button>
          <span className="pb-1.5 text-[10px] text-muted-foreground/60">
            共 {audit.length} 条
          </span>
        </div>

        {/* 时间线列表 */}
        {audit.length === 0 ? (
          <EmptyMini
            loading={loadingAudit}
            icon={History}
            title="暂无审计日志"
            hint="调整数量后点击「查询审计」"
          />
        ) : (
          <ol className="relative space-y-2 border-l border-border pl-4">
            {audit.map((entry, idx) => (
              <li key={idx} className="relative">
                <span className="absolute -left-[1.3rem] top-1 h-2 w-2 rounded-full bg-primary" />
                <div className="rounded-md border border-border bg-muted/30 px-2.5 py-1.5">
                  <div className="flex items-center justify-between gap-2">
                    <div className="flex min-w-0 items-center gap-1.5">
                      <Badge
                        variant="outline"
                        className="shrink-0 px-1.5 py-0 font-mono text-[9px]"
                      >
                        {entry.action}
                      </Badge>
                      <span className="truncate font-mono text-[11px] text-foreground/80">
                        {entry.actor}
                      </span>
                    </div>
                    <span
                      className="shrink-0 text-[10px] text-muted-foreground/60"
                      title={formatDate(entry.ts) + " " + formatTime(entry.ts)}
                    >
                      {relativeTime(entry.ts)}
                    </span>
                  </div>
                  {entry.detail && (
                    <div className="mt-1 text-[11px] text-muted-foreground">
                      {entry.detail}
                    </div>
                  )}
                </div>
              </li>
            ))}
          </ol>
        )}
      </CardContent>
    </Card>
  );
}

// ============================================================
// 内部辅助组件
// ============================================================
function EmptyMini({
  loading,
  icon: Icon,
  title,
  hint,
}: {
  loading: boolean;
  icon: typeof Inbox;
  title: string;
  hint: string;
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 px-4 py-8 text-center">
      <div className="flex h-9 w-9 items-center justify-center rounded-full bg-muted text-muted-foreground">
        {loading ? (
          <Loader2 size={14} className="animate-spin" />
        ) : (
          <Icon size={14} />
        )}
      </div>
      <div className="space-y-0.5">
        <div className="text-[12px] font-medium text-foreground">
          {loading ? "加载中…" : title}
        </div>
        <div className="text-[10px] text-muted-foreground">{hint}</div>
      </div>
    </div>
  );
}
