import { invoke as tauriInvoke } from '@tauri-apps/api/core';
import { invokeWails, isWailsRuntime } from './wailsBridge';

export { isWailsRuntime } from './wailsBridge';

type NativeArgs = Record<string, unknown> | undefined;
type NativeCommand =
  | 'check_anki_connect_status'
  | 'get_anki_deck_names'
  | 'anki_get_deck_names'
  | 'get_anki_model_names'
  | 'get_model_adapter_options'
  | 'memory_get_config'
  | 'get_setting'
  | 'get_settings'
  | 'get_settings_by_prefix'
  | 'save_setting'
  | 'save_settings'
  | 'delete_setting'
  | 'get_backup_config'
  | 'set_backup_config'
  | 'check_api_config_status'
  | 'restore_default_api_configs'
  | 'test_ocr_engine'
  | 'test_search_engine'
  | 'test_web_search_connectivity'
  | 'test_all_search_engines'
  | 'get_provider_strategies_config'
  | 'save_provider_strategies_config'
  | 'get_statistics'
  | 'get_enhanced_statistics'
  | 'preheat_mcp_tools'
  | 'get_mcp_status'
  | 'get_mcp_tools'
  | 'reload_mcp_client'
  | 'get_app_data_dir'
  | 'ensure_debug_log_dir'
  | 'save_webview_settings'
  | 'open_logs_folder'
  | 'report_frontend_log'
  | 'read_file_bytes'
  | 'read_file_text'
  | 'save_text_to_file'
  | 'get_file_size'
  | 'copy_file'
  | 'get_image_as_base64'
  | string;

declare global {
  interface Window {
    __TAURI_INTERNALS__?: any;
    __TAURI_IPC__?: unknown;
    __DEEP_STUDENT_NATIVE__?: {
      invoke?: <T>(command: string, args?: NativeArgs) => Promise<T>;
    };
  }
}

export const isTauriRuntime = (): boolean => {
  if (typeof window === 'undefined') return false;
  return Boolean(window.__TAURI_INTERNALS__ || window.__TAURI_IPC__);
};

export const isInjectedNativeRuntime = (): boolean => {
  if (typeof window === 'undefined') return false;
  return typeof window.__DEEP_STUDENT_NATIVE__?.invoke === 'function';
};

const mcpFallbackBeforeTauriCommands = new Set<string>([
  'preheat_mcp_tools',
  'get_mcp_status',
  'get_mcp_tools',
  'reload_mcp_client',
  'get_model_adapter_options',
]);

function shouldUseFallbackBeforeTauri(command: NativeCommand): boolean {
  return mcpFallbackBeforeTauriCommands.has(command);
}

function storageGet(key: string): string | null {
  if (typeof localStorage === 'undefined') return null;
  return localStorage.getItem(key);
}

const fallbackModelAdapterOptions = [
  { value: 'general', label: 'OpenAI Compatible', description: 'Standard OpenAI-compatible request format', is_default: true },
  { value: 'google', label: 'Google Gemini', description: 'Gemini request format with thinking controls', is_default: true },
  { value: 'anthropic', label: 'Anthropic Claude', description: 'Claude request format with extended thinking controls', is_default: true },
  { value: 'deepseek', label: 'DeepSeek', description: 'DeepSeek reasoning and thinking parameter format', is_default: true },
  { value: 'qwen', label: 'Qwen', description: 'DashScope/Qwen thinking parameter format', is_default: true },
  { value: 'zhipu', label: 'Zhipu GLM', description: 'GLM reasoning parameter format', is_default: true },
  { value: 'doubao', label: 'Doubao', description: 'Doubao thinking model format', is_default: true },
  { value: 'moonshot', label: 'Kimi/Moonshot', description: 'Kimi/Moonshot reasoning content format', is_default: true },
  { value: 'grok', label: 'xAI Grok', description: 'Grok request constraints and reasoning effort format', is_default: true },
  { value: 'minimax', label: 'MiniMax', description: 'MiniMax reasoning split format', is_default: true },
  { value: 'mimo', label: 'Xiaomi MiMo', description: 'MiMo thinking.type and reasoning_content format', is_default: true },
];

function storageSet(key: string, value: string): void {
  if (typeof localStorage === 'undefined') return;
  localStorage.setItem(key, value);
}

function storageDelete(key: string): void {
  if (typeof localStorage === 'undefined') return;
  localStorage.removeItem(key);
}

function fallbackPdfProcessingStatus(fileId: unknown): Record<string, unknown> {
  const id = typeof fileId === 'string' ? fileId : '';
  return {
    fileId: id,
    stage: 'pending',
    percent: 0,
    readyModes: [],
    progress: {
      stage: 'pending',
      percent: 0,
      readyModes: [],
    },
  };
}

function fallbackStatistics(): Record<string, unknown> {
  return {
    total_mistakes: 0,
    total_reviews: 0,
    type_stats: {},
    tag_stats: {},
    recent_mistakes: [],
  };
}

const attachmentRootFolderIdKey = 'attachment_root_folder_id';
const attachmentRootFolderTitleKey = 'attachment_root_folder_title';
const backupConfigKey = 'backup.config';
const apiConfigurationsKey = 'api_configurations';
const vendorConfigsKey = 'vendor_configs';
const modelProfilesKey = 'model_profiles';
const modelAssignmentsKey = 'model_assignments';
const ocrAvailableModelsKey = 'ocr.available_models';
const ocrEngineTypeKey = 'ocr.engine_type';
const ocrEnableThinkingKey = 'ocr.enable_thinking';
const fallbackResourceIndexKey = 'go_hybrid_vfs_resource_sync_index';
const fallbackDstuFoldersKey = 'go_dstu_folder_store';
const fallbackQbankSyncConfigsKey = 'go_qbank_sync_configs';
const systemOcrConfigId = '__system_ocr__';
const defaultCnTrustedSites = [
  'edu.cn',
  'tsinghua.edu.cn',
  'pku.edu.cn',
  'fudan.edu.cn',
  'sjtu.edu.cn',
  'zju.edu.cn',
  'nju.edu.cn',
  'ustc.edu.cn',
  'bit.edu.cn',
  'buaa.edu.cn',
  'gov.cn',
  'beijing.gov.cn',
  'shanghai.gov.cn',
  'guangzhou.gov.cn',
  'shenzhen.gov.cn',
  'xinhuanet.com',
  'people.com.cn',
  'cctv.com',
  'chinanews.com.cn',
  'ce.cn',
  'runoob.com',
  'w3school.com.cn',
  'liaoxuefeng.com',
  'cnblogs.com',
  'csdn.net',
  'jianshu.com',
  'segmentfault.com',
  'juejin.cn',
  'zhihu.com',
  'oschina.net',
  'github.com',
  'gitee.com',
  'coding.net',
  'cas.cn',
  'cass.cn',
  'cnki.net',
  'wanfangdata.com.cn',
  'baidu.com',
  'tencent.com',
  'alibaba.com',
  'huawei.com',
  'xiaomi.com',
  'bytedance.com',
  'infoq.cn',
  '51cto.com',
  'iteye.com',
  'ibiblio.org',
  'apache.org',
  'python.org',
  'nodejs.org',
  'mysql.com',
  'postgresql.org',
];

const defaultModelAssignments = {
  model2_config_id: null,
  review_analysis_model_config_id: null,
  anki_card_model_config_id: null,
  qbank_ai_grading_model_config_id: null,
  embedding_model_config_id: null,
  reranker_model_config_id: null,
  chat_title_model_config_id: null,
  exam_sheet_ocr_model_config_id: null,
  translation_model_config_id: null,
  vl_embedding_model_config_id: null,
  vl_reranker_model_config_id: null,
  memory_decision_model_config_id: null,
  voice_input_asr_model_config_id: null,
  image_generation_model_config_id: null,
  translation_display_mode: null,
};

function storageGetJson<T>(key: string, fallback: T): T {
  const raw = storageGet(key);
  if (!raw) return fallback;
  try {
    return JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
}

function storageSetJson(key: string, value: unknown): void {
  storageSet(key, JSON.stringify(value));
}

function parseBooleanSetting(value: string | null, fallback: boolean): boolean {
  const normalized = (value ?? '').trim().toLowerCase();
  if (['true', '1', 'yes', 'on'].includes(normalized)) return true;
  if (['false', '0', 'no', 'off'].includes(normalized)) return false;
  return fallback;
}

function parseSiteList(value: string | null): string[] {
  const raw = (value ?? '').trim();
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    if (Array.isArray(parsed)) {
      return cleanSiteList(parsed);
    }
  } catch {
    // Keep comma-separated compatibility below.
  }
  return cleanSiteList(raw.split(','));
}

