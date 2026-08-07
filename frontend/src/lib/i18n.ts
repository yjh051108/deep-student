// 轻量 i18n：en-US / zh-CN 双语言，无外部依赖。
// 通过 zustand 管理语言状态；t(key) 按当前语言取词，缺失回退 en-US 再回退 key。

import { create } from "zustand";

export type Lang = "en-US" | "zh-CN";

/** 扁平 key → 文案 */
type Dict = Record<string, string>;

const en: Dict = {
  // 通用
  "common.save": "Save",
  "common.cancel": "Cancel",
  "common.delete": "Delete",
  "common.search": "Search",
  "common.add": "Add",
  "common.refresh": "Refresh",
  "common.loading": "Loading…",
  "common.confirm": "Confirm",
  "common.edit": "Edit",
  "common.copy": "Copy",
  "common.export": "Export",
  "common.import": "Import",

  // 侧边栏
  "sidebar.hub": "Hub",
  "sidebar.chat": "Chat",
  "sidebar.mindmap": "Mindmap",
  "sidebar.notes": "Notes",
  "sidebar.todo": "Todo",
  "sidebar.pomodoro": "Pomodoro",
  "sidebar.qbank": "QBank",
  "sidebar.anki": "Anki",
  "sidebar.reader": "Reader",
  "sidebar.translate": "Translate",
  "sidebar.essay": "Essay",
  "sidebar.memory": "Memory",
  "sidebar.research": "Research",
  "sidebar.paper": "Paper",
  "sidebar.skills": "Skills",
  "sidebar.governance": "Governance",
  "sidebar.settings": "Settings",
  "sidebar.llmUsage": "LLM Usage",

  // 待办
  "todo.today": "Today",
  "todo.overdue": "Overdue",
  "todo.upcoming": "Next 7 days",
  "todo.all": "All todos",
  "todo.trash": "Trash",
  "todo.lists": "Lists",
  "todo.newList": "New list…",
  "todo.addItem": "Add a todo, press Enter…",
  "todo.aiBreakdown": "AI breakdown",
  "todo.inbox": "Inbox",

  // 番茄钟
  "pomodoro.focus": "Focus",
  "pomodoro.shortBreak": "Short break",
  "pomodoro.longBreak": "Long break",
  "pomodoro.start": "Start",
  "pomodoro.pause": "Pause",
  "pomodoro.resume": "Resume",
  "pomodoro.reset": "Reset",
  "pomodoro.interrupt": "Interrupt",
  "pomodoro.todayStats": "Today stats",
  "pomodoro.focusMinutes": "Focus",
  "pomodoro.completed": "Completed",
  "pomodoro.interrupted": "Interrupted",

  // 设置
  "settings.language": "Language",
  "settings.vault": "Vault directory",
  "settings.providers": "LLM Providers",
  "settings.theme": "Theme",
};

const zh: Dict = {
  "common.save": "保存",
  "common.cancel": "取消",
  "common.delete": "删除",
  "common.search": "搜索",
  "common.add": "添加",
  "common.refresh": "刷新",
  "common.loading": "加载中…",
  "common.confirm": "确认",
  "common.edit": "编辑",
  "common.copy": "复制",
  "common.export": "导出",
  "common.import": "导入",

  "sidebar.hub": "Hub · 资源中枢",
  "sidebar.chat": "Chat · 对话",
  "sidebar.mindmap": "Mindmap · 导图",
  "sidebar.notes": "Notes · 笔记",
  "sidebar.todo": "Todo · 待办",
  "sidebar.pomodoro": "Pomodoro · 番茄钟",
  "sidebar.qbank": "QBank · 题库",
  "sidebar.anki": "Anki · 卡片",
  "sidebar.reader": "Reader · 阅读",
  "sidebar.translate": "Translate · 翻译",
  "sidebar.essay": "Essay · 作文",
  "sidebar.memory": "Memory · 记忆",
  "sidebar.research": "Research · 研究",
  "sidebar.paper": "Paper · 论文",
  "sidebar.skills": "Skills · 技能",
  "sidebar.governance": "Governance · 治理",
  "sidebar.settings": "Settings · 设置",
  "sidebar.llmUsage": "LLM Usage · 用量",

  "todo.today": "今日",
  "todo.overdue": "逾期",
  "todo.upcoming": "未来 7 天",
  "todo.all": "全部待办",
  "todo.trash": "回收站",
  "todo.lists": "列表",
  "todo.newList": "新列表…",
  "todo.addItem": "添加待办，回车确认…",
  "todo.aiBreakdown": "AI 拆解",
  "todo.inbox": "收件箱",

  "pomodoro.focus": "专注",
  "pomodoro.shortBreak": "短休",
  "pomodoro.longBreak": "长休",
  "pomodoro.start": "开始",
  "pomodoro.pause": "暂停",
  "pomodoro.resume": "继续",
  "pomodoro.reset": "重置",
  "pomodoro.interrupt": "中断",
  "pomodoro.todayStats": "今日统计",
  "pomodoro.focusMinutes": "专注",
  "pomodoro.completed": "完成",
  "pomodoro.interrupted": "中断",

  "settings.language": "语言",
  "settings.vault": "知识库目录",
  "settings.providers": "LLM 供应商",
  "settings.theme": "主题",
};

const dicts: Record<Lang, Dict> = { "en-US": en, "zh-CN": zh };

interface I18nState {
  lang: Lang;
  setLang: (l: Lang) => void;
  t: (key: string) => string;
}

export const useI18n = create<I18nState>((set, get) => ({
  lang: (localStorage.getItem("ds.lang") as Lang) || "zh-CN",
  setLang: (l) => {
    localStorage.setItem("ds.lang", l);
    set({ lang: l });
  },
  t: (key) => {
    const { lang } = get();
    return dicts[lang][key] ?? dicts["en-US"][key] ?? key;
  },
}));

/** 组件内便捷 hook */
export function useT() {
  return useI18n((s) => s.t);
}
