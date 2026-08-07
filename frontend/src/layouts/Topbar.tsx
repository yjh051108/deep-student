// 顶部栏 —— 对照原版 Topbar
// ------------------------------------------------------------
// 设计要点：
// - 暗色背景 (--card) + 底部 border 分隔
// - 左侧：当前页面标题（由路由推导）+ 后退/前进按钮
// - 中间：全局搜索/命令面板触发器（Ctrl+K）
// - 右侧：Provider 切换 + A/B 槽位 + 用户头像

import { useLocation, useNavigate } from "react-router-dom";
import {
  Search,
  ArrowLeft,
  ArrowRight,
  ChevronDown,
  Bot,
} from "lucide-react";
import { useMemo, useState } from "react";
import { useSessionStore, PROVIDERS, SlotKey, ProviderKey } from "@/state/session";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/Button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/DropdownMenu";

// 路由 → 页面标题映射
const ROUTE_TITLES: Record<string, { title: string; subtitle: string }> = {
  "/hub": { title: "Hub", subtitle: "资源中枢" },
  "/chat": { title: "Chat", subtitle: "多模型对话" },
  "/mindmap": { title: "Mindmap", subtitle: "思维导图" },
  "/qbank": { title: "QBank", subtitle: "题库练习" },
  "/anki": { title: "Anki", subtitle: "卡片记忆" },
  "/reader": { title: "Reader", subtitle: "文档阅读" },
  "/translate": { title: "Translate", subtitle: "翻译工作台" },
  "/essay": { title: "Essay", subtitle: "作文批改" },
  "/research": { title: "Research", subtitle: "研究助手" },
  "/paper": { title: "Paper", subtitle: "论文检索" },
  "/memory": { title: "Memory", subtitle: "记忆中枢" },
  "/skills": { title: "Skills", subtitle: "技能与 MCP" },
  "/governance": { title: "Governance", subtitle: "数据治理" },
  "/settings": { title: "Settings", subtitle: "应用设置" },
};

export function Topbar() {
  const navigate = useNavigate();
  const location = useLocation();
  const provider = useSessionStore((s) => s.provider);
  const slot = useSessionStore((s) => s.slot);
  const user = useSessionStore((s) => s.user);
  const setProvider = useSessionStore((s) => s.setProvider);
  const setSlot = useSessionStore((s) => s.setSlot);
  const setPaletteOpen = useSessionStore((s) => s.setCommandPaletteOpen);
  const setUser = useSessionStore((s) => s.setUser);

  const [editingUser, setEditingUser] = useState(false);

  // 由路由推导当前页面标题
  const current = useMemo(() => {
    return ROUTE_TITLES[location.pathname] ?? { title: "DeepStudent", subtitle: "" };
  }, [location.pathname]);

  return (
    <header className="flex h-14 shrink-0 items-center justify-between border-b border-border bg-card px-4">
      {/* —— 左侧：页面标题 + 历史导航 —— */}
      <div className="flex min-w-0 items-center gap-2">
        <div className="flex items-center gap-0.5">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate(-1)}
            aria-label="后退"
            className="h-8 w-8"
          >
            <ArrowLeft size={14} />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate(1)}
            aria-label="前进"
            className="h-8 w-8"
          >
            <ArrowRight size={14} />
          </Button>
        </div>

        <div className="min-w-0">
          <div className="flex items-baseline gap-2">
            <h1 className="truncate text-sm font-semibold tracking-tight text-foreground">
              {current.title}
            </h1>
            {current.subtitle && (
              <span className="truncate text-xs text-muted-foreground">
                · {current.subtitle}
              </span>
            )}
          </div>
        </div>
      </div>

      {/* —— 中间：命令面板触发器 —— */}
      <button
        type="button"
        onClick={() => setPaletteOpen(true)}
        className="hidden md:flex w-80 items-center gap-2 rounded-md border border-input bg-transparent px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
      >
        <Search size={14} />
        <span className="flex-1 text-left">搜索 / 命令</span>
        <kbd className="rounded border border-border bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
          Ctrl+K
        </kbd>
      </button>

      {/* —— 右侧：Provider + 槽位 + 用户 —— */}
      <div className="flex items-center gap-3">
        {/* Provider 下拉 */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="gap-1.5">
              <Bot size={14} />
              <span className="hidden sm:inline">
                {PROVIDERS.find((p) => p.key === provider)?.label ?? provider}
              </span>
              <ChevronDown size={12} className="text-muted-foreground" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-48">
            <DropdownMenuLabel>LLM Provider</DropdownMenuLabel>
            <DropdownMenuSeparator />
            {PROVIDERS.map((p) => (
              <DropdownMenuItem
                key={p.key}
                onClick={() => setProvider(p.key as ProviderKey)}
                className={cn(
                  "justify-between",
                  provider === p.key && "bg-accent text-accent-foreground"
                )}
              >
                {p.label}
                {provider === p.key && <span className="text-primary">●</span>}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>

        {/* A/B 槽位切换 */}
        <div className="flex items-center gap-0.5 rounded-md border border-border bg-muted/30 p-0.5">
          {(["A", "B"] as SlotKey[]).map((s) => (
            <button
              key={s}
              type="button"
              onClick={() => setSlot(s)}
              className={cn(
                "rounded px-2 py-0.5 text-[11px] font-medium transition-colors",
                slot === s
                  ? "bg-primary text-primary-foreground shadow-soft"
                  : "text-muted-foreground hover:bg-accent hover:text-foreground"
              )}
              aria-label={`切换到槽 ${s}`}
            >
              槽 {s}
            </button>
          ))}
        </div>

        {/* 用户标识 */}
        <div className="flex items-center gap-2 border-l border-border pl-3">
          {editingUser ? (
            <input
              autoFocus
              value={user}
              onChange={(e) => setUser(e.target.value)}
              onBlur={() => setEditingUser(false)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === "Escape") {
                  setEditingUser(false);
                }
              }}
              className="w-24 rounded-md border border-input bg-transparent px-2 py-1 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          ) : (
            <button
              type="button"
              onClick={() => setEditingUser(true)}
              className="flex h-8 w-8 items-center justify-center rounded-full bg-gradient-to-br from-primary to-brand-primary-dark text-xs font-semibold text-primary-foreground transition-transform hover:scale-105"
              aria-label="切换用户名"
            >
              {user.slice(0, 1).toUpperCase()}
            </button>
          )}
        </div>
      </div>
    </header>
  );
}
