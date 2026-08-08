// ============================================================
// Tauri → Wails 适配层：@tauri-apps/api/core
// ------------------------------------------------------------
// 原版前端（Rust/Tauri）通过 invoke(cmd, payload) 调用后端命令；
// Go 版（Wails v2）将方法绑定到 window.go.deepstudent.App（PascalCase）。
// 本 shim 负责：
//   - 命令名映射（snake_case → Go 方法名，支持显式桥接表）
//   - 参数形态适配（原版单对象 payload → Go 位置参数）
//   - 错误透传（Wails rejection → 原版 catch 语义）
// ============================================================

declare global {
  interface Window {
    go?: {
      deepstudent?: {
        App?: Record<string, (...args: unknown[]) => unknown>;
      };
      main?: {
        App?: Record<string, (...args: unknown[]) => unknown>;
      };
    };
  }
}

// ------------------------------------------------------------
// Wails 绑定获取（与 lib/wails.ts 相同的探测逻辑，独立实现避免循环依赖）
// ------------------------------------------------------------
function getWailsApp(): Record<string, (...args: unknown[]) => unknown> | null {
  return window.go?.deepstudent?.App ?? window.go?.main?.App ?? null;
}

function wailsAvailable(): boolean {
  return Boolean(getWailsApp());
}

// ------------------------------------------------------------
// 命令桥接表
// ------------------------------------------------------------
export interface CommandBridge {
  /** Go 端 Wails 方法名（PascalCase） */
  method: string;
  /** 把原版单对象 payload 转换为 Go 方法位置参数；缺省时原样传单参数 */
  args?: (payload: Record<string, unknown>) => unknown[];
  /** 返回值适配（默认原样透传） */
  result?: (raw: unknown) => unknown;
}

