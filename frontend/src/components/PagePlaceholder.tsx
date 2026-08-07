import { ReactNode } from "react";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";

export interface PagePlaceholderProps {
  title: string;
  description: string;
  buttonLabel?: string;
  onTry?: () => void;
  buttonDisabled?: boolean;
  /** 调用进行中：禁用按钮并切换按钮文案 */
  loading?: boolean;
  /** 错误提示（红色横幅） */
  error?: string | null;
  result?: string;
  children?: ReactNode;
}

// 阶段 1 临时占位组件：使用新设计系统
// 阶段 3+ 将由各页面的真实实现替换
export function PagePlaceholder({
  title,
  description,
  buttonLabel = "试一下",
  onTry,
  buttonDisabled,
  loading,
  error,
  result,
  children,
}: PagePlaceholderProps) {
  const disabled = buttonDisabled || loading;
  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <header>
        <h1 className="text-2xl font-semibold text-foreground">{title}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{description}</p>
      </header>
      <Card className="p-5">
        <div className="flex items-center justify-between">
          <p className="text-sm text-foreground">
            点击按钮可触发对应的 Wails 后端方法（无绑定时仅 console.warn）。
          </p>
          <Button onClick={onTry} disabled={disabled}>
            {loading ? "加载中..." : buttonLabel}
          </Button>
        </div>
        {error && (
          <div className="mt-4 rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
            {error}
          </div>
        )}
        {result !== undefined && !error && (
          <pre className="mt-4 max-h-64 overflow-auto rounded-md bg-muted p-3 text-xs text-foreground">
            {result || "(空响应)"}
          </pre>
        )}
        {children && <div className="mt-4">{children}</div>}
      </Card>
    </div>
  );
}
