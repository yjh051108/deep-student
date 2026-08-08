// SkillsPage —— 技能与 MCP 管理
// ------------------------------------------------------------
// 三栏布局：
// 1. 左：技能列表（SkillsList）+ 加载 SKILL.md 表单
// 2. 中：工具列表（SkillsTools）+ 工具调用面板
// 3. 右：MCP 服务器管理（启用 / Spawn / 禁用）

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { Badge } from "@/components/ui/Badge";
import { cn } from "@/lib/utils";
import {
  useSkillsStore,
  TIERS,
  type SkillTier,
  type MCPServerConfig,
} from "@/state/skills";
import {
  Sparkles,
  Wrench,
  Server,
  Play,
  Loader2,
  AlertCircle,
  CheckCircle2,
  Plus,
  Trash2,
  Upload,
  Power,
  FileCode2,
  Inbox,
  RefreshCw,
  Terminal,
} from "lucide-react";

export function SkillsPage() {
  const refresh = useSkillsStore((s) => s.refresh);
  const refreshTools = useSkillsStore((s) => s.refreshTools);
  const refreshMCP = useSkillsStore((s) => s.refreshMCP);
  const error = useSkillsStore((s) => s.error);
  const notice = useSkillsStore((s) => s.notice);
  const clearError = useSkillsStore((s) => s.clearError);
  const clearNotice = useSkillsStore((s) => s.clearNotice);

  useEffect(() => {
    void refresh();
    void refreshTools();
    void refreshMCP();
  }, [refresh, refreshTools, refreshMCP]);

  return (
    <div className="flex h-full w-full min-h-0 flex-col bg-background">
      {/* —— 顶部标题栏 —— */}
      <header className="shrink-0 flex items-center justify-between border-b border-[var(--shell-seam)] bg-[var(--shell-inspector-panel)] px-4 py-3">
        <div className="flex items-center gap-2 text-sm">
          <Sparkles size={16} className="text-primary" />
          <h1 className="font-semibold text-foreground">技能与 MCP</h1>
          <span className="text-[11px] text-muted-foreground/60">
            SKILL.md / 工具调用 / MCP 服务器
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          <Button
            variant="ghost"
            size="sm"
            className="h-7"
            onClick={() => {
              void refresh();
              void refreshTools();
              void refreshMCP();
            }}
          >
            <RefreshCw size={12} />
            刷新
          </Button>
        </div>
      </header>

      {/* —— 错误 / 通知横幅 —— */}
      {error && (
        <div className="shrink-0 flex items-center justify-between gap-2 border-b border-destructive/30 bg-destructive/10 px-4 py-2 text-xs text-destructive">
          <div className="flex min-w-0 items-center gap-2">
            <AlertCircle size={12} className="shrink-0" />
            <span className="truncate">{error}</span>
          </div>
          <button
            type="button"
            onClick={clearError}
            className="shrink-0 text-destructive/70 hover:text-destructive"
          >
            ×
          </button>
        </div>
      )}
      {notice && (
        <div className="shrink-0 flex items-center justify-between gap-2 border-b border-emerald-500/30 bg-emerald-500/10 px-4 py-2 text-xs text-emerald-500">
          <div className="flex min-w-0 items-center gap-2">
            <CheckCircle2 size={12} className="shrink-0" />
            <span className="truncate">{notice}</span>
          </div>
          <button
            type="button"
            onClick={clearNotice}
            className="shrink-0 text-emerald-500/70 hover:text-emerald-500"
          >
            ×
          </button>
        </div>
      )}

      {/* —— 三栏主体 —— */}
      <div className="flex min-h-0 flex-1 gap-px bg-border">
        <aside className="w-72 shrink-0 overflow-y-auto bg-[var(--shell-inspector-panel)] scrollbar-dark">
          <SkillListPanel />
        </aside>
        <section className="flex min-w-0 flex-1 flex-col overflow-y-auto bg-background scrollbar-dark">
          <ToolPanel />
        </section>
        <aside className="w-80 shrink-0 overflow-y-auto bg-[var(--shell-inspector-panel)] scrollbar-dark">
          <MCPPanel />
        </aside>
      </div>
    </div>
  );
}

