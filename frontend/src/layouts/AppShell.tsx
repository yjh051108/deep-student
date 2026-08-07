// 应用外壳 —— 对照原版 AppShell
// ------------------------------------------------------------
// 设计要点：
// - 暗色主题（--background / --foreground）
// - 三段式布局：Sidebar（左）+ 主区（右上 Topbar + 中间 Outlet + 右下 Statusbar）
// - 命令面板（Ctrl+K）浮层
// - TooltipProvider 包裹全应用
// - 拖拽窗口区（顶栏上方 8px）

import { Outlet, useNavigate } from "react-router-dom";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";
import { Statusbar } from "./Statusbar";
import { useHotkeys } from "@/hooks/useHotkeys";
import { useSessionStore } from "@/state/session";
import { TooltipProvider } from "@/components/ui/Tooltip";
import { useEffect, useMemo, useRef, useState } from "react";
import {
  LayoutGrid,
  MessagesSquare,
  Network,
  ListChecks,
  CreditCard,
  BookOpen,
  Languages,
  PenSquare,
  Search,
  FileText,
  Brain,
  Wrench,
  ShieldCheck,
  Settings as SettingsIcon,
} from "lucide-react";
import { cn } from "@/lib/utils";

// 命令面板项定义
interface PaletteCommand {
  id: string;
  label: string;
  hint?: string;
  icon: typeof LayoutGrid;
  action: () => void;
}

const ROUTE_COMMANDS: { id: string; label: string; to: string; icon: typeof LayoutGrid }[] = [
  { id: "hub", label: "前往 Hub · 资源中枢", to: "/hub", icon: LayoutGrid },
  { id: "chat", label: "前往 Chat · 多模型对话", to: "/chat", icon: MessagesSquare },
  { id: "mindmap", label: "前往 Mindmap · 思维导图", to: "/mindmap", icon: Network },
  { id: "qbank", label: "前往 QBank · 题库", to: "/qbank", icon: ListChecks },
  { id: "anki", label: "前往 Anki · 卡片", to: "/anki", icon: CreditCard },
  { id: "reader", label: "前往 Reader · 文档阅读", to: "/reader", icon: BookOpen },
  { id: "translate", label: "前往 Translate · 翻译", to: "/translate", icon: Languages },
  { id: "essay", label: "前往 Essay · 作文批改", to: "/essay", icon: PenSquare },
  { id: "research", label: "前往 Research · 研究", to: "/research", icon: Search },
  { id: "paper", label: "前往 Paper · 论文", to: "/paper", icon: FileText },
  { id: "memory", label: "前往 Memory · 记忆", to: "/memory", icon: Brain },
  { id: "skills", label: "前往 Skills · 技能", to: "/skills", icon: Wrench },
  { id: "governance", label: "前往 Governance · 治理", to: "/governance", icon: ShieldCheck },
  { id: "settings", label: "前往 Settings · 设置", to: "/settings", icon: SettingsIcon },
];

export function AppShell() {
  const navigate = useNavigate();
  const setPaletteOpen = useSessionStore((s) => s.setCommandPaletteOpen);
  const paletteOpen = useSessionStore((s) => s.commandPaletteOpen);
  const sidebarCollapsed = useSessionStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useSessionStore((s) => s.toggleSidebar);

  useHotkeys(
    [
      {
        combo: "Ctrl+N",
        description: "新建 chat tab",
        handler: () => navigate("/chat?new=1"),
      },
      {
        combo: "Ctrl+K",
        description: "打开 command palette",
        handler: () => setPaletteOpen(true),
      },
      {
        combo: "Ctrl+B",
        description: "折叠/展开侧边栏",
        handler: () => toggleSidebar(),
      },
      {
        combo: "Ctrl+,",
        description: "打开设置",
        handler: () => navigate("/settings"),
      },
    ],
    true
  );

  return (
    <TooltipProvider delayDuration={300}>
      <div className="flex h-full w-full flex-col bg-background text-foreground">
        <div className="flex min-h-0 flex-1">
          <Sidebar
            collapsed={sidebarCollapsed}
            onToggleCollapsed={toggleSidebar}
          />
          <div className="flex min-w-0 flex-1 flex-col">
            <Topbar />
            <main className="min-h-0 flex-1 overflow-auto bg-background scrollbar-dark">
              <Outlet />
            </main>
            <Statusbar />
          </div>
        </div>
        {paletteOpen && (
          <CommandPalette onClose={() => setPaletteOpen(false)} />
        )}
      </div>
    </TooltipProvider>
  );
}