function cleanSiteList(values: unknown[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const value of values) {
    if (typeof value !== 'string') continue;
    const site = value.trim();
    if (!site || seen.has(site)) continue;
    seen.add(site);
    out.push(site);
  }
  return out;
}

type FallbackProviderStrategy = Record<string, unknown>;
type FallbackProviderStrategies = Record<string, FallbackProviderStrategy>;

function defaultProviderStrategy(overrides: FallbackProviderStrategy = {}): FallbackProviderStrategy {
  return {
    timeout_ms: 8000,
    max_retries: 2,
    initial_retry_delay_ms: 200,
    max_retry_delay_ms: 5000,
    backoff_multiplier: 2,
    max_concurrent_requests: 5,
    rate_limit_per_minute: 60,
    cache_enabled: true,
    cache_ttl_seconds: 300,
    cache_max_entries: 128,
    special_handling: {
      handle_429_retry_after: true,
      exponential_backoff_on_5xx: true,
      circuit_breaker_enabled: false,
      circuit_breaker_failure_threshold: 5,
      circuit_breaker_recovery_timeout_ms: 30000,
    },
    ...overrides,
  };
}

function defaultProviderStrategies(): FallbackProviderStrategies {
  return {
    default: defaultProviderStrategy(),
    google_cse: defaultProviderStrategy({ timeout_ms: 6000, max_retries: 2, rate_limit_per_minute: 100 }),
    serpapi: defaultProviderStrategy({
      timeout_ms: 15000,
      max_retries: 2,
      rate_limit_per_minute: 20,
      special_handling: {
        handle_429_retry_after: true,
        exponential_backoff_on_5xx: true,
        circuit_breaker_enabled: true,
        circuit_breaker_failure_threshold: 3,
        circuit_breaker_recovery_timeout_ms: 60000,
      },
    }),
    tavily: defaultProviderStrategy({ timeout_ms: 8000, max_retries: 3, rate_limit_per_minute: 50 }),
    brave: defaultProviderStrategy({ timeout_ms: 12000, max_retries: 2, rate_limit_per_minute: 30 }),
    searxng: defaultProviderStrategy({
      timeout_ms: 20000,
      max_retries: 1,
      rate_limit_per_minute: 30,
      special_handling: {
        handle_429_retry_after: false,
        exponential_backoff_on_5xx: false,
        circuit_breaker_enabled: false,
        circuit_breaker_failure_threshold: 5,
        circuit_breaker_recovery_timeout_ms: 30000,
      },
    }),
    zhipu: defaultProviderStrategy({ timeout_ms: 10000, max_retries: 2, rate_limit_per_minute: 60 }),
    bocha: defaultProviderStrategy({ timeout_ms: 10000, max_retries: 2, rate_limit_per_minute: 60 }),
  };
}

type FallbackResourceSyncRecord = {
  resourceId: string;
  hash: string;
  type: string;
  sourceId: string;
};

type FallbackDstuFolder = {
  id: string;
  parentId: string | null;
  title: string;
  icon?: string;
  color?: string;
  isExpanded: boolean;
  sortOrder: number;
  createdAt: number;
  updatedAt: number;
};

type FallbackDstuFolderItem = {
  id: string;
  folderId: string | null;
  itemType: string;
  itemId: string;
  sortOrder: number;
  createdAt: number;
  updatedAt?: number;
  cachedPath?: string;
};

type FallbackDstuFolderStore = {
  folders: FallbackDstuFolder[];
  items: FallbackDstuFolderItem[];
};

type FallbackOcrModel = {
  configId: string;
  model: string;
  engineType: string;
  name: string;
  isFree: boolean;
  enabled: boolean;
  priority: number;
};

type FallbackOcrEngineInfo = {
  engineType: string;
  name: string;
  description: string;
  recommendedModel: string;
  supportsGrounding: boolean;
  isFree: boolean;
};

type FallbackQbankSyncConfig = {
  default_strategy: string;
  auto_sync: boolean;
  sync_interval_secs: number;
  sync_progress: boolean;
  sync_notes: boolean;
};

const fallbackOcrEngineInfos: FallbackOcrEngineInfo[] = [
  {
    engineType: 'deepseek_ocr',
    name: 'DeepSeek-OCR',
    description: '专业 OCR 模型，支持 Grounding 坐标输出，适合题目集识别',
    recommendedModel: 'deepseek-ai/DeepSeek-OCR',
    supportsGrounding: true,
    isFree: false,
  },
  {
    engineType: 'paddle_ocr_vl',
    name: 'PaddleOCR-VL-1.5',
    description: '百度开源 OCR 视觉语言模型 1.5 版，支持 109 种语言，精度 94.5%，完全免费',
    recommendedModel: 'PaddlePaddle/PaddleOCR-VL-1.5',
    supportsGrounding: true,
    isFree: true,
  },
  {
    engineType: 'paddle_ocr_vl_v1',
    name: 'PaddleOCR-VL',
    description: '百度开源 OCR 视觉语言模型旧版，支持坐标输出，完全免费，作为 1.5 版的备用',
    recommendedModel: 'PaddlePaddle/PaddleOCR-VL',
    supportsGrounding: true,
    isFree: true,
  },
  {
    engineType: 'glm4v_ocr',
    name: 'GLM-4.6V',
    description: '智谱 106B MoE 多模态模型，支持 bbox_2d 坐标输出，题目集导入优先引擎',
    recommendedModel: 'zai-org/GLM-4.6V',
    supportsGrounding: true,
    isFree: false,
  },
  {
    engineType: 'generic_vlm',
    name: '通用多模态模型',
    description: '使用通用 VLM 进行 OCR，适合简单文档识别',
    recommendedModel: 'Qwen/Qwen2.5-VL-7B-Instruct',
    supportsGrounding: false,
    isFree: false,
  },
];

function fallbackResourceIndex(): Record<string, FallbackResourceSyncRecord> {
  return storageGetJson<Record<string, FallbackResourceSyncRecord>>(fallbackResourceIndexKey, {});
}

function fallbackDstuFolders(): FallbackDstuFolderStore {
  return storageGetJson<FallbackDstuFolderStore>(fallbackDstuFoldersKey, { folders: [], items: [] });
}

function saveFallbackDstuFolders(store: FallbackDstuFolderStore): void {
  storageSetJson(fallbackDstuFoldersKey, store);
}

function defaultFallbackQbankSyncConfig(): FallbackQbankSyncConfig {
  return {
    default_strategy: 'keep_newer',
    auto_sync: false,
    sync_interval_secs: 300,
    sync_progress: true,
    sync_notes: true,
  };
}

function normalizeFallbackQbankSyncConfig(value: unknown): FallbackQbankSyncConfig {
  const defaults = defaultFallbackQbankSyncConfig();
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return defaults;
  }
  const source = value as Record<string, unknown>;
  const defaultStrategy =
    typeof source.default_strategy === 'string'
      ? source.default_strategy
      : typeof source.defaultStrategy === 'string'
        ? source.defaultStrategy
        : defaults.default_strategy;
  const syncInterval =
    typeof source.sync_interval_secs === 'number' && Number.isFinite(source.sync_interval_secs)
      ? source.sync_interval_secs
      : typeof source.syncIntervalSecs === 'number' && Number.isFinite(source.syncIntervalSecs)
        ? source.syncIntervalSecs
        : defaults.sync_interval_secs;
  return {
    default_strategy: defaultStrategy.trim() || defaults.default_strategy,
    auto_sync: typeof source.auto_sync === 'boolean'
      ? source.auto_sync
      : typeof source.autoSync === 'boolean'
        ? source.autoSync
        : defaults.auto_sync,
    sync_interval_secs: syncInterval > 0 ? Math.floor(syncInterval) : defaults.sync_interval_secs,
    sync_progress: typeof source.sync_progress === 'boolean'
      ? source.sync_progress
      : typeof source.syncProgress === 'boolean'
        ? source.syncProgress
        : defaults.sync_progress,
    sync_notes: typeof source.sync_notes === 'boolean'
      ? source.sync_notes
      : typeof source.syncNotes === 'boolean'
        ? source.syncNotes
        : defaults.sync_notes,
  };
}

function fallbackQbankSyncConfigs(): Record<string, FallbackQbankSyncConfig> {
  const raw = storageGetJson<Record<string, unknown>>(fallbackQbankSyncConfigsKey, {});
  return Object.fromEntries(
    Object.entries(raw).map(([examId, config]) => [examId, normalizeFallbackQbankSyncConfig(config)])
  );
}

function saveFallbackQbankSyncConfigs(configs: Record<string, FallbackQbankSyncConfig>): void {
  storageSetJson(fallbackQbankSyncConfigsKey, configs);
}

