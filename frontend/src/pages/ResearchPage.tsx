// ResearchPage —— 深度调研
// ------------------------------------------------------------
// 顶部：控制条（主题 + 深度 + 格式 + 引擎 + 按钮）
// 主体：计划步骤 / 调研中 / 报告视图（章节列表 + 内容 + 来源）

import { ResearchControlBar } from "@/components/research/ResearchControlBar";
import { ResearchReportPanel } from "@/components/research/ResearchReportPanel";

export function ResearchPage() {
  return (
    <div className="flex h-full w-full min-h-0 flex-col bg-background">
      {/* —— 顶部：控制条 —— */}
      <ResearchControlBar />

      {/* —— 主体：报告 / 计划 / 加载 —— */}
      <section className="flex min-h-0 flex-1 flex-col">
        <ResearchReportPanel />
      </section>
    </div>
  );
}
