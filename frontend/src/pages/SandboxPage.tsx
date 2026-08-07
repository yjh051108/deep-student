// SandboxPage —— 代码沙盒工作台
// ------------------------------------------------------------
// 支持 HTML/CSS/JS 的 iframe 沙盒预览（safe-preview 模式），
// 纯前端实现（对齐 Rust 原版 src/features/sandbox 的核心能力）。

import { useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/Button";
import { Textarea } from "@/components/ui/Textarea";
import { Badge } from "@/components/ui/Badge";
import { Play, RotateCcw, Monitor, Code2, Loader2 } from "lucide-react";

type Mode = "html" | "css" | "js" | "full";

const DEFAULT_HTML = `<!DOCTYPE html>
<html>
<head><style>
  body { font-family: system-ui; display: flex; align-items: center;
         justify-content: center; height: 100vh; margin: 0; background: #f5f5f5; }
  .card { background: white; padding: 32px 48px; border-radius: 12px;
          box-shadow: 0 8px 24px rgba(0,0,0,.1); text-align: center; }
  h1 { color: #6d28d9; }
</style></head>
<body>
  <div class="card">
    <h1>Hello, DeepStudent 🎉</h1>
    <p>这是沙盒预览。修改左侧代码试试。</p>
  </div>
</body>
</html>`;

const DEFAULT_CSS = `/* 输入自定义 CSS，将注入到预览 */
.highlight { color: #6d28d9; font-weight: bold; }`;

const DEFAULT_JS = `// 输入自定义 JS（预览页加载后执行）
console.log("sandbox ready");`;

export function SandboxPage() {
  const [mode, setMode] = useState<Mode>("html");
  const [html, setHtml] = useState(DEFAULT_HTML);
  const [css, setCss] = useState(DEFAULT_CSS);
  const [js, setJs] = useState(DEFAULT_JS);
  const [src, setSrc] = useState("");
  const [running, setRunning] = useState(false);

  const run = () => {
    setRunning(true);
    const doc = html.includes("<html")
      ? html
      : `<!DOCTYPE html><html><head><style>${css}</style></head><body>${html}<script>${js}<\/script></body></html>`;
    // 延迟以显示加载动画
    setTimeout(() => {
      setSrc(doc);
      setRunning(false);
    }, 120);
  };

  const reset = () => {
    setHtml(DEFAULT_HTML);
    setCss(DEFAULT_CSS);
    setJs(DEFAULT_JS);
    setSrc("");
  };

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
        e.preventDefault();
        run();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [html, css, js]);

  const codeValue = useMemo(() => {
    switch (mode) {
      case "css":
        return css;
      case "js":
        return js;
      default:
        return html;
    }
  }, [mode, html, css, js]);

  const onCodeChange = (v: string) => {
    if (mode === "css") setCss(v);
    else if (mode === "js") setJs(v);
    else setHtml(v);
  };

  return (
    <div className="flex h-full w-full min-h-0 flex-col bg-background">
      {/* 头部 */}
      <div className="flex shrink-0 items-center gap-2 border-b border-border bg-card px-4 py-2.5">
        <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary/15 text-primary">
          <Code2 size={13} />
        </div>
        <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
          代码沙盒
        </span>
        <div className="ml-auto flex items-center gap-2">
          <Badge variant="secondary">safe-preview</Badge>
          <Button size="sm" variant="ghost" className="h-7" onClick={reset}>
            <RotateCcw size={12} className="mr-1" />
            重置
          </Button>
          <Button size="sm" className="h-7" onClick={run} disabled={running}>
            {running ? (
              <Loader2 size={12} className="mr-1 animate-spin" />
            ) : (
              <Play size={12} className="mr-1" />
            )}
            运行
          </Button>
        </div>
      </div>

      <div className="flex min-h-0 flex-1">
        {/* 左：代码编辑 */}
        <section className="flex w-[46%] shrink-0 flex-col border-r border-border">
          {/* 模式切换 */}
          <div className="flex shrink-0 gap-1 border-b border-border px-3 py-1.5">
            {(
              [
                { key: "html", label: "HTML" },
                { key: "css", label: "CSS" },
                { key: "js", label: "JS" },
              ] as { key: Mode; label: string }[]
            ).map((m) => (
              <button
                key={m.key}
                onClick={() => setMode(m.key)}
                className={`rounded-md px-2.5 py-1 text-[11px] transition-colors ${
                  mode === m.key
                    ? "bg-primary/15 font-medium text-primary"
                    : "text-muted-foreground hover:bg-accent"
                }`}
              >
                {m.label}
              </button>
            ))}
          </div>
          <div className="min-h-0 flex-1">
            <Textarea
              value={codeValue}
              onChange={(e) => onCodeChange(e.target.value)}
              spellCheck={false}
              className="h-full w-full resize-none rounded-none border-0 bg-background p-3 font-mono text-[12px] leading-relaxed shadow-none focus-visible:ring-0"
            />
          </div>
        </section>

        {/* 右：预览 */}
        <section className="flex min-w-0 flex-1 flex-col bg-muted/30">
          <div className="flex shrink-0 items-center gap-2 border-b border-border px-3 py-1.5">
            <Monitor size={11} className="text-muted-foreground" />
            <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              预览
            </span>
          </div>
          <div className="min-h-0 flex-1 p-2">
            {src ? (
              <iframe
                title="sandbox-preview"
                srcDoc={src}
                sandbox="allow-scripts"
                className="h-full w-full rounded-md border border-border bg-white"
              />
            ) : (
              <div className="flex h-full flex-col items-center justify-center gap-2 text-muted-foreground">
                <Monitor size={28} />
                <span className="text-[12px]">点击「运行」预览效果</span>
              </div>
            )}
          </div>
        </section>
      </div>
    </div>
  );
}
