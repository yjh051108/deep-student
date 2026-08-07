// 现代侧边栏 —— 对照原版 ModernSidebar
// ------------------------------------------------------------
// 设计要点：
// - 暗色背景 (--card) + 精致 hover/active 状态
// - 分组导航：核心 / 工具 / 系统
// - 折叠模式：宽度从 240px 收缩到 64px，仅显示图标
// - 顶部品牌区 + 底部用户/版本区
// - 使用 lucide-react 图标（对照原版 phosphor-icons 视觉）

import { NavLink, useLocation } from "react-router-dom";
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
  PanelLeftClose,
  PanelLeftOpen,
  Sparkles,
  NotebookPen,
  Timer,
  Gauge,
  CheckSquare,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/Tooltip";
import { useSessionStore } from "@/state/session";

interface NavItem {
  to: string;
  label: string;
  icon: typeof LayoutGrid;
  /** 可选的角标（如未读数） */
  badge?: number;
}

// 分组定义：核心 / 工具 / 系统
const NAV_GROUPS: { title: string; items: NavItem[] }[] = [
  {
    title: "核心",
    items: [
      { to: "/hub", label: "Hub · 资源中枢", icon: LayoutGrid },
      { to: "/chat", label: "Chat · 多模型对话", icon: MessagesSquare },
      { to: "/mindmap", label: "Mindmap · 思维导图", icon: Network },
      { to: "/reader", label: "Reader · 文档阅读", icon: BookOpen },
      { to: "/notes", label: "Notes · 笔记", icon: NotebookPen },
    ],
  },
  {
    title: "学习工具",
    items: [
      { to: "/todo", label: "Todo · 待办", icon: CheckSquare },
      { to: "/pomodoro", label: "Pomodoro · 番茄钟", icon: Timer },
      { to: "/qbank", label: "QBank · 题库", icon: ListChecks },
      { to: "/anki", label: "Anki · 卡片", icon: CreditCard },
      { to: "/translate", label: "Translate · 翻译", icon: Languages },
      { to: "/essay", label: "Essay · 作文批改", icon: PenSquare },
      { to: "/memory", label: "Memory · 记忆", icon: Brain },
    ],
  },
  {
    title: "研究",
    items: [
      { to: "/research", label: "Research · 研究", icon: Search },
      { to: "/paper", label: "Paper · 论文", icon: FileText },
    ],
  },
  {
    title: "系统",
    items: [
      { to: "/llm-usage", label: "LLM Usage · 用量", icon: Gauge },
      { to: "/skills", label: "Skills · 技能", icon: Wrench },
      { to: "/governance", label: "Governance · 治理", icon: ShieldCheck },
      { to: "/settings", label: "Settings · 设置", icon: SettingsIcon },
    ],
  },
];

interface SidebarProps {
  collapsed: boolean;
  onToggleCollapsed: () => void;
}

export function Sidebar({ collapsed, onToggleCollapsed }: SidebarProps) {
  const user = useSessionStore((s) => s.user);
  const version = useSessionStore((s) => s.version);
  const location = useLocation();

  return (
    <aside
      className={cn(
        "flex shrink-0 flex-col border-r border-border bg-card text-foreground transition-[width] duration-200 ease-out",
        collapsed ? "w-16" : "w-60"
      )}
      data-collapsed={collapsed}
    >
      {/* —— 品牌区 —— */}
      <div
        className={cn(
          "flex h-14 shrink-0 items-center gap-2 border-b border-border px-3",
          collapsed && "justify-center px-0"
        )}
      >
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-primary to-brand-primary-dark text-primary-foreground shadow-soft">
          <Sparkles size={16} />
        </div>
        {!collapsed && (
          <div className="min-w-0 flex-1">
            <div className="truncate text-sm font-semibold tracking-tight">
              DeepStudent
            </div>
            <div className="truncate text-[10px] text-muted-foreground">
              本地优先 · AI 学习工作台
            </div>
          </div>
        )}
      </div>

      {/* —— 导航区 —— */}
      <nav className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto py-3 scrollbar-dark">
        {NAV_GROUPS.map((group) => (
          <div key={group.title} className="px-2">
            {!collapsed && (
              <div className="mb-1 px-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                {group.title}
              </div>
            )}
            <div className="space-y-0.5">
              {group.items.map((item) => {
                const isActive = location.pathname === item.to;
                const linkInner = (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    className={cn(
                      "group relative flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm transition-colors",
                      collapsed && "justify-center px-0",
                      isActive
                        ? "bg-primary/12 text-primary font-medium"
                        : "text-muted-foreground hover:bg-accent hover:text-foreground"
                    )}
                  >
                    {/* 激活态左侧指示条 */}
                    {isActive && (
                      <span className="absolute left-0 top-1/2 h-4 w-0.5 -translate-y-1/2 rounded-r-full bg-primary" />
                    )}
                    <item.icon
                      size={16}
                      className={cn(
                        "shrink-0 transition-colors",
                        isActive
                          ? "text-primary"
                          : "text-muted-foreground group-hover:text-foreground"
                      )}
                    />
                    {!collapsed && (
                      <span className="truncate">{item.label}</span>
                    )}
                    {!collapsed && item.badge !== undefined && item.badge > 0 && (
                      <span className="ml-auto rounded-full bg-primary/15 px-1.5 py-0.5 text-[10px] font-medium text-primary">
                        {item.badge}
                      </span>
                    )}
                  </NavLink>
                );
                // 折叠态：包装 Tooltip 显示完整标签
                if (collapsed) {
                  return (
                    <Tooltip key={item.to}>
                      <TooltipTrigger asChild>
                        <div>{linkInner}</div>
                      </TooltipTrigger>
                      <TooltipContent side="right">{item.label}</TooltipContent>
                    </Tooltip>
                  );
                }
                return linkInner;
              })}
            </div>
          </div>
        ))}
      </nav>

      {/* —— 底部用户/版本区 —— */}
      <div className="shrink-0 border-t border-border p-2">
        {!collapsed ? (
          <div className="flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-accent transition-colors">
            <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-primary to-brand-primary-dark text-xs font-semibold text-primary-foreground">
              {user.slice(0, 1).toUpperCase()}
            </div>
            <div className="min-w-0 flex-1">
              <div className="truncate text-xs font-medium text-foreground">
                {user}
              </div>
              <div className="truncate text-[10px] text-muted-foreground">
                {version}
              </div>
            </div>
          </div>
        ) : (
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="flex justify-center py-1.5">
                <div className="flex h-7 w-7 items-center justify-center rounded-full bg-gradient-to-br from-primary to-brand-primary-dark text-xs font-semibold text-primary-foreground">
                  {user.slice(0, 1).toUpperCase()}
                </div>
              </div>
            </TooltipTrigger>
            <TooltipContent side="right">
              {user} · {version}
            </TooltipContent>
          </Tooltip>
        )}
      </div>

      {/* —— 折叠按钮 —— */}
      <button
        type="button"
        onClick={onToggleCollapsed}
        className={cn(
          "flex h-9 shrink-0 items-center gap-2 border-t border-border px-3 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-foreground",
          collapsed && "justify-center px-0"
        )}
        aria-label={collapsed ? "展开侧边栏" : "折叠侧边栏"}
      >
        {collapsed ? (
          <PanelLeftOpen size={14} />
        ) : (
          <>
            <PanelLeftClose size={14} />
            <span>折叠</span>
          </>
        )}
      </button>
    </aside>
  );
}