function fallbackQbankSyncConfig(examId: unknown): FallbackQbankSyncConfig {
  const id = typeof examId === 'string' ? examId.trim() : '';
  if (!id) return defaultFallbackQbankSyncConfig();
  return fallbackQbankSyncConfigs()[id] ?? defaultFallbackQbankSyncConfig();
}

function requireFallbackStringArg(command: string, args: NativeArgs | undefined, name: string): string {
  const value = args?.[name];
  if (typeof value !== 'string' || !value.trim()) {
    throw new Error(`Native fallback command ${command} requires string argument "${name}"`);
  }
  return value.trim();
}

function requireFallbackObjectArg(command: string, args: NativeArgs | undefined, name: string): Record<string, unknown> {
  const value = args?.[name];
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`Native fallback command ${command} requires object argument "${name}"`);
  }
  return value as Record<string, unknown>;
}

function fallbackFolderId(value: unknown): string | null {
  if (typeof value !== 'string') return null;
  const trimmed = value.trim();
  return trimmed && trimmed !== 'root' ? trimmed : null;
}

function fallbackDstuType(id: string, explicit?: unknown): string {
  if (typeof explicit === 'string' && explicit.trim()) return explicit.trim();
  if (id.startsWith('note_')) return 'note';
  if (id.startsWith('tb_')) return 'textbook';
  if (id.startsWith('exam_')) return 'exam';
  if (id.startsWith('tr_')) return 'translation';
  if (id.startsWith('essay_')) return 'essay';
  if (id.startsWith('img_')) return 'image';
  if (id.startsWith('mm_')) return 'mindmap';
  return 'file';
}

function fallbackFolderPath(store: FallbackDstuFolderStore, folderId: string | null): string {
  if (!folderId) return '/';
  const byId = new Map(store.folders.map(folder => [folder.id, folder]));
  const chain: FallbackDstuFolder[] = [];
  let current: string | null = folderId;
  const seen = new Set<string>();
  while (current && !seen.has(current)) {
    seen.add(current);
    const folder = byId.get(current);
    if (!folder) break;
    chain.push(folder);
    current = folder.parentId;
  }
  if (chain.length === 0) return `/${folderId}`;
  return `/${chain.reverse().map(folder => folder.title || folder.id).join('/')}`;
}

function fallbackResourcePath(store: FallbackDstuFolderStore, folderId: string | null, resourceId: string): string {
  return `${fallbackFolderPath(store, folderId).replace(/\/$/, '')}/${resourceId}`;
}

function fallbackFolderTree(store: FallbackDstuFolderStore, parentId: string | null = null): unknown[] {
  return store.folders
    .filter(folder => folder.parentId === parentId)
    .sort((a, b) => a.sortOrder - b.sortOrder || a.title.localeCompare(b.title))
    .map(folder => ({
      folder,
      children: fallbackFolderTree(store, folder.id),
      items: store.items.filter(item => item.folderId === folder.id).sort((a, b) => a.sortOrder - b.sortOrder),
    }));
}

function fallbackHash(value: string): string {
  let hash = 0x811c9dc5;
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index);
    hash = Math.imul(hash, 0x01000193);
  }
  return (hash >>> 0).toString(16).padStart(64, '0');
}

function normalizeOcrEngineType(value: unknown): string {
  const normalized = typeof value === 'string' ? value.trim().toLowerCase() : '';
  if (['deepseek_ocr', 'deepseek-ocr', 'deepseek'].includes(normalized)) return 'deepseek_ocr';
  if (['paddle_ocr_vl', 'paddleocr-vl', 'paddleocr_vl', 'paddle'].includes(normalized)) return 'paddle_ocr_vl';
  if (['paddle_ocr_vl_v1', 'paddleocr-vl-v1', 'paddleocr_vl_v1'].includes(normalized)) return 'paddle_ocr_vl_v1';
  if (['glm4v_ocr', 'glm-4.6v', 'glm4v', 'glm-4v'].includes(normalized)) return 'glm4v_ocr';
  if (['generic_vlm', 'generic', 'vlm'].includes(normalized)) return 'generic_vlm';
  if (['system_ocr', 'system', 'native'].includes(normalized)) return 'system_ocr';
  return 'paddle_ocr_vl';
}

function inferOcrEngineFromModel(model: string): string {
  const lower = model.trim().toLowerCase();
  if (lower === 'system') return 'system_ocr';
  if (/glm-(?:4\.[5-9]|4\.\d{2,}|[5-9](?:\.\d+)?)v/i.test(model)) return 'glm4v_ocr';
  if (lower.includes('deepseek') && lower.includes('ocr')) return 'deepseek_ocr';
  if (lower.includes('paddleocr-vl-1') || lower.includes('paddleocr_vl_1')) return 'paddle_ocr_vl';
  if (lower.includes('paddle') || lower.includes('paddleocr')) return 'paddle_ocr_vl_v1';
  return 'generic_vlm';
}

function fallbackOcrModels(): FallbackOcrModel[] {
  return storageGetJson<FallbackOcrModel[]>(ocrAvailableModelsKey, [])
    .map((model, index) => ({
      configId: String(model.configId ?? '').trim(),
      model: String(model.model ?? '').trim(),
      engineType: normalizeOcrEngineType(model.engineType),
      name: String(model.name || model.model || model.configId || '').trim(),
      isFree: Boolean(model.isFree),
      enabled: model.enabled !== false,
      priority: typeof model.priority === 'number' && Number.isFinite(model.priority) ? model.priority : index,
    }))
    .filter(model => model.configId && model.configId !== systemOcrConfigId)
    .sort((left, right) => left.priority - right.priority || left.configId.localeCompare(right.configId))
    .map((model, index) => ({ ...model, priority: index }));
}

function saveFallbackOcrModels(models: FallbackOcrModel[]): void {
  storageSetJson(ocrAvailableModelsKey, models.map((model, index) => ({ ...model, priority: index })));
}

function withOcrMetadata(model: FallbackOcrModel): FallbackOcrModel & {
  description?: string;
  supportsGrounding: boolean;
} {
  const info = fallbackOcrEngineInfos.find(item => item.engineType === model.engineType);
  return {
    ...model,
    description: info?.description,
    supportsGrounding: info?.supportsGrounding ?? false,
  };
}

function fallbackSyncResource(type: string, sourceId: string, content: string): FallbackResourceSyncRecord & { isNew: boolean } {
  const trimmedSourceId = sourceId.trim();
  if (!trimmedSourceId) {
    throw new Error(`resource sync requires sourceId for ${type}`);
  }
  const index = fallbackResourceIndex();
  const key = `${type}:${trimmedSourceId}`;
  const hash = fallbackHash(content);
  const existing = index[key];
  if (existing && existing.hash === hash) {
    return { ...existing, isNew: false };
  }
  const record: FallbackResourceSyncRecord = {
    resourceId: existing?.resourceId ?? `res_${fallbackHash(key).slice(0, 12)}`,
    hash,
    type,
    sourceId: trimmedSourceId,
  };
  index[key] = record;
  storageSetJson(fallbackResourceIndexKey, index);
  return { ...record, isNew: !existing };
}

function fallbackReviewStats(examId: unknown = null): Record<string, unknown> {
  return {
    exam_id: typeof examId === 'string' ? examId : null,
    total_plans: 0,
    new_count: 0,
    learning_count: 0,
    reviewing_count: 0,
    graduated_count: 0,
    suspended_count: 0,
    due_today: 0,
    overdue_count: 0,
    difficult_count: 0,
    total_reviews: 0,
    total_correct: 0,
    avg_correct_rate: 0,
    avg_ease_factor: 0,
    updated_at: new Date().toISOString(),
  };
}

