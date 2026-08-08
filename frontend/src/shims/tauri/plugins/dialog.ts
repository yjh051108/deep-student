// ============================================================
// Tauri → Wails 适配层：@tauri-apps/plugin-dialog
// ------------------------------------------------------------
// 文件对话框优先走 Wails 侧 Go 暴露的对话框能力（若有），
// 否则回退到浏览器原生 <input type="file"> / window.confirm。
// ============================================================

export interface OpenDialogOptions {
  directory?: boolean;
  multiple?: boolean;
  defaultPath?: string;
  filters?: Array<{ name: string; extensions: string[] }>;
  title?: string;
}

export interface SaveDialogOptions {
  title?: string;
  defaultPath?: string;
  filters?: Array<{ name: string; extensions: string[] }>;
}

export interface MessageDialogOptions {
  title?: string;
  kind?: 'info' | 'warning' | 'error';
  okLabel?: string;
  cancelLabel?: string;
}

function pickFileViaInput(directory: boolean, multiple: boolean, filters?: OpenDialogOptions['filters']): Promise<string | string[] | null> {
  return new Promise((resolve) => {
    const input = document.createElement('input');
    input.type = 'file';
    if (directory) {
      // 浏览器不支持目录选择（webkitdirectory 可作降级）
      (input as HTMLInputElement & { webkitdirectory?: boolean }).webkitdirectory = true;
    }
    input.multiple = multiple;
    if (filters && filters.length > 0) {
      const exts = filters.flatMap((f) => f.extensions);
      input.accept = exts.map((e) => `.${e.replace(/^\./, '')}`).join(',');
    }
    input.style.display = 'none';
    document.body.appendChild(input);
    input.onchange = () => {
      const files = Array.from(input.files ?? []);
      document.body.removeChild(input);
      if (files.length === 0) return resolve(null);
      const paths = files.map((f) => f.name);
      resolve(multiple ? paths : paths[0] ?? null);
    };
    input.oncancel = () => {
      document.body.removeChild(input);
      resolve(null);
    };
    input.click();
  });
}

/** 打开文件/目录选择对话框；返回选中路径（multiple 时返回数组），取消返回 null */
export async function open(options: OpenDialogOptions = {}): Promise<string | string[] | null> {
  const { directory = false, multiple = false, filters } = options;
  return pickFileViaInput(directory, multiple, filters);
}

/** 保存对话框；浏览器环境回退为输入文件名 */
export async function save(options: SaveDialogOptions = {}): Promise<string | null> {
  const name = window.prompt('保存为（文件对话框不可用）', options.defaultPath ?? 'untitled');
  return name ?? null;
}

/** 消息对话框 */
export async function message(message: string, options: MessageDialogOptions = {}): Promise<void> {
  void options;
  window.alert(message);
}

/** 确认对话框（确认返回 true） */
export async function confirm(message: string, options: MessageDialogOptions = {}): Promise<boolean> {
  void options;
  return window.confirm(message);
}

/** 询问对话框 */
export async function ask(message: string, options: MessageDialogOptions = {}): Promise<boolean> {
  void options;
  return window.confirm(message);
}
