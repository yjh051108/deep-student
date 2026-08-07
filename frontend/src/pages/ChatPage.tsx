// Chat 页面 —— 多模型对话
// ------------------------------------------------------------
// 对照原版 features/chat/pages/ChatV2Page.tsx
// 三栏布局：
//   左 (260px) - SessionList 会话列表
//   中 (flex-1) - MessageList 对话区 + InputBar 输入栏
//   右 (300px) - RefsPanel 引用面板（可折叠）

import { useEffect } from "react";
import { useChatStore } from "@/state/chat";
import { SessionSidebar } from "@/components/chat/SessionList";
import { MessageList } from "@/components/chat/MessageList";
import { InputBar } from "@/components/chat/InputBar";
import { RefsPanel } from "@/components/chat/RefsPanel";
import { useSearchParams } from "react-router-dom";
import { cn } from "@/lib/utils";
import { useState } from "react";
import { PanelRightClose, PanelRightOpen } from "lucide-react";
import { Button } from "@/components/ui/Button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/Tooltip";

export function ChatPage() {
  const init = useChatStore((s) => s.init);
  const setView = useChatStore((s) => s.setView);
  const createSession = useChatStore((s) => s.createSession);
  const sessions = useChatStore((s) => s.sessions);
  const activeSessionId = useChatStore((s) => s.activeSessionId);
  const [searchParams, setSearchParams] = useSearchParams();
  const [refsPanelOpen, setRefsPanelOpen] = useState(true);

  // 初始化（加载分组 + 会话）
  useEffect(() => {
    void init();
  }, [init]);

  // 处理 ?new=1 自动创建会话
  useEffect(() => {
    if (searchParams.get("new") === "1") {
      createSession("新会话").then(() => {
        setSearchParams({}, { replace: true });
      });
    }
  }, [searchParams, createSession, setSearchParams]);

  const activeSession = sessions.find((s) => s.id === activeSessionId);

  return (
    <div className="flex h-full w-full min-h-0 bg-background">
      {/* —— 左：会话列表 —— */}
      <SessionSidebar />

      {/* —— 中：对话区 —— */}
      <div className="flex min-w-0 flex-1 flex-col">
        <MessageList
          session={activeSession ?? null}
          onDeleteMessage={(msgId) => {
            if (activeSession) void deleteOneMessage(activeSession.id, msgId);
          }}
        />
        <InputBar />
      </div>

      {/* —— 右：引用面板（可折叠） —— */}
      <div
        className={cn(
          "shrink-0 border-l border-border bg-card transition-[width] duration-200 ease-out",
          refsPanelOpen ? "w-72" : "w-12"
        )}
      >
        <div className="flex h-full flex-col">
          <div className="flex h-14 shrink-0 items-center justify-between border-b border-border px-3">
            {refsPanelOpen && (
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                引用 Refs
              </span>
            )}
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7"
                  onClick={() => setRefsPanelOpen((v) => !v)}
                  aria-label={refsPanelOpen ? "折叠引用面板" : "展开引用面板"}
                >
                  {refsPanelOpen ? (
                    <PanelRightClose size={14} />
                  ) : (
                    <PanelRightOpen size={14} />
                  )}
                </Button>
              </TooltipTrigger>
              <TooltipContent side="left">
                {refsPanelOpen ? "折叠" : "展开"}
              </TooltipContent>
            </Tooltip>
          </div>
          {refsPanelOpen && <RefsPanel />}
        </div>
      </div>
    </div>
  );
}

// 删除单条消息（后端 + 本地状态同步）
async function deleteOneMessage(sessionId: string, messageId: string) {
  const { chatV2Api } = await import("@/lib/chat");
  await chatV2Api.deleteMessage(sessionId, messageId);
  const store = useChatStore.getState();
  // 本地移除
  useChatStore.setState({
    sessions: store.sessions.map((sess) =>
      sess.id === sessionId
        ? { ...sess, messages: sess.messages.filter((m) => m.id !== messageId) }
        : sess
    ),
  });
}
