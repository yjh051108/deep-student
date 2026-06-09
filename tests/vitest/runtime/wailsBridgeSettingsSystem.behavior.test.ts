import { afterEach, describe, expect, it, vi } from 'vitest';

const mockSetBackupConfig = vi.hoisted(() => vi.fn());
const mockGetBackupConfig = vi.hoisted(() => vi.fn());
const mockCheckAPIConfigStatus = vi.hoisted(() => vi.fn());
const mockRestoreDefaultAPIConfigs = vi.hoisted(() => vi.fn());
const mockTestAPIConnection = vi.hoisted(() => vi.fn());
const mockTestOCREngine = vi.hoisted(() => vi.fn());
const mockTestSearchEngine = vi.hoisted(() => vi.fn());
const mockTestWebSearchConnectivity = vi.hoisted(() => vi.fn());
const mockTestAllSearchEngines = vi.hoisted(() => vi.fn());
const mockGetProviderStrategiesConfig = vi.hoisted(() => vi.fn());
const mockSaveProviderStrategiesConfig = vi.hoisted(() => vi.fn());
const mockGetStatistics = vi.hoisted(() => vi.fn());
const mockGetEnhancedStatistics = vi.hoisted(() => vi.fn());
const mockGetMemoryConfig = vi.hoisted(() => vi.fn());
const mockGetModelAdapterOptions = vi.hoisted(() => vi.fn());
const mockSaveWebviewSettings = vi.hoisted(() => vi.fn());
const mockCheckConnectStatus = vi.hoisted(() => vi.fn());
const mockListDeckNames = vi.hoisted(() => vi.fn());
const mockListModelNames = vi.hoisted(() => vi.fn());

vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/settingsservice', () => ({
  SetBackupConfig: mockSetBackupConfig,
  GetBackupConfig: mockGetBackupConfig,
  CheckAPIConfigStatus: mockCheckAPIConfigStatus,
  RestoreDefaultAPIConfigs: mockRestoreDefaultAPIConfigs,
  TestAPIConnection: mockTestAPIConnection,
  TestOCREngine: mockTestOCREngine,
  TestSearchEngine: mockTestSearchEngine,
  TestWebSearchConnectivity: mockTestWebSearchConnectivity,
  TestAllSearchEngines: mockTestAllSearchEngines,
  GetProviderStrategiesConfig: mockGetProviderStrategiesConfig,
  SaveProviderStrategiesConfig: mockSaveProviderStrategiesConfig,
  GetStatistics: mockGetStatistics,
  GetEnhancedStatistics: mockGetEnhancedStatistics,
  GetMemoryConfig: mockGetMemoryConfig,
  GetModelAdapterOptions: mockGetModelAdapterOptions,
}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/skillservice', () => ({}));

vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/systemservice', () => ({
  SaveWebviewSettings: mockSaveWebviewSettings,
}));

vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/ankiservice', () => ({
  CheckConnectStatus: mockCheckConnectStatus,
  ListDeckNames: mockListDeckNames,
  ListModelNames: mockListModelNames,
}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/chatservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/dstuservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/fileservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/notesservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/qbankservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/reviewplanservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/todoservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/vfsservice', () => ({}));

function setWindowProperty(name: string, value: unknown): () => void {
  const previous = Object.getOwnPropertyDescriptor(window, name);
  Object.defineProperty(window, name, {
    configurable: true,
    writable: true,
    value,
  });

  return () => {
    if (previous) {
      Object.defineProperty(window, name, previous);
    } else {
      delete (window as unknown as Record<string, unknown>)[name];
    }
  };
}

