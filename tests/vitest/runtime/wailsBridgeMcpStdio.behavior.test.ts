import { afterEach, describe, expect, it, vi } from 'vitest';

const mockStartStdioSession = vi.hoisted(() => vi.fn());
const mockSendStdioMessage = vi.hoisted(() => vi.fn());
const mockCloseStdioSession = vi.hoisted(() => vi.fn());

vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/mcpservice', () => ({
  StartStdioSession: mockStartStdioSession,
  SendStdioMessage: mockSendStdioMessage,
  CloseStdioSession: mockCloseStdioSession,
}));

vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/ankiservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/chatservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/dstuservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/fileservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/notesservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/qbankservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/reviewplanservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/settingsservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/skillservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/systemservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/todoservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/vfsservice', () => ({}));

describe('wails bridge MCP stdio payload forwarding', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('routes mcp_stdio_start to McpService.StartStdioSession', async () => {
    const { invokeWails } = await import('@/runtime/wailsBridge');
    const env = {
      API_KEY: 'test-key',
      MCP_TRACE: '1',
    };
    const args = ['server.js', '--stdio'];

    mockStartStdioSession.mockResolvedValue('stdio-session-1');

    await expect(invokeWails('mcp_stdio_start', {
      command: 'node',
      args,
      env,
      framing: 'content_length',
      cwd: 'D:/workspace/mcp-server',
    })).resolves.toBe('stdio-session-1');

    expect(mockStartStdioSession).toHaveBeenCalledTimes(1);
    expect(mockStartStdioSession).toHaveBeenCalledWith(
      'node',
      args,
      env,
      'content_length',
      'D:/workspace/mcp-server',
    );
  });

  it('routes mcp_stdio_send to McpService.SendStdioMessage', async () => {
    const { invokeWails } = await import('@/runtime/wailsBridge');
    const payload = '{"jsonrpc":"2.0","id":1,"method":"tools/list"}';

    mockSendStdioMessage.mockResolvedValue(undefined);

    await expect(invokeWails('mcp_stdio_send', {
      sessionId: 'stdio-session-1',
      payload,
    })).resolves.toBeUndefined();

    expect(mockSendStdioMessage).toHaveBeenCalledTimes(1);
    expect(mockSendStdioMessage).toHaveBeenCalledWith('stdio-session-1', payload);
  });

  it('routes mcp_stdio_close to McpService.CloseStdioSession', async () => {
    const { invokeWails } = await import('@/runtime/wailsBridge');

    mockCloseStdioSession.mockResolvedValue(undefined);

    await expect(invokeWails('mcp_stdio_close', {
      sessionId: 'stdio-session-1',
    })).resolves.toBeUndefined();

    expect(mockCloseStdioSession).toHaveBeenCalledTimes(1);
    expect(mockCloseStdioSession).toHaveBeenCalledWith('stdio-session-1');
  });
});
