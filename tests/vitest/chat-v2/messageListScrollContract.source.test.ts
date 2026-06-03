import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('MessageList scroll and streaming contract', () => {
  const source = readFileSync(
    resolve(process.cwd(), 'src/features/chat/components/MessageList.tsx'),
    'utf-8'
  );
  const smoothWheelSource = readFileSync(
    resolve(process.cwd(), 'src/features/chat/hooks/useSmoothWheel.ts'),
    'utf-8'
  );

  it('cancels smooth wheel inertia before programmatic scroll-to-bottom', () => {
    expect(source).toContain("containerRef.current?.dispatchEvent(new CustomEvent('smooth-wheel:cancel'))");
    expect(source).toContain('const handleScrollToBottomClick');
    expect(source).toContain('autoFollowEnabledRef.current = true;');
    expect(source).toContain('setShowScrollToBottom(false);');
    expect(source).toContain('data-slot="message-list-scroll-to-bottom"');

    expect(smoothWheelSource).toContain("hostElement.addEventListener('smooth-wheel:cancel'");
    expect(smoothWheelSource).toContain('target = el.scrollTop;');
    expect(smoothWheelSource).toContain('stop();');
  });

  it('lets user scroll intent break streaming auto-bottom lock', () => {
    expect(source).toContain('const markUserScrollIntent = () => {');
    expect(source).toContain('autoFollowEnabledRef.current = false;');
    expect(source).toContain('setShowScrollToBottom(true);');
    expect(source).toContain('const intentElements = Array.from(');
    expect(source).toContain('new Set([viewportElement, containerRef.current].filter(Boolean))');
    expect(source).toContain('if (isStreaming || event.deltaY < 0) markUserScrollIntent();');
    expect(source).toContain("element.addEventListener('wheel', handleWheelIntent");
    expect(source).toContain("element.addEventListener('touchmove', handleTouchMoveIntent");
    expect(source).toContain("element.addEventListener('pointerdown', handlePointerDownIntent");
    expect(source).toContain("element.addEventListener('keydown', handleKeyIntent");

    const streamingLoopStart = source.indexOf('const scrollLoop = () => {');
    expect(streamingLoopStart).toBeGreaterThan(-1);
    const streamingLoopEnd = source.indexOf('const maxScroll = Math.max(0, viewportElement.scrollHeight - viewportElement.clientHeight);', streamingLoopStart);
    const streamingLoop = source.slice(streamingLoopStart, streamingLoopEnd);
    expect(streamingLoop).toContain('currentScrollTop < lastAutoScrollTopRef.current - 2');
    expect(streamingLoop).toContain('if (!autoFollowEnabledRef.current) {');
    expect(streamingLoop).toContain('return;');
  });

  it('does not let captured smooth-wheel handling hide upward wheel intent', () => {
    expect(smoothWheelSource).toContain('passive: false, capture: true');
    expect(smoothWheelSource).toContain('e.preventDefault();');
    expect(smoothWheelSource).not.toContain('stopPropagation');
    expect(smoothWheelSource).toContain('if (e.deltaY < 0) optsRef.current.onUserScrollUp?.();');
    expect(source).toContain('onUserScrollUp: () => {');
    expect(source).toContain('autoFollowEnabledRef.current = false;');
  });

  it('keeps position sync separate from user intent and uses deterministic programmatic scroll', () => {
    expect(source).toContain('const distanceToBottom = viewportElement.scrollHeight - scrollTop - viewportElement.clientHeight;');
    expect(source).toContain('const nearBottom = distanceToBottom < 50;');
    expect(source).toContain('lastObservedScrollTopRef.current = scrollTop;');
    expect(source).toContain('viewportElement.scrollTop = maxScroll;');
    expect(source).not.toContain('viewportClassName="scroll-smooth"');
  });
});
