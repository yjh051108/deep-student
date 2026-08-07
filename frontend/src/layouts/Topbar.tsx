// Topbar —— 1:1 对齐原版 desktop-shell-titlebar（40px 固定顶栏）
// ------------------------------------------------------------
// - 左侧：侧栏折叠钮（CaretLeft/CaretRight）+ 后退/前进
// - 中间：视图标题（按路由映射）
// - 右侧：命令面板钮（Terminal + ⌘K）+ Windows 窗口控制

import { useLocation, useNavigate } from "react-router-dom";
import {
  CaretLeft,
  CaretRight,
  Terminal,
  Minus,
  Square,
  X,
} from "@phosphor-icons/react";
import { useSessionStore } from "@/state/session";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/Tooltip";

const VIEW_TITLES: Record<string, string> = {
  "/chat": "新会话",
  "/hub": "学习资源",
  "/notes": "笔记",
  "/mindmap": "思维导图",
  "/todo": "待办",
  "/pomodoro": "番茄钟",
  "/qbank": "题库",
  "/anki": "制卡任务",
  "/fsrs": "间隔复习",
  "/reader": "阅读器",
  "/ocr": "OCR 识别",
  "/translate": "翻译",
  "/essay": "作文批改",
  "/memory": "记忆库",
  "/research": "深度调研",
  "/paper": "论文检索",
  "/skills": "技能管理",
  "/template-manager": "模板管理",
  "/sync": "云同步",
  "/sandbox": "代码沙盒",
  "/llm-usage": "LLM 用量",
  "/governance": "数据治理",
  "/settings": "设置",
};

export function Topbar() {
  const location = useLocation();
  const navigate = useNavigate();
  const sidebarCollapsed = useSessionStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useSessionStore((s) => s.toggleSidebar);
  const setPaletteOpen = useSessionStore((s) => s.setCommandPaletteOpen);

  const title = VIEW_TITLES[location.pathname] ?? "DeepStudent";

  return (
    <header
      className="flex shrink-0 items-center border-b border-[var(--shell-seam)] bg-[var(--titlebar-background)] px-3"
      style={{ height: 40 }}
    >
      {/* —— 左侧：折叠 + 导航 —— */}
      <div className="flex items-center gap-1">
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              onClick={toggleSidebar}
              className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground hover:bg-[var(--interactive-hover)] hover:text-foreground"
            >
              {sidebarCollapsed ? <CaretRight size={16} /> : <CaretLeft size={16} />}
            </button>
          </TooltipTrigger>
          <TooltipContent side="bottom">{sidebarCollapsed ? "展开侧栏" : "折叠侧栏"}</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              onClick={() => navigate(-1)}
              className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground hover:bg-[var(--interactive-hover)] hover:text-foreground"
            >
              <CaretLeft size={14} />
            </button>
          </TooltipTrigger>
          <TooltipContent side="bottom">后退</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              onClick={() => navigate(1)}
              className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground hover:bg-[var(--interactive-hover)] hover:text-foreground"
            >
              <CaretRight size={14} />
            </button>
          </TooltipTrigger>
          <TooltipContent side="bottom">前进</TooltipContent>
        </Tooltip>
      </div>

      {/* —— 中间：视图标题 —— */}
      <div className="flex min-w-0 flex-1 items-center justify-center px-2">
        <span className="truncate text-[13px] font-medium text-foreground/90">{title}</span>
      </div>

      {/* —— 右侧：命令面板 + 窗口控制 —— */}
      <div className="flex items-center gap-1">
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              onClick={() => setPaletteOpen(true)}
              className="flex h-7 items-center gap-1.5 rounded-md px-2 text-muted-foreground hover:bg-[var(--interactive-hover)] hover:text-foreground"
            >
              <Terminal size={14} />
              <kbd className="rounded border border-[var(--border-default)] px-1 text-[9px]">⌘K</kbd>
            </button>
          </TooltipTrigger>
          <TooltipContent side="bottom">命令面板</TooltipContent>
        </Tooltip>
        <WindowControls />
      </div>
    </header>
  );
}

// —— Windows 窗口控制 ——
function WindowControls() {
  const isWindows = navigator.userAgent.includes("Windows");
  if (!isWindows) return null;

  const send = (action: string) => {
    try {
      const w = window as unknown as Record<string, unknown>;
      const wails = w.wails as Record<string, unknown> | undefined;
      const runtime = wails?.Window as Record<string, (...a: unknown[]) => unknown> | undefined;
      if (action === "minimize") runtime?.Minimise?.();
      else if (action === "maximize") runtime?.Maximise?.();
      else if (action === "close") runtime?.Close?.();
    } catch {
      // 忽略（浏览器预览时无窗口控制）
    }
  };

  return (
    <div className="ml-1 flex items-center gap-0.5">
      <button
        onClick={() => send("minimize")}
        className="flex h-6 w-8 items-center justify-center rounded text-muted-foreground/70 hover:bg-[var(--interactive-hover)] hover:text-foreground"
      >
        <Minus size={12} />
      </button>
      <button
        onClick={() => send("maximize")}
        className="flex h-6 w-8 items-center justify-center rounded text-muted-foreground/70 hover:bg-[var(--interactive-hover)] hover:text-foreground"
      >
        <Square size={10} />
      </button>
      <button
        onClick={() => send("close")}
        className="flex h-6 w-8 items-center justify-center rounded text-muted-foreground/70 hover:bg-destructive hover:text-white"
      >
        <X size={13} />
      </button>
    </div>
  );
}
