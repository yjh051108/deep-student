// FSRS 前端类型与 Wails 封装
// ------------------------------------------------------------
// 与后端 internal/fsrs 对齐。

import { callWails } from "@/lib/wails";

/** 卡片状态 —— 与后端 fsrs.CardState 对齐 */
export interface FSRSCard {
  cardId: string;
  deck: string;
  front: string;
  back: string;
  stability: number;
  difficulty: number;
  dueAt: string;
  lastReview?: string | null;
  reps: number;
  lapses: number;
  state: "new" | "learning" | "review" | "relearning";
  createdAt: string;
  updatedAt: string;
}

/** 复习记录 —— 与后端 fsrs.ReviewLog 对齐 */
export interface FSRSReviewLog {
  id: string;
  cardId: string;
  rating: 1 | 2 | 3 | 4;
  reviewAt: string;
  stability: number;
  difficulty: number;
  prevDue: string;
}

/** 牌组统计 —— 与后端 fsrs.DeckStat 对齐 */
export interface FSRSDeckStat {
  deck: string;
  total: number;
  due: number;
}

export const fsrsApi = {
  addCards: (deck: string, cards: { front: string; back: string }[]) =>
    callWails<FSRSCard[]>("FSRSAddCards", deck, cards),
  due: (deck = "", limit = 50) =>
    callWails<FSRSCard[]>("FSRSDue", deck, limit),
  all: (deck = "", limit = 200) =>
    callWails<FSRSCard[]>("FSRSAll", deck, limit),
  review: (cardId: string, rating: number) =>
    callWails<FSRSCard>("FSRSReview", cardId, rating),
  get: (cardId: string) => callWails<FSRSCard>("FSRSGet", cardId),
  reviewLogs: (cardId: string, limit = 20) =>
    callWails<FSRSReviewLog[]>("FSRSReviewLogs", cardId, limit),
  remove: (cardId: string) => callWails<void>("FSRSDelete", cardId),
  dueCount: () => callWails<number>("FSRSDueCount"),
  deckStats: () => callWails<FSRSDeckStat[]>("FSRSDeckStats"),
};
