// PracticePanel —— 题目练习区
// ------------------------------------------------------------
// 显示当前题目（题干 + 选项 / 填空 / 简答）
// 支持题目导航（上一题 / 下一题 / 跳转）
// 开始 attempt 后才允许作答

import { useQBankStore, type Question } from "@/state/qbank";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { Badge } from "@/components/ui/Badge";
import { cn } from "@/lib/utils";
import {
  Play,
  ChevronLeft,
  ChevronRight,
  Loader2,
  Inbox,
  CheckCircle2,
  Save,
} from "lucide-react";

/** 题型徽章颜色映射 */
const TYPE_LABEL: Record<string, string> = {
  single: "单选",
  multi: "多选",
  fill: "填空",
  essay: "简答",
};

export function PracticePanel() {
  const activeSet = useQBankStore((s) => s.activeSet);
  const attempt = useQBankStore((s) => s.attempt);
  const currentIndex = useQBankStore((s) => s.currentIndex);
  const draftAnswers = useQBankStore((s) => s.draftAnswers);
  const loading = useQBankStore((s) => s.loading);
  const startAttempt = useQBankStore((s) => s.startAttempt);
  const next = useQBankStore((s) => s.next);
  const prev = useQBankStore((s) => s.prev);
  const save = useQBankStore((s) => s.save);

  if (!activeSet) {
    return <EmptyState />;
  }

  const questions = activeSet.questions ?? [];
  const current = questions[currentIndex];

  return (
    <div className="flex h-full w-full min-w-0 flex-col">
      {/* 头部：题集信息 + 操作 */}
      <div className="shrink-0 space-y-2 border-b border-border bg-card px-4 py-3">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <h2
              className="truncate text-sm font-semibold text-foreground"
              title={activeSet.title}
            >
              {activeSet.title}
            </h2>
            <div className="mt-0.5 flex items-center gap-2 text-[10px] text-muted-foreground/70">
              <span>共 {questions.length} 题</span>
              <span>·</span>
              <span>当前第 {Math.min(currentIndex + 1, questions.length)} 题</span>
              {attempt && (
                <>
                  <span>·</span>
                  <span className="text-primary">答题中</span>
                </>
              )}
            </div>
          </div>
          <div className="flex items-center gap-1.5">
            {!attempt ? (
              <Button
                size="sm"
                className="h-8"
                onClick={() => void startAttempt()}
                disabled={loading || questions.length === 0}
              >
                {loading ? (
                  <Loader2 size={12} className="animate-spin" />
                ) : (
                  <Play size={12} />
                )}
                {loading ? "准备中…" : "开始答题"}
              </Button>
            ) : (
              <Button
                variant="outline"
                size="sm"
                className="h-8"
                onClick={() => void save(activeSet)}
                title="保存题集到 Hub"
              >
                <Save size={12} />
                保存
              </Button>
            )}
          </div>
        </div>
      </div>

      {/* 题目内容 */}
      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        {current ? (
          <div className="space-y-4 px-6 py-5 animate-fade-in" key={current.id}>
            <QuestionView
              question={current}
              index={currentIndex}
              total={questions.length}
              draft={draftAnswers[current.id] ?? ""}
              disabled={!attempt}
              onChange={(ans) => useQBankStore.getState().setDraft(current.id, ans)}
              onCommit={(ans) => void useQBankStore.getState().answer(current.id, ans)}
            />
          </div>
        ) : (
          <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
            无题目
          </div>
        )}
      </div>

      {/* 底部导航 */}
      <div className="shrink-0 flex items-center justify-between gap-2 border-t border-border bg-card px-4 py-2">
        <Button
          variant="ghost"
          size="sm"
          className="h-8"
          onClick={prev}
          disabled={currentIndex === 0}
        >
          <ChevronLeft size={12} />
          上一题
        </Button>
        <div className="flex items-center gap-1">
          {questions.map((q, idx) => {
            const answered = !!draftAnswers[q.id]?.trim();
            return (
              <button
                key={q.id}
                type="button"
                onClick={() => useQBankStore.getState().jumpTo(idx)}
                className={cn(
                  "h-6 w-6 rounded text-[10px] font-medium transition-colors",
                  idx === currentIndex
                    ? "bg-primary text-primary-foreground"
                    : answered
                      ? "bg-primary/15 text-primary"
                      : "bg-muted text-muted-foreground hover:bg-accent"
                )}
                title={`第 ${idx + 1} 题`}
              >
                {idx + 1}
              </button>
            );
          })}
        </div>
        <Button
          variant="ghost"
          size="sm"
          className="h-8"
          onClick={next}
          disabled={currentIndex >= questions.length - 1}
        >
          下一题
          <ChevronRight size={12} />
        </Button>
      </div>
    </div>
  );
}

interface QuestionViewProps {
  question: Question;
  index: number;
  total: number;
  draft: string;
  disabled: boolean;
  onChange: (ans: string) => void;
  onCommit: (ans: string) => void;
}

