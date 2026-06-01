import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { StoreApi } from 'zustand';
import type { ChatStore } from '@/features/chat/core/types';

let mockMessageOrder = ['message-1'];
let mockSessionStatus = 'idle';
let mockIsDataLoaded = true;
let latestViewport: HTMLDivElement | null = null;

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (_key: string, options?: { defaultValue?: string; count?: number }) =>
      options?.defaultValue ?? _key,
  }),
}));

vi.mock('@tanstack/react-virtual', () => ({
  useVirtualizer: () => ({
    getVirtualItems: () => [],
    getTotalSize: () => 0,
    measure: vi.fn(),
    measureElement: vi.fn(),
  }),
}));

vi.mock('@/components/custom-scroll-area', () => ({
  CustomScrollArea: React.forwardRef(function MockCustomScrollArea(
    {
      children,
      className,
      viewportClassName,
      viewportRef,
    }: {
      children: React.ReactNode;
      className?: string;
      viewportClassName?: string;
      viewportRef?: React.Ref<HTMLDivElement>;
    },
    ref: React.ForwardedRef<HTMLDivElement>
  ) {
    const hostRef = React.useRef<HTMLDivElement>(null);
    const viewportInnerRef = React.useRef<HTMLDivElement>(null);

    React.useImperativeHandle(ref, () => hostRef.current as HTMLDivElement);

    React.useEffect(() => {
      latestViewport = viewportInnerRef.current;

      if (typeof viewportRef === 'function') {
        viewportRef(viewportInnerRef.current);
      } else if (viewportRef && 'current' in viewportRef) {
        viewportRef.current = viewportInnerRef.current;
      }

      return () => {
        latestViewport = null;
        if (typeof viewportRef === 'function') {
          viewportRef(null);
        } else if (viewportRef && 'current' in viewportRef) {
          viewportRef.current = null;
        }
      };
    }, [viewportRef]);

    return (
      <div ref={hostRef} className={className}>
        <div ref={viewportInnerRef} className={viewportClassName}>
          {children}
        </div>
      </div>
    );
  }),
}));

vi.mock('@/hooks/useBreakpoint', () => ({
  useBreakpoint: () => ({
    isSmallScreen: false,
  }),
}));

vi.mock('@/features/chat/hooks/useChatStore', () => ({
  useMessageOrder: () => mockMessageOrder,
  useSessionStatus: () => mockSessionStatus,
  useIsDataLoaded: () => mockIsDataLoaded,
}));

vi.mock('@/features/chat/debug/sessionSwitchPerf', () => ({
  sessionSwitchPerf: {
    mark: vi.fn(),
    endTrace: vi.fn(),
  },
}));

vi.mock('@/features/chat/components/MessageItem', () => ({
  MessageItem: ({ messageId }: { messageId: string }) => (
    <div data-testid={`message-${messageId}`}>{messageId}</div>
  ),
}));

vi.mock('@/features/chat/components/ui/ThreadEmptyStateShell', () => ({
  ThreadEmptyStateShell: ({ title }: { title: string }) => <div>{title}</div>,
}));

import { MessageList } from '@/features/chat/components/MessageList';

function renderMessageList() {
  const store = {
    getState: () => ({
      getMessage: (messageId: string) => ({
        id: messageId,
        role: messageId.includes('user') ? 'user' : 'assistant',
      }),
    }),
  } as unknown as StoreApi<ChatStore>;
  return render(<MessageList store={store} />);
}

function requireViewport() {
  if (!latestViewport) {
    throw new Error('Viewport was not mounted');
  }
  return latestViewport;
}

function configureViewportMetrics(
  viewport: HTMLDivElement,
  {
    scrollHeight = 1000,
    clientHeight = 400,
    scrollTop = 200,
  }: {
    scrollHeight?: number;
    clientHeight?: number;
    scrollTop?: number;
  } = {}
) {
  let currentScrollTop = scrollTop;
  let currentScrollHeight = scrollHeight;

  Object.defineProperty(viewport, 'scrollHeight', {
    configurable: true,
    get: () => currentScrollHeight,
  });
  Object.defineProperty(viewport, 'clientHeight', {
    configurable: true,
    get: () => clientHeight,
  });
  Object.defineProperty(viewport, 'scrollTop', {
    configurable: true,
    get: () => currentScrollTop,
    set: (value: number) => {
      currentScrollTop = value;
    },
  });

  const scrollTo = vi.fn(({ top }: { top: number }) => {
    currentScrollTop = top;
    fireEvent.scroll(viewport);
  });

  Object.defineProperty(viewport, 'scrollTo', {
    configurable: true,
    value: scrollTo,
  });

  return {
    scrollTo,
    getScrollTop: () => currentScrollTop,
    setScrollTop: (value: number) => {
      currentScrollTop = value;
    },
    setScrollHeight: (value: number) => {
      currentScrollHeight = value;
    },
  };
}