// 显式桥接表：命令名 → Go 方法。
// 缺省回退：snake_case → PascalCase 直接匹配；再缺省则单对象透传。
const BRIDGES: Record<string, CommandBridge> = {
  // —— todo ——
  todo_list_items: { method: 'TodoListItems', args: (p) => [p.listId ?? 'inbox', p.filter ?? 'all'] },
  todo_create_item: { method: 'TodoCreateItem', args: (p) => [p.listId ?? 'inbox', p.title, p.notes, p.due, p.priority, p.tags, p.parentId] },
  todo_update_item: { method: 'TodoUpdateItem', args: (p) => [p.params ?? p] },
  todo_toggle_item: { method: 'TodoToggleItem', args: (p) => [p.itemId ?? p.id] },
  todo_delete_item: { method: 'TodoDeleteItem', args: (p) => [p.itemId ?? p.id] },
  todo_trash_item: { method: 'TodoTrashItem', args: (p) => [p.itemId ?? p.id] },
  todo_restore_item: { method: 'TodoRestoreItem', args: (p) => [p.itemId ?? p.id] },
  todo_list_lists: { method: 'TodoListLists' },
  todo_create_list: { method: 'TodoCreateList', args: (p) => [p.name] },
  todo_rename_list: { method: 'TodoRenameList', args: (p) => [p.listId, p.name] },
  todo_delete_list: { method: 'TodoDeleteList', args: (p) => [p.listId] },
  todo_list_trash: { method: 'TodoListTrash' },
  todo_empty_trash: { method: 'TodoEmptyTrash' },
  todo_ai_split: { method: 'TodoAISplit', args: (p) => [p.itemId, p.prompt] },

  // —— hub / vfs 资源 ——
  hub_import_resource: { method: 'HubImportResource', args: (p) => [p.typ ?? p.type, p.title, p.data, p.tags] },
  hub_list: { method: 'HubList', args: (p) => [p.typ ?? p.type ?? 'note'] },
  hub_search: { method: 'HubSearch', args: (p) => [p.typ ?? p.type, p.tag] },
  hub_get: { method: 'HubGet', args: (p) => [p.uri] },
  hub_delete: { method: 'HubDelete', args: (p) => [p.uri] },
  hub_continue_note: { method: 'HubContinueNote', args: (p) => [p.uri, p.prompt] },

  // —— chat v2 ——
  chat_v2_create_group: { method: 'ChatCreateGroup', args: (p) => [p.name] },
  chat_v2_create_session: { method: 'ChatCreateSession', args: (p) => [p.groupId ?? p.group_id ?? '', p.title ?? '', p.model ?? '', p.systemPrompt ?? p.system_prompt ?? ''] },
  chat_v2_branch: { method: 'ChatBranch', args: (p) => [p.sessionId ?? p.session_id, p.messageId ?? p.message_id, p.title] },
  chat_v2_send: { method: 'ChatSend', args: (p) => [p.sessionId, p.content, p.refs ?? [], p.deep ?? false] },
  chat_v2_send_message: { method: 'ChatV2Send', args: (p) => [p.request ?? p] },
  chat_v2_compare: { method: 'ChatCompare', args: (p) => [p.sessionId, p.leftMessageId, p.rightMessageId] },
  chat_v2_list_groups: { method: 'ChatV2ListGroups', args: (p) => [p.includeDeleted ?? false] },
  chat_v2_update_group: { method: 'ChatV2UpdateGroup', args: (p) => [p.group ?? p] },
  chat_v2_delete_group: { method: 'ChatV2DeleteGroup', args: (p) => [p.id ?? p.groupId ?? p.group_id] },
  chat_v2_restore_group: { method: 'ChatV2RestoreGroup', args: (p) => [p.id ?? p.groupId ?? p.group_id] },
  chat_v2_purge_group: { method: 'ChatV2PurgeGroup', args: (p) => [p.id ?? p.groupId ?? p.group_id] },
  chat_v2_list_sessions: { method: 'ChatV2ListSessions', args: (p) => [p.filter ?? p] },
  chat_v2_get_session: { method: 'ChatV2GetSession', args: (p) => [p.id ?? p.sessionId ?? p.session_id] },
  chat_v2_load_session: { method: 'ChatV2GetSession', args: (p) => [p.id ?? p.sessionId ?? p.session_id] },
  chat_v2_update_title: { method: 'ChatV2UpdateTitle', args: (p) => [p.id ?? p.sessionId ?? p.session_id, p.title] },
  chat_v2_pin: { method: 'ChatV2Pin', args: (p) => [p.id ?? p.sessionId ?? p.session_id, p.pinned ?? true] },
  chat_v2_soft_delete: { method: 'ChatV2SoftDelete', args: (p) => [p.id ?? p.sessionId ?? p.session_id] },
  chat_v2_delete_session: { method: 'ChatV2SoftDelete', args: (p) => [p.id ?? p.sessionId ?? p.session_id] },
  chat_v2_archive_session: { method: 'ChatV2SoftDelete', args: (p) => [p.id ?? p.sessionId ?? p.session_id] },
  chat_v2_restore: { method: 'ChatV2Restore', args: (p) => [p.id ?? p.sessionId ?? p.session_id] },
  chat_v2_purge: { method: 'ChatV2Purge', args: (p) => [p.id ?? p.sessionId ?? p.session_id] },
  chat_v2_update_tags: { method: 'ChatV2UpdateTags', args: (p) => [p.id ?? p.sessionId ?? p.session_id, p.tags ?? []] },
  chat_v2_search_content: { method: 'ChatV2Search', args: (p) => [p.query, p.limit ?? 50] },
  chat_v2_count_sessions: { method: 'ChatV2Count', args: (p) => [p.status ?? p] },
  chat_v2_delete_message: { method: 'ChatV2DeleteMessage', args: (p) => [p.sessionId ?? p.session_id, p.messageId ?? p.message_id] },
  chat_v2_register_tool: { method: 'ChatV2RegisterTool', args: (p) => [p.tool ?? p] },
  chat_v2_tools: { method: 'ChatV2Tools' },
  chat_v2_export: { method: 'ChatV2Export', args: (p) => [p.sessionId ?? p.session_id, p.format ?? 'md'] },

  // —— 设置 KV ——
  get_setting: { method: 'GetSetting', args: (p) => [p.key] },
  save_setting: { method: 'SaveSetting', args: (p) => [p.key, p.value] },
  delete_setting: { method: 'DeleteSetting', args: (p) => [p.key] },
  get_settings_by_prefix: { method: 'GetSettingsByPrefix', args: (p) => [p.prefix ?? ''] },

  // —— LLM 配置 ——
  get_api_configurations: { method: 'LLMCfgListApiConfigurations' },
  get_model_assignments: { method: 'LLMCfgGetAssignments' },
  save_model_assignments: { method: 'LLMCfgSaveAssignments', args: (p) => [p.assignments ?? p] },
  get_vendor_configs: { method: 'LLMCfgGetVendors' },
  get_model_profiles: { method: 'LLMCfgGetProfiles' },
  test_api_connection: { method: 'LLMCfgTestConnection', args: (p) => [p.profileId ?? p.profileID ?? p.id] },

  // —— notes ——
  notes_create: { method: 'NotesCreate', args: (p) => [p.params ?? p] },
  notes_get: { method: 'NotesGet', args: (p) => [p.id] },
  notes_update: { method: 'NotesUpdate', args: (p) => [p.params ?? p] },
  notes_list: { method: 'NotesList' },
  notes_list_meta: { method: 'NotesListMeta' },
  notes_move_to_trash: { method: 'NotesMoveToTrash', args: (p) => [p.id] },
  notes_restore: { method: 'NotesRestore', args: (p) => [p.id] },
  notes_hard_delete: { method: 'NotesHardDelete', args: (p) => [p.id] },
  notes_empty_trash: { method: 'NotesEmptyTrash' },
  notes_trash_count: { method: 'NotesTrashCount' },
  notes_list_folders: { method: 'NotesListFolders' },
  notes_create_folder: { method: 'NotesCreateFolder', args: (p) => [p.name, p.parentId] },
  notes_update_folder: { method: 'NotesUpdateFolder', args: (p) => [p.folderId, p.name] },
  notes_delete_folder: { method: 'NotesDeleteFolder', args: (p) => [p.folderId] },

  // —— mindmap ——
  mindmap_generate: { method: 'MindmapGenerate', args: (p) => [p.topic, p.depth ?? 2] },
  mindmap_save: { method: 'MindmapSave', args: (p) => [p.params ?? p] },
  mindmap_load: { method: 'MindmapLoad', args: (p) => [p.id] },
  mindmap_edit: { method: 'MindmapEdit', args: (p) => [p.id, p.nodes, p.edges] },
  mindmap_to_outline: { method: 'MindmapToOutline', args: (p) => [p.id] },
  mindmap_from_outline: { method: 'MindmapFromOutline', args: (p) => [p.outline] },
  mindmap_mask: { method: 'MindmapMask', args: (p) => [p.id, p.level] },

  // —— qbank ——
  qbank_extract: { method: 'QBankExtract', args: (p) => [p.content, p.source] },
  qbank_save: { method: 'QBankSave', args: (p) => [p.params ?? p] },
  qbank_start_attempt: { method: 'QBankStartAttempt', args: (p) => [p.qbankId, p.questionIds] },
  qbank_answer: { method: 'QBankAnswer', args: (p) => [p.attemptId, p.questionId, p.choice] },
  qbank_submit: { method: 'QBankSubmit', args: (p) => [p.attemptId] },
  qbank_analyze: { method: 'QBankAnalyze', args: (p) => [p.attemptId] },
  qbank_mastery: { method: 'QBankMastery', args: (p) => [p.qbankId] },

  // —— anki ——
  anki_generate: { method: 'AnkiGenerate', args: (p) => [p.source, p.templateId] },
  anki_templates: { method: 'AnkiTemplates' },
  anki_add_template: { method: 'AnkiAddTemplate', args: (p) => [p.name, p.front, p.back] },
  anki_save: { method: 'AnkiSave', args: (p) => [p.card] },
  anki_export: { method: 'AnkiExport', args: (p) => [p.deckName] },

  // —— reader ——
  reader_open: { method: 'ReaderOpen', args: (p) => [p.uri] },
  reader_summarize: { method: 'ReaderSummarize', args: (p) => [p.uri, p.scope] },
  reader_inject: { method: 'ReaderInject', args: (p) => [p.uri, p.chunk] },

  // —— translate / essay / research / paper ——
  translate_text: { method: 'TranslateText', args: (p) => [p.text, p.target] },
  translate_document: { method: 'TranslateDocument', args: (p) => [p.uri, p.target] },
  essay_grade: { method: 'EssayGrade', args: (p) => [p.content, p.prompt] },
  essay_save: { method: 'EssaySave', args: (p) => [p.params ?? p] },
  research_plan: { method: 'ResearchPlan', args: (p) => [p.topic] },
  research_run: { method: 'ResearchRun', args: (p) => [p.planId] },
  research_save: { method: 'ResearchSave', args: (p) => [p.params ?? p] },
  paper_search_arxiv: { method: 'PaperSearchArXiv', args: (p) => [p.query, p.limit ?? 10] },
  paper_search_openalex: { method: 'PaperSearchOpenAlex', args: (p) => [p.query, p.limit ?? 10] },
  paper_download: { method: 'PaperDownload', args: (p) => [p.paperId] },
  paper_cite: { method: 'PaperCite', args: (p) => [p.paperId, p.style] },
  paper_resolve_doi: { method: 'PaperResolveDOI', args: (p) => [p.doi] },

  // —— memory ——
  memory_ingest: { method: 'MemoryIngest', args: (p) => [p.content, p.tags] },
  memory_search: { method: 'MemorySearch', args: (p) => [p.query, p.limit ?? 20] },
  memory_profile: { method: 'MemoryProfile' },
  memory_privacy_mode: { method: 'MemoryPrivacyMode', args: (p) => [p.enabled] },
  memory_decay: { method: 'MemoryDecay' },

  // —— skills / governance ——
  skills_list: { method: 'SkillsList' },
  skills_tools: { method: 'SkillsTools' },
  skills_call: { method: 'SkillsCall', args: (p) => [p.name, p.args] },
  skills_spawn_mcp: { method: 'SkillsSpawnMCP', args: (p) => [p.serverName] },
  skills_enable_server: { method: 'SkillsEnableServer', args: (p) => [p.name] },
  skills_disable_server: { method: 'SkillsDisableServer', args: (p) => [p.name] },
  skills_list_mcp_servers: { method: 'SkillsListMCPServers' },
  skills_load_skillmd: { method: 'SkillsLoadSKILLMD', args: (p) => [p.name] },
  gov_backup: { method: 'GovBackup' },
  gov_restore: { method: 'GovRestore', args: (p) => [p.path] },
  gov_switch_slot: { method: 'GovSwitchSlot', args: (p) => [p.slot] },
  gov_export: { method: 'GovExport', args: (p) => [p.path] },
  gov_import: { method: 'GovImport', args: (p) => [p.path] },
  gov_audit: { method: 'GovAudit' },
  gov_status: { method: 'GovStatus' },
  gov_integrity_check: { method: 'GovIntegrityCheck' },

  // —— voice / ocr / multimodal ——
  voice_input: { method: 'VoiceInput' },
  ocr_recognize: { method: 'OcrRecognize', args: (p) => [p.imageBase64] },
  multimodal_search: { method: 'MultimodalSearch', args: (p) => [p.query] },
};

