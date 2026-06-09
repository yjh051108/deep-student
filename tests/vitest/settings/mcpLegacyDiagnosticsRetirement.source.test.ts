import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('MCP legacy diagnostics retirement source contract', () => {
  const settingsApiSource = readFileSync(resolve(process.cwd(), 'src/utils/settingsApi.ts'), 'utf-8');
  const mcpEditorSource = readFileSync(
    resolve(process.cwd(), 'src/features/settings/components/McpEditorSection.tsx'),
    'utf-8'
  );

  it('keeps legacy MCP backend diagnostic command names out of settingsApi', () => {
    [
      'test_mcp_connection',
      'test_mcp_http',
      'test_mcp_sse',
      'test_mcp_websocket',
      'test_mcp_modelscope',
    ].forEach(commandName => {
      expect(settingsApiSource).not.toContain(commandName);
    });
  });

  it('keeps legacy MCP diagnostic facade exports out of settingsApi', () => {
    [
      'testMcpConnection',
      'testMcpHttp',
      'testMcpSse',
      'testMcpWebsocket',
      'testMcpModelScope',
    ].forEach(exportName => {
      expect(settingsApiSource).not.toContain(exportName);
    });
  });

  it('routes streamable HTTP diagnostics through the frontend tester in McpEditorSection', () => {
    expect(mcpEditorSource).toContain('testMcpHttpFrontend');
    expect(mcpEditorSource).toMatch(/transport === 'streamable_http'[\s\S]*testMcpHttpFrontend/);
  });

  it('routes SSE diagnostics through the frontend tester in McpEditorSection', () => {
    expect(mcpEditorSource).toContain('testMcpSseFrontend');
    expect(mcpEditorSource).toMatch(/testMcpSseFrontend\(endpoint, String\(tool\?\.apiKey \|\| ''\), headerCandidates\)/);
  });

  it('routes websocket diagnostics through the frontend tester in McpEditorSection', () => {
    expect(mcpEditorSource).toContain('testMcpWebsocketFrontend');
    expect(mcpEditorSource).toMatch(/transport === 'websocket'[\s\S]*testMcpWebsocketFrontend/);
  });

  it('routes stdio diagnostics through the frontend tester in McpEditorSection', () => {
    expect(mcpEditorSource).toContain('testMcpStdioFrontend');
    expect(mcpEditorSource).toMatch(/transport === 'stdio'[\s\S]*testMcpStdioFrontend/);
  });
});
