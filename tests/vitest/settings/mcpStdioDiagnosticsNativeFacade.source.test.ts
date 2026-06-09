import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('MCP stdio settings diagnostics native facade source contract', () => {
  const editorSource = readFileSync(
    resolve(process.cwd(), 'src/features/settings/components/McpEditorSection.tsx'),
    'utf-8'
  );
  const testerSource = readFileSync(
    resolve(process.cwd(), 'src/mcp/mcpFrontendTester.ts'),
    'utf-8'
  );

  it('routes settings-page stdio diagnostics through the frontend/native stdio tester', () => {
    expect(editorSource).not.toContain("from '@tauri-apps/api/core'");
    expect(editorSource).not.toContain("from '@tauri-apps/api/event'");
    expect(editorSource).not.toContain("tauriInvoke");
    expect(editorSource).not.toContain("tauriListen");
    expect(editorSource).not.toContain("'test_mcp_connection'");

    expect(editorSource).toContain('testMcpStdioFrontend');
    expect(editorSource).toContain('onProgress: setMcpTestStep');
    expect(testerSource).toContain("options.onProgress?.('spawn_process')");
    expect(testerSource).toContain("options.onProgress?.('connecting')");
    expect(testerSource).toContain("options.onProgress?.('initializing')");
    expect(testerSource).toContain("options.onProgress?.('listing_tools')");
    expect(testerSource).toContain("options.onProgress?.('disconnecting')");
    expect(testerSource).toContain("options.onProgress?.('done')");
  });
});