// ------------------------------------------------------------
// snake_case → PascalCase 回退转换
// ------------------------------------------------------------
function snakeToPascal(snake: string): string {
  return snake
    .split('_')
    .map((part) => (part ? part[0]!.toUpperCase() + part.slice(1) : part))
    .join('');
}

// 方法解析链：
//   1. 精确匹配（Go 方法名也可能是 snake，罕见）
//   2. snake_case → PascalCase（get_setting → GetSetting）
//   3. 剥离版本/命名空间前缀后再转（chat_v2_xxx → ChatXxx；vfs_xxx → Xxx）
function resolveMethod(app: Record<string, (...args: unknown[]) => unknown>, cmd: string): string | null {
  if (typeof app[cmd] === 'function') return cmd;
  const pascal = snakeToPascal(cmd);
  if (typeof app[pascal] === 'function') return pascal;
  const stripped = cmd.replace(/^(chat_v2|vfs|notes|qbank|anki|todo|fsrs|skills|gov|memory|paper|research|essay|translate|reader|mindmap|ocr|voice)_/, '');
  if (stripped !== cmd) {
    const p2 = snakeToPascal(stripped);
    if (typeof app[p2] === 'function') return p2;
  }
  return null;
}

// ------------------------------------------------------------
// invoke 核心
// ------------------------------------------------------------
export interface InvokeOptions {
  timeoutMs?: number;
}