async function fallbackInvoke<T>(command: NativeCommand, args?: NativeArgs): Promise<T> {
  const key = typeof args?.key === 'string' ? args.key : '';

  if (command === 'get_setting') {
    return storageGet(key) as T;
  }

  if (command === 'get_settings') {
    const keys = Array.isArray(args?.keys) ? args.keys.filter((item): item is string => typeof item === 'string') : [];
    return Object.fromEntries(keys.flatMap(item => {
      const value = storageGet(item);
      return value == null ? [] : [[item, value]];
    })) as T;
  }

  if (command === 'get_settings_by_prefix') {
    const prefix = typeof args?.prefix === 'string' ? args.prefix : '';
    if (typeof localStorage === 'undefined') {
      return [] as T;
    }
    const rows: string[][] = [];
    for (let index = 0; index < localStorage.length; index += 1) {
      const itemKey = localStorage.key(index);
      if (!itemKey || !itemKey.startsWith(prefix)) continue;
      rows.push([itemKey, localStorage.getItem(itemKey) ?? '', '']);
    }
    rows.sort((left, right) => left[0].localeCompare(right[0]));
    return rows as T;
  }

  if (command === 'save_setting') {
    storageSet(key, String(args?.value ?? ''));
    return undefined as T;
  }

  if (command === 'save_settings') {
    const values = args?.values;
    if (values && typeof values === 'object' && !Array.isArray(values)) {
      for (const [settingKey, settingValue] of Object.entries(values)) {
        storageSet(settingKey, String(settingValue ?? ''));
      }
    }
    return undefined as T;
  }

  if (command === 'delete_setting') {
    storageDelete(key);
    return undefined as T;
  }

  if (command === 'get_backup_config') {
    return storageGetJson(backupConfigKey, {
      backupDirectory: null,
      autoBackupEnabled: false,
      autoBackupIntervalHours: 24,
      maxBackupCount: 5,
      slimBackup: false,
    }) as T;
  }

  if (command === 'set_backup_config') {
    storageSetJson(backupConfigKey, args?.config ?? {});
    return undefined as T;
  }

  if (command === 'check_api_config_status') {
    const configs = storageGetJson<any[]>(apiConfigurationsKey, []);
    const assignments = storageGetJson<Record<string, unknown>>(modelAssignmentsKey, {});
    return {
      config_count: configs.length,
      enabled_count: configs.filter(config => config?.enabled === true).length,
      has_assignments: Boolean(assignments.model2_config_id || assignments.review_analysis_model_config_id),
      needs_recovery: configs.length === 0,
    } as T;
  }

  if (command === 'restore_default_api_configs') {
    storageSetJson(apiConfigurationsKey, [
      {
        id: 'openai-gpt4',
        name: 'OpenAI GPT-4',
        providerType: 'openai',
        providerScope: 'openai',
        apiProtocol: 'openai_responses',
        supportsOpenAIResponses: true,
        apiKey: '',
        baseUrl: 'https://api.openai.com/v1',
        model: 'gpt-4-turbo-preview',
        isMultimodal: true,
        enabled: false,
        modelAdapter: 'general',
        maxOutputTokens: 4096,
        temperature: 0.7,
        supportsTools: true,
        geminiApiVersion: 'v1',
      },
      {
        id: 'claude-sonnet',
        name: 'Claude 3.5 Sonnet',
        providerType: 'anthropic',
        providerScope: 'anthropic',
        apiProtocol: 'anthropic',
        supportsOpenAIResponses: false,
        apiKey: '',
        baseUrl: 'https://api.anthropic.com/v1',
        model: 'claude-3-5-sonnet-20241022',
        isMultimodal: true,
        enabled: false,
        modelAdapter: 'anthropic',
        maxOutputTokens: 4096,
        temperature: 0.7,
        supportsTools: true,
        geminiApiVersion: 'v1',
      },
    ]);
    storageSetJson(modelAssignmentsKey, defaultModelAssignments);
    return '✅ 默认API配置已恢复！请填入您的API密钥并启用相应配置。' as T;
  }

  if (command === 'import_builtin_templates') {
    return undefined as T;
  }

  if (command === 'get_all_custom_templates') {
    return [] as T;
  }

  if (command === 'get_default_template_id') {
    return null as T;
  }

  if (
    command === 'create_custom_template' ||
    command === 'update_custom_template' ||
    command === 'delete_custom_template' ||
    command === 'set_default_template' ||
    command === 'import_custom_templates_bulk' ||
    command === 'export_template'
  ) {
    throw new Error(`Template command is not available in this native fallback: ${command}`);
  }

  if (command === 'vfs_get_attachment_config') {
    return {
      attachmentRootFolderId: storageGet(attachmentRootFolderIdKey),
      attachmentRootFolderTitle: storageGet(attachmentRootFolderTitleKey),
    } as T;
  }

  if (command === 'vfs_set_attachment_root_folder') {
    const folderId = typeof args?.folderId === 'string' ? args.folderId.trim() : '';
    if (!folderId) {
      throw new Error('vfs_set_attachment_root_folder requires folderId');
    }
    storageSet(attachmentRootFolderIdKey, folderId);
    storageDelete(attachmentRootFolderTitleKey);
    return undefined as T;
  }

  if (command === 'vfs_create_attachment_root_folder') {
    const title = typeof args?.title === 'string' && args.title.trim() ? args.title.trim() : 'Attachments';
    const folderId = `folder_${Date.now().toString(36)}${Math.random().toString(36).slice(2, 10)}`;
    storageSet(attachmentRootFolderIdKey, folderId);
    storageSet(attachmentRootFolderTitleKey, title);
    return folderId as T;
  }

  if (command === 'get_cn_whitelist_config') {
    return {
      default_sites: defaultCnTrustedSites,
      user_config: {
        enabled: parseBooleanSetting(storageGet('web_search.cn_whitelist.enabled'), false),
        use_default_list: parseBooleanSetting(storageGet('web_search.cn_whitelist.use_default'), true),
        custom_sites: parseSiteList(storageGet('web_search.cn_whitelist.custom_sites')),
      },
    } as T;
  }

  if (command === 'get_provider_strategies_config') {
    const fallbackStrategies = defaultProviderStrategies();
    const raw = storageGet('web_search.provider_strategies');
    let providerStrategies = fallbackStrategies;
    if (raw && raw.trim()) {
      try {
        const parsed = JSON.parse(raw);
        if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
          providerStrategies = {
            ...fallbackStrategies,
            ...parsed,
            default: {
              ...fallbackStrategies.default,
              ...(typeof parsed.default === 'object' && parsed.default ? parsed.default : {}),
            },
          };
        }
      } catch {
        providerStrategies = fallbackStrategies;
      }
    }
    return {
      provider_strategies: providerStrategies,
      config_keys: { provider_strategies: 'web_search.provider_strategies' },
    } as T;
  }

  if (command === 'save_provider_strategies_config') {
    const strategies = args?.strategies;
    if (!strategies || typeof strategies !== 'object' || Array.isArray(strategies)) {
      throw new Error('save_provider_strategies_config requires object argument "strategies"');
    }
    storageSet('web_search.provider_strategies', JSON.stringify(strategies));
    return true as T;
  }

  if (command === 'preheat_mcp_tools') {
    const raw = storageGet('mcp.tools.list');
    let count = 0;
    if (raw && raw.trim()) {
      try {
        const parsed = JSON.parse(raw);
        count = Array.isArray(parsed) ? parsed.length : 0;
      } catch {
        count = 0;
      }
    }
    return { ok: true, count } as T;
  }

  if (command === 'get_mcp_status') {
    const selected = storageGet('session.selected_mcp_tools');
    const selectedIsKnown = selected != null;
    const enabled = selectedIsKnown ? selected.trim().length > 0 : false;
    return {
      available: false,
      enabled,
      connected: false,
      enabled_reason: selectedIsKnown && !enabled ? '会话未选择MCP工具' : null,
      server_info: null,
      tools_count: 0,
      last_error: 'backend_mcp_disabled',
      namespace_prefix: storageGet('mcp.tools.namespace_prefix') ?? '',
      conflict_resolution: storageGet('mcp.tools.conflict_resolution') || 'use_namespace',
      cache_state: {
        ttl_ms: Number.parseInt(storageGet('mcp.tools.cache_ttl_ms') || '300000', 10) || 300000,
        last_built_at: null,
      },
    } as T;
  }

  if (command === 'get_mcp_tools') {
    return [] as T;
  }

  if (command === 'get_ocr_engines') {
    return fallbackOcrEngineInfos as T;
  }

  if (command === 'get_ocr_engine_type') {
    return (storageGet(ocrEngineTypeKey) || 'paddle_ocr_vl') as T;
  }

  if (command === 'get_ocr_thinking_enabled') {
    return parseBooleanSetting(storageGet(ocrEnableThinkingKey), false) as T;
  }

  if (command === 'set_ocr_thinking_enabled') {
    storageSet(ocrEnableThinkingKey, args?.enabled === true ? 'true' : 'false');
    return true as T;
  }

  if (command === 'get_available_ocr_models') {
    return fallbackOcrModels().map(withOcrMetadata) as T;
  }

  if (command === 'test_ocr_engine') {
    const request = args?.request && typeof args.request === 'object' && !Array.isArray(args.request)
      ? args.request as Record<string, unknown>
      : {};
    const engineType = normalizeOcrEngineType(request.engineType);
    const configId = typeof request.configId === 'string' ? request.configId : '';
    const model = fallbackOcrModels().find(item => item.configId === configId);
    const info = fallbackOcrEngineInfos.find(item => item.engineType === (model?.engineType ?? engineType));
    return {
      engineType: model?.engineType ?? engineType,
      engineName: model?.name || info?.name || 'OCR',
      text: '',
      regions: [],
      elapsedMs: 0,
      success: false,
      error: 'OCR engine test requires the Go/Wails native runtime',
    } as T;
  }

  if (command === 'save_available_ocr_models') {
    const models = Array.isArray(args?.models) ? args.models : [];
    saveFallbackOcrModels(models.map((model, index) => {
      const entry = model as Record<string, unknown>;
      return {
        configId: String(entry.configId ?? '').trim(),
        model: String(entry.model ?? '').trim(),
        engineType: normalizeOcrEngineType(entry.engineType),
        name: String(entry.name || entry.model || entry.configId || '').trim(),
        isFree: Boolean(entry.isFree),
        enabled: entry.enabled !== false,
        priority: typeof entry.priority === 'number' ? entry.priority : index,
      };
    }).filter(model => model.configId));
    return true as T;
  }

  if (command === 'update_ocr_engine_priority') {
    const models = fallbackOcrModels();
    const byId = new Map(models.map((model, index) => [model.configId, { model, index }]));
    const engineList = Array.isArray(args?.engineList) ? args.engineList : [];
    for (const [index, item] of engineList.entries()) {
      if (!item || typeof item !== 'object') continue;
      const configId = String((item as Record<string, unknown>).configId ?? '').trim();
      const existing = byId.get(configId);
      if (!existing) continue;
      existing.model.priority = index;
      existing.model.enabled = (item as Record<string, unknown>).enabled === true;
    }
    saveFallbackOcrModels(models.sort((left, right) => left.priority - right.priority));
    return true as T;
  }

  if (command === 'add_ocr_engine') {
    const configId = typeof args?.configId === 'string' ? args.configId.trim() : '';
    const model = typeof args?.model === 'string' ? args.model.trim() : '';
    const name = typeof args?.name === 'string' ? args.name.trim() : model || configId;
    if (!configId || configId === systemOcrConfigId) {
      throw new Error('Invalid OCR engine configId');
    }
    const models = fallbackOcrModels();
    if (models.some(item => item.configId === configId)) {
      throw new Error('OCR engine already exists');
    }
    const engineType = typeof args?.engineType === 'string' && args.engineType.trim()
      ? normalizeOcrEngineType(args.engineType)
      : inferOcrEngineFromModel(model);
    const info = fallbackOcrEngineInfos.find(item => item.engineType === engineType);
    models.push({
      configId,
      model,
      engineType,
      name,
      isFree: info?.isFree ?? false,
      enabled: true,
      priority: models.length,
    });
    saveFallbackOcrModels(models);
    return true as T;
  }

  if (command === 'remove_ocr_engine') {
    const configId = typeof args?.configId === 'string' ? args.configId.trim() : '';
    if (!configId || configId === systemOcrConfigId) {
      throw new Error('Invalid OCR engine configId');
    }
    const models = fallbackOcrModels();
    const filtered = models.filter(model => model.configId !== configId);
    if (filtered.length === models.length) {
      throw new Error('OCR engine not found');
    }
    saveFallbackOcrModels(filtered);
    return true as T;
  }

  if (command === 'reload_mcp_client') {
    return {
      success: true,
      message: 'Backend MCP disabled; frontend SDK in use',
    } as T;
  }

  if (command === 'resource_sync_note') {
    const noteId = typeof args?.noteId === 'string' ? args.noteId : '';
    return fallbackSyncResource('note', noteId, `# Synced note\n\nSource note: ${noteId.trim()}\n`) as T;
  }

  if (command === 'resource_sync_exam') {
    const sessionId = typeof args?.sessionId === 'string' ? args.sessionId : '';
    return fallbackSyncResource('exam', sessionId, `# Synced exam\n\nSource exam session: ${sessionId.trim()}\n`) as T;
  }

  if (command === 'resource_sync_textbook_pages') {
    const textbookId = typeof args?.textbookId === 'string' ? args.textbookId.trim() : '';
    const pageRange = Array.isArray(args?.pageRange)
      ? args.pageRange.filter((item): item is number => typeof item === 'number' && Number.isFinite(item))
      : [];
    const rangeLabel = pageRange.length === 2 ? `${pageRange[0]}-${pageRange[1]}` : 'all';
    const sourceId = rangeLabel === 'all' ? textbookId : `${textbookId}:pages:${rangeLabel}`;
    return [fallbackSyncResource('textbook', sourceId, `# Synced textbook pages\n\nSource textbook: ${textbookId}\nPage range: ${rangeLabel}\n`)] as T;
  }

  if (command === 'resource_check_sync_needed') {
    const resourceType = typeof args?.resourceType === 'string' ? args.resourceType.trim() : '';
    const sourceId = typeof args?.sourceId === 'string' ? args.sourceId.trim() : '';
    const currentHash = typeof args?.currentHash === 'string' ? args.currentHash.trim() : '';
    const record = fallbackResourceIndex()[`${resourceType}:${sourceId}`];
    if (!record) {
      return { needsSync: true } as T;
    }
    return {
      needsSync: currentHash !== '' && currentHash !== record.hash,
      existingResourceId: record.resourceId,
      existingHash: record.hash,
    } as T;
  }

  if (command === 'get_api_configurations') {
    return storageGetJson(apiConfigurationsKey, []) as T;
  }

  if (command === 'save_api_configurations') {
    storageSetJson(apiConfigurationsKey, Array.isArray(args?.configs) ? args.configs : []);
    return undefined as T;
  }

  if (command === 'get_vendor_configs') {
    return storageGetJson(vendorConfigsKey, []) as T;
  }

  if (command === 'save_vendor_configs') {
    storageSetJson(vendorConfigsKey, Array.isArray(args?.configs) ? args.configs : []);
    return undefined as T;
  }

  if (command === 'get_model_profiles') {
    return storageGetJson(modelProfilesKey, []) as T;
  }

  if (command === 'save_model_profiles') {
    storageSetJson(modelProfilesKey, Array.isArray(args?.profiles) ? args.profiles : []);
    return undefined as T;
  }

  if (command === 'get_model_assignments') {
    return {
      ...defaultModelAssignments,
      ...storageGetJson(modelAssignmentsKey, {}),
    } as T;
  }

  if (command === 'save_model_assignments') {
    const assignments = args?.assignments && typeof args.assignments === 'object' && !Array.isArray(args.assignments)
      ? args.assignments
      : {};
    storageSetJson(modelAssignmentsKey, { ...defaultModelAssignments, ...assignments });
    return undefined as T;
  }

  if (command === 'vfs_unified_index_status') {
    return {
      totalUnits: 0,
      textStats: { pending: 0, indexing: 0, indexed: 0, failed: 0, disabled: 0 },
      mmStats: { pending: 0, indexing: 0, indexed: 0, failed: 0, disabled: 0 },
      dimensions: [],
    } as T;
  }

  if (command === 'vfs_get_resource_units' || command === 'vfs_sync_resource_units') {
    return [] as T;
  }

  if (command === 'vfs_get_all_index_status') {
    return {
      totalResources: 0,
      indexedCount: 0,
      pendingCount: 0,
      indexingCount: 0,
      failedCount: 0,
      disabledCount: 0,
      staleCount: 0,
      textQueueCount: 0,
      textTotalResources: 0,
      textIndexedCount: 0,
      textPendingCount: 0,
      textIndexingCount: 0,
      textFailedCount: 0,
      textDisabledCount: 0,
      displayTotalResources: 0,
      displayIndexedCount: 0,
      displayPendingCount: 0,
      displayIndexingCount: 0,
      displayFailedCount: 0,
      displayDisabledCount: 0,
      mmTotalResources: 0,
      mmIndexedCount: 0,
      mmPendingCount: 0,
      mmIndexingCount: 0,
      mmFailedCount: 0,
      mmDisabledCount: 0,
      resources: [],
    } as T;
  }

  if (command === 'vfs_list_embedding_dims' || command === 'vfs_list_dimensions' || command === 'vfs_get_resource_text_chunks') {
    return [] as T;
  }

  if (command === 'vfs_reindex_unit' || command === 'vfs_clear_resource_ocr') {
    return false as T;
  }

  if (command === 'vfs_reindex_resource') {
    return 0 as T;
  }

  if (command === 'vfs_unified_batch_index' || command === 'vfs_batch_index_pending') {
    return { successCount: 0, failCount: 0, total: 0 } as T;
  }

  if (command === 'vfs_delete_resource_index') {
    return {
      sqliteOk: true,
      lanceTextOk: true,
      lanceMmOk: true,
      warnings: [],
      retryable: false,
    } as T;
  }

  if (command === 'vfs_get_resource_ocr_info') {
    const resourceId = typeof args?.resourceId === 'string' ? args.resourceId : '';
    return {
      resourceId,
      resourceType: '',
      hasOcr: false,
      ocrText: null,
      ocrTextLength: 0,
      extractedText: null,
      extractedTextLength: 0,
      activeSource: 'none',
      ocrPages: [],
    } as T;
  }

  if (command === 'vfs_rag_search') {
    return {
      results: [],
      count: 0,
      elapsedMs: 0,
    } as T;
  }

  if (command === 'vfs_list_files') {
    return [] as T;
  }

  if (command === 'vfs_list_mindmaps') {
    return [] as T;
  }

  if (
    command === 'vfs_create_mindmap' ||
    command === 'vfs_get_mindmap' ||
    command === 'vfs_get_mindmap_content' ||
    command === 'vfs_update_mindmap' ||
    command === 'vfs_delete_mindmap' ||
    command === 'vfs_set_mindmap_favorite' ||
    command === 'vfs_get_mindmap_versions' ||
    command === 'vfs_get_mindmap_version' ||
    command === 'vfs_get_mindmap_version_content'
  ) {
    throw new Error(`${command} is not available outside a native runtime`);
  }

  if (command === 'vfs_get_pdf_processing_status') {
    return fallbackPdfProcessingStatus(args?.fileId) as T;
  }

  if (command === 'vfs_get_batch_pdf_processing_status') {
    const fileIds = Array.isArray(args?.fileIds) ? args.fileIds.filter((fileId): fileId is string => typeof fileId === 'string') : [];
    return Object.fromEntries(fileIds.map(fileId => [fileId, fallbackPdfProcessingStatus(fileId)])) as T;
  }

  if (command === 'vfs_cancel_pdf_processing') {
    return false as T;
  }

  if (command === 'vfs_retry_pdf_processing' || command === 'vfs_start_pdf_processing') {
    return undefined as T;
  }

  if (command === 'vfs_get_pdf_page_image') {
    throw new Error('vfs_get_pdf_page_image is not available outside a native runtime');
  }

  if (command === 'vfs_get_blob_base64') {
    throw new Error('vfs_get_blob_base64 is not available outside a native runtime');
  }

  if (command === 'vfs_get_file') {
    return null as T;
  }

  if (command === 'vfs_get_file_content') {
    return { content: null, found: false } as T;
  }

  if (command === 'vfs_upload_file') {
    throw new Error('vfs_upload_file is not available outside a native runtime');
  }

  if (command === 'vfs_delete_file') {
    return undefined as T;
  }

  if (command === 'textbooks_update_bookmarks') {
    return false as T;
  }

  if (command === 'textbooks_add') {
    throw new Error('textbooks_add is not available outside a native runtime');
  }

  if (command === 'dstu_folder_create') {
    const store = fallbackDstuFolders();
    const now = Date.now();
    const parentId = fallbackFolderId(args?.parentId);
    if (parentId && !store.folders.some(folder => folder.id === parentId)) {
      throw new Error(`folder not found: ${parentId}`);
    }
    const folder: FallbackDstuFolder = {
      id: `fld_${now.toString(36)}${Math.random().toString(36).slice(2, 8)}`,
      parentId,
      title: typeof args?.title === 'string' && args.title.trim() ? args.title.trim() : 'Untitled',
      icon: typeof args?.icon === 'string' ? args.icon : undefined,
      color: typeof args?.color === 'string' ? args.color : undefined,
      isExpanded: true,
      sortOrder: store.folders.filter(item => item.parentId === parentId).length,
      createdAt: now,
      updatedAt: now,
    };
    store.folders.push(folder);
    saveFallbackDstuFolders(store);
    return folder as T;
  }

  if (command === 'dstu_folder_get') {
    const folderId = typeof args?.folderId === 'string' ? args.folderId : '';
    return (fallbackDstuFolders().folders.find(folder => folder.id === folderId) ?? null) as T;
  }

  if (command === 'dstu_folder_rename') {
    const store = fallbackDstuFolders();
    const folderId = typeof args?.folderId === 'string' ? args.folderId : '';
    const folder = store.folders.find(item => item.id === folderId);
    if (!folder) throw new Error(`folder not found: ${folderId}`);
    folder.title = typeof args?.title === 'string' && args.title.trim() ? args.title.trim() : folder.title;
    folder.updatedAt = Date.now();
    saveFallbackDstuFolders(store);
    return undefined as T;
  }

  if (command === 'dstu_folder_delete') {
    const store = fallbackDstuFolders();
    const folderId = typeof args?.folderId === 'string' ? args.folderId : '';
    const removed = new Set<string>([folderId]);
    let changed = true;
    while (changed) {
      changed = false;
      for (const folder of store.folders) {
        if (folder.parentId && removed.has(folder.parentId) && !removed.has(folder.id)) {
          removed.add(folder.id);
          changed = true;
        }
      }
    }
    store.folders = store.folders.filter(folder => !removed.has(folder.id));
    store.items = store.items.map(item => removed.has(item.folderId ?? '') ? { ...item, folderId: null, cachedPath: `/${item.itemId}` } : item);
    saveFallbackDstuFolders(store);
    return undefined as T;
  }

  if (command === 'dstu_folder_move') {
    const store = fallbackDstuFolders();
    const folderId = typeof args?.folderId === 'string' ? args.folderId : '';
    const folder = store.folders.find(item => item.id === folderId);
    if (!folder) throw new Error(`folder not found: ${folderId}`);
    folder.parentId = fallbackFolderId(args?.newParentId);
    folder.updatedAt = Date.now();
    saveFallbackDstuFolders(store);
    return undefined as T;
  }

  if (command === 'dstu_folder_set_expanded') {
    const store = fallbackDstuFolders();
    const folderId = typeof args?.folderId === 'string' ? args.folderId : '';
    const folder = store.folders.find(item => item.id === folderId);
    if (!folder) throw new Error(`folder not found: ${folderId}`);
    folder.isExpanded = args?.isExpanded === true;
    folder.updatedAt = Date.now();
    saveFallbackDstuFolders(store);
    return undefined as T;
  }

  if (command === 'dstu_folder_add_item' || command === 'dstu_folder_move_item') {
    const store = fallbackDstuFolders();
    const folderId = fallbackFolderId(command === 'dstu_folder_add_item' ? args?.folderId : args?.newFolderId);
    const itemId = typeof args?.itemId === 'string' ? args.itemId.trim() : '';
    if (!itemId) throw new Error(`${command} requires itemId`);
    const itemType = fallbackDstuType(itemId, args?.itemType);
    let item = store.items.find(entry => entry.itemType === itemType && entry.itemId === itemId);
    if (!item) {
      item = {
        id: `fi_${Date.now().toString(36)}${Math.random().toString(36).slice(2, 8)}`,
        folderId,
        itemType,
        itemId,
        sortOrder: store.items.filter(entry => entry.folderId === folderId).length,
        createdAt: Date.now(),
      };
      store.items.push(item);
    }
    item.folderId = folderId;
    item.updatedAt = Date.now();
    item.cachedPath = fallbackResourcePath(store, folderId, itemId);
    saveFallbackDstuFolders(store);
    return (command === 'dstu_folder_add_item' ? item : undefined) as T;
  }

  if (command === 'dstu_folder_remove_item') {
    const store = fallbackDstuFolders();
    const itemId = typeof args?.itemId === 'string' ? args.itemId.trim() : '';
    const itemType = fallbackDstuType(itemId, args?.itemType);
    store.items = store.items.filter(item => item.itemId !== itemId || item.itemType !== itemType);
    saveFallbackDstuFolders(store);
    return undefined as T;
  }

  if (command === 'dstu_folder_list') {
    return fallbackDstuFolders().folders.sort((a, b) => a.sortOrder - b.sortOrder) as T;
  }

  if (command === 'dstu_folder_get_tree') {
    return fallbackFolderTree(fallbackDstuFolders()) as T;
  }

  if (command === 'dstu_folder_get_items') {
    const store = fallbackDstuFolders();
    const folderId = fallbackFolderId(args?.folderId);
    return store.items.filter(item => item.folderId === folderId).sort((a, b) => a.sortOrder - b.sortOrder) as T;
  }

  if (command === 'dstu_folder_reorder') {
    const store = fallbackDstuFolders();
    const folderIds = Array.isArray(args?.folderIds) ? args.folderIds.filter((item): item is string => typeof item === 'string') : [];
    folderIds.forEach((folderId, index) => {
      const folder = store.folders.find(item => item.id === folderId);
      if (folder) folder.sortOrder = index;
    });
    saveFallbackDstuFolders(store);
    return undefined as T;
  }

  if (command === 'dstu_folder_reorder_items') {
    const store = fallbackDstuFolders();
    const itemIds = Array.isArray(args?.itemIds) ? args.itemIds.filter((item): item is string => typeof item === 'string') : [];
    itemIds.forEach((itemId, index) => {
      const item = store.items.find(entry => entry.itemId === itemId || entry.id === itemId);
      if (item) item.sortOrder = index;
    });
    saveFallbackDstuFolders(store);
    return undefined as T;
  }

  if (command === 'dstu_folder_get_breadcrumbs') {
    const store = fallbackDstuFolders();
    const folderId = typeof args?.folderId === 'string' ? args.folderId : '';
    const path = fallbackFolderPath(store, folderId);
    const titles = path.split('/').filter(Boolean);
    return titles.map((name, index) => ({ id: index === titles.length - 1 ? folderId : name, name })) as T;
  }

  if (command === 'dstu_parse_path') {
    const fullPath = typeof args?.path === 'string' && args.path.trim() ? `/${args.path.split('/').filter(Boolean).join('/')}` : '/';
    const parts = fullPath.split('/').filter(Boolean);
    const resourceId = parts.length > 0 ? parts[parts.length - 1] : null;
    const resourceType = resourceId ? fallbackDstuType(resourceId) : null;
    return {
      fullPath,
      folderPath: parts.length > 1 ? `/${parts.slice(0, -1).join('/')}` : resourceId ? '/' : null,
      resourceId,
      id: resourceId,
      resourceType,
      isRoot: fullPath === '/',
      isVirtual: parts[0]?.startsWith('@') ?? false,
      virtualType: parts[0]?.startsWith('@') ? parts[0].slice(1) : undefined,
    } as T;
  }

  if (command === 'dstu_build_path') {
    const store = fallbackDstuFolders();
    const resourceId = typeof args?.resourceId === 'string' ? args.resourceId : '';
    return fallbackResourcePath(store, fallbackFolderId(args?.folderId), resourceId) as T;
  }

  if (command === 'dstu_move_to_folder') {
    const store = fallbackDstuFolders();
    const resourceId = typeof args?.resourceId === 'string' ? args.resourceId : '';
    const folderId = fallbackFolderId(args?.targetFolderId);
    const itemType = fallbackDstuType(resourceId);
    let item = store.items.find(entry => entry.itemId === resourceId);
    if (!item) {
      item = { id: `fi_${Date.now().toString(36)}`, folderId, itemType, itemId: resourceId, sortOrder: 0, createdAt: Date.now() };
      store.items.push(item);
    }
    item.folderId = folderId;
    item.cachedPath = fallbackResourcePath(store, folderId, resourceId);
    saveFallbackDstuFolders(store);
    return { id: resourceId, resourceType: itemType, folderId, folderPath: fallbackFolderPath(store, folderId), fullPath: item.cachedPath } as T;
  }

  if (command === 'dstu_batch_move') {
    const request = args?.request && typeof args.request === 'object' && !Array.isArray(args.request)
      ? args.request as Record<string, unknown>
      : {};
    const itemIds = Array.isArray(request.itemIds) ? request.itemIds.filter((item): item is string => typeof item === 'string') : [];
    const successes = await Promise.all(itemIds.map(itemId => fallbackInvoke('dstu_move_to_folder', {
      resourceId: itemId,
      targetFolderId: request.targetFolderId,
    })));
    return { successes, failedItems: [], totalCount: itemIds.length } as T;
  }

  if (command === 'dstu_refresh_path_cache') {
    const store = fallbackDstuFolders();
    const resourceId = typeof args?.resourceId === 'string' ? args.resourceId : '';
    let count = 0;
    for (const item of store.items) {
      if (resourceId && item.itemId !== resourceId) continue;
      item.cachedPath = fallbackResourcePath(store, item.folderId, item.itemId);
      count += 1;
    }
    saveFallbackDstuFolders(store);
    return count as T;
  }

  if (command === 'dstu_get_path_by_id') {
    const store = fallbackDstuFolders();
    const resourceId = typeof args?.resourceId === 'string' ? args.resourceId : '';
    const folder = store.folders.find(item => item.id === resourceId);
    if (folder) return fallbackFolderPath(store, folder.id) as T;
    const item = store.items.find(entry => entry.itemId === resourceId);
    return (item?.cachedPath ?? `/${resourceId}`) as T;
  }

  if (command === 'dstu_folder_get_all_resources') {
    const folderId = typeof args?.folderId === 'string' ? args.folderId : '';
    return {
      folderId,
      folderTitle: folderId || 'Folder',
      path: folderId ? `/${folderId}` : '/',
      totalCount: 0,
      resources: [],
    } as T;
  }

  if (command === 'dstu_get_resource_by_path') {
    return null as T;
  }

  if (command === 'dstu_get_resource_location') {
    throw new Error('dstu_get_resource_location is not available outside a native runtime');
  }

  if (command === 'get_csv_preview' || command === 'import_questions_csv' || command === 'export_questions_csv') {
    throw new Error(`${command} is not available outside a native runtime`);
  }

  if (command === 'get_csv_exportable_fields') {
    return [
      ['content', '题干内容'],
      ['question_type', '题目类型'],
      ['options', '选项'],
      ['answer', '答案'],
      ['explanation', '解析'],
      ['difficulty', '难度'],
      ['tags', '标签'],
      ['images', '关联图片'],
      ['question_label', '题号'],
      ['user_answer', '用户答案'],
      ['is_correct', '是否正确'],
      ['attempt_count', '答题次数'],
      ['correct_count', '正确次数'],
      ['status', '学习状态'],
      ['is_favorite', '收藏'],
      ['user_note', '笔记'],
      ['created_at', '创建时间'],
      ['updated_at', '更新时间'],
    ] as T;
  }

  if (
    command === 'qbank_get_history' ||
    command === 'qbank_get_submissions' ||
    command === 'qbank_get_learning_trend' ||
    command === 'qbank_get_activity_heatmap' ||
    command === 'qbank_get_knowledge_stats'
  ) {
    return [] as T;
  }

  if (command === 'qbank_get_knowledge_stats_with_comparison') {
    return { current: [], previous: [] } as T;
  }

  if (command === 'qbank_get_check_in_calendar') {
    const request = args?.request && typeof args.request === 'object' && !Array.isArray(args.request)
      ? args.request as Record<string, unknown>
      : {};
    return {
      year: typeof request.year === 'number' ? request.year : new Date().getFullYear(),
      month: typeof request.month === 'number' ? request.month : new Date().getMonth() + 1,
      days: [],
      streak_days: 0,
      month_check_in_days: 0,
      month_total_questions: 0,
    } as T;
  }

  if (
    command === 'qbank_start_timed_practice' ||
    command === 'qbank_generate_mock_exam' ||
    command === 'qbank_submit_mock_exam' ||
    command === 'qbank_get_daily_practice' ||
    command === 'qbank_generate_paper'
  ) {
    throw new Error(`${command} is not available outside a native runtime`);
  }

  if (command === 'qbank_cancel_grading') {
    return true as T;
  }

  if (command === 'qbank_ai_grade') {
    throw new Error('qbank_ai_grade is not available outside a native runtime');
  }

  if (command === 'qbank_sync_check') {
    const syncConfig = fallbackQbankSyncConfig(args?.examId);
    return {
      sync_enabled: syncConfig.auto_sync,
      local_modified_count: 0,
      pending_conflict_count: 0,
      total_count: 0,
      synced_count: 0,
      sync_config: syncConfig,
    } as T;
  }

  if (command === 'qbank_get_sync_conflicts' || command === 'qbank_batch_resolve_conflicts') {
    return [] as T;
  }

  if (command === 'qbank_set_sync_enabled') {
    const examId = requireFallbackStringArg(command, args, 'examId');
    if (typeof args?.enabled !== 'boolean') {
      throw new Error(`Native fallback command ${command} requires boolean argument "enabled"`);
    }
    const configs = fallbackQbankSyncConfigs();
    configs[examId] = {
      ...fallbackQbankSyncConfig(examId),
      auto_sync: args.enabled,
    };
    saveFallbackQbankSyncConfigs(configs);
    return true as T;
  }

  if (command === 'qbank_update_sync_config') {
    const examId = requireFallbackStringArg(command, args, 'examId');
    const config = requireFallbackObjectArg(command, args, 'config');
    const configs = fallbackQbankSyncConfigs();
    configs[examId] = normalizeFallbackQbankSyncConfig(config);
    saveFallbackQbankSyncConfigs(configs);
    return true as T;
  }

  if (command === 'qbank_resolve_sync_conflict') {
    throw new Error('qbank_resolve_sync_conflict is not available outside a native runtime');
  }

  if (command === 'check_anki_connect_status') {
    return false as T;
  }

  if (command === 'get_anki_deck_names' || command === 'anki_get_deck_names') {
    return ['Default'] as T;
  }

  if (command === 'get_anki_model_names') {
    return ['Basic', 'Cloze'] as T;
  }

  if (command === 'get_model_adapter_options') {
    return fallbackModelAdapterOptions as T;
  }

  if (command === 'memory_get_config') {
    return {
      memoryRootFolderId: null,
      memoryRootFolderTitle: null,
      autoCreateSubfolders: true,
      defaultCategory: '通用',
      privacyMode: false,
      autoExtractFrequency: 'balanced',
    } as T;
  }

  if (
    command === 'review_plan_get_due' ||
    command === 'review_plan_get_due_with_filter' ||
    command === 'review_plan_list_by_exam'
  ) {
    return { plans: [], total: 0, has_more: false } as T;
  }

  if (command === 'review_plan_get_stats' || command === 'review_plan_refresh_stats') {
    return fallbackReviewStats(args?.examId) as T;
  }

  if (command === 'review_plan_get_by_question' || command === 'review_plan_get') {
    return null as T;
  }

  if (command === 'review_plan_get_history' || command === 'review_plan_get_calendar_data') {
    return [] as T;
  }

  if (
    command === 'review_plan_create' ||
    command === 'review_plan_process' ||
    command === 'review_plan_suspend' ||
    command === 'review_plan_resume' ||
    command === 'review_plan_delete' ||
    command === 'review_plan_batch_create' ||
    command === 'review_plan_create_for_exam' ||
    command === 'review_plan_get_or_create'
  ) {
    throw new Error(`${command} is not available outside a native runtime`);
  }

  if (command === 'get_document_tasks' || command === 'get_document_cards') {
    return [] as T;
  }

  if (command === 'start_enhanced_document_processing') {
    throw new Error('start_enhanced_document_processing is not available outside a native runtime');
  }

  if (
    command === 'pause_document_processing' ||
    command === 'resume_document_processing' ||
    command === 'delete_document_session'
  ) {
    return true as T;
  }

  if (command === 'trigger_task_processing') {
    return undefined as T;
  }

  if (command === 'recover_stuck_document_tasks') {
    return 0 as T;
  }

  if (command === 'get_document_processing_state' || command === 'get_document_state') {
    return {
      status: 'pending',
      total_tasks: 0,
      completed_tasks: 0,
      failed_tasks: 0,
      paused_tasks: 0,
    } as T;
  }

  if (command === 'notes_list_tags' || command === 'notes_search') {
    return [] as T;
  }

  if (
    command === 'canvas_note_read' ||
    command === 'canvas_note_append' ||
    command === 'canvas_note_replace' ||
    command === 'canvas_note_set'
  ) {
    throw new Error(`${command} is not available outside a native runtime`);
  }

  if (command === 'notes_get_pref') {
    return storageGet(`notes.pref.${key}`) as T;
  }

  if (command === 'notes_set_pref') {
    storageSet(`notes.pref.${key}`, String(args?.value ?? ''));
    return true as T;
  }

  if (command === 'get_app_data_dir') {
    return '' as T;
  }

  if (command === 'ensure_debug_log_dir') {
    return '' as T;
  }

  if (command === 'open_logs_folder' || command === 'report_frontend_log') {
    return undefined as T;
  }

  if (command === 'save_webview_settings') {
    const settings = args?.settings && typeof args.settings === 'object' ? args.settings : {};
    return {
      success: true,
      path: '',
      size: JSON.stringify(settings).length,
    } as T;
  }

  if (command === 'get_statistics') {
    return fallbackStatistics() as T;
  }

  if (command === 'get_enhanced_statistics') {
    return {
      basic_stats: fallbackStatistics(),
      image_stats: { total_files: 0, total_size_bytes: 0 },
      recent_additions: 0,
      quality_score: 0,
      monthly_trend: [],
      timestamp: new Date().toISOString(),
    } as T;
  }

  if (command === 'read_file_bytes') {
    throw new Error('read_file_bytes is not available outside a native runtime');
  }

  if (command === 'read_file_text') {
    throw new Error('read_file_text is not available outside a native runtime');
  }

  if (command === 'save_text_to_file') {
    throw new Error('save_text_to_file is not available outside a native runtime');
  }

  if (command === 'get_file_size') {
    throw new Error('get_file_size is not available outside a native runtime');
  }

  if (command === 'copy_file') {
    throw new Error('copy_file is not available outside a native runtime');
  }

  if (command === 'get_image_as_base64') {
    throw new Error('get_image_as_base64 is not available outside a native runtime');
  }

  throw new Error(`Native command is not available outside a native runtime: ${command}`);
}

