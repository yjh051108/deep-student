// FSRS 复习页 —— 间隔重复复习（到期队列 / 评分 / 牌组 / 统计）
// ------------------------------------------------------------
// 布局：左牌组列表 + 中复习区（正面→翻面→评分）+ 右统计/全部卡片
// 对接 fsrsApi（FSRS* 方法）。

import { useEffect, useMemo, useState } from "react";
import { fsrsApi, type FSRSCard, type FSRSDeckStat } from "@/lib/fsrs";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { Input } from "@/components/ui/Input";
import {
  Layers,
  Brain,
  Plus,
  Trash2,
  Loader2,
  CheckCircle2,
  XCircle,
  ChevronRight,
  AlertCircle,
} from "lucide-react";
import { cn } from "@/lib/utils";

export function FSRSReviewPage() {
  const [due, setDue] = useState<FSRSCard[]>([]);
  const [all, setAll] = useState<FSRSCard[]>([]);
  const [decks, setDecks] = useState<FSRSDeckStat[]>([]);
  const [current, setCurrent] = useState<FSRSCard | null>(null);
  const [flipped, setFlipped] = useState(false);
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState<string | null>(null);
  const [newDeck, setNewDeck] = useState("");
  const [newFront, setNewFront] = useState("");
  const [newBack, setNewBack] = useState("");

  const load = async () => {
    setLoading(true);
    const [d, a, ds] = await Promise.all([
      fsrsApi.due("", 50),
      fsrsApi.all("", 200),
      fsrsApi.deckStats(),
    ]);
    setDue(d ?? []);
    setAll(a ?? []);
    setDecks(ds ?? []);
    if (!current && d && d.length > 0) setCurrent(d[0]);
    setLoading(false);
  };

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const addCard = async () => {
    if (!newDeck.trim() || !newFront.trim() || !newBack.trim()) return;
    await fsrsApi.addCards(newDeck.trim(), [{ front: newFront.trim(), back: newBack.trim() }]);
    setNewFront("");
    setNewBack("");
    setToast(`已添加到「${newDeck.trim()}」`);
    await load();
  };

  const rate = async (rating: number) => {
    if (!current) return;
    await fsrsApi.review(current.cardId, rating);
    const rest = due.filter((c) => c.cardId !== current.cardId);
    setDue(rest);
    setCurrent(rest[0] ?? null);
    setFlipped(false);
    setToast(rating === 1 ? "重来（10 分钟后）" : rating === 4 ? "简单！" : "已复习");
    await load();
  };

  const removeCard = async (id: string) => {
    await fsrsApi.remove(id);
    setAll(all.filter((c) => c.cardId !== id));
    setToast("已删除");
  };

  const dueCount = due.length;

  return (
    <div className="flex h-full w-full min-h-0 bg-background">
      {/* —— 左：牌组 + 添加 —— */}
      <aside className="w-60 shrink-0 border-r border-border bg-card">
        <div className="flex h-full flex-col">
          <div className="shrink-0 border-b border-border px-3 py-2.5">
            <div className="flex items-center gap-2">
              <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary/15 text-primary">
                <Layers size={13} />
              </div>
              <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                FSRS 牌组
              </span>
            </div>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark p-2">
            {decks.length === 0 ? (
              <p className="px-2 py-4 text-center text-[11px] text-muted-foreground">
                暂无牌组
              </p>
            ) : (
              <div className="space-y-1">
                {decks.map((d) => (
                  <div key={d.deck} className="flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-accent">
                    <Brain size={12} className="shrink-0 text-primary" />
                    <span className="min-w-0 flex-1 truncate text-[12px] text-foreground">
                      {d.deck}
                    </span>
                    {d.due > 0 && (
                      <Badge className="text-[9px]">{d.due} 待复习</Badge>
                    )}
                    <span className="text-[9px] text-muted-foreground/60">
                      {d.total}
                    </span>
                  </div>
                ))}
              </div>
            )}
            <div className="mt-3 border-t border-border/60 pt-3">
              <div className="mb-1.5 px-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                添加卡片
              </div>
              <div className="space-y-1.5">
                <Input
                  value={newDeck}
                  onChange={(e) => setNewDeck(e.target.value)}
                  placeholder="牌组名（如 英语单词）"
                  className="h-7 text-[11px]"
                />
                <Input
                  value={newFront}
                  onChange={(e) => setNewFront(e.target.value)}
                  placeholder="正面"
                  className="h-7 text-[11px]"
                />
                <Input
                  value={newBack}
                  onChange={(e) => setNewBack(e.target.value)}
                  placeholder="背面"
                  onKeyDown={(e) => e.key === "Enter" && void addCard()}
                  className="h-7 text-[11px]"
                />
                <Button size="sm" className="h-7 w-full" disabled={!newDeck.trim() || !newFront.trim() || !newBack.trim()} onClick={() => void addCard()}>
                  <Plus size={12} className="mr-1" />
                  添加
                </Button>
              </div>
            </div>
          </div>
        </div>
      </aside>

      {/* —— 中：复习区 —— */}
      <section className="flex min-w-0 flex-1 flex-col">
        <div className="shrink-0 border-b border-border bg-card px-4 py-2.5">
          <div className="flex items-center justify-between">
            <h1 className="text-sm font-semibold text-foreground">间隔重复复习</h1>
            <span className="text-[11px] text-muted-foreground">
              待复习 <b className="text-primary">{dueCount}</b> 张
            </span>
          </div>
        </div>

        <div className="flex min-h-0 flex-1 items-center justify-center bg-muted/20 p-6">
          {loading && !current ? (
            <Loader2 size={24} className="animate-spin text-muted-foreground" />
          ) : !current ? (
            <div className="text-center">
              <CheckCircle2 size={36} className="mx-auto mb-3 text-emerald-500" />
              <p className="text-sm font-medium text-foreground">今日复习完成 🎉</p>
              <p className="mt-1 text-xs text-muted-foreground">
                {dueCount === 0 ? "没有到期卡片" : "稍后再来看看"}
              </p>
              <Button size="sm" variant="outline" className="mt-4" onClick={() => void load()}>
                <Loader2 size={12} className="mr-1" />
                刷新
              </Button>
            </div>
          ) : (
            <div className="w-full max-w-xl">
              {/* 卡片 */}
              <div
                onClick={() => setFlipped((v) => !v)}
                className={cn(
                  "flex min-h-56 cursor-pointer flex-col items-center justify-center rounded-xl border bg-card p-8 text-center shadow-soft transition-transform",
                  flipped ? "border-primary/40" : "border-border"
                )}
              >
                <Badge variant="secondary" className="mb-3 self-start text-[9px]">
                  {current.deck} · {current.state}
                </Badge>
                <p className="text-lg font-medium leading-relaxed text-foreground">
                  {flipped ? current.back : current.front}
                </p>
                <p className="mt-4 text-[10px] text-muted-foreground">
                  {flipped ? "点卡片返回" : "点击卡片查看答案"}
                </p>
              </div>

              {/* 评分按钮（翻面后显示） */}
              {flipped ? (
                <div className="mt-4 grid grid-cols-4 gap-2">
                  <RateButton label="重来" color="destructive" icon={<XCircle size={14} />} onClick={() => void rate(1)} />
                  <RateButton label="困难" color="amber" icon={<AlertCircle size={14} />} onClick={() => void rate(2)} />
                  <RateButton label="良好" color="emerald" icon={<CheckCircle2 size={14} />} onClick={() => void rate(3)} />
                  <RateButton label="简单" color="primary" icon={<ChevronRight size={14} />} onClick={() => void rate(4)} />
                </div>
              ) : (
                <div className="mt-4 text-center text-[10px] text-muted-foreground">
                  剩余 {dueCount} 张 · 难度 {current.difficulty.toFixed(1)} · 稳定性 {current.stability.toFixed(1)}d
                </div>
              )}
            </div>
          )}
        </div>
      </section>

      {/* —— 右：全部卡片 + 统计 —— */}
      <aside className="w-72 shrink-0 border-l border-border bg-card">
        <div className="flex h-full flex-col">
          <div className="shrink-0 border-b border-border px-3 py-2.5">
            <div className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              全部卡片（{all.length}）
            </div>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
            {all.length === 0 ? (
              <p className="px-4 py-6 text-center text-[11px] text-muted-foreground">
                还没有卡片
              </p>
            ) : (
              <div className="space-y-1 p-2">
                {all.map((c) => (
                  <div key={c.cardId} className="group flex items-center gap-2 rounded-md border border-border/50 px-2 py-1.5">
                    <span className="min-w-0 flex-1 truncate text-[11px] text-foreground">
                      {c.front}
                    </span>
                    <span className={cn("text-[9px]", new Date(c.dueAt).getTime() <= Date.now() ? "text-amber-500" : "text-muted-foreground/50")}>
                      {new Date(c.dueAt) <= new Date() ? "待复习" : `${Math.ceil((new Date(c.dueAt).getTime() - Date.now()) / 86400000)}d`}
                    </span>
                    <button
                      onClick={() => void removeCard(c.cardId)}
                      className="hidden rounded p-0.5 text-muted-foreground hover:text-destructive group-hover:block"
                    >
                      <Trash2 size={11} />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
          {toast && (
            <div className="shrink-0 border-t border-border px-3 py-2 text-[11px] text-emerald-500">
              {toast}
            </div>
          )}
        </div>
      </aside>
    </div>
  );
}

function RateButton({
  label,
  color,
  icon,
  onClick,
}: {
  label: string;
  color: "destructive" | "amber" | "emerald" | "primary";
  icon: React.ReactNode;
  onClick: () => void;
}) {
  const colorClass =
    color === "destructive"
      ? "bg-destructive/15 text-destructive hover:bg-destructive/25"
      : color === "amber"
        ? "bg-amber-500/15 text-amber-500 hover:bg-amber-500/25"
        : color === "emerald"
          ? "bg-emerald-500/15 text-emerald-600 hover:bg-emerald-500/25"
          : "bg-primary/15 text-primary hover:bg-primary/25";
  return (
    <Button size="sm" className={cn("h-10", colorClass)} onClick={onClick}>
      {icon}
      <span className="ml-1">{label}</span>
    </Button>
  );
}