describe('MessageList scroll-to-bottom control', () => {
  beforeEach(() => {
    mockMessageOrder = ['message-1'];
    mockSessionStatus = 'idle';
    mockIsDataLoaded = true;
    latestViewport = null;
    vi.clearAllMocks();
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      callback(0);
      return 1;
    });
    vi.stubGlobal('cancelAnimationFrame', vi.fn());
  });

  it('shows an icon-only scroll-to-bottom control whenever the thread is away from the bottom', async () => {
    renderMessageList();

    const viewport = requireViewport();
    configureViewportMetrics(viewport, { scrollTop: 220 });

    fireEvent.scroll(viewport);

    const button = await screen.findByRole('button', { name: 'Scroll to bottom' });
    expect(button).toBeInTheDocument();
    expect(button.querySelector('span')).toBeNull();
    expect(screen.queryByText('新内容')).not.toBeInTheDocument();

    const animatedContainer = button.parentElement;
    expect(animatedContainer).toHaveAttribute('data-open', 'true');
    expect(animatedContainer).toHaveAttribute('aria-hidden', 'false');
  });

  it('smooth-scrolls to the latest message and fades the control into its closed state after click', async () => {
    renderMessageList();

    const viewport = requireViewport();
    const { scrollTo, getScrollTop } = configureViewportMetrics(viewport, { scrollTop: 240 });
    const host = viewport.parentElement;
    const cancelListener = vi.fn();
    host?.addEventListener('smooth-wheel:cancel', cancelListener);

    fireEvent.scroll(viewport);

    const button = await screen.findByRole('button', { name: 'Scroll to bottom' });
    const animatedContainer = button.parentElement;

    fireEvent.click(button);

    expect(cancelListener).toHaveBeenCalledTimes(1);
    expect(scrollTo).toHaveBeenCalledWith({ top: 600, behavior: 'auto' });
    expect(getScrollTop()).toBe(600);
    expect(animatedContainer).toHaveAttribute('data-open', 'false');
    expect(animatedContainer).toHaveAttribute('aria-hidden', 'true');

    await waitFor(() => {
      expect(screen.getByRole('button', { hidden: true, name: 'Scroll to bottom' })).toHaveAttribute('tabindex', '-1');
    });
  });

  it('releases streaming auto-scroll as soon as the user wheels upward', async () => {
    const rafQueue: FrameRequestCallback[] = [];
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      rafQueue.push(callback);
      return rafQueue.length;
    });

    mockSessionStatus = 'streaming';
    mockMessageOrder = ['message-user', 'message-assistant'];

    renderMessageList();

    const viewport = requireViewport();
    const { getScrollTop } = configureViewportMetrics(viewport, {
      scrollHeight: 1000,
      clientHeight: 400,
      scrollTop: 580,
    });

    fireEvent.wheel(viewport, { deltaY: -1 });

    const button = await screen.findByRole('button', { name: 'Scroll to bottom' });
    expect(button.parentElement).toHaveAttribute('data-open', 'true');

    const queuedFrame = rafQueue.shift();
    expect(queuedFrame).toBeTypeOf('function');
    queuedFrame?.(0);

    expect(getScrollTop()).toBe(580);
  });

  it('keeps streaming auto-scroll paused after a small upward wheel near the bottom', async () => {
    const rafQueue: FrameRequestCallback[] = [];
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      rafQueue.push(callback);
      return rafQueue.length;
    });

    mockSessionStatus = 'streaming';
    mockMessageOrder = ['message-user', 'message-assistant'];

    renderMessageList();

    const viewport = requireViewport();
    const { getScrollTop, setScrollTop } = configureViewportMetrics(viewport, {
      scrollHeight: 1000,
      clientHeight: 400,
      scrollTop: 580,
    });

    fireEvent.wheel(viewport, { deltaY: -1 });
    setScrollTop(575);
    fireEvent.scroll(viewport);

    const button = await screen.findByRole('button', { name: 'Scroll to bottom' });
    expect(button.parentElement).toHaveAttribute('data-open', 'true');

    const queuedFrame = rafQueue.shift();
    expect(queuedFrame).toBeTypeOf('function');
    queuedFrame?.(0);

    expect(getScrollTop()).toBe(575);
  });

  it('releases streaming auto-scroll when wheel intent lands on the scroll host', async () => {
    const rafQueue: FrameRequestCallback[] = [];
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      rafQueue.push(callback);
      return rafQueue.length;
    });

    mockSessionStatus = 'streaming';
    mockMessageOrder = ['message-user', 'message-assistant'];

    renderMessageList();

    const viewport = requireViewport();
    const host = viewport.parentElement as HTMLElement;
    const { getScrollTop } = configureViewportMetrics(viewport, {
      scrollHeight: 1200,
      clientHeight: 400,
      scrollTop: 760,
    });

    fireEvent.wheel(host, { deltaY: -20 });

    const button = await screen.findByRole('button', { name: 'Scroll to bottom' });
    expect(button.parentElement).toHaveAttribute('data-open', 'true');

    const queuedFrame = rafQueue.shift();
    expect(queuedFrame).toBeTypeOf('function');
    queuedFrame?.(0);

    expect(getScrollTop()).toBe(760);
  });

  it('resumes streaming auto-follow after the scroll-to-bottom control is clicked', async () => {
    const rafQueue: FrameRequestCallback[] = [];
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      rafQueue.push(callback);
      return rafQueue.length;
    });

    mockSessionStatus = 'streaming';
    mockMessageOrder = ['message-user', 'message-assistant'];

    renderMessageList();

    const viewport = requireViewport();
    const { getScrollTop, setScrollHeight } = configureViewportMetrics(viewport, {
      scrollHeight: 1200,
      clientHeight: 400,
      scrollTop: 500,
    });

    fireEvent.wheel(viewport, { deltaY: -120 });
    const button = await screen.findByRole('button', { name: 'Scroll to bottom' });
    expect(button.parentElement).toHaveAttribute('data-open', 'true');

    fireEvent.click(button);
    expect(getScrollTop()).toBe(800);

    setScrollHeight(1400);
    for (let i = 0; i < 5 && getScrollTop() !== 1000; i += 1) {
      const queuedFrame = rafQueue.shift();
      expect(queuedFrame).toBeTypeOf('function');
      queuedFrame?.(0);
    }

    expect(getScrollTop()).toBe(1000);
    expect(button.parentElement).toHaveAttribute('data-open', 'false');
  });

  it('keeps streaming auto-follow after programmatic user-message positioning', async () => {
    const rafQueue: FrameRequestCallback[] = [];
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      rafQueue.push(callback);
      return rafQueue.length;
    });

    mockSessionStatus = 'idle';
    mockMessageOrder = ['message-user', 'message-assistant'];

    const store = {
      getState: () => ({
        getMessage: (messageId: string) => ({
          id: messageId,
          role: messageId.includes('user') ? 'user' : 'assistant',
        }),
      }),
    } as unknown as StoreApi<ChatStore>;

    const { rerender } = render(<MessageList store={store} className="idle" />);
    const viewport = requireViewport();
    const { getScrollTop, setScrollTop } = configureViewportMetrics(viewport, {
      scrollHeight: 1200,
      clientHeight: 400,
      scrollTop: 700,
    });

    const originalScrollIntoView = HTMLElement.prototype.scrollIntoView;
    Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
      configurable: true,
      value: vi.fn(() => {
        setScrollTop(320);
        fireEvent.scroll(viewport);
      }),
    });

    rafQueue.length = 0;
    mockSessionStatus = 'streaming';
    rerender(<MessageList store={store} className="streaming" />);

    const autoFollowFrame = rafQueue.shift();
    const positioningFrame = rafQueue.shift();
    expect(autoFollowFrame).toBeTypeOf('function');
    expect(positioningFrame).toBeTypeOf('function');

    autoFollowFrame?.(0);
    expect(getScrollTop()).toBe(800);

    positioningFrame?.(0);
    expect(getScrollTop()).toBe(320);

    const nextAutoFollowFrame = rafQueue.shift();
    expect(nextAutoFollowFrame).toBeTypeOf('function');
    nextAutoFollowFrame?.(0);

    expect(getScrollTop()).toBe(800);
    expect(screen.getByRole('button', { hidden: true, name: 'Scroll to bottom' }).parentElement)
      .toHaveAttribute('data-open', 'false');

    if (originalScrollIntoView) {
      Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
        configurable: true,
        value: originalScrollIntoView,
      });
    } else {
      delete (HTMLElement.prototype as HTMLElement & { scrollIntoView?: unknown }).scrollIntoView;
    }
  });

  it('does not treat streaming auto-scroll writes as user scroll intent', async () => {
    const rafQueue: FrameRequestCallback[] = [];
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      rafQueue.push(callback);
      return rafQueue.length;
    });

    mockSessionStatus = 'streaming';
    mockMessageOrder = ['message-user', 'message-assistant'];

    renderMessageList();

    const viewport = requireViewport();
    const { getScrollTop } = configureViewportMetrics(viewport, {
      scrollHeight: 1200,
      clientHeight: 400,
      scrollTop: 300,
    });

    for (let i = 0; i < 5 && getScrollTop() === 300; i += 1) {
      const queuedFrame = rafQueue.shift();
      expect(queuedFrame).toBeTypeOf('function');
      queuedFrame?.(0);
    }

    expect(getScrollTop()).toBeGreaterThan(300);
    fireEvent.scroll(viewport);

    const button = screen.getByRole('button', { hidden: true, name: 'Scroll to bottom' });
    expect(button.parentElement).toHaveAttribute('data-open', 'false');

    const nextFrame = rafQueue.shift();
    expect(nextFrame).toBeTypeOf('function');
  });
});