// 插件命令（plugin:xxx|yyy）路由
async function invokePlugin(cmd: string, payload?: Record<string, unknown>): Promise<unknown> {
  // plugin:clipboard-manager|write_text / read_text
  if (cmd.startsWith('plugin:clipboard-manager|')) {
    const op = cmd.split('|')[1];
    if (op === 'write_text') {
      const text = (payload?.text as string) ?? '';
      const { writeText } = await import('../plugins/clipboard-manager');
      await writeText(text);
      return null;
    }
    if (op === 'read_text') {
      const { readText } = await import('../plugins/clipboard-manager');
      return readText();
    }
  }
  console.warn(`[tauri-shim] unsupported plugin command: ${cmd}`);
  return undefined;
}

export async function invoke<T = unknown>(
  cmd: string,
  payload?: Record<string, unknown>,
  options?: InvokeOptions
): Promise<T> {
  if (cmd.startsWith('plugin:')) {
    return (await invokePlugin(cmd, payload)) as T;
  }
  const app = getWailsApp();
  if (!app) {
    throw new Error(`[tauri-shim] wails binding not available for command: ${cmd}`);
  }

  const bridge = BRIDGES[cmd];
  const methodName =
    bridge?.method ?? resolveMethod(app, cmd) ?? snakeToPascal(cmd);
  const fn = app[methodName];
  if (typeof fn !== 'function') {
    // 后端无对应方法：返回 undefined（模拟空结果），避免页面崩溃
    console.warn(`[tauri-shim] no backend method for command: ${cmd} (tried ${methodName})`);
    return undefined as T;
  }

  const args = bridge?.args ? bridge.args(payload ?? {}) : payload ? [payload] : [];
  const raw = await fn(...args);
  return (bridge?.result ? bridge.result(raw) : raw) as T;
}

