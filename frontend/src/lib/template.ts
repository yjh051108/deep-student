// Template 模板管理前端封装
// ------------------------------------------------------------
// 对接后端 templatemgr（Template* 方法）。

import { callWails } from "@/lib/wails";

/** 模板 —— 与后端 templatemgr.Template 对齐 */
export interface Template {
  id: string;
  name: string;
  front: string;
  back: string;
  style?: string;
  css?: string;
  isBuiltin: boolean;
  preview?: string;
  sortOrder: number;
  createdAt: string;
  updatedAt: string;
}

export const templateApi = {
  list: () => callWails<Template[]>("TemplateList"),
  get: (id: string) => callWails<Template>("TemplateGet", id),
  create: (p: {
    name: string;
    front: string;
    back: string;
    style?: string;
    css?: string;
    preview?: string;
  }) => callWails<Template>("TemplateCreate", p),
  update: (p: {
    id: string;
    name?: string | null;
    front?: string | null;
    back?: string | null;
    style?: string | null;
    css?: string | null;
    preview?: string | null;
  }) => callWails<Template>("TemplateUpdate", p),
  remove: (id: string) => callWails<void>("TemplateDelete", id),
  export: (id: string) => callWails<number[]>("TemplateExport", id),
  import: (data: number[]) => callWails<Template>("TemplateImport", data),
  importBulk: (data: number[]) => callWails<[number, number]>("TemplateImportBulk", data),
  importBuiltins: () => callWails<number>("TemplateImportBuiltins"),
  setDefault: (id: string) => callWails<void>("TemplateSetDefault", id),
  getDefaultID: () => callWails<string>("TemplateGetDefaultID"),
};
