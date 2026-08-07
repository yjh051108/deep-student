import { useEffect, useState } from "react";

export type Hotkey = {
  combo: string; // e.g. "Ctrl+N", "Ctrl+Shift+P"
  description?: string;
  handler: (e: KeyboardEvent) => void;
};

function eventToCombo(e: KeyboardEvent): string {
  const parts: string[] = [];
  if (e.ctrlKey || e.metaKey) parts.push("Ctrl");
  if (e.shiftKey) parts.push("Shift");
  if (e.altKey) parts.push("Alt");
  const key = e.key.length === 1 ? e.key.toUpperCase() : e.key;
  parts.push(key);
  return parts.join("+");
}

function normalize(combo: string): string {
  return combo
    .split("+")
    .map((p) => p.trim())
    .map((p) =>
      p.toLowerCase() === "cmd" || p.toLowerCase() === "meta" ? "Ctrl" : p
    )
    .map((p) => (p.length === 1 ? p.toUpperCase() : p))
    .join("+");
}

export function useHotkeys(hotkeys: Hotkey[], enabled = true) {
  const [last, setLast] = useState<string | null>(null);

  useEffect(() => {
    if (!enabled) return;
    const map = new Map<string, Hotkey["handler"]>();
    for (const h of hotkeys) {
      map.set(normalize(h.combo), h.handler);
    }

    const onKey = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement | null;
      if (target) {
        const tag = target.tagName;
        if (
          tag === "INPUT" ||
          tag === "TEXTAREA" ||
          target.isContentEditable
        ) {
          // 仅在文本框里允许纯字符热键；带 Ctrl/Alt 的仍触发
          if (!(e.ctrlKey || e.metaKey || e.altKey)) return;
        }
      }
      const combo = eventToCombo(e);
      const handler = map.get(normalize(combo));
      if (handler) {
        e.preventDefault();
        setLast(combo);
        handler(e);
      }
    };

    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [hotkeys, enabled]);

  return { last };
}
