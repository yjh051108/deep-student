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
    expect(source).toContain('userHasScrolledRef.current = false;');
    expect(source).toContain('setShowScrollToBottom(false);');
    expect(source).toContain('holdProgrammaticScrollLockUntilBottom();');
    expect(source).toContain('data-slot="message-list-scroll-to-bottom"');

    expect(smoothWheelSource).toContain("hostElement.addEventListener('smooth-wheel:cancel'");
    expect(smoothWheelSource).toContain('target = el.scrollTop;');
    expect(smoothWheelSource).toContain('stop();');
  });

  it('lets user scroll intent break streaming auto-bottom lock', () => {
    expect(source).toContain('const markUserScrollIntent = () => {');
    expect(source).toContain('releaseProgrammaticScrollLock();');
    expect(source).toContain('userHasScrolledRef.current = true;');
    expect(source).toContain('setShowScrollToBottom(true);');
    expect(source).toContain("viewportElement.addEventListener('wheel', handleWheelIntent");
    expect(source).toContain("viewportElement.addEventListener('touchmove', handleTouchMoveIntent");
    expect(source).toContain("viewportElement.addEventListener('pointerdown', handlePointerDownIntent");
    expect(source).toContain("viewportElement.addEventListener('keydown', handleKeyIntent");

    const streamingLoopStart = source.indexOf('const scrollLoop = () => {');
    expect(streamingLoopStart).toBeGreaterThan(-1);
    const streamingLoopEnd = source.indexOf('rafIdRef.current = requestAnimationFrame(scrollLoop);', streamingLoopStart);
    const streamingLoop = source.slice(streamingLoopStart, streamingLoopEnd);
    expect(streamingLoop).toContain('if (userHasScrolledRef.current) {');
    expect(streamingLoop).toContain('return;');
  });

  it('does not let captured smooth-wheel handling hide upward wheel intent', () => {
    expect(smoothWheelSource).toContain('passive: false, capture: true');
    expect(smoothWheelSource).toContain('e.preventDefault();');
    expect(smoothWheelSource).not.toContain('stopPropagation');
    expect(smoothWheelSource).toContain('if (e.deltaY < 0) optsRef.current.onUserScrollUp?.();');
    expect(source).toContain('onUserScrollUp: () => {');
    expect(source).toContain('releaseProgrammaticScrollLock();');
  });
});
