// QBankPage —— 1:1 对齐原版 PracticeLauncher（练习模式选择入口）
// ------------------------------------------------------------
// - 顶部：快速统计摘要
// - 模式卡片网格：顺序 / 随机 / 错题优先 / 按标签（基础模式）
//   高级模式：限时 / 模拟考 / 每日 / 组卷
// - 有题集时进入做题界面（复用 PracticePanel 三栏）

import { useEffect, useState } from "react";
import { useQBankStore } from "@/state/qbank";
import { cn } from "@/lib/utils";
import {
  ListNumbers,
  Shuffle,
  WarningCircle,
  Tag,
  Clock,
  Exam,
  Target,
  FileText,
  MagnifyingGlass,
  Plus,
} from "@phosphor-icons/react";
import { SetList } from "@/components/qbank/SetList";
import { PracticePanel } from "@/components/qbank/PracticePanel";
import { ProgressPanel } from "@/components/qbank/ProgressPanel";
import { ImportDialog } from "@/components/hub/ImportDialog";

interface ModeCard {
  key: string;
  title: string;
  desc: string;
  icon: typeof ListNumbers;
  advanced?: boolean;
  color: string;
}

const MODES: ModeCard[] = [
  { key: "sequential", title: "顺序练习", desc: "按题目顺序逐题作答", icon: ListNumbers, color: "text-slate-500 bg-slate-500/10" },
  { key: "random", title: "随机练习", desc: "随机打乱题目顺序", icon: Shuffle, color: "text-sky-600 bg-sky-500/10" },
  { key: "mistakes", title: "错题优先", desc: "优先复习做错的题目", icon: WarningCircle, color: "text-rose-500 bg-rose-500/10" },
  { key: "tagged", title: "按标签练习", desc: "按知识点标签筛选", icon: Tag, color: "text-violet-500 bg-violet-500/10" },
  { key: "timed", title: "限时挑战", desc: "设定时长倒计时答题", icon: Clock, color: "text-amber-500 bg-amber-500/10", advanced: true },
  { key: "mock", title: "模拟考", desc: "完整试卷模式 + 成绩单", icon: Exam, color: "text-emerald-600 bg-emerald-500/10", advanced: true },
  { key: "daily", title: "每日目标", desc: "每天完成目标题数", icon: Target, color: "text-blue-600 bg-blue-500/10", advanced: true },
  { key: "generate", title: "生成试卷", desc: "按范围组卷导出", icon: FileText, color: "text-fuchsia-600 bg-fuchsia-500/10", advanced: true },
];