function QuestionView({
  question,
  index,
  total,
  draft,
  disabled,
  onChange,
  onCommit,
}: QuestionViewProps) {
  const type = question.type ?? "single";
  const label = TYPE_LABEL[type] ?? type;

  return (
    <div className="space-y-4">
      {/* 题头 */}
      <div className="flex items-center gap-2">
        <Badge variant="secondary" className="text-[10px]">
          第 {index + 1} / {total} 题
        </Badge>
        <Badge variant="outline" className="text-[10px]">
          {label}
        </Badge>
        {question.knowledge && question.knowledge.length > 0 && (
          <div className="flex items-center gap-1 text-[10px] text-muted-foreground/70">
            <span>知识点：</span>
            {question.knowledge.map((k) => (
              <span
                key={k}
                className="rounded bg-muted px-1.5 py-0.5 font-mono"
              >
                {k}
              </span>
            ))}
          </div>
        )}
      </div>

      {/* 题干 */}
      <div className="rounded-md border border-border bg-background px-4 py-3">
        <pre className="whitespace-pre-wrap break-words font-sans text-[14px] leading-relaxed text-foreground">
          {question.stem}
        </pre>
      </div>

      {/* 作答区 */}
      <div className="space-y-2">
        <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
          作答
        </div>
        {disabled ? (
          <div className="rounded-md border border-dashed border-border bg-muted/30 px-4 py-3 text-[12px] text-muted-foreground">
            点击右上角"开始答题"后即可作答
          </div>
        ) : (
          <AnswerInput
            type={type}
            options={question.options}
            draft={draft}
            onChange={onChange}
            onCommit={onCommit}
          />
        )}
      </div>
    </div>
  );
}

interface AnswerInputProps {
  type: string;
  options?: string[];
  draft: string;
  onChange: (ans: string) => void;
  onCommit: (ans: string) => void;
}

function AnswerInput({
  type,
  options,
  draft,
  onChange,
  onCommit,
}: AnswerInputProps) {
  // 单选：从 options 中选一个（draft 存索引或值）
  if (type === "single" && options && options.length > 0) {
    const selectedIdx = draft ? Number(draft) : -1;
    return (
      <div className="space-y-1.5">
        {options.map((opt, idx) => (
          <button
            key={idx}
            type="button"
            disabled={false}
            onClick={() => {
              onChange(String(idx));
              onCommit(String(idx));
            }}
            className={cn(
              "flex w-full items-start gap-2.5 rounded-md border px-3 py-2 text-left text-[13px] transition-colors",
              selectedIdx === idx
                ? "border-primary/40 bg-primary/10 text-foreground"
                : "border-border bg-background text-foreground/90 hover:bg-accent/50"
            )}
          >
            <span
              className={cn(
                "flex h-5 w-5 shrink-0 items-center justify-center rounded-full border text-[10px] font-medium",
                selectedIdx === idx
                  ? "border-primary bg-primary text-primary-foreground"
                  : "border-border text-muted-foreground"
              )}
            >
              {String.fromCharCode(65 + idx)}
            </span>
            <span className="flex-1 whitespace-pre-wrap break-words">
              {opt}
            </span>
          </button>
        ))}
      </div>
    );
  }

  // 多选：从 options 中选多个（draft 存逗号分隔索引）
  if (type === "multi" && options && options.length > 0) {
    const selectedSet = new Set(
      draft
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean)
    );
    return (
      <div className="space-y-1.5">
        {options.map((opt, idx) => {
          const checked = selectedSet.has(String(idx));
          return (
            <button
              key={idx}
              type="button"
              onClick={() => {
                const next = new Set(selectedSet);
                if (checked) next.delete(String(idx));
                else next.add(String(idx));
                const arr = Array.from(next).sort();
                const ans = arr.join(",");
                onChange(ans);
                onCommit(ans);
              }}
              className={cn(
                "flex w-full items-start gap-2.5 rounded-md border px-3 py-2 text-left text-[13px] transition-colors",
                checked
                  ? "border-primary/40 bg-primary/10 text-foreground"
                  : "border-border bg-background text-foreground/90 hover:bg-accent/50"
              )}
            >
              <span
                className={cn(
                  "flex h-5 w-5 shrink-0 items-center justify-center rounded border text-[10px] font-medium",
                  checked
                    ? "border-primary bg-primary text-primary-foreground"
                    : "border-border text-muted-foreground"
                )}
              >
                {checked ? <CheckCircle2 size={11} /> : String.fromCharCode(65 + idx)}
              </span>
              <span className="flex-1 whitespace-pre-wrap break-words">
                {opt}
              </span>
            </button>
          );
        })}
      </div>
    );
  }

  // 填空：单行输入
  if (type === "fill") {
    return (
      <Input
        value={draft}
        onChange={(e) => onChange(e.target.value)}
        onBlur={() => onCommit(draft)}
        placeholder="填入答案…"
        className="h-9 text-[13px]"
      />
    );
  }

  // 简答：多行输入
  return (
    <Textarea
      value={draft}
      onChange={(e) => onChange(e.target.value)}
      onBlur={() => onCommit(draft)}
      placeholder="请输入你的解答…"
      rows={8}
      className="resize-none text-[13px]"
    />
  );
}

function EmptyState() {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
        <Inbox size={20} />
      </div>
      <div className="space-y-1">
        <div className="text-sm font-medium text-foreground">未选中题集</div>
        <div className="text-xs text-muted-foreground">
          从左侧选择题集，或通过 URI 抽题生成新题集
        </div>
      </div>
    </div>
  );
}
