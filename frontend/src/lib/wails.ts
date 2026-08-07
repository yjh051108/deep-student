// Wails 绑定在 `wails dev` 期间被注入到 window.go.deepstudent.App。
// Go 入口包名为 deepstudent（cmd/deepstudent/main.go → package deepstudent），
// 因此 Wails 生成的绑定命名空间是 deepstudent，而非 main。
//
// 调用处统一通过 callWails<T> 走 runtime 检测。

declare global {
  interface Window {
    go?: {
      deepstudent?: {
        App?: Record<string, (...args: unknown[]) => unknown>;
      };
      // 兼容：部分旧代码可能仍引用 main 命名空间
      main?: {
        App?: Record<string, (...args: unknown[]) => unknown>;
      };
    };
  }
}

/** 获取 Wails 绑定对象 —— 自动适配 deepstudent / main 命名空间 */
function getWailsApp(): Record<string, (...args: unknown[]) => unknown> | null {
  return window.go?.deepstudent?.App ?? window.go?.main?.App ?? null;
}

export async function callWails<T = unknown>(
  method: string,
  ...args: unknown[]
): Promise<T | null> {
  const app = getWailsApp();
  if (!app || typeof app[method] !== "function") {
    // eslint-disable-next-line no-console
    console.warn(`[wails] binding not available: ${method}`);
    return null;
  }
  try {
    const result = await (app[method] as (...a: unknown[]) => unknown)(
      ...args
    );
    return result as T;
  } catch (err) {
    // eslint-disable-next-line no-console
    console.warn(`[wails] ${method} failed`, err);
    return null;
  }
}

export function wailsAvailable(): boolean {
  return Boolean(getWailsApp());
}
