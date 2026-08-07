// QBankPage —— 题库与练习
// ------------------------------------------------------------
// 三栏布局：
// 1. 左 (256px)：题集列表 + 抽题入口（SetList）
// 2. 中 (flex-1)：题目练习区（PracticePanel）
// 3. 右 (320px)：进度 + 提交 + AI 解析（ProgressPanel）
//
// 顶部由 ProgressPanel 内嵌的掌握度雷达承担

import { useEffect } from "react";
import { SetList } from "@/components/qbank/SetList";
import { PracticePanel } from "@/components/qbank/PracticePanel";
import { ProgressPanel } from "@/components/qbank/ProgressPanel";
import { useQBankStore } from "@/state/qbank";

export function QBankPage() {
  const loadMastery = useQBankStore((s) => s.loadMastery);

  // 挂载时拉取一次掌握度
  useEffect(() => {
    void loadMastery();
  }, [loadMastery]);

  return (
    <div className="flex h-full w-full min-h-0 bg-background">
      {/* —— 左：题集列表 —— */}
      <aside className="w-64 shrink-0 border-r border-border bg-card">
        <SetList />
      </aside>

      {/* —— 中：题目练习区 —— */}
      <section className="flex min-w-0 flex-1 flex-col">
        <PracticePanel />
      </section>

      {/* —— 右：进度 + AI 解析 —— */}
      <aside className="w-80 shrink-0 border-l border-border bg-background">
        <ProgressPanel />
      </aside>
    </div>
  );
}
