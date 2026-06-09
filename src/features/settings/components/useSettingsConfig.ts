import { useCallback } from 'react';
import { ApiConfig, ModelAssignments } from '@/types';
import { showGlobalNotification } from '@/components/UnifiedNotification';
import { getErrorMessage } from '@/utils/errorUtils';
import { debugLog } from '@/debug-panel/debugMasterSwitch';
import { normalizeMcpToolList } from './mcpUtils';
import { DEFAULT_STDIO_ARGS, DEFAULT_STDIO_ARGS_STORAGE } from './constants';
import type { ThemeMode, ThemePalette } from '@/hooks/useTheme';
import type { UseSettingsConfigDeps } from './hookDepsTypes';
import type { SystemConfig } from './types';
import { BUILTIN_SERVER_ID } from '@/mcp/builtinMcpServer';
import {
  getSettings,
  saveSettings,
  invoke as nativeInvoke,
  isInjectedNativeRuntime,
  isTauriRuntime,
  isWailsRuntime,
} from '@/runtime/native';

const console = debugLog as Pick<typeof debugLog, 'log' | 'warn' | 'error' | 'info' | 'debug'>;
const hasNativeRuntime =
  typeof window !== 'undefined' &&
  (isTauriRuntime() || isWailsRuntime() || isInjectedNativeRuntime());
const invoke = hasNativeRuntime ? nativeInvoke : null;

const SETTINGS_DEFAULTS = {
  'auto_save': 'true',
  'theme': 'light',
  'theme_palette': 'default',
  'debug_mode': 'false',
  'rag_enabled': 'false',
  'rag_top_k': '5',
  'anki_connect_enabled': 'false',
  'mcp.transport.command': 'npx',
  'mcp.transport.args': DEFAULT_STDIO_ARGS_STORAGE,
  'mcp.transport.type': 'stdio',
  'mcp.transport.url': 'ws://localhost:8000',
  'mcp.tools.advertise_all_tools': 'false',
  'mcp.tools.whitelist': 'read_file, write_file, list_directory',
  'mcp.tools.blacklist': 'delete_file, execute_command, rm, sudo',
  'mcp.tools.list': '[]',
  'mcp.performance.timeout_ms': '15000',
  'mcp.performance.rate_limit_per_second': '10',
  'mcp.performance.cache_max_size': '500',
  'mcp.performance.cache_ttl_ms': '300000',
  'web_search.engine': '',
  'web_search.timeout_ms': '15000',
  'web_search.api_key.google_cse': '',
  'web_search.google_cse.cx': '',
  'web_search.api_key.serpapi': '',
  'web_search.api_key.tavily': '',
  'web_search.api_key.brave': '',
  'web_search.searxng.endpoint': '',
  'web_search.searxng.api_key': '',
  'web_search.api_key.zhipu': '',
  'web_search.api_key.bocha': '',
  'web_search.site_whitelist': '',
  'web_search.site_blacklist': '',
  'web_search.inject.snippet_max_chars': '180',
  'web_search.inject.total_max_chars': '1900',
} as const;

type SettingsKey = keyof typeof SETTINGS_DEFAULTS;
const SETTINGS_KEYS = Object.keys(SETTINGS_DEFAULTS) as SettingsKey[];

const readSettingsWithDefaults = async (): Promise<Record<SettingsKey, string>> => {
  const values = await getSettings([...SETTINGS_KEYS]).catch(() => ({}));
  return Object.fromEntries(
    SETTINGS_KEYS.map(key => [key, values[key] ?? SETTINGS_DEFAULTS[key]]),
  ) as Record<SettingsKey, string>;
};

const writeSetting = (key: SettingsKey, value: string): Promise<void> =>
  saveSettings({ [key]: value });

const normalizeThemeMode = (value: unknown): ThemeMode => {
  if (value === 'dark' || value === 'auto') return value;
  return 'light';
};