export function QBankPage() {
  const loadMastery = useQBankStore((s) => s.loadMastery);
  const sets = useQBankStore((s) => s.sets);
  const activeSetId = useQBankStore((s) => s.activeSetId);
  const selectSet = useQBankStore((s) => s.selectSet);
  const extract = useQBankStore((s) => s.extract);
  const [mode, setMode] = useState<string | null>(null);
  const [importOpen, setImportOpen] = useState(false);
  const [extractUri, setExtractUri] = useState("");
  const [extractTitle, setExtractTitle] = useState("");
  const [extracting, setExtracting] = useState(false);

  useEffect(() => {
    void loadMastery();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 进入做题模式
  const startMode = (m: string) => {
    // 无题集时提示先导入/抽题
    if (sets.length === 0) {
      setExtractUri("");
      setExtractTitle("");
      setImportOpen(true);
      return;
    }
    setMode(m);
  };

  // 做题界面（有激活题集时）
  if (mode && activeSetId) {
    return (
      <div className="flex h-full w-full min-h-0 bg-background">
        {/* 返回模式选择 */}
        <div className="absolute left-2 top-2 z-10">
          <button
            onClick={() => setMode(null)}
            className="rounded-md bg-[var(--shell-inspector-panel)] px-2.5 py-1 text-[11px] text-muted-foreground shadow-[var(--shadow-shell-soft)] hover:text-foreground"
          >
            ← 返回模式选择
          </button>
        </div>
        <aside className="w-64 shrink-0 border-r border-[var(--shell-seam)] bg-[var(--shell-navigation-surface)]">
          <SetList />
        </aside>
        <section className="flex min-w-0 flex-1 flex-col">
          <PracticePanel />
        </section>
        <aside className="w-80 shrink-0 border-l border-[var(--shell-seam)] bg-background">
          <ProgressPanel />
        </aside>
      </div>
    );
  }

  return (
    <div className="flex h-full w-full min-h-0 flex-col bg-background">
      {/* —— 顶部：统计摘要 + 抽题入口 —— */}
      <div className="shrink-0 border-b border-[var(--shell-seam)] bg-[var(--shell-workspace-panel)] px-5 py-3">
        <div className="flex items-center gap-4">
          <h1 className="text-[14px] font-semibold text-foreground">题库与练习</h1>
          <span className="text-[11px] text-muted-foreground">
            {sets.length} 个题集 · 开始练习吧
          </span>
          <div className="ml-auto flex items-center gap-2">
            <button
              onClick={() => setImportOpen(true)}
              className="flex h-7 items-center gap-1.5 rounded-md bg-primary/15 px-3 text-[11px] font-medium text-primary hover:bg-primary/25"
            >
              <Plus size={12} weight="bold" />
              导入资源
            </button>
          </div>
        </div>
      </div>

      {/* —— 模式卡片网格 —— */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark p-5">
        <div className="mx-auto max-w-4xl">
          {/* 基础模式 */}
          <div className="mb-2 px-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">
            基础模式
          </div>
          <div className="mb-5 grid grid-cols-2 gap-3 sm:grid-cols-4">
            {MODES.filter((m) => !m.advanced).map((m) => (
              <ModeCardBtn key={m.key} mode={m} onClick={() => startMode(m.key)} />
            ))}
          </div>

          {/* 高级模式 */}
          <div className="mb-2 px-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">
            高级模式
          </div>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            {MODES.filter((m) => m.advanced).map((m) => (
              <ModeCardBtn key={m.key} mode={m} onClick={() => startMode(m.key)} />
            ))}
          </div>

          {/* 题集列表（选择后做题） */}
          {sets.length > 0 && (
            <div className="mt-6">
              <div className="mb-2 px-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">
                我的题集
              </div>
              <div className="space-y-1">
                {sets.map((s) => (
                  <button
                    key={s.id}
                    onClick={() => { selectSet(s.id); setMode("sequential"); }}
                    className="flex w-full items-center gap-3 rounded-lg border border-[var(--border-default)] bg-[var(--shell-inspector-panel)] px-3.5 py-2.5 text-left transition-colors hover:border-[var(--primary-color)]/40"
                  >
                    <ListNumbers size={16} className="shrink-0 text-primary" />
                    <span className="min-w-0 flex-1 truncate text-[13px] text-foreground">{s.title}</span>
                    <span className="shrink-0 text-[10px] text-muted-foreground/60">{s.questions?.length ?? 0} 题</span>
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* 抽题对话框（简易：输入 URI + 标题） */}
      {importOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={() => setImportOpen(false)}>
          <div className="w-[420px] rounded-[var(--radius-shell-dialog)] border border-[var(--border-default)] bg-[var(--shell-inspector-panel)] p-5 shadow-[var(--shadow-shell-floating)]" onClick={(e) => e.stopPropagation()}>
            <div className="mb-3 flex items-center gap-2">
              <FileText size={16} className="text-primary" />
              <span className="text-[13px] font-semibold text-foreground">从资源抽题</span>
            </div>
            <input
              value={extractUri}
              onChange={(e) => setExtractUri(e.target.value)}
              placeholder="资源 URI（如 vfs://textbook/xxx）"
              className="mb-2 w-full rounded-md border border-[var(--border-default)] bg-transparent px-2.5 py-1.5 text-[12px] text-foreground focus:outline-none focus:border-[var(--primary-color)]/40"
            />
            <input
              value={extractTitle}
              onChange={(e) => setExtractTitle(e.target.value)}
              placeholder="题集标题"
              className="mb-4 w-full rounded-md border border-[var(--border-default)] bg-transparent px-2.5 py-1.5 text-[12px] text-foreground focus:outline-none focus:border-[var(--primary-color)]/40"
            />
            <div className="flex justify-end gap-2">
              <button onClick={() => setImportOpen(false)} className="rounded-md px-3 py-1.5 text-[12px] text-muted-foreground hover:bg-[var(--interactive-hover)]">
                取消
              </button>
              <button
                disabled={!extractUri.trim() || extracting}
                onClick={async () => {
                  setExtracting(true);
                  const set = await extract(extractUri.trim(), extractTitle.trim() || "新题集");
                  setExtracting(false);
                  if (set) {
                    setImportOpen(false);
                    selectSet(set.id);
                    setMode("sequential");
                  }
                }}
                className="rounded-md bg-black px-3 py-1.5 text-[12px] font-medium text-white hover:bg-black/80 disabled:opacity-40 dark:bg-white dark:text-black"
              >
                {extracting ? "抽题中…" : "开始抽题"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// —— 模式卡片 ——
function ModeCardBtn({ mode, onClick }: { mode: ModeCard; onClick: () => void }) {
  const Icon = mode.icon;
  return (
    <button
      onClick={onClick}
      className="group flex flex-col gap-2 rounded-lg border border-[var(--border-default)] bg-[var(--shell-inspector-panel)] p-3.5 text-left transition-all hover:border-[var(--primary-color)]/40 hover:shadow-[var(--shadow-shell-soft)]"
    >
      <div className={cn("flex h-8 w-8 items-center justify-center rounded-md", mode.color)}>
        <Icon size={16} weight="regular" />
      </div>
      <div>
        <div className="text-[12.5px] font-medium text-foreground">{mode.title}</div>
        <div className="mt-0.5 text-[10px] leading-relaxed text-muted-foreground">{mode.desc}</div>
      </div>
    </button>
  );
}

// 保留图标引用
void MagnifyingGlass;
void Tag;
