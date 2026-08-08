// QBankApp —— Hub 内嵌题库应用（对齐原版 exam→题库）
// ------------------------------------------------------------
// 显示 qbank 类型题集列表，选中后做题（复用 PracticePanel 单栏版）。

import { useEffect, useState } from "react";
import { useHubStore } from "@/state/hub";
import { useQBankStore } from "@/state/qbank";
import { cn } from "@/lib/utils";
import { ListChecks, MagnifyingGlass, CaretLeft, CaretRight, CheckCircle, XCircle } from "@phosphor-icons/react";

export function QBankApp() {
  const entries = useHubStore((s) => s.entries);
  const sets = useQBankStore((s) => s.sets);
  const loadMastery = useQBankStore((s) => s.loadMastery);
  const selectSet = useQBankStore((s) => s.selectSet);
  const startAttempt = useQBankStore((s) => s.startAttempt);
  const attempt = useQBankStore((s) => s.attempt);
  const activeSet = useQBankStore((s) => s.activeSet);
  const currentIndex = useQBankStore((s) => s.currentIndex);
  const draftAnswers = useQBankStore((s) => s.draftAnswers);
  const setDraft = useQBankStore((s) => s.setDraft);
  const submit = useQBankStore((s) => s.submit);
  const [keyword, setKeyword] = useState("");

  useEffect(() => {
    void loadMastery();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const qbankSets = sets.length > 0 ? sets : [];
  const visible = keyword.trim()
    ? qbankSets.filter((s) => s.title.toLowerCase().includes(keyword.toLowerCase()))
    : qbankSets;

  // 做题中
  if (attempt && activeSet) {
    const q = activeSet.questions[currentIndex];
    const isSubmitted = attempt.score !== undefined;
    return (
      <div className="flex h-full w-full min-h-0 flex-col">
        <div className="flex shrink-0 items-center gap-2 border-b border-[var(--shell-seam)] px-4 py-2">
          <ListChecks size={14} className="text-primary" />
          <span className="truncate text-[12.5px] font-medium text-foreground">{activeSet.title}</span>
          <span className="text-[11px] text-muted-foreground/60">
            {currentIndex + 1} / {activeSet.questions.length}
          </span>
          <button
            onClick={() => void startAttempt()}
            className="ml-auto rounded px-2 py-0.5 text-[11px] text-muted-foreground hover:bg-[var(--interactive-hover)] hover:text-foreground"
          >
            重做
          </button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark p-5">
          {q && (
            <div className="mx-auto max-w-2xl">
              {/* 题干 */}
              <div className="rounded-lg border border-[var(--border-default)] bg-[var(--shell-inspector-panel)] p-4">
                <div className="mb-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">
                  {q.type ?? "选择题"}
                </div>
                <div className="whitespace-pre-wrap text-[14px] leading-relaxed text-foreground">{q.stem}</div>
              </div>

              {/* 选项 */}
              <div className="mt-3 space-y-1.5">
                {(q.options ?? []).map((opt, i) => {
                  const letter = String.fromCharCode(65 + i);
                  const selected = draftAnswers[q.id] === opt;
                  return (
                    <button
                      key={i}
                      disabled={isSubmitted}
                      onClick={() => setDraft(q.id, opt)}
                      className={cn(
                        "flex w-full items-center gap-2.5 rounded-lg border px-3.5 py-2.5 text-left transition-colors",
                        selected
                          ? "border-[var(--primary-color)]/50 bg-primary/10"
                          : "border-[var(--border-default)] bg-[var(--shell-inspector-panel)] hover:border-[var(--primary-color)]/30",
                        isSubmitted && opt === q.answer && "border-emerald-500/50 bg-emerald-500/10"
                      )}
                    >
                      <span className={cn("flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-[11px]", selected ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground")}>
                        {letter}
                      </span>
                      <span className="text-[13px] text-foreground">{opt}</span>
                      {isSubmitted && opt === q.answer && <CheckCircle size={15} className="ml-auto shrink-0 text-emerald-500" weight="fill" />}
                      {isSubmitted && selected && opt !== q.answer && <XCircle size={15} className="ml-auto shrink-0 text-destructive" weight="fill" />}
                    </button>
                  );
                })}
              </div>

              {/* 操作 */}
              {isSubmitted ? (
                <div className="mt-4 flex items-center gap-2">
                  <span className="rounded-md bg-emerald-500/15 px-2.5 py-1 text-[11px] text-emerald-600">
                    得分 {attempt.score}/{attempt.total}
                  </span>
                  <button
                    onClick={() => void startAttempt()}
                    className="ml-auto rounded-md bg-black px-3 py-1.5 text-[12px] font-medium text-white hover:bg-black/80 dark:bg-white dark:text-black"
                  >
                    重新开始
                  </button>
                </div>
              ) : (
                <div className="mt-4 flex items-center gap-2">
                  <button
                    disabled={currentIndex <= 0}
                    onClick={() => selectSet(activeSet.id)}
                    className="rounded-md border border-[var(--border-default)] px-3 py-1.5 text-[12px] text-muted-foreground hover:bg-[var(--interactive-hover)] disabled:opacity-30"
                  >
                    <CaretLeft size={12} className="mr-1 inline" />
                    上一题
                  </button>
                  <button
                    onClick={() => void submit()}
                    className="ml-auto rounded-md bg-black px-3 py-1.5 text-[12px] font-medium text-white hover:bg-black/80 dark:bg-white dark:text-black"
                  >
                    提交
                  </button>
                  <button
                    disabled={currentIndex >= activeSet.questions.length - 1}
                    onClick={() => selectSet(activeSet.id)}
                    className="rounded-md border border-[var(--border-default)] px-3 py-1.5 text-[12px] text-muted-foreground hover:bg-[var(--interactive-hover)] disabled:opacity-30"
                  >
                    下一题
                    <CaretRight size={12} className="ml-1 inline" />
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    );
  }

  // 题集列表
  return (
    <div className="flex h-full w-full min-h-0 flex-col">
      <div className="flex shrink-0 items-center gap-2 border-b border-[var(--shell-seam)] px-3 py-2">
        <div className="flex flex-1 items-center gap-2 rounded-md bg-[var(--interactive-hover)] px-2 py-1.5">
          <MagnifyingGlass size={13} className="text-muted-foreground" />
          <input
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder="搜索题集…"
            className="w-full bg-transparent text-[12px] text-foreground placeholder:text-muted-foreground/60 focus:outline-none"
          />
        </div>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark p-3">
        {visible.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 text-muted-foreground">
            <ListChecks size={26} className="opacity-40" />
            <span className="text-[12px]">没有题集，去聊天生成或导入</span>
          </div>
        ) : (
          <div className="mx-auto max-w-2xl space-y-1">
            {visible.map((s) => (
              <button
                key={s.id}
                onClick={() => { selectSet(s.id); void startAttempt(); }}
                className="flex w-full items-center gap-3 rounded-lg border border-[var(--border-default)] bg-[var(--shell-inspector-panel)] px-3.5 py-2.5 text-left transition-colors hover:border-[var(--primary-color)]/40"
              >
                <ListChecks size={16} className="shrink-0 text-primary" />
                <span className="min-w-0 flex-1 truncate text-[13px] text-foreground">{s.title}</span>
                <span className="shrink-0 text-[10px] text-muted-foreground/60">{s.questions?.length ?? 0} 题</span>
                <CaretRight size={12} className="shrink-0 text-muted-foreground/50" />
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