export function useSettingsConfig(deps: UseSettingsConfigDeps) {
  const { setLoading, configLoadedRef, setExtra, setActiveTab, activeTab, modelAssignments, vendors, modelProfiles, resolvedApiConfigs, refreshVendors, refreshProfiles, refreshApiConfigsFromBackend, persistAssignments, saving, setSaving, t, config, setConfig, loading, updateIndicatorRaf } = deps;

const normalizeThemePalette = (value: unknown): ThemePalette => {
  if (value === 'colorsafe' || value === 'accessible') return 'muted';
  const validPalettes: ThemePalette[] = ['default', 'purple', 'green', 'orange', 'pink', 'teal', 'muted', 'paper', 'custom'];
  if (validPalettes.includes(value as ThemePalette)) return value as ThemePalette;
  return 'default';
};

  const loadConfig = async () => {
    setLoading(true);
    try {
      if (invoke) {
        // 使用新的专用API配置管理命令
        const [apiConfigs, modelAssignments, settingsValues] = await Promise.all([
          invoke('get_api_configurations').catch(() => []) as Promise<ApiConfig[]>,
          invoke('get_model_assignments').catch(() => ({
            model2_config_id: null,
            anki_card_model_config_id: null,
            qbank_ai_grading_model_config_id: null,
            reranker_model_config_id: null,
            exam_sheet_ocr_model_config_id: null,
            translation_model_config_id: null,
            chat_title_model_config_id: null,
            // 多模态知识库模型（嵌入模型通过维度管理设置）
            vl_reranker_model_config_id: null,
            memory_decision_model_config_id: null,
            voice_input_asr_model_config_id: null,
            image_generation_model_config_id: null,
            translation_display_mode: null,
          })) as Promise<{
            model2_config_id: string | null,
            anki_card_model_config_id: string | null,
            qbank_ai_grading_model_config_id: string | null,
            reranker_model_config_id: string | null,
            exam_sheet_ocr_model_config_id: string | null,
            translation_model_config_id: string | null,
            chat_title_model_config_id: string | null,
            // 多模态知识库模型（嵌入模型通过维度管理设置）
            vl_reranker_model_config_id: string | null,
            memory_decision_model_config_id: string | null,
            voice_input_asr_model_config_id: string | null,
            image_generation_model_config_id: string | null,
            translation_display_mode: string | null,
          }>,
          readSettingsWithDefaults(),
        ]);

        const autoSave = settingsValues['auto_save'];
        const theme = settingsValues['theme'];
        const themePaletteSetting = settingsValues['theme_palette'];
        const debugMode = settingsValues['debug_mode'];
        const ragEnabled = settingsValues['rag_enabled'];
        const ragTopK = settingsValues['rag_top_k'];
        const ankiConnectEnabled = settingsValues['anki_connect_enabled'];
        const mcpCommand = settingsValues['mcp.transport.command'];
        const mcpArgs = settingsValues['mcp.transport.args'];
        const mcpTransportType = settingsValues['mcp.transport.type'];
        const mcpUrl = settingsValues['mcp.transport.url'];
        const mcpAdvertiseAll = settingsValues['mcp.tools.advertise_all_tools'];
        const mcpWhitelist = settingsValues['mcp.tools.whitelist'];
        const mcpBlacklist = settingsValues['mcp.tools.blacklist'];
        const mcpToolsJson = settingsValues['mcp.tools.list'];
        const mcpTimeoutMs = settingsValues['mcp.performance.timeout_ms'];
        const mcpRateLimit = settingsValues['mcp.performance.rate_limit_per_second'];
        const mcpCacheMax = settingsValues['mcp.performance.cache_max_size'];
        const mcpCacheTtlMs = settingsValues['mcp.performance.cache_ttl_ms'];
        const webEngine = settingsValues['web_search.engine'];
        const webTimeoutMs = settingsValues['web_search.timeout_ms'];
        const webGoogleKey = settingsValues['web_search.api_key.google_cse'];
        const webGoogleCx = settingsValues['web_search.google_cse.cx'];
        const webSerpKey = settingsValues['web_search.api_key.serpapi'];
        const webTavilyKey = settingsValues['web_search.api_key.tavily'];
        const webBraveKey = settingsValues['web_search.api_key.brave'];
        const webSearxngEndpoint = settingsValues['web_search.searxng.endpoint'];
        const webSearxngKey = settingsValues['web_search.searxng.api_key'];
        const webZhipuKey = settingsValues['web_search.api_key.zhipu'];
        const webBochaKey = settingsValues['web_search.api_key.bocha'];
        const webWhitelist = settingsValues['web_search.site_whitelist'];
        const webBlacklist = settingsValues['web_search.site_blacklist'];
        const webInjectSnippet = settingsValues['web_search.inject.snippet_max_chars'];
        const webInjectTotal = settingsValues['web_search.inject.total_max_chars'];

        // 处理API配置的字段映射（snake_case to camelCase）
        const mappedApiConfigs = (apiConfigs || []).map((c: ApiConfig) => ({
          ...c,
          maxOutputTokens: c.maxOutputTokens,
          temperature: c.temperature,
        }));

        const parsedMcpTimeout = (() => {
          const parsed = parseInt(mcpTimeoutMs || '15000', 10);
          return Number.isFinite(parsed) ? parsed : 15000;
        })();
        const parsedMcpRateLimit = (() => {
          const parsed = parseInt(mcpRateLimit || '10', 10);
          return Number.isFinite(parsed) ? parsed : 10;
        })();
        const parsedMcpCacheMax = (() => {
          const parsed = parseInt(mcpCacheMax || '500', 10);
          const val = Number.isFinite(parsed) ? parsed : 500;
          if (val <= 100) {
            writeSetting('mcp.performance.cache_max_size', '500').catch(() => {});
            return 500;
          }
          return val;
        })();
        const parsedMcpCacheTtl = (() => {
          const parsed = parseInt(mcpCacheTtlMs || '300000', 10);
          return Number.isFinite(parsed) && parsed >= 0 ? parsed : 300000;
        })();

        const parsedMcpTools = (() => {
          try {
            return JSON.parse(mcpToolsJson || '[]');
          } catch {
            return [];
          }
        })();
        const normalizedMcpTools = normalizeMcpToolList(parsedMcpTools);

        const migratedCommandSegments = (() => {
          if (typeof mcpCommand === 'string' && /@modelcontextprotocol\//.test(mcpCommand || '')) {
            const pieces = mcpCommand.split(' ').filter(Boolean);
            if (pieces.length > 1) {
              return pieces;
            }
          }
          return null;
        })();
        const normalizedMcpCommand = (() => {
          if (migratedCommandSegments && migratedCommandSegments.length > 0) {
            return migratedCommandSegments[0];
          }
          if (typeof mcpCommand === 'string' && mcpCommand.trim().length > 0) {
            return mcpCommand.trim();
          }
          return 'npx';
        })();
        const normalizedMcpArgsString = (() => {
          let argsArray: string[] = [];
          if (migratedCommandSegments && migratedCommandSegments.length > 1) {
            argsArray = migratedCommandSegments.slice(1);
          } else if (typeof mcpArgs === 'string' && mcpArgs.trim().length > 0) {
            argsArray = mcpArgs
              .split(',')
              .map(segment => segment.trim())
              .filter(Boolean);
          }
          if (argsArray.length === 0) {
            argsArray = [...DEFAULT_STDIO_ARGS];
          }
          return argsArray.join(',');
        })();

        const newConfig = {
          apiConfigs: mappedApiConfigs,
          model2ConfigId: modelAssignments?.model2_config_id || '',
          ankiCardModelConfigId: modelAssignments?.anki_card_model_config_id || '',
          qbank_ai_grading_model_config_id: modelAssignments?.qbank_ai_grading_model_config_id || '',
          rerankerModelConfigId: modelAssignments?.reranker_model_config_id || '',
          chat_title_model_config_id: modelAssignments?.chat_title_model_config_id || '',
          exam_sheet_ocr_model_config_id: modelAssignments?.exam_sheet_ocr_model_config_id || '',
          translation_model_config_id: modelAssignments?.translation_model_config_id || '',
          // 多模态知识库模型配置（嵌入模型通过维度管理设置）
          vl_reranker_model_config_id: modelAssignments?.vl_reranker_model_config_id || '',
          memory_decision_model_config_id: modelAssignments?.memory_decision_model_config_id || '',
          voice_input_asr_model_config_id: modelAssignments?.voice_input_asr_model_config_id || '',
          image_generation_model_config_id: modelAssignments?.image_generation_model_config_id || '',
          translation_display_mode: (modelAssignments?.translation_display_mode === 'streaming' ? 'streaming' : 'aligned') as 'aligned' | 'streaming',
          autoSave: (autoSave || 'true') === 'true',
          theme: normalizeThemeMode(theme),
          themePalette: normalizeThemePalette(themePaletteSetting),
          debugMode: (debugMode || 'false') === 'true',
          ragEnabled: (ragEnabled || 'false') === 'true',
          ragTopK: parseInt(ragTopK || '5', 10),
          ankiConnectEnabled: (ankiConnectEnabled || 'false') === 'true',

          // MCP 工具协议设置（不再设置全局启用项）
          mcpCommand: normalizedMcpCommand,
          mcpTransportType: (mcpTransportType === 'websocket' ? 'websocket' : 'stdio') as 'stdio' | 'websocket',
          mcpUrl: mcpUrl || 'ws://localhost:8000',
          mcpArgs: normalizedMcpArgsString,
          mcpAdvertiseAll: (mcpAdvertiseAll || 'false') === 'true',
          mcpWhitelist: mcpWhitelist || 'read_file, write_file, list_directory',
          mcpBlacklist: mcpBlacklist || 'delete_file, execute_command, rm, sudo',
          mcpTimeoutMs: parsedMcpTimeout,
          mcpRateLimit: parsedMcpRateLimit,
          mcpCacheMax: parsedMcpCacheMax,
          mcpCacheTtlMs: parsedMcpCacheTtl,
          mcpTools: normalizedMcpTools,

          // Web Search 设置（UI 层存储，仅供保存使用）
          // 为保持与其他页面一致，全部使用简单原生控件，不在此定义专门类型
          // 外部搜索设置（不再设置全局启用项）
          webSearchEngine: webEngine || '',
          webSearchTimeoutMs: parseInt(webTimeoutMs || '15000', 10),
          webSearchGoogleKey: webGoogleKey || '',
          webSearchGoogleCx: webGoogleCx || '',
          webSearchSerpApiKey: webSerpKey || '',
          webSearchTavilyKey: webTavilyKey || '',
          webSearchBraveKey: webBraveKey || '',
          webSearchSearxngEndpoint: webSearxngEndpoint || '',
          webSearchSearxngKey: webSearxngKey || '',
          webSearchZhipuKey: webZhipuKey || '',
          webSearchBochaKey: webBochaKey || '',
          webSearchWhitelist: webWhitelist || '',
          webSearchBlacklist: webBlacklist || '',
          webSearchInjectSnippetMax: parseInt(webInjectSnippet || '180', 10) || 180,
          webSearchInjectTotalMax: parseInt(webInjectTotal || '1900', 10) || 1900,
        };
        
        console.log('加载的配置:', {
          apiConfigs: newConfig.apiConfigs.length,
          model2ConfigId: newConfig.model2ConfigId,
          modelAssignments
        });
        
        setConfig(newConfig);
        // 🔧 标记 config 已成功加载，允许 auto-save
        if (configLoadedRef) configLoadedRef.current = true;

        // 注意：不要用后端存储的 theme/themePalette 覆盖前端 useTheme 的状态
        // useTheme 使用 localStorage 作为主题的 single source of truth
        // 后端存储可能是旧值，会导致从暗色模式意外切换回亮色模式
        // 相反，我们应该将前端的主题状态同步到 config 中（已在 useEffect 中处理）
      } else {
        // 浏览器环境 - 支持从旧键名迁移
        let savedConfig = localStorage.getItem('deep-student-config');
        if (!savedConfig) {
          // 尝试从旧键名迁移
          const oldConfig = localStorage.getItem('ai-mistake-manager-config');
          if (oldConfig) {
            savedConfig = oldConfig;
            // 保存到新键名
            localStorage.setItem('deep-student-config', oldConfig);
            // 删除旧键名
            localStorage.removeItem('ai-mistake-manager-config');
            console.log('✅ 已自动迁移配置从旧键名到新键名');
          }
        }
        if (savedConfig) {
          try {
            const parsed = JSON.parse(savedConfig) as Partial<SystemConfig> & { mcpServers?: unknown };
            const normalized = normalizeMcpToolList(parsed?.mcpTools ?? parsed?.mcpServers);
            setConfig(prev => ({
              ...prev,
              ...parsed,
              theme: normalizeThemeMode(parsed?.theme),
              themePalette: normalizeThemePalette(parsed?.themePalette),
              mcpTools: normalized,
            }));
            if (configLoadedRef) configLoadedRef.current = true;
          } catch (e) {
            console.error('Browser config load failed:', e);
          }
        }
      }
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      console.error('Config load failed:', error);
      showGlobalNotification('error', t('settings:mcp.load_config_failed', { error: errorMessage }));
    } finally {
      setLoading(false);
    }
  };
  const handleSave = useCallback(async (silent = false) => {
    setSaving(true);
    try {
      if (invoke) {
        await saveSettings({
          auto_save: config.autoSave.toString(),
          theme: config.theme,
          theme_palette: config.themePalette ?? 'default',
          rag_enabled: config.ragEnabled.toString(),
          rag_top_k: config.ragTopK.toString(),
          anki_connect_enabled: config.ankiConnectEnabled.toString(),
          debug_mode: config.debugMode.toString(),
          // MCP 工具协议设置保存（移除全局启用项）
          'mcp.transport.type': String(config.mcpTransportType || 'stdio'),
          'mcp.transport.command': config.mcpCommand,
          'mcp.transport.args': config.mcpArgs,
          'mcp.transport.url': String(config.mcpUrl || ''),
          'mcp.tools.advertise_all_tools': config.mcpAdvertiseAll.toString(),
          'mcp.tools.whitelist': config.mcpWhitelist,
          'mcp.tools.blacklist': config.mcpBlacklist,
          'mcp.performance.timeout_ms': String(config.mcpTimeoutMs ?? 15000),
          'mcp.performance.rate_limit_per_second': String(config.mcpRateLimit ?? 10),
          'mcp.performance.cache_max_size': String(config.mcpCacheMax ?? 500),
          'mcp.performance.cache_ttl_ms': String(config.mcpCacheTtlMs ?? 300000),
          // 保存多工具配置（过滤掉内置服务器）
          'mcp.tools.list': JSON.stringify((config.mcpTools || []).filter(s => s.id !== BUILTIN_SERVER_ID)),
          // 强制使用前端SDK模式
          'mcp.mode': 'frontend',

          // Web Search 设置保存
          // 外部搜索保存（移除全局启用项）
          'web_search.engine': config.webSearchEngine ?? '',
          'web_search.timeout_ms': String(config.webSearchTimeoutMs ?? 15000),
          'web_search.api_key.google_cse': config.webSearchGoogleKey ?? '',
          'web_search.google_cse.cx': config.webSearchGoogleCx ?? '',
          'web_search.api_key.serpapi': config.webSearchSerpApiKey ?? '',
          'web_search.api_key.tavily': config.webSearchTavilyKey ?? '',
          'web_search.api_key.brave': config.webSearchBraveKey ?? '',
          'web_search.searxng.endpoint': config.webSearchSearxngEndpoint ?? '',
          'web_search.searxng.api_key': config.webSearchSearxngKey ?? '',
          'web_search.api_key.zhipu': config.webSearchZhipuKey ?? '',
          'web_search.api_key.bocha': config.webSearchBochaKey ?? '',
          'web_search.site_whitelist': config.webSearchWhitelist ?? '',
          'web_search.site_blacklist': config.webSearchBlacklist ?? '',
          'web_search.inject.snippet_max_chars': String(config.webSearchInjectSnippetMax ?? 180),
          'web_search.inject.total_max_chars': String(config.webSearchInjectTotalMax ?? 1900),
        });
        if (!silent) {
          showGlobalNotification('success', t('settings:notifications.config_save_success'));
        }
        
        // 广播：API 配置已变更（仅非静默保存时广播，避免 auto-save 触发自身 refreshApiConfigsFromBackend 形成无限循环）
        if (!silent) {
          try {
            if (typeof window !== 'undefined' && typeof window.dispatchEvent === 'function') {
              window.dispatchEvent(new CustomEvent('api_configurations_changed'));
            }
          } catch {}
        }

        // 触发设置变更事件，通知其他组件
        // 静默保存（auto-save）时不标记 mcpChanged，避免每次 auto-save 都触发 MCP bootstrap 全链路
        window.dispatchEvent(new CustomEvent('systemSettingsChanged', { 
          detail: { 
            ankiConnectEnabled: config.ankiConnectEnabled,
            theme: config.theme,
            themePalette: config.themePalette,
            debugMode: config.debugMode,
            mcpChanged: !silent,
          } 
        }));
      } else {
        localStorage.setItem('deep-student-config', JSON.stringify(config));
        if (!silent) {
          showGlobalNotification('success', t('settings:notifications.config_save_success_browser'));
        }
      }
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      console.error('保存配置失败:', error);
      if (silent) {
        showGlobalNotification('warning', t('settings:notifications.silent_save_failed'));
      } else {
        showGlobalNotification('error', t('settings:notifications.config_save_failed', { error: errorMessage }));
      }
    } finally {
      setSaving(false);
    }
  }, [config, invoke]);

  // 仅更新模型分配的某一个字段：读取后端当前 assignments 合并，再保存，避免空字段覆盖。
  const saveSingleAssignmentField = useCallback(
    async (field: keyof ModelAssignments, value: string | null) => {
      const merged: ModelAssignments = { ...modelAssignments, [field]: value };
      try {
        await persistAssignments(merged);
        return merged;
      } catch (error) {
        const errorMessage = getErrorMessage(error);
        console.error('Save model assignment failed:', error);
        showGlobalNotification('error', t('settings:mcp.save_model_assignment_failed', { error: errorMessage }));
        throw error;
      }
    },
    [modelAssignments, persistAssignments]
  );

  // 更新标签页切换处理函数，添加动画效果
  const handleTabChange = async (newTab: string) => {
    if (!loading) {
      // 在切换标签页前先保存当前配置
      await handleSave(true);
    }
    setActiveTab(newTab);
    
    // 更新指示器位置
    updateIndicatorRaf(newTab);
  };

  return { loadConfig, handleSave, saveSingleAssignmentField, handleTabChange };
}