// 带超时包装（原版 tauriInvoke 语义）
export async function invokeWithTimeout<T = unknown>(
  cmd: string,
  payload?: Record<string, unknown>,
  options?: InvokeOptions
): Promise<T> {
  const timeoutMs = options?.timeoutMs;
  if (!timeoutMs || timeoutMs <= 0) {
    return invoke<T>(cmd, payload, options);
  }
  return new Promise<T>((resolve, reject) => {
    const timer = setTimeout(() => {
      reject(new Error(`[tauri-shim] command timed out: ${cmd}`));
    }, timeoutMs);
    invoke<T>(cmd, payload, options)
      .then((v) => {
        clearTimeout(timer);
        resolve(v);
      })
      .catch((e) => {
        clearTimeout(timer);
        reject(e);
      });
  });
}

// ------------------------------------------------------------
// convertFileSrc —— 原版把后端文件路径转成可访问 URL。
// Wails 下：http(s) 原样返回；其余路径原样返回（由后端 Get 取字节）。
// ------------------------------------------------------------
export function convertFileSrc(filePath: string, protocol = 'asset'): string {
  void protocol;
  if (/^https?:\/\//i.test(filePath)) return filePath;
  if (filePath.startsWith('data:')) return filePath;
  return filePath;
}

// ------------------------------------------------------------
// Channel —— 原版流式通道（Tauri v2）。Wails 无对应物，返回空实现。
// ------------------------------------------------------------
export class Channel<T = unknown> {
  id = 0;
  private handler?: (value: T) => void;
  onmessage(handler: (value: T) => void) {
    this.handler = handler;
    return this;
  }
  // 供 invoke shim 使用：本实现不支持流式，仅占位
  get rid(): number {
    return this.id;
  }
}

export function transformCallback<T = unknown>(callback?: (response: T) => void): number {
  void callback;
  return 0;
}

export function wailsAvailableCheck(): boolean {
  return wailsAvailable();
}