export async function invoke<T = unknown>(command: NativeCommand, args?: NativeArgs): Promise<T> {
  if (isInjectedNativeRuntime()) {
    return window.__DEEP_STUDENT_NATIVE__!.invoke!<T>(command, args);
  }

  if (isWailsRuntime()) {
    return invokeWails<T>(command, args);
  }

  if (shouldUseFallbackBeforeTauri(command)) {
    return fallbackInvoke<T>(command, args);
  }

  if (isTauriRuntime()) {
    return tauriInvoke<T>(command, args);
  }

  return fallbackInvoke<T>(command, args);
}

export async function getSetting(key: string): Promise<string | null> {
  return invoke<string | null>('get_setting', { key });
}

export async function getSettings(keys: string[]): Promise<Record<string, string>> {
  if (isWailsRuntime() || isInjectedNativeRuntime()) {
    return invoke<Record<string, string>>('get_settings', { keys });
  }

  const entries = await Promise.all(keys.map(async key => {
    const value = await getSetting(key);
    return [key, value] as const;
  }));
  return Object.fromEntries(entries.filter((entry): entry is readonly [string, string] => entry[1] != null));
}

export async function saveSetting(key: string, value: string): Promise<void> {
  await invoke<void>('save_setting', { key, value });
}

export async function saveSettings(values: Record<string, string>): Promise<void> {
  if (isWailsRuntime() || isInjectedNativeRuntime()) {
    await invoke<void>('save_settings', { values });
    return;
  }

  await Promise.all(Object.entries(values).map(([key, value]) => saveSetting(key, value)));
}

