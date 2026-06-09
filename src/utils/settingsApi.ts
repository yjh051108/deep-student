import {
  deleteSetting as nativeDeleteSetting,
  getSetting as nativeGetSetting,
  invoke as nativeInvoke,
  readFileText as nativeReadFileText,
  saveSetting as nativeSaveSetting,
  saveTextToFile as nativeSaveTextToFile,
} from '../runtime/native';

export interface SearchEngineTestResponse {
  ok?: boolean;
  message?: string;
  response_time?: number;
  test_query?: string;
  error_details?: string;
  results_count?: number;
}

export interface SearchEngineHealthReport {
  results: Record<string, {
    name: string;
    status: 'success' | 'failed' | 'not_configured';
    message: string;
    elapsed_ms: number;
    results_count?: number;
  }>;
  summary: {
    total: number;
    configured: number;
    success: number;
    failed: number;
  };
  timestamp: string;
}

export async function saveSetting(key: string, value: string): Promise<void> {
  try {
    await nativeSaveSetting(key, value);
  } catch (error) {
    console.error('Failed to save setting:', error);
    throw new Error(`Failed to save setting: ${error}`);
  }
}

export async function getSetting(key: string): Promise<string | null> {
  try {
    return await nativeGetSetting(key);
  } catch (error) {
    console.error('Failed to get setting:', error);
    const fallback = typeof localStorage !== 'undefined' ? localStorage.getItem(key) : null;
    return fallback;
  }
}

export async function deleteSetting(key: string): Promise<void> {
  try {
    await nativeDeleteSetting(key);
  } catch (error) {
    console.error('Failed to delete setting:', error);
    if (typeof localStorage !== 'undefined') {
      localStorage.removeItem(key);
    }
  }
}

// Deep Research command wrappers were retired with the lean Go rewrite. Do not
// reintroduce direct Tauri research_* invokes here; rebuild reachable research
// workflows as explicit Go/Wails product APIs instead.

// Utilities
export async function saveTextToFile(path: string, content: string): Promise<void> {
  await nativeSaveTextToFile(path, content);
}

export async function readFileText(path: string): Promise<string> {
  return await nativeReadFileText(path);
}

export async function reloadMcpClient(): Promise<{ success: boolean; message?: string; error?: string }> {
  try {
    const response = await nativeInvoke<{ success: boolean; message?: string; error?: string }>('reload_mcp_client');
    return response;
  } catch (error) {
    console.error('Failed to reload MCP client:', error);
    throw new Error(`Failed to reload MCP client: ${error}`);
  }
}

// 外部搜索连通性测试
export async function testWebSearchConnectivity(engine?: string): Promise<any> {
  try {
    const response = await nativeInvoke<any>('test_web_search_connectivity', { engine: engine || null });
    return response;
  } catch (error) {
    console.error('Failed to test external search connection:', error);
    throw new Error(`Failed to test external search connection: ${error}`);
  }
}

export async function testSearchEngine(engine: string): Promise<SearchEngineTestResponse> {
  try {
    return await nativeInvoke<SearchEngineTestResponse>('test_search_engine', { engine });
  } catch (error) {
    console.error('Failed to test search engine:', error);
    throw new Error(`Failed to test search engine: ${error}`);
  }
}

// MCP 状态与工具
export async function getMcpStatus(): Promise<any> {
  try {
    return await nativeInvoke<any>('get_mcp_status');
  } catch (error) {
    console.error('Failed to get MCP status:', error);
    throw new Error(`Failed to get MCP status: ${error}`);
  }
}

export async function getMcpTools(): Promise<Array<{ name: string; description?: string; input_schema: any }>> {
  try {
    return await nativeInvoke<Array<{ name: string; description?: string; input_schema: any }>>('get_mcp_tools');
  } catch (error) {
    console.error('Failed to get MCP tools:', error);
    throw new Error(`Failed to get MCP tools: ${error}`);
  }
}

export async function testAllSearchEngines(): Promise<SearchEngineHealthReport> {
  try {
    return await nativeInvoke<SearchEngineHealthReport>('test_all_search_engines');
  } catch (error) {
    console.error('Search engine health check failed:', error);
    throw new Error(`Search engine health check failed: ${error}`);
  }
}

export async function testApiConnection(
  apiKey: string,
  apiBase: string,
  model?: string,
  options: {
    apiProtocol?: string | null;
    supportsOpenAIResponses?: boolean | null;
    vendorId?: string | null;
  } = {}
): Promise<boolean> {
  try {
    const response = await nativeInvoke<boolean>('test_api_connection', {
      api_key: apiKey,
      apiKey,
      api_base: apiBase,
      apiBase,
      api_protocol: options.apiProtocol ?? null,
      apiProtocol: options.apiProtocol ?? null,
      supports_openai_responses: options.supportsOpenAIResponses ?? null,
      supportsOpenAIResponses: options.supportsOpenAIResponses ?? null,
      model: model || null,
      vendor_id: options.vendorId ?? null,
      vendorId: options.vendorId ?? null,
    });
    return response;
  } catch (error) {
    console.error('Failed to test API connection:', error);
    throw new Error(`Failed to test API connection: ${error}`);
  }
}

// 统计信息API
export async function getStatistics(): Promise<any> {
  try {
    const response = await nativeInvoke<any>('get_statistics');
    return response;
  } catch (error) {
    console.error('Failed to get statistics:', error);
    throw new Error(`Failed to get statistics: ${error}`);
  }
}

// 获取增强版统计信息（包含所有模块）
export async function getEnhancedStatistics(): Promise<any> {
  try {
    const response = await nativeInvoke<any>('get_enhanced_statistics');
    return response;
  } catch (error) {
    console.error('Failed to get enhanced statistics:', error);
    // 降级到基础统计
    return getStatistics();
  }
}

// 文档31清理：getSupportedSubjects 已彻底删除
