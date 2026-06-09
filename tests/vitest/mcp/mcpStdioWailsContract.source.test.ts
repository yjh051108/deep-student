import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('MCP stdio native facade source contract', () => {
  const source = readFileSync(resolve(process.cwd(), 'src/mcp/tauriStdioTransport.ts'), 'utf-8');

  it('uses runtime native facades instead of importing Tauri APIs directly', () => {
    expect(source).not.toContain('@tauri-apps/api/core');
    expect(source).not.toContain('@tauri-apps/api/event');

    expect(source).toMatch(
      /from\s+['"](?:@\/runtime\/native|\.\.\/runtime\/native)['"]|import\(\s*['"](?:@\/runtime\/native|\.\.\/runtime\/native)['"]\s*\)/,
    );
    expect(source).toMatch(/\binvoke\b/);

    expect(source).toMatch(
      /from\s+['"](?:@\/runtime\/nativeEvents|\.\.\/runtime\/nativeEvents)['"]|import\(\s*['"](?:@\/runtime\/nativeEvents|\.\.\/runtime\/nativeEvents)['"]\s*\)/,
    );
    expect(source).toMatch(/\blisten\b/);
  });

  it('preserves stdio command names and server config fields', () => {
    expect(source).toContain('mcp_stdio_start');
    expect(source).toContain('mcp_stdio_send');
    expect(source).toContain('mcp_stdio_close');

    for (const field of ['command', 'args', 'env', 'cwd', 'framing']) {
      expect(source).toMatch(new RegExp(`\\b${field}\\b`));
    }
  });

  it('binds native stdio commands to per-session message, error, and closed events', () => {
    expect(source).toContain("await invoke<string>('mcp_stdio_start'");
    expect(source).toContain("await invoke('mcp_stdio_send'");
    expect(source).toContain("await invoke('mcp_stdio_close'");
    expect(source).toContain('const eventPrefix = `mcp-stdio-${sessionId}`');
    expect(source).toContain('listen(`${eventPrefix}-message`');
    expect(source).toContain('listen(`${eventPrefix}-error`');
    expect(source).toContain('listen(`${eventPrefix}-closed`');
    expect(source).toContain('this.onmessage?.(parsed)');
    expect(source).toContain('this.onerror(error)');
    expect(source).toContain('this.onclose?.()');
  });

  it('keeps Wails stdio command routing on Go bindings rather than browser or Tauri fallback', () => {
    const nativeRuntime = readFileSync(resolve(process.cwd(), 'src/runtime/native.ts'), 'utf-8');
    const nativeEvents = readFileSync(resolve(process.cwd(), 'src/runtime/nativeEvents.ts'), 'utf-8');
    const wailsBridge = readFileSync(resolve(process.cwd(), 'src/runtime/wailsBridge.ts'), 'utf-8');

    expect(nativeRuntime).toMatch(/if\s*\(\s*isWailsRuntime\(\)\s*\)\s*{\s*return invokeWails<T>\(command, args\);/);
    expect(nativeRuntime).toMatch(/if\s*\(\s*shouldUseFallbackBeforeTauri\(command\)\s*\)/);
    expect(nativeEvents).toContain('isWailsEventEnvelope');
    expect(nativeEvents).toContain('(payload as { name?: unknown }).name === event');
    expect(nativeEvents).toContain('payload.data');
    expect(wailsBridge).toContain("command === 'mcp_stdio_start'");
    expect(wailsBridge).toContain('McpService.StartStdioSession');
    expect(wailsBridge).toContain("command === 'mcp_stdio_send'");
    expect(wailsBridge).toContain('McpService.SendStdioMessage');
    expect(wailsBridge).toContain("command === 'mcp_stdio_close'");
    expect(wailsBridge).toContain('McpService.CloseStdioSession');
  });
});