export async function deleteSetting(key: string): Promise<void> {
  await invoke<void>('delete_setting', { key });
}

export async function getAppDataDir(): Promise<string> {
  return invoke<string>('get_app_data_dir');
}

export async function ensureDebugLogDir(): Promise<string> {
  return invoke<string>('ensure_debug_log_dir');
}

export async function openLogsFolder(logType: string): Promise<void> {
  await invoke<void>('open_logs_folder', { logType });
}

export async function reportFrontendLog(payload: Record<string, unknown>): Promise<void> {
  await invoke<void>('report_frontend_log', { payload });
}

export async function readFileBytes(path: string): Promise<number[]> {
  return invoke<number[]>('read_file_bytes', { path });
}

export async function readFileText(path: string): Promise<string> {
  return invoke<string>('read_file_text', { path });
}

export async function saveTextToFile(path: string, content: string): Promise<void> {
  await invoke<void>('save_text_to_file', { path, content });
}

export async function getFileSize(path: string): Promise<number> {
  return invoke<number>('get_file_size', { path });
}

export async function copyFile(sourcePath: string, destPath: string): Promise<void> {
  await invoke<void>('copy_file', {
    sourcePath,
    destPath,
    source_path: sourcePath,
    dest_path: destPath,
  });
}

export async function getImageAsBase64(relativePath: string): Promise<string> {
  return invoke<string>('get_image_as_base64', {
    relativePath,
    relative_path: relativePath,
  });
}

export const native = {
  invoke,
  settings: {
    get: getSetting,
    getMany: getSettings,
    save: saveSetting,
    saveMany: saveSettings,
    delete: deleteSetting,
  },
  system: {
    getAppDataDir,
    ensureDebugLogDir,
    openLogsFolder,
    reportFrontendLog,
  },
  files: {
    readBytes: readFileBytes,
    readText: readFileText,
    saveText: saveTextToFile,
    getSize: getFileSize,
    copy: copyFile,
    getImageAsBase64,
  },
  runtime: {
    isTauri: isTauriRuntime,
    isInjected: isInjectedNativeRuntime,
    isWails: isWailsRuntime,
  },
};
