// 工具函数集合
// 对照原版 src/lib/utils.ts，提供 cn 类名合并与常用工具

import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * cn —— 类名合并工具
 *
 * 结合 clsx（条件类名）与 tailwind-merge（解决 Tailwind 类冲突），
 * 作为所有 shad 组件的统一类名拼接入口。
 *
 * @example
 *   cn("px-2 py-1", isActive && "bg-primary", "px-4")  // "py-1 bg-primary px-4"
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

/**
 * delay —— 异步延迟（用于流式模拟、防抖等）
 */
export function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * formatTime —— 将时间戳格式化为 HH:mm
 */
export function formatTime(ts: number): string {
  const d = new Date(ts);
  const hh = String(d.getHours()).padStart(2, "0");
  const mm = String(d.getMinutes()).padStart(2, "0");
  return `${hh}:${mm}`;
}

/**
 * formatDate —— 将时间戳格式化为 YYYY-MM-DD
 */
export function formatDate(ts: number): string {
  const d = new Date(ts);
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

/**
 * relativeTime —— 相对时间描述（刚刚 / N 分钟前 / N 小时前 / N 天前）
 */
export function relativeTime(ts: number): string {
  const now = Date.now();
  const diff = Math.max(0, now - ts);
  const sec = Math.floor(diff / 1000);
  const min = Math.floor(sec / 60);
  const hr = Math.floor(min / 60);
  const day = Math.floor(hr / 24);

  if (sec < 60) return "刚刚";
  if (min < 60) return `${min} 分钟前`;
  if (hr < 24) return `${hr} 小时前`;
  if (day < 30) return `${day} 天前`;
  return formatDate(ts);
}

/**
 * truncate —— 截断字符串并附加省略号
 */
export function truncate(s: string, max: number): string {
  if (s.length <= max) return s;
  return s.slice(0, max - 1) + "…";
}

/**
 * uid —— 生成短随机 id（用于客户端临时 id）
 */
export function uid(prefix = "id"): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 10)}${Date.now().toString(36)}`;
}
