// SyncPage —— 云同步与数据治理面板
// ------------------------------------------------------------
// - 云存储配置（WebDAV / S3）
// - 同步状态 + 执行同步
// - 加密备份上传 / 版本列表 / 下载 / 删除
// - 隔离区（重试 / 丢弃）

import { useEffect, useState } from "react";
import { cloudApi, syncApi, type CloudConfig, type CloudVersion, type SyncStatus, type QuarantineEntry } from "@/lib/cloud";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Badge } from "@/components/ui/Badge";
import { Switch } from "@/components/ui/Switch";
import {
  Cloud,
  CloudUpload,
  CloudDownload,
  RefreshCw,
  AlertTriangle,
  CheckCircle2,
  Loader2,
  Trash2,
  RotateCcw,
  Server,
  ShieldCheck,
} from "lucide-react";
import { cn } from "@/lib/utils";

export function SyncPage() {
  const [provider, setProvider] = useState<"webdav" | "s3">("webdav");
  const [cfg, setCfg] = useState<CloudConfig>({ provider: "webdav" });
  const [configured, setConfigured] = useState(false);
  const [status, setStatus] = useState<SyncStatus | null>(null);
  const [versions, setVersions] = useState<CloudVersion[]>([]);
  const [quarantine, setQuarantine] = useState<QuarantineEntry[]>([]);
  const [busy, setBusy] = useState<string | null>(null);
  const [toast, setToast] = useState<{ kind: "success" | "error"; text: string } | null>(null);

  const loadAll = async () => {
    const [cfgRes, statusRes, verRes, qRes] = await Promise.all([
      cloudApi.loadConfig(),
      syncApi.status(),
      cloudApi.listVersions(),
      syncApi.listQuarantine(100),
    ]);
    if (cfgRes) {
      setCfg({ ...cfgRes[0], provider: cfgRes[0].provider || "webdav" });
      setProvider(cfgRes[0].provider || "webdav");
      setConfigured(cfgRes[1]);
    }
    if (statusRes) setStatus(statusRes);
    if (verRes) setVersions(verRes);
    if (qRes) setQuarantine(qRes);
  };

  useEffect(() => {
    void loadAll();
  }, []);

  const save = async () => {
    setBusy("save");
    const ok = await cloudApi.saveConfig({ ...cfg, provider });
    if (ok === null) setToast({ kind: "error", text: "保存失败" });
    else { setConfigured(true); setToast({ kind: "success", text: "配置已保存" }); }
    setBusy(null);
    await loadAll();
  };

  const runSync = async () => {
    setBusy("sync");
    const res = await syncApi.run();
    if (res) setToast({ kind: "success", text: `同步完成：上传 ${res.uploaded} · 下载 ${res.downloaded} · 隔离 ${res.quarantined}` });
    setBusy(null);
    await loadAll();
  };

  const upload = async () => {
    setBusy("upload");
    const v = await cloudApi.uploadBackup("", "手动备份");
    if (v) setToast({ kind: "success", text: `已上传 ${v.key.split("/").pop()}` });
    setBusy(null);
    await loadAll();
  };

  return (
    <div className="flex h-full w-full min-h-0 flex-col bg-background">
      <div className="shrink-0 border-b border-border bg-card px-4 py-2.5">
        <div className="flex items-center gap-2">
          <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary/15 text-primary">
            <Cloud size={13} />
          </div>
          <h1 className="text-sm font-semibold text-foreground">云同步与治理</h1>
          {toast && (
            <span className={cn("ml-auto text-[11px]", toast.kind === "success" ? "text-emerald-500" : "text-destructive")}>
              {toast.text}
            </span>
          )}
        </div>
      </div>

      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto scrollbar-dark p-4">
        {/* —— 云存储配置 —— */}
        <div className="rounded-md border border-border bg-card p-4">
          <div className="mb-3 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            <Server size={12} className="text-primary" />
            云存储配置
          </div>
          <div className="mb-3 flex items-center gap-3">
            <div className="flex items-center gap-1.5">
              <Switch checked={provider === "webdav"} onCheckedChange={(v) => setProvider(v ? "webdav" : "s3")} />
              <span className="text-[12px] text-foreground">WebDAV</span>
            </div>
            <div className="flex items-center gap-1.5">
              <Switch checked={provider === "s3"} onCheckedChange={(v) => setProvider(v ? "s3" : "webdav")} />
              <span className="text-[12px] text-foreground">S3 兼容</span>
            </div>
            {configured && <Badge variant="secondary" className="ml-auto">已配置</Badge>}
          </div>

          {provider === "webdav" ? (
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
              <Input value={cfg.webdavUrl ?? ""} onChange={(e) => setCfg({ ...cfg, webdavUrl: e.target.value })} placeholder="WebDAV URL（如 https://dav.jianguoyun.com/dav）" className="h-8 text-[11px]" />
              <Input value={cfg.webdavUser ?? ""} onChange={(e) => setCfg({ ...cfg, webdavUser: e.target.value })} placeholder="用户名" className="h-8 text-[11px]" />
              <Input type="password" value={cfg.webdavPass ?? ""} onChange={(e) => setCfg({ ...cfg, webdavPass: e.target.value })} placeholder="密码（加密存储）" className="h-8 text-[11px]" />
            </div>
          ) : (
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              <Input value={cfg.s3Endpoint ?? ""} onChange={(e) => setCfg({ ...cfg, s3Endpoint: e.target.value })} placeholder="S3 Endpoint（R2/OSS/MinIO）" className="h-8 text-[11px]" />
              <Input value={cfg.s3Bucket ?? ""} onChange={(e) => setCfg({ ...cfg, s3Bucket: e.target.value })} placeholder="Bucket" className="h-8 text-[11px]" />
              <Input value={cfg.s3AccessKey ?? ""} onChange={(e) => setCfg({ ...cfg, s3AccessKey: e.target.value })} placeholder="Access Key" className="h-8 text-[11px]" />
              <Input type="password" value={cfg.s3SecretKey ?? ""} onChange={(e) => setCfg({ ...cfg, s3SecretKey: e.target.value })} placeholder="Secret Key（加密存储）" className="h-8 text-[11px]" />
            </div>
          )}
          <div className="mt-2">
            <Input value={cfg.remoteDir ?? ""} onChange={(e) => setCfg({ ...cfg, remoteDir: e.target.value })} placeholder="远端目录（默认 deepstudent-backups）" className="h-8 text-[11px]" />
          </div>
          <div className="mt-3 flex items-center gap-2">
            <Button size="sm" className="h-7" disabled={busy !== null} onClick={() => void save()}>
              {busy === "save" ? <Loader2 size={12} className="mr-1 animate-spin" /> : <ShieldCheck size={12} className="mr-1" />}
              保存配置
            </Button>
            <Button size="sm" variant="outline" className="h-7" disabled={busy !== null} onClick={() => void (async () => {
              setBusy("check");
              const r = await cloudApi.checkConnection();
              setToast(r === null ? { kind: "error", text: "连接失败" } : { kind: "success", text: "连接成功" });
              setBusy(null);
            })()}>
              {busy === "check" ? <Loader2 size={12} className="mr-1 animate-spin" /> : <RefreshCw size={12} className="mr-1" />}
              测试连接
            </Button>
          </div>
        </div>

        {/* —— 同步状态 + 执行 —— */}
        <div className="rounded-md border border-border bg-card p-4">
          <div className="mb-3 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            <RefreshCw size={12} className="text-primary" />
            增量同步
          </div>
          <div className="mb-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
            <StatusChip label="待同步变更" value={String(status?.pending ?? "-")} />
            <StatusChip label="已同步游标" value={String(status?.cursor ?? "-")} />
            <StatusChip label="隔离区" value={String(status?.quarantine ?? "-")} />
            <StatusChip label="云端状态" value={status?.cloud?.connected ? "已连接" : "未连接"} ok={status?.cloud?.connected} />
          </div>
          <div className="flex items-center gap-2">
            <Button size="sm" className="h-7" disabled={busy !== null || !configured} onClick={() => void runSync()}>
              {busy === "sync" ? <Loader2 size={12} className="mr-1 animate-spin" /> : <RefreshCw size={12} className="mr-1" />}
              执行同步
            </Button>
            <Button size="sm" variant="outline" className="h-7" disabled={busy !== null || !configured} onClick={() => void upload()}>
              {busy === "upload" ? <Loader2 size={12} className="mr-1 animate-spin" /> : <CloudUpload size={12} className="mr-1" />}
              上传加密备份
            </Button>
            {!configured && (
              <span className="text-[10px] text-muted-foreground">先保存云配置</span>
            )}
          </div>
        </div>

        {/* —— 版本列表 —— */}
        <div className="rounded-md border border-border bg-card p-4">
          <div className="mb-2 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            <CloudDownload size={12} className="text-primary" />
            云端备份版本（{versions.length}）
          </div>
          {versions.length === 0 ? (
            <p className="py-3 text-center text-[11px] text-muted-foreground">暂无云端备份</p>
          ) : (
            <div className="space-y-1">
              {versions.map((v) => (
                <div key={v.key} className="group flex items-center gap-2 rounded-md border border-border/50 px-2.5 py-1.5">
                  <Badge variant="secondary" className="text-[9px]">{new Date(v.created).toLocaleString()}</Badge>
                  <span className="min-w-0 flex-1 truncate font-mono text-[10px] text-muted-foreground">
                    {v.key.split("/").pop()} · {(v.size / 1024).toFixed(1)}KB
                  </span>
                  {v.note && <span className="text-[10px] text-muted-foreground/70">{v.note}</span>}
                  <div className="hidden items-center gap-0.5 group-hover:flex">
                    <button title="下载" onClick={() => void (async () => {
                      setBusy("dl");
                      const path = await cloudApi.downloadVersion(v.key);
                      if (path) setToast({ kind: "success", text: `已下载：${path}` });
                      setBusy(null);
                    })()} className="rounded p-0.5 text-muted-foreground hover:text-primary">
                      <CloudDownload size={11} />
                    </button>
                    <button title="删除" onClick={() => void (async () => {
                      await cloudApi.deleteVersion(v.key);
                      await loadAll();
                    })()} className="rounded p-0.5 text-muted-foreground hover:text-destructive">
                      <Trash2 size={11} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* —— 隔离区 —— */}
        <div className="rounded-md border border-border bg-card p-4">
          <div className="mb-2 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            <AlertTriangle size={12} className="text-amber-500" />
            同步隔离区（{quarantine.length}）
          </div>
          {quarantine.length === 0 ? (
            <p className="py-2 text-[11px] text-muted-foreground">没有隔离记录</p>
          ) : (
            <div className="space-y-1">
              {quarantine.slice(0, 20).map((q) => (
                <div key={q.id} className="flex items-center gap-2 rounded-md border border-amber-500/20 bg-amber-500/5 px-2.5 py-1.5">
                  <Badge className="text-[9px]">{q.table}</Badge>
                  <span className="min-w-0 flex-1 truncate font-mono text-[10px] text-muted-foreground">{q.recordId}</span>
                  <span className="truncate text-[10px] text-amber-600/80">{q.reason}</span>
                  <div className="flex shrink-0 items-center gap-0.5">
                    <button title="重试" onClick={() => void (async () => {
                      await syncApi.retryQuarantine(q.id);
                      await loadAll();
                    })()} className="rounded p-0.5 text-muted-foreground hover:text-primary">
                      <RotateCcw size={11} />
                    </button>
                    <button title="丢弃" onClick={() => void (async () => {
                      await syncApi.discardQuarantine(q.id);
                      await loadAll();
                    })()} className="rounded p-0.5 text-muted-foreground hover:text-destructive">
                      <Trash2 size={11} />
                    </button>
                  </div>
                </div>
              ))}
              {quarantine.length > 0 && (
                <Button size="sm" variant="outline" className="mt-1 h-6 text-[10px]" onClick={() => void (async () => {
                  await syncApi.discardAllQuarantine();
                  await loadAll();
                })()}>
                  清空隔离区
                </Button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function StatusChip({ label, value, ok }: { label: string; value: string; ok?: boolean }) {
  return (
    <div className="rounded-md border border-border/60 bg-background px-2.5 py-1.5">
      <div className="text-[9px] uppercase tracking-wider text-muted-foreground/70">{label}</div>
      <div className={cn("mt-0.5 flex items-center gap-1 text-[13px] font-semibold", ok === false ? "text-amber-500" : ok === true ? "text-emerald-500" : "text-foreground")}>
        {ok === true && <CheckCircle2 size={11} />}
        {value}
      </div>
    </div>
  );
}
