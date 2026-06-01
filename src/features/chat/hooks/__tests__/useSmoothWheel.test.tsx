import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, waitFor } from '@testing-library/react';
import { useSmoothWheel } from '../useSmoothWheel';

vi.mock('@/lib/scroll-platform', () => ({
  detectScrollPlatform: () => ({
    isIOSWebView: false,
    isTauri: false,
    isTouchPrimary: false,
    preferNativeScrollbars: false,
  }),
}));

function configureViewportMetrics(
  viewport: HTMLDivElement,
  {
    scrollHeight = 1000,
    clientHeight = 400,
    scrollTop = 0,
  }: {
    scrollHeight?: number;
    clientHeight?: number;
    scrollTop?: number;
  } = {}
) {
  let currentScrollTop = scrollTop;

  Object.defineProperty(viewport, 'scrollHeight', {
    configurable: true,
    get: () => scrollHeight,
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

  return {
    getScrollTop: () => currentScrollTop,
  };
}

function SmoothWheelHarness() {
  const hostRef = React.useRef<HTMLDivElement>(null);
  const viewportRef = React.useRef<HTMLDivElement>(null);
  const [host, setHost] = React.useState<HTMLDivElement | null>(null);

  React.useEffect(() => {
    setHost(hostRef.current);
  }, []);

  useSmoothWheel(host, {
    getScrollElement: () => viewportRef.current,
    intensity: 0.5,
  });

  return (
    <div data-testid="host" ref={hostRef}>
      <div data-testid="viewport" ref={viewportRef} />
    </div>
  );
}

describe('useSmoothWheel', () => {
  it('cancels pending inertial wheel writes when the host receives smooth-wheel:cancel', async () => {
    const rafQueue: FrameRequestCallback[] = [];
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      rafQueue.push(callback);
      return rafQueue.length;
    });
    vi.stubGlobal('cancelAnimationFrame', vi.fn());

    const { getByTestId } = render(<SmoothWheelHarness />);
    const host = getByTestId('host') as HTMLDivElement;
    const viewport = getByTestId('viewport') as HTMLDivElement;
    const { getScrollTop } = configureViewportMetrics(viewport);

    await waitFor(() => {
      fireEvent.wheel(host, { deltaY: 120, wheelDeltaY: 120 });
      expect(rafQueue.length).toBeGreaterThan(0);
    });

    host.dispatchEvent(new CustomEvent('smooth-wheel:cancel'));
    const queuedFrame = rafQueue.shift();
    expect(queuedFrame).toBeTypeOf('function');
    queuedFrame?.(0);

    expect(getScrollTop()).toBe(0);
  });
});