// ============================================================
// 左：技能列表 + 加载 SKILL.md
// ============================================================
function SkillListPanel() {
  const skills = useSkillsStore((s) => s.skills);
  const activeSkillName = useSkillsStore((s) => s.activeSkillName);
  const selectSkill = useSkillsStore((s) => s.selectSkill);
  const loading = useSkillsStore((s) => s.loading);
  const loadSkillMD = useSkillsStore((s) => s.loadSkillMD);

  const [tier, setTier] = useState<SkillTier>("project");
  const [path, setPath] = useState("");

  const activeSkill = skills.find((s) => s.name === activeSkillName) ?? null;

  const handleLoad = async () => {
    if (!path.trim()) return;
    await loadSkillMD(tier, path.trim());
    setPath("");
  };

  return (
    <div className="flex h-full flex-col">
      <div className="shrink-0 border-b border-[var(--shell-seam)] px-3 py-2.5">
        <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
          <Sparkles size={11} className="text-primary" />
          技能列表
          {skills.length > 0 && (
            <span className="font-normal normal-case text-muted-foreground/50">
              · {skills.length}
            </span>
          )}
        </div>
      </div>

      {/* 技能列表 */}
      <div className="min-h-0 flex-1 overflow-y-auto px-2 py-2 scrollbar-dark">
        {skills.length === 0 ? (
          <EmptyMini
            loading={loading}
            icon={Sparkles}
            title="暂无技能"
            hint="加载 SKILL.md 或刷新列表"
          />
        ) : (
          <ul className="space-y-1">
            {skills.map((s) => {
              const active = s.name === activeSkillName;
              return (
                <li key={s.name}>
                  <button
                    type="button"
                    onClick={() => selectSkill(s.name)}
                    className={cn(
                      "group flex w-full items-start gap-2 rounded-md px-2.5 py-2 text-left text-sm transition-colors",
                      active
                        ? "bg-primary/12 text-primary"
                        : "text-muted-foreground hover:bg-accent hover:text-foreground"
                    )}
                  >
                    <FileCode2
                      size={14}
                      className={cn(
                        "mt-0.5 shrink-0",
                        active ? "text-primary" : "text-muted-foreground"
                      )}
                    />
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-[13px] font-medium">
                        {s.title || s.name}
                      </div>
                      <div className="truncate font-mono text-[10px] text-muted-foreground/60">
                        {s.name}
                      </div>
                      <div className="mt-1 flex items-center gap-1">
                        <Badge
                          variant="outline"
                          className="px-1 py-0 text-[9px] font-normal"
                        >
                          {s.tier}
                        </Badge>
                        {s.tools?.length > 0 && (
                          <span className="text-[9px] text-muted-foreground/60">
                            {s.tools.length} 工具
                          </span>
                        )}
                      </div>
                    </div>
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </div>

      {/* 选中技能的详情 */}
      {activeSkill && (
        <div className="shrink-0 space-y-2 border-t border-[var(--shell-seam)] bg-muted/20 px-3 py-3">
          <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            技能详情
          </div>
          <div className="text-[12px] leading-relaxed text-foreground/80">
            {activeSkill.description || "（无描述）"}
          </div>
          {activeSkill.prompt && (
            <div>
              <div className="mb-1 text-[10px] text-muted-foreground/60">
                Prompt：
              </div>
              <pre className="max-h-32 overflow-y-auto whitespace-pre-wrap break-words rounded border border-[var(--shell-seam)] bg-background p-2 font-mono text-[10px] leading-relaxed text-foreground/70 scrollbar-dark">
                {activeSkill.prompt}
              </pre>
            </div>
          )}
        </div>
      )}

      {/* 加载 SKILL.md 表单 */}
      <div className="shrink-0 space-y-2 border-t border-[var(--shell-seam)] px-3 py-3">
        <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
          加载 SKILL.md
        </div>
        <div className="flex flex-wrap gap-1">
          {TIERS.map((t) => (
            <button
              key={t.key}
              type="button"
              onClick={() => setTier(t.key)}
              title={t.description}
              className={cn(
                "rounded-full border px-2 py-0.5 text-[10px] font-medium transition-colors",
                tier === t.key
                  ? "border-primary/40 bg-primary/15 text-primary"
                  : "border-[var(--shell-seam)] bg-transparent text-muted-foreground hover:bg-accent hover:text-foreground"
              )}
            >
              {t.label}
            </button>
          ))}
        </div>
        <Input
          value={path}
          onChange={(e) => setPath(e.target.value)}
          placeholder="SKILL.md 路径（如 ./.skills/research/SKILL.md）"
          className="h-8 font-mono text-[11px]"
        />
        <Button
          size="sm"
          className="h-7 w-full"
          onClick={handleLoad}
          disabled={loading || !path.trim()}
        >
          {loading ? (
            <Loader2 size={12} className="animate-spin" />
          ) : (
            <Upload size={12} />
          )}
          {loading ? "加载中…" : "加载"}
        </Button>
      </div>
    </div>
  );
}

// ============================================================
// 中：工具列表 + 工具调用面板
// ============================================================
function ToolPanel() {
  const tools = useSkillsStore((s) => s.tools);
  const loading = useSkillsStore((s) => s.loading);
  const callTool = useSkillsStore((s) => s.callTool);
  const callResult = useSkillsStore((s) => s.callResult);
  const callingTool = useSkillsStore((s) => s.callingTool);
  const refreshTools = useSkillsStore((s) => s.refreshTools);

  const [selectedTool, setSelectedTool] = useState<string | null>(null);
  const [args, setArgs] = useState("{}");

  const handleCall = async () => {
    if (!selectedTool) return;
    await callTool(selectedTool, args);
  };

  return (
    <div className="flex h-full flex-col">
      {/* 工具列表 */}
      <div className="shrink-0 border-b border-[var(--shell-seam)] bg-[var(--shell-inspector-panel)] px-4 py-3">
        <div className="mb-2 flex items-center justify-between">
          <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            <Wrench size={11} className="text-primary" />
            工具列表
            {tools.length > 0 && (
              <span className="font-normal normal-case text-muted-foreground/50">
                · {tools.length}
              </span>
            )}
          </div>
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={() => void refreshTools()}
            title="刷新工具"
          >
            <RefreshCw size={11} className={cn(loading && "animate-spin")} />
          </Button>
        </div>
        {tools.length === 0 ? (
          <EmptyMini
            loading={loading}
            icon={Wrench}
            title="暂无工具"
            hint="刷新工具或加载带工具的 SKILL.md"
          />
        ) : (
          <ul className="grid grid-cols-2 gap-1.5 lg:grid-cols-3">
            {tools.map((t) => {
              const active = t.name === selectedTool;
              return (
                <li key={t.name}>
                  <button
                    type="button"
                    onClick={() => setSelectedTool(t.name)}
                    className={cn(
                      "w-full rounded-md border px-2.5 py-1.5 text-left transition-colors",
                      active
                        ? "border-primary/40 bg-primary/10"
                        : "border-[var(--shell-seam)] bg-background hover:bg-accent"
                    )}
                  >
                    <div className="flex items-center gap-1">
                      <Wrench
                        size={10}
                        className={cn(
                          "shrink-0",
                          active ? "text-primary" : "text-muted-foreground"
                        )}
                      />
                      <span
                        className={cn(
                          "truncate font-mono text-[11px] font-medium",
                          active ? "text-primary" : "text-foreground"
                        )}
                      >
                        {t.name}
                      </span>
                    </div>
                    <div className="mt-0.5 truncate text-[10px] text-muted-foreground/70">
                      {t.description || "（无描述）"}
                    </div>
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </div>

      {/* 工具调用面板 */}
      <div className="flex min-h-0 flex-1 flex-col">
        {selectedTool ? (
          <div className="flex min-h-0 flex-1 flex-col">
            <div className="shrink-0 space-y-2 border-b border-[var(--shell-seam)] bg-[var(--shell-inspector-panel)] px-4 py-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                  <Terminal size={11} className="text-primary" />
                  调用工具
                </div>
                <Badge variant="outline" className="font-mono text-[10px]">
                  {selectedTool}
                </Badge>
              </div>
              <div>
                <div className="mb-1 text-[10px] text-muted-foreground/60">
                  参数（JSON）
                </div>
                <Textarea
                  value={args}
                  onChange={(e) => setArgs(e.target.value)}
                  placeholder='{\n  "key": "value"\n}'
                  rows={5}
                  className="resize-none font-mono text-[12px]"
                />
              </div>
              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  className="h-7"
                  onClick={handleCall}
                  disabled={callingTool !== null}
                >
                  {callingTool !== null ? (
                    <Loader2 size={12} className="animate-spin" />
                  ) : (
                    <Play size={12} />
                  )}
                  {callingTool !== null ? "调用中…" : "调用"}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7"
                  onClick={() => setArgs("{}")}
                >
                  重置
                </Button>
              </div>
            </div>

            {/* 调用结果 */}
            <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
              <div className="px-4 py-3">
                <div className="mb-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                  调用结果
                </div>
                {callResult ? (
                  <pre className="whitespace-pre-wrap break-words rounded-md border border-[var(--shell-seam)] bg-muted/30 p-3 font-mono text-[11px] leading-relaxed text-foreground/90">
                    {callResult}
                  </pre>
                ) : (
                  <div className="py-8 text-center text-xs text-muted-foreground">
                    {callingTool !== null
                      ? "正在调用…"
                      : "尚未调用或结果为空"}
                  </div>
                )}
              </div>
            </div>
          </div>
        ) : (
          <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
              <Terminal size={20} />
            </div>
            <div className="space-y-1">
              <div className="text-sm font-medium text-foreground">
                未选择工具
              </div>
              <div className="text-xs text-muted-foreground">
                从上方列表选择一个工具，输入 JSON 参数后调用
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// ============================================================
// 右：MCP 服务器管理
// ============================================================
function MCPPanel() {
  const mcpServers = useSkillsStore((s) => s.mcpServers);
  const mcpLoading = useSkillsStore((s) => s.mcpLoading);
  const spawnMCP = useSkillsStore((s) => s.spawnMCP);
  const enableServer = useSkillsStore((s) => s.enableServer);
  const disableServer = useSkillsStore((s) => s.disableServer);
  const refreshMCP = useSkillsStore((s) => s.refreshMCP);

  // Spawn 表单
  const [spawnName, setSpawnName] = useState("");
  const [spawnCmd, setSpawnCmd] = useState("");
  const [spawnArgs, setSpawnArgs] = useState("");
  const [spawnEnv, setSpawnEnv] = useState("");

  // 启用服务器表单
  const [enableName, setEnableName] = useState("");
  const [enableCmd, setEnableCmd] = useState("");
  const [enableArgs, setEnableArgs] = useState("");
  const [enableEnv, setEnableEnv] = useState("");
  const [enableUrl, setEnableUrl] = useState("");

  const handleSpawn = async () => {
    if (!spawnName.trim() || !spawnCmd.trim()) return;
    const args = parseLines(spawnArgs);
    const env = parseLines(spawnEnv);
    await spawnMCP(spawnName.trim(), spawnCmd.trim(), args, env);
    setSpawnName("");
    setSpawnCmd("");
    setSpawnArgs("");
    setSpawnEnv("");
  };

  const handleEnable = async () => {
    if (!enableName.trim()) return;
    const cfg: MCPServerConfig = {
      command: enableCmd.trim(),
      args: parseLines(enableArgs),
      env: parseLines(enableEnv),
      url: enableUrl.trim(),
      enabled: true,
    };
    await enableServer(enableName.trim(), cfg);
    setEnableName("");
    setEnableCmd("");
    setEnableArgs("");
    setEnableEnv("");
    setEnableUrl("");
  };

  return (
    <div className="flex h-full flex-col">
      <div className="shrink-0 border-b border-[var(--shell-seam)] px-3 py-2.5">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            <Server size={11} className="text-primary" />
            MCP 服务器
          </div>
          <Button
            variant="ghost"
            size="icon"
            className="h-5 w-5"
            onClick={() => void refreshMCP()}
            title="刷新"
          >
            <RefreshCw size={10} className={cn(mcpLoading && "animate-spin")} />
          </Button>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto scrollbar-dark">
        {/* 已启用列表 */}
        <div className="space-y-2 px-3 py-3">
          <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            已启用（{mcpServers.length}）
          </div>
          {mcpServers.length === 0 ? (
            <EmptyMini
              loading={mcpLoading}
              icon={Server}
              title="暂无 MCP 服务器"
              hint="使用下方表单启用或 Spawn"
            />
          ) : (
            <ul className="space-y-1">
              {mcpServers.map((name) => (
                <li
                  key={name}
                  className="flex items-center gap-2 rounded-md border border-[var(--shell-seam)] bg-background px-2.5 py-1.5"
                >
                  <Power size={11} className="shrink-0 text-success" />
                  <span className="min-w-0 flex-1 truncate font-mono text-[11px] text-foreground">
                    {name}
                  </span>
                  <button
                    type="button"
                    onClick={() => void disableServer(name)}
                    disabled={mcpLoading}
                    title="禁用"
                    className="shrink-0 rounded p-1 text-muted-foreground/60 transition-colors hover:bg-destructive/15 hover:text-destructive"
                  >
                    <Trash2 size={11} />
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* Spawn MCP 表单 */}
        <div className="space-y-2 border-t border-[var(--shell-seam)] px-3 py-3">
          <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            <Terminal size={11} className="text-primary" />
            Spawn MCP 进程
          </div>
          <MCPInput
            label="名称"
            value={spawnName}
            onChange={setSpawnName}
            placeholder="my-mcp"
          />
          <MCPInput
            label="命令"
            value={spawnCmd}
            onChange={setSpawnCmd}
            placeholder="npx -y @modelcontextprotocol/server-filesystem"
            mono
          />
          <MCPInput
            label="args（每行一个）"
            value={spawnArgs}
            onChange={setSpawnArgs}
            placeholder={"/tmp\ndocuments"}
            mono
          />
          <MCPInput
            label="env（KEY=VALUE，每行一个）"
            value={spawnEnv}
            onChange={setSpawnEnv}
            placeholder={"API_KEY=sk-xxx\nNODE_ENV=production"}
            mono
          />
          <Button
            size="sm"
            className="h-7 w-full"
            onClick={handleSpawn}
            disabled={mcpLoading || !spawnName.trim() || !spawnCmd.trim()}
          >
            {mcpLoading ? (
              <Loader2 size={12} className="animate-spin" />
            ) : (
              <Plus size={12} />
            )}
            Spawn
          </Button>
        </div>

        {/* 启用新服务器表单 */}
        <div className="space-y-2 border-t border-[var(--shell-seam)] px-3 py-3">
          <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
            <Power size={11} className="text-primary" />
            启用 MCP 服务器
          </div>
          <MCPInput
            label="名称"
            value={enableName}
            onChange={setEnableName}
            placeholder="filesystem"
          />
          <MCPInput
            label="command"
            value={enableCmd}
            onChange={setEnableCmd}
            placeholder="npx"
            mono
          />
          <MCPInput
            label="args（每行一个）"
            value={enableArgs}
            onChange={setEnableArgs}
            placeholder="-y\n@modelcontextprotocol/server-filesystem"
            mono
          />
          <MCPInput
            label="env（KEY=VALUE，每行一个）"
            value={enableEnv}
            onChange={setEnableEnv}
            placeholder="API_KEY=sk-xxx"
            mono
          />
          <MCPInput
            label="url（可选，SSE/HTTP 模式）"
            value={enableUrl}
            onChange={setEnableUrl}
            placeholder="http://localhost:8080/sse"
            mono
          />
          <Button
            size="sm"
            className="h-7 w-full"
            onClick={handleEnable}
            disabled={mcpLoading || !enableName.trim()}
          >
            {mcpLoading ? (
              <Loader2 size={12} className="animate-spin" />
            ) : (
              <Power size={12} />
            )}
            启用
          </Button>
        </div>
      </div>
    </div>
  );
}

// ============================================================
// 内部辅助组件
// ============================================================
function MCPInput({
  label,
  value,
  onChange,
  placeholder,
  mono,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  mono?: boolean;
}) {
  return (
    <label className="block">
      <span className="mb-1 block text-[10px] text-muted-foreground/70">
        {label}
      </span>
      {mono ? (
        <Textarea
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          rows={2}
          className="resize-none font-mono text-[11px]"
        />
      ) : (
        <Input
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          className="h-8 font-mono text-[11px]"
        />
      )}
    </label>
  );
}

function EmptyMini({
  loading,
  icon: Icon,
  title,
  hint,
}: {
  loading: boolean;
  icon: typeof Inbox;
  title: string;
  hint: string;
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 px-4 py-8 text-center">
      <div className="flex h-9 w-9 items-center justify-center rounded-full bg-muted text-muted-foreground">
        {loading ? (
          <Loader2 size={14} className="animate-spin" />
        ) : (
          <Icon size={14} />
        )}
      </div>
      <div className="space-y-0.5">
        <div className="text-[12px] font-medium text-foreground">
          {loading ? "加载中…" : title}
        </div>
        <div className="text-[10px] text-muted-foreground">{hint}</div>
      </div>
    </div>
  );
}

/** 把多行字符串解析为字符串数组（空行过滤） */
function parseLines(s: string): string[] {
  return s
    .split(/\r?\n/)
    .map((l) => l.trim())
    .filter(Boolean);
}
