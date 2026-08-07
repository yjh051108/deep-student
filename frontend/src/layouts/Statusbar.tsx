// 状态栏 —— 对照原版底部状态栏
// ------------------------------------------------------------
// 设计要点：
// - 极小高度（28px）+ 顶部 border 分隔
// - 左侧：数据目录 + 版本 + 当前页面
// - 右侧：Wails 连接状态 + 当前时间

import { useSessionStore } from "@/state/session";
import { wailsAvailable } from "@/lib/wails";
import { useLocation } from "react-router-dom";
import { useEffect, useState } from "react";
import { cn } from "@/lib/utils";

export function Statusbar() {
  const dataDir = useSessionStore((s) => s.dataDir);
  const version = useSessionStore((s) => s.version);
  const location = useLocation();
  const [available, setAvailable] = useState(false);
  const [now, setNow] = useState(() => new Date());

  useEffect(() => {
    setAvailable(wailsAvailable());
    const t = window.setInterval(() => setAvailable(wailsAvailable()), 1000);
    return () => window.clearInterval(t);
  }, []);

  useEffect(() => {
    const t = window.setInterval(() => setNow(new Date()), 1000);
    return () => window.clearInterval(t);
  }, []);

  const timeStr = `${String(now.getHours()).padStart(2, "0")}:${String(
    now.getMinutes()
  ).padStart(2, "0")}:${String(now.getSeconds()).padStart(2, "0")}`;

  return (
    <footer className="flex h-7 shrink-0 items-center justify-between border-t border-border bg-card px-3 text-[11px] text-muted-foreground">
      <div className="flex items-center gap-2.5">
        <span className="flex items-center gap-1.5">
          <span
            className={cn(
              "inline-block h-1.5 w-1.5 rounded-full transition-colors",
              available ? "bg-success" : "bg-muted-foreground/40"
            )}
          />
          <span>{available ? "Wails 已连接" : "Wails 未连接"}</span>
        </span>
        <span className="text-muted-foreground/40">·</span>
        <span>路由 {location.pathname}</span>
        <span className="text-muted-foreground/40">·</span>
        <span>数据 {dataDir}</span>
      </div>
      <div className="flex items-center gap-2.5">
        <span>v{version}</span>
        <span className="text-muted-foreground/40">·</span>
        <span className="font-mono tabular-nums">{timeStr}</span>
      </div>
    </footer>
  );
}
