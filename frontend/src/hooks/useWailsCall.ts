import { useCallback, useState } from "react";
import { callWails } from "@/lib/wails";

/**
 * useWailsCall —— 统一封装"占位页试一下"场景的 Wails 调用模式。
 *
 * - result: 展示在 <PagePlaceholder> 的结果字符串
 * - loading: 调用进行中，用于禁用按钮 / 切换文案
 * - error: 当绑定不可用或调用失败时的错误消息（与 result 同步呈现）
 * - execute: 触发调用的异步函数，直接传给 PagePlaceholder 的 onTry
 */
export interface UseWailsCallOptions<T> {
  /** Wails 绑定方法名，例如 "ChatSend" */
  methodName: string;
  /** 调用参数列表，按顺序传给 callWails */
  args?: unknown[];
  /** 将返回值格式化为展示字符串；默认对字符串原样返回、其它 JSON.stringify */
  transformResult?: (result: T) => string;
  /** 绑定不可用 / 调用抛错时的统一兜底文案 */
  fallbackMessage?: string;
  /** 自定义错误处理钩子：返回最终的展示字符串 */
  onError?: (err: unknown) => string;
}

export interface UseWailsCallReturn {
  result: string;
  loading: boolean;
  error: string | null;
  execute: () => Promise<void>;
}

const DEFAULT_FALLBACK = "(绑定不可用)";

function defaultTransform<T>(result: T): string {
  if (typeof result === "string") return result;
  return JSON.stringify(result ?? null, null, 2);
}

export function useWailsCall<T = unknown>(
  options: UseWailsCallOptions<T>
): UseWailsCallReturn {
  const {
    methodName,
    args = [],
    transformResult = defaultTransform,
    fallbackMessage = DEFAULT_FALLBACK,
    onError,
  } = options;

  const [result, setResult] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const execute = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await callWails<T>(methodName, ...args);
      if (response === null || response === undefined) {
        const msg = onError
          ? onError(new Error("binding unavailable"))
          : fallbackMessage;
        setError(msg);
        setResult(msg);
        return;
      }
      setResult(transformResult(response));
    } catch (err) {
      const msg = onError ? onError(err) : `${fallbackMessage}: ${String(err)}`;
      setError(msg);
      setResult(msg);
    } finally {
      setLoading(false);
    }
  }, [methodName, args, transformResult, fallbackMessage, onError]);

  return { result, loading, error, execute };
}
