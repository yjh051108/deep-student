// WorkbenchGrid —— 学习工作台视图（多应用网格）
// ------------------------------------------------------------
// 对照原版 workbench 概念：把各功能模块呈现为应用卡片，
// 点击跳转对应页面。展示各模块的实时摘要。

import { useNavigate } from "react-router-dom";
import {
  MessagesSquare,
  Network,
  NotebookPen,
  CheckSquare,
  Timer,
  ListChecks,
  CreditCard,
  BookOpen,
  Languages,
  PenSquare,
  Brain,
  Search,
  FileText,
  Layers,
  ScanLine,
  LayoutTemplate,
  RefreshCw,
  Code2,
} from "lucide-react";
import { useEffect, useState } from "react";
import { callWails } from "@/lib/wails";
import { cn } from "@/lib/utils";

interface AppCard {
  to: string;
  title: string;
  desc: string;
  icon: typeof MessagesSquare;
  /** 摘要查询方法 */
  badge?: () => Promise<string | null>;
}

const APPS: AppCard[] = [
  { to: "/chat", title: "智能对话", desc: "多模型聊天 · 工具循环", icon: MessagesSquare },
  { to: "/notes", title: "笔记", desc: "Obsidian 式 Markdown", icon: NotebookPen, badge: async () => {
    const r = await callWails<{ total: number }>("NotesStats");
    return r ? `${r.total} 篇` : null;
  } },
  { to: "/todo", title: "待办", desc: "任务管理 · AI 拆解", icon: CheckSquare },
  { to: "/pomodoro", title: "番茄钟", desc: "专注计时 · 统计", icon: Timer },
  { to: "/mindmap", title: "思维导图", desc: "AI 生成 · 大纲互转", icon: Network },
  { to: "/qbank", title: "题库", desc: "抽题 · 阅卷 · 掌握度", icon: ListChecks },
  { to: "/anki", title: "Anki 制卡", desc: "批量制卡 · 模板", icon: CreditCard },
  { to: "/fsrs", title: "间隔复习", desc: "FSRS 调度 · 复习队列", icon: Layers, badge: async () => {
    const r = await callWails<number>("FSRSDueCount");
    return r != null && r > 0 ? `${r} 待复习` : null;
  } },
  { to: "/reader", title: "阅读器", desc: "PDF / DOCX / MD", icon: BookOpen },
  { to: "/ocr", title: "OCR 识别", desc: "图片 / PDF 文字提取", icon: ScanLine },
  { to: "/translate", title: "翻译", desc: "全文 · 术语表", icon: Languages },
  { to: "/essay", title: "作文批改", desc: "维度评分 · 润色", icon: PenSquare },
  { to: "/memory", title: "记忆库", desc: "事实抽取 · 文件夹", icon: Brain },
  { to: "/research", title: "深度调研", desc: "计划 · 执行 · 报告", icon: Search },
  { to: "/paper", title: "论文检索", desc: "arXiv · 引用", icon: FileText },
  { to: "/template-manager", title: "模板管理", desc: "Anki 模板编辑器", icon: LayoutTemplate },
  { to: "/sync", title: "云同步", desc: "增量同步 · 隔离区", icon: RefreshCw },
  { to: "/sandbox", title: "代码沙盒", desc: "HTML/CSS/JS 预览", icon: Code2 },
];

export function WorkbenchGrid() {
  const navigate = useNavigate();
  const [badges, setBadges] = useState<Record<string, string | null>>({});

  useEffect(() => {
    const load = async () => {
      const entries: Record<string, string | null> = {};
      await Promise.all(
        APPS.map(async (app) => {
          if (app.badge) {
            try {
              entries[app.to] = (await app.badge()) ?? null;
            } catch {
              entries[app.to] = null;
            }
          }
        })
      );
      setBadges(entries);
    };
    void load();
  }, []);

  return (
    <div className="h-full w-full overflow-y-auto scrollbar-dark p-5">
      <div className="mx-auto max-w-6xl">
        <div className="mb-4">
          <h1 className="text-base font-semibold text-foreground">学习工作台</h1>
          <p className="text-[11px] text-muted-foreground">
            全部学习能力入口 · 点击进入对应模块
          </p>
        </div>
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
          {APPS.map((app) => {
            const Icon = app.icon;
            const badge = badges[app.to];
            return (
              <button
                key={app.to}
                onClick={() => navigate(app.to)}
                className="group flex flex-col gap-2.5 rounded-lg border border-border bg-card p-4 text-left transition-all hover:border-primary/40 hover:shadow-soft"
              >
                <div className="flex items-start justify-between">
                  <div className="flex h-9 w-9 items-center justify-center rounded-md bg-primary/12 text-primary transition-colors group-hover:bg-primary group-hover:text-primary-foreground">
                    <Icon size={17} />
                  </div>
                  {badge && (
                    <span className="rounded-full bg-primary/15 px-1.5 py-0.5 text-[9px] font-medium text-primary">
                      {badge}
                    </span>
                  )}
                </div>
                <div>
                  <div className="text-[13px] font-medium text-foreground">{app.title}</div>
                  <div className="mt-0.5 text-[10px] leading-relaxed text-muted-foreground">{app.desc}</div>
                </div>
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}

// 图标引用保留（避免 tree-shake 告警）
void cn;