// —— 命令面板 ——
function CommandPalette({ onClose }: { onClose: () => void }) {
  const navigate = useNavigate();
  const [query, setQuery] = useState("");
  const [activeIdx, setActiveIdx] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  // 过滤命令
  const commands = useMemo<PaletteCommand[]>(() => {
    const q = query.trim().toLowerCase();
    const all: PaletteCommand[] = ROUTE_COMMANDS.map((c) => ({
      id: c.id,
      label: c.label,
      icon: c.icon,
      action: () => {
        navigate(c.to);
        onClose();
      },
    }));
    if (!q) return all;
    return all.filter((c) => c.label.toLowerCase().includes(q));
  }, [query, navigate, onClose]);

  // 索引重置
  useEffect(() => {
    setActiveIdx(0);
  }, [query]);

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      e.preventDefault();
      onClose();
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      setActiveIdx((i) => Math.min(i + 1, commands.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActiveIdx((i) => Math.max(i - 1, 0));
    } else if (e.key === "Enter") {
      e.preventDefault();
      const cmd = commands[activeIdx];
      if (cmd) cmd.action();
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-background/80 pt-32 backdrop-blur-sm animate-fade-in"
      onClick={onClose}
    >
      <div
        className="w-[560px] max-h-[60vh] flex flex-col overflow-hidden rounded-dialog border border-border bg-popover shadow-floating animate-zoom-in"
        onClick={(e) => e.stopPropagation()}
      >
        {/* 搜索输入 */}
        <div className="flex items-center gap-2 border-b border-border px-4 py-3">
          <Search size={16} className="text-muted-foreground" />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={onKeyDown}
            placeholder="搜索命令或页面…"
            className="flex-1 bg-transparent text-sm text-foreground placeholder:text-muted-foreground focus:outline-none"
          />
          <kbd className="rounded border border-border bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
            ESC
          </kbd>
        </div>

        {/* 命令列表 */}
        <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
          {commands.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm text-muted-foreground">
              没有匹配的命令
            </div>
          ) : (
            <ul className="py-1">
              {commands.map((cmd, idx) => {
                const Icon = cmd.icon;
                const isActive = idx === activeIdx;
                return (
                  <li key={cmd.id}>
                    <button
                      type="button"
                      onMouseMove={() => setActiveIdx(idx)}
                      onClick={cmd.action}
                      className={cn(
                        "flex w-full items-center gap-3 px-4 py-2 text-left text-sm transition-colors",
                        isActive
                          ? "bg-accent text-foreground"
                          : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
                      )}
                    >
                      <Icon size={14} className="shrink-0" />
                      <span className="flex-1 truncate">{cmd.label}</span>
                      {isActive && (
                        <kbd className="text-[10px] text-muted-foreground">↵</kbd>
                      )}
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
        </div>

        {/* 底部提示 */}
        <div className="border-t border-border px-4 py-2 text-[10px] text-muted-foreground">
          <span className="flex items-center gap-3">
            <span><kbd className="font-mono">↑↓</kbd> 导航</span>
            <span><kbd className="font-mono">↵</kbd> 选择</span>
            <span><kbd className="font-mono">ESC</kbd> 关闭</span>
          </span>
        </div>
      </div>
    </div>
  );
}
