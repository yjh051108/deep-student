// Sidebar —— 1:1 对齐原版 ModernSidebar
// ------------------------------------------------------------
// - 宽度 272px（可折叠为 0，由顶栏悬浮钮控制）
// - 主导航 7 项（新会话/学习资源/待办/技能管理/制卡任务/模板管理/设置）
// - 会话滚动区（置顶/最近，对齐原版 pinned/conversations 分组）
// - 浅灰激活态（interactive-selected），非品牌色填充
// - Phosphor 图标（与原版同库）

import { NavLink, useLocation, useNavigate } from "react-router-dom";
import {
  ChatCenteredText,
  Books,
  ClipboardText,
  MagicWand,
  Stack,
  SquaresFour,
  Gear,
  PushPin,
  DotsThree,
  Plus,
  CaretRight,
} from "@phosphor-icons/react";
import { cn } from "@/lib/utils";
import { useEffect, useState } from "react";
import { useSessionStore } from "@/state/session";

interface NavItem {
  to: string;
  label: string;
  icon: typeof ChatCenteredText;
  /** 新建会话等快捷动作 */
  action?: () => void;
  /** mac 快捷键提示 */
  kbd?: string;
}

const NAV_ITEMS: NavItem[] = [
  { to: "/chat", label: "新会话", icon: ChatCenteredText, kbd: "⌘N" },
  { to: "/hub", label: "学习资源", icon: Books },
  { to: "/todo", label: "待办", icon: ClipboardText },
  { to: "/skills", label: "技能管理", icon: MagicWand },
  { to: "/anki", label: "制卡任务", icon: Stack },
  { to: "/template-manager", label: "模板管理", icon: SquaresFour },
];

export function Sidebar({
  collapsed,
  onToggleCollapsed,
}: {
  collapsed: boolean;
  onToggleCollapsed: () => void;
}) {
  const location = useLocation();
  const navigate = useNavigate();
  const [pinnedOpen, setPinnedOpen] = useState(true);
  const [recentOpen, setRecentOpen] = useState(true);
  const user = useSessionStore((s) => s.user);
  const version = useSessionStore((s) => s.version);

  useEffect(() => {
    if (location.pathname !== "/chat") return;
    // 到达 /chat 时若带 ?new=1 触发新会话（由 ChatPage 处理）
  }, [location]);

  if (collapsed) {
    return (
      <aside className="shrink-0 overflow-hidden" style={{ width: 0 }}>
        <div className="w-[272px] border-r border-[var(--shell-seam)] bg-[var(--shell-navigation-surface)]" />
      </aside>
    );
  }

  return (
    <aside
      className="flex shrink-0 flex-col border-r border-[var(--shell-seam)] bg-[var(--shell-navigation-surface)]"
      style={{ width: 272 }}
    >
      {/* —— 主导航区 —— */}
      <nav className="shrink-0 space-y-0.5 p-2">
        {NAV_ITEMS.map((item) => {
          const isActive = location.pathname === item.to;
          return (
            <NavLink
              key={item.to}
              to={item.to}
              className={cn("sidebar-row flex items-center gap-2.5 px-2.5")}
              data-active={isActive}
            >
              <item.icon size={18} weight={isActive ? "fill" : "regular"} className="shrink-0 opacity-80" />
              <span className="min-w-0 flex-1 truncate text-[13px]">{item.label}</span>
              {item.kbd && (
                <kbd className="rounded border border-[var(--border-default)] bg-transparent px-1 text-[9px] text-muted-foreground/60">
                  {item.kbd}
                </kbd>
              )}
            </NavLink>
          );
        })}
      </nav>

      {/* —— 会话区 —— */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark px-2 pb-2">
        {/* 置顶 */}
        <SectionHeader
          title="置顶"
          open={pinnedOpen}
          onToggle={() => setPinnedOpen((v) => !v)}
          onNew={() => navigate("/chat?new=1")}
        />
        {pinnedOpen && (
          <div className="mb-1 space-y-0.5">
            <SidebarEmptyRow text="没有置顶会话" />
          </div>
        )}

        {/* 最近 */}
        <SectionHeader
          title="最近"
          open={recentOpen}
          onToggle={() => setRecentOpen((v) => !v)}
          onNew={() => navigate("/chat?new=1")}
        />
        {recentOpen && (
          <div className="space-y-0.5">
            <SidebarEmptyRow text="还没有会话，点击「新会话」开始" />
          </div>
        )}
      </div>

      {/* —— 底部：设置 —— */}
      <div className="shrink-0 border-t border-[var(--shell-seam)] p-2">
        <NavLink
          to="/settings"
          className={cn("sidebar-row flex items-center gap-2.5 px-2.5")}
          data-active={location.pathname === "/settings"}
        >
          <Gear size={18} className="shrink-0 opacity-80" />
          <span className="min-w-0 flex-1 truncate text-[13px]">设置</span>
          <span className="text-[9px] text-muted-foreground/50">{version}</span>
        </NavLink>
        <div className="mt-1 flex items-center gap-1.5 px-2.5 text-[10px] text-muted-foreground/50">
          <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
          {user}
        </div>
      </div>

      {/* 折叠按钮由顶栏悬浮控制（保留 prop 引用） */}
      <span className="hidden">{void onToggleCollapsed}</span>
    </aside>
  );
}

// —— 分组标题（对齐原版 renderSidebarSectionHeader）——
function SectionHeader({
  title,
  open,
  onToggle,
  onNew,
}: {
  title: string;
  open: boolean;
  onToggle: () => void;
  onNew: () => void;
}) {
  return (
    <div className="group flex items-center px-1 pt-2">
      <button
        onClick={onToggle}
        className="flex flex-1 items-center gap-0.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70 hover:text-foreground"
      >
        <CaretRight size={10} className={cn("transition-transform", !open && "rotate-90")} weight="bold" />
        {title}
      </button>
      <button
        onClick={onNew}
        className="hidden rounded p-0.5 text-muted-foreground/60 hover:text-foreground group-hover:block"
        title={`新建${title}`}
      >
        <Plus size={11} weight="bold" />
      </button>
    </div>
  );
}

function SidebarEmptyRow({ text }: { text: string }) {
  return (
    <div className="flex items-center gap-1.5 px-2.5 py-1.5 text-[11px] text-muted-foreground/50">
      <DotsThree size={12} className="opacity-40" />
      {text}
    </div>
  );
}

// 图标引用保留
void PushPin;
