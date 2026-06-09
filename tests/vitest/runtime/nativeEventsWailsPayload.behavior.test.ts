import { afterEach, describe, expect, it, vi } from 'vitest';

const mockWailsOn = vi.hoisted(() => vi.fn());

vi.mock('@wailsio/runtime', () => ({
  Events: {
    On: mockWailsOn,
    Emit: vi.fn(),
  },
}));

vi.mock('@/runtime/native', () => ({
  isTauriRuntime: () => false,
  isWailsRuntime: () => true,
}));

describe('nativeEvents Wails payload normalization', () => {
  afterEach(() => {
    mockWailsOn.mockReset();
    vi.resetModules();
  });

  it('unwraps Wails event envelopes by matching the event name', async () => {
    let registered: ((payload: unknown) => void) | undefined;
    mockWailsOn.mockImplementation((_eventName: string, callback: (payload: unknown) => void) => {
      registered = callback;
      return () => undefined;
    });

    const { listen } = await import('@/runtime/nativeEvents');
    const received: unknown[] = [];

    await listen('mcp-stdio-session-message', event => received.push(event.payload));
    registered?.({
      name: 'mcp-stdio-session-message',
      data: { message: '{"jsonrpc":"2.0","id":1}' },
      sender: 'main',
    });

    expect(received).toEqual([{ message: '{"jsonrpc":"2.0","id":1}' }]);
  });

  it('preserves raw business payloads that legitimately contain data', async () => {
    let registered: ((payload: unknown) => void) | undefined;
    mockWailsOn.mockImplementation((_eventName: string, callback: (payload: unknown) => void) => {
      registered = callback;
      return () => undefined;
    });

    const { listen } = await import('@/runtime/nativeEvents');
    const received: unknown[] = [];

    await listen('anki-generation-event', event => received.push(event.payload));
    registered?.({
      type: 'card_generated',
      data: { cardId: 'card-1' },
    });

    expect(received).toEqual([
      {
        type: 'card_generated',
        data: { cardId: 'card-1' },
      },
    ]);
  });

  it('preserves data-bearing objects when the Wails envelope name does not match', async () => {
    let registered: ((payload: unknown) => void) | undefined;
    mockWailsOn.mockImplementation((_eventName: string, callback: (payload: unknown) => void) => {
      registered = callback;
      return () => undefined;
    });

    const { listen } = await import('@/runtime/nativeEvents');
    const received: unknown[] = [];

    await listen('expected-event', event => received.push(event.payload));
    registered?.({
      name: 'other-event',
      data: { value: 1 },
    });

    expect(received).toEqual([
      {
        name: 'other-event',
        data: { value: 1 },
      },
    ]);
  });
});
