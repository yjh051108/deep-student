// EssayPage —— 作文批改
// ------------------------------------------------------------
// 双栏布局：
// 1. 左 (flex-1)：作文输入区（EssayInput）
// 2. 右 (480px)：批改结果展示（EssayResultPanel）

import { EssayInput } from "@/components/essay/EssayInput";
import { EssayResultPanel } from "@/components/essay/EssayResultPanel";

export function EssayPage() {
  return (
    <div className="flex h-full w-full min-h-0 bg-background">
      {/* —— 左：作文输入 —— */}
      <section className="flex min-w-0 flex-1 flex-col">
        <EssayInput />
      </section>

      {/* —— 右：批改结果 —— */}
      <aside className="w-[30rem] shrink-0 border-l border-[var(--shell-seam)] bg-background">
        <EssayResultPanel />
      </aside>
    </div>
  );
}