describe('wails bridge settings/system payload forwarding', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('forwards backup config and webview settings payloads unchanged', async () => {
    const { invokeWails } = await import('@/runtime/wailsBridge');
    const backupConfig = {
      backupDirectory: null,
      autoBackupEnabled: true,
      autoBackupIntervalHours: 12,
      maxBackupCount: null,
      slimBackup: true,
      backupTiers: ['core', 'important'],
    };
    const webviewSettings = {
      theme: 'dark',
      zoom: 1.2,
      nested: { compact: true },
    };

    mockGetBackupConfig.mockResolvedValue(backupConfig);
    mockSetBackupConfig.mockResolvedValue(undefined);
    mockCheckAPIConfigStatus.mockResolvedValue({ config_count: 0, enabled_count: 0, has_assignments: false, needs_recovery: true });
    mockRestoreDefaultAPIConfigs.mockResolvedValue('ok');
    mockTestAPIConnection.mockResolvedValue(true);
    mockTestOCREngine.mockResolvedValue({
      engineType: 'paddle_ocr_vl',
      engineName: 'Paddle UI',
      text: 'recognized text',
      regions: [{ text: 'recognized text', bbox: null, label: 'text' }],
      elapsedMs: 10,
      success: true,
      error: null,
    });
    mockTestSearchEngine.mockResolvedValue({ ok: true, message: 'ok', response_time: 12 });
    mockTestWebSearchConnectivity.mockResolvedValue({ success: true, usage: { provider: 'searxng' } });
    mockTestAllSearchEngines.mockResolvedValue({
      results: { searxng: { name: 'SearXNG', status: 'success', message: '连接成功', elapsed_ms: 12 } },
      summary: { total: 7, configured: 1, success: 1, failed: 0 },
      timestamp: '2026-06-08T00:00:00Z',
    });
    const providerStrategies = {
      default: { timeout_ms: 8000, max_retries: 2 },
      searxng: { timeout_ms: 20000, max_retries: 1 },
    };
    mockGetProviderStrategiesConfig.mockResolvedValue({
      provider_strategies: providerStrategies,
      config_keys: { provider_strategies: 'web_search.provider_strategies' },
    });
    mockSaveProviderStrategiesConfig.mockResolvedValue(true);
    mockGetStatistics.mockResolvedValue({ total_mistakes: 0, total_reviews: 0, type_stats: {}, tag_stats: {}, recent_mistakes: [] });
    mockGetEnhancedStatistics.mockResolvedValue({
      basic_stats: { total_mistakes: 0, total_reviews: 0, type_stats: {}, tag_stats: {}, recent_mistakes: [] },
      image_stats: { total_files: 1, total_size_bytes: 5 },
      timestamp: '2026-06-08T00:00:00Z',
    });
    mockGetMemoryConfig.mockResolvedValue({
      memoryRootFolderId: null,
      memoryRootFolderTitle: null,
      autoCreateSubfolders: true,
      defaultCategory: '通用',
      privacyMode: false,
      autoExtractFrequency: 'balanced',
    });
    mockGetModelAdapterOptions.mockResolvedValue([
      { value: 'general', label: 'OpenAI Compatible', description: 'standard', is_default: true },
      { value: 'deepseek', label: 'DeepSeek', description: 'reasoning', is_default: true },
    ]);
    mockSaveWebviewSettings.mockResolvedValue({ success: true, path: 'webview_settings.json', size: 42 });
    mockCheckConnectStatus.mockResolvedValue(true);
    mockListDeckNames.mockResolvedValue(['Default', 'Biology']);
    mockListModelNames.mockResolvedValue(['Basic', 'Cloze']);

    await expect(invokeWails('check_anki_connect_status')).resolves.toBe(true);
    await expect(invokeWails('get_anki_deck_names')).resolves.toEqual(['Default', 'Biology']);
    await expect(invokeWails('anki_get_deck_names')).resolves.toEqual(['Default', 'Biology']);
    await expect(invokeWails('get_anki_model_names')).resolves.toEqual(['Basic', 'Cloze']);
    await expect(invokeWails('get_backup_config')).resolves.toBe(backupConfig);
    await invokeWails('set_backup_config', { config: backupConfig });
    await expect(invokeWails('check_api_config_status')).resolves.toEqual({
      config_count: 0,
      enabled_count: 0,
      has_assignments: false,
      needs_recovery: true,
    });
    await expect(invokeWails('restore_default_api_configs')).resolves.toBe('ok');
    await expect(invokeWails('test_api_connection', {
      api_key: 'masked',
      apiBase: 'https://api.example.test/v1',
      api_protocol: 'openai_chat_completions',
      supportsOpenAIResponses: false,
      model: 'model-a',
      vendor_id: 'vendor-a',
    })).resolves.toBe(true);
    await expect(invokeWails('test_api_connection', {
      apiKey: 'masked-2',
      api_base: 'https://api.responses.test/v1',
      apiProtocol: 'openai_responses',
      supports_openai_responses: true,
      model: 'gpt-5',
      vendorId: 'vendor-b',
    })).resolves.toBe(true);
    const ocrRequest = {
      imageBase64: 'data:image/png;base64,AQID',
      engineType: 'paddle_ocr_vl',
      configId: 'api_paddle',
    };
    await expect(invokeWails('test_ocr_engine', { request: ocrRequest })).resolves.toEqual({
      engineType: 'paddle_ocr_vl',
      engineName: 'Paddle UI',
      text: 'recognized text',
      regions: [{ text: 'recognized text', bbox: null, label: 'text' }],
      elapsedMs: 10,
      success: true,
      error: null,
    });
    await expect(invokeWails('test_search_engine', { engine: 'searxng' })).resolves.toEqual({
      ok: true,
      message: 'ok',
      response_time: 12,
    });
    await expect(invokeWails('test_web_search_connectivity', { engine: 'searxng' })).resolves.toEqual({
      success: true,
      usage: { provider: 'searxng' },
    });
    await expect(invokeWails('test_all_search_engines')).resolves.toEqual({
      results: { searxng: { name: 'SearXNG', status: 'success', message: '连接成功', elapsed_ms: 12 } },
      summary: { total: 7, configured: 1, success: 1, failed: 0 },
      timestamp: '2026-06-08T00:00:00Z',
    });
    await expect(invokeWails('get_provider_strategies_config')).resolves.toEqual({
      provider_strategies: providerStrategies,
      config_keys: { provider_strategies: 'web_search.provider_strategies' },
    });
    await expect(invokeWails('save_provider_strategies_config', { strategies: providerStrategies })).resolves.toBe(true);
    await expect(invokeWails('get_statistics')).resolves.toEqual({
      total_mistakes: 0,
      total_reviews: 0,
      type_stats: {},
      tag_stats: {},
      recent_mistakes: [],
    });
    await expect(invokeWails('get_enhanced_statistics')).resolves.toEqual({
      basic_stats: { total_mistakes: 0, total_reviews: 0, type_stats: {}, tag_stats: {}, recent_mistakes: [] },
      image_stats: { total_files: 1, total_size_bytes: 5 },
      timestamp: '2026-06-08T00:00:00Z',
    });
    await expect(invokeWails('memory_get_config')).resolves.toEqual({
      memoryRootFolderId: null,
      memoryRootFolderTitle: null,
      autoCreateSubfolders: true,
      defaultCategory: '通用',
      privacyMode: false,
      autoExtractFrequency: 'balanced',
    });
    await expect(invokeWails('get_model_adapter_options')).resolves.toEqual([
      { value: 'general', label: 'OpenAI Compatible', description: 'standard', is_default: true },
      { value: 'deepseek', label: 'DeepSeek', description: 'reasoning', is_default: true },
    ]);
    await invokeWails('save_webview_settings', { settings: webviewSettings });

    expect(mockSetBackupConfig).toHaveBeenCalledWith(backupConfig);
    expect(mockTestAPIConnection).toHaveBeenCalledWith(
      'masked',
      'https://api.example.test/v1',
      'openai_chat_completions',
      false,
      'model-a',
      'vendor-a'
    );
    expect(mockTestAPIConnection).toHaveBeenCalledWith(
      'masked-2',
      'https://api.responses.test/v1',
      'openai_responses',
      true,
      'gpt-5',
      'vendor-b'
    );
    expect(mockTestOCREngine).toHaveBeenCalledWith({
      imageBase64: 'data:image/png;base64,AQID',
      engineType: 'paddle_ocr_vl',
      configId: 'api_paddle',
    });
    expect(mockTestSearchEngine).toHaveBeenCalledWith('searxng');
    expect(mockTestWebSearchConnectivity).toHaveBeenCalledWith('searxng');
    expect(mockTestAllSearchEngines).toHaveBeenCalled();
    expect(mockGetProviderStrategiesConfig).toHaveBeenCalled();
    expect(mockSaveProviderStrategiesConfig).toHaveBeenCalledWith(providerStrategies);
    expect(mockGetStatistics).toHaveBeenCalled();
    expect(mockGetEnhancedStatistics).toHaveBeenCalled();
    expect(mockGetMemoryConfig).toHaveBeenCalled();
    expect(mockGetModelAdapterOptions).toHaveBeenCalled();
    expect(mockSaveWebviewSettings).toHaveBeenCalledWith(webviewSettings);
    expect(mockCheckConnectStatus).toHaveBeenCalled();
    expect(mockListDeckNames).toHaveBeenCalledTimes(2);
    expect(mockListModelNames).toHaveBeenCalled();
  });

  it('treats Wails v3 WebView bridge markers as native and avoids browser fallbacks', async () => {
    const restoreWails = setWindowProperty('_wails', {});
    const restoreChrome = setWindowProperty('chrome', {
      webview: {
        postMessage: vi.fn(),
      },
    });

    try {
      const { invoke, isWailsRuntime } = await import('@/runtime/native');
      const memoryConfig = {
        memoryRootFolderId: 'native-root',
        memoryRootFolderTitle: 'Native Memory',
        autoCreateSubfolders: false,
        defaultCategory: 'native-category',
        privacyMode: true,
        autoExtractFrequency: 'manual',
      };

      mockListModelNames.mockResolvedValue(['Wails Native Model']);
      mockGetMemoryConfig.mockResolvedValue(memoryConfig);

      expect(isWailsRuntime()).toBe(true);
      await expect(invoke('get_anki_model_names')).resolves.toEqual(['Wails Native Model']);
      await expect(invoke('memory_get_config')).resolves.toEqual(memoryConfig);

      expect(mockListModelNames).toHaveBeenCalled();
      expect(mockGetMemoryConfig).toHaveBeenCalled();
    } finally {
      restoreChrome();
      restoreWails();
    }
  });
});
