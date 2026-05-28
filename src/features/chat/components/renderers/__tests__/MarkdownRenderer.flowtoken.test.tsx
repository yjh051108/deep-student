import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { render } from '@testing-library/react';
import { MarkdownRenderer } from '../MarkdownRenderer';
import { StreamingMarkdownRenderer } from '../StreamingMarkdownRenderer';
import { StreamingBlockRenderer } from '../StreamingBlockRenderer';
import { FlowTokenMarkdownRenderer } from '../FlowTokenMarkdownRenderer';
import { ThinkingBlock } from '../../../plugins/blocks/thinking';

vi.mock('@tauri-apps/api/core', () => ({
  convertFileSrc: (path: string) => `asset://mock${path}`,
}));

vi.mock('@/features/mindmap/components/mindmap/MindMapEmbed', () => ({
  MindMapEmbed: ({
    height,
    displayTitle,
    expandLargeMaps,
    zoomOnScroll,
  }: {
    height: number;
    displayTitle?: string;
    expandLargeMaps?: boolean;
    zoomOnScroll?: boolean;
  }) => (
    <div
      className="mindmap-container"
      data-height={height}
      data-expand-large-maps={String(expandLargeMaps)}
      data-zoom-on-scroll={String(zoomOnScroll)}
      style={{ height: `${height}px` }}
    >
      {displayTitle}
    </div>
  ),
}));

const FLOWTOKEN_ANIMATION_SELECTOR =
  '[style*="animation-name: ft-fadeIn"]';

describe('MarkdownRenderer flowtoken streaming animation', () => {
  it('does not emit flowtoken spans from MarkdownRenderer in streaming mode', () => {
    const { container } = render(<MarkdownRenderer content="流式输出正在变得更自然。" isStreaming />);

    expect(container.querySelector(FLOWTOKEN_ANIMATION_SELECTOR)).toBeNull();
  });

  it('does not add flowtoken animation spans for completed prose', () => {
    const { container } = render(<MarkdownRenderer content="流式输出已经完成。" isStreaming={false} />);

    expect(container.querySelector('[style*="animation-name: ft-fadeIn"]')).toBeNull();
  });

  it('FlowTokenMarkdownRenderer animates streaming prose with flowtoken diff spans', () => {
    const { container } = render(<FlowTokenMarkdownRenderer content="流式输出正在变得更自然。" isStreaming />);

    expect(container.querySelector(FLOWTOKEN_ANIMATION_SELECTOR)).not.toBeNull();
  });

  it('keeps fenced code blocks out of the flowtoken text animation path', () => {
    const { container } = render(
      <MarkdownRenderer content={'```ts\nconst answer = 42;\n```'} isStreaming />
    );

    expect(container.querySelector('pre [style*="animation-name: ft-fadeIn"]')).toBeNull();
  });

  it('uses the same flowtoken animation path in streaming block mode', () => {
    const { container } = render(
      <StreamingBlockRenderer content="当前聊天流式块正在输出。" isStreaming />
    );

    expect(container.querySelector(FLOWTOKEN_ANIMATION_SELECTOR)).not.toBeNull();
  });

  it('uses flowtoken markdown list styling for supported streaming blocks', () => {
    const { container } = render(
      <StreamingBlockRenderer content={'- 第一项\n- 第二项'} isStreaming />
    );

    expect(container.querySelector('li.ft-custom-li')).not.toBeNull();
  });

  it('uses flowtoken fade-in for the streaming thinking chain', () => {
    const { container } = render(
      <StreamingBlockRenderer content={'<thinking>先想一想</thinking>\n最终答案'} isStreaming />
    );

    expect(container.querySelector('.chain-of-thought [style*="animation-name: ft-fadeIn"]')).not.toBeNull();
  });

  it('routes streaming main content with thinking tags through the full flowtoken renderer', () => {
    const { container } = render(
      <StreamingMarkdownRenderer content={'<thinking>先想一想</thinking>\n最终答案正在输出。'} isStreaming />
    );

    expect(container.querySelector('.main-content .flowtoken-markdown')).not.toBeNull();
  });

  it('routes standalone streaming thinking blocks through the full flowtoken renderer', () => {
    const { container } = render(
      <ThinkingBlock
        block={{
          id: 'thinking-1',
          type: 'thinking',
          status: 'running',
          messageId: 'message-1',
          content: '先拆解问题，再组织答案。',
        }}
        isStreaming
      />
    );

    expect(container.querySelector('.think-content .flowtoken-markdown')).not.toBeNull();
    expect(container.querySelector('.think-content [style*="animation-name: ft-fadeIn"]')).not.toBeNull();
  });

  it('keeps citation-like streaming blocks on the app markdown path', () => {
    const { container } = render(
      <StreamingBlockRenderer content="参考这个结论 [知识库-1]" isStreaming />
    );

    expect(container.querySelector('.stream-block')?.getAttribute('data-flowtoken')).toBe('false');
    expect(container.querySelector('.markdown-content')).not.toBeNull();
  });

  it('keeps citation-like streaming blocks off flowtoken while content grows', () => {
    const { container, rerender } = render(
      <StreamingBlockRenderer content="参考这个结论 [知识库-1]" isStreaming />
    );

    rerender(
      <StreamingBlockRenderer content="参考这个结论 [知识库-1]，并继续补充说明。" isStreaming />
    );

    const block = container.querySelector('.stream-block');
    expect(block?.getAttribute('data-flowtoken')).toBe('false');
    expect(block?.getAttribute('data-motion-layer')).toBe('inline');
  });

  it('renders streaming mindmap citations through the app markdown renderer', () => {
    const content = '<span data-mindmap-citation="true" data-mindmap-id="mm_123" data-mindmap-title="%E5%AF%BC%E5%9B%BE">导图</span>';
    const { container } = render(
      <StreamingBlockRenderer content={content} isStreaming />
    );

    expect(container.querySelector('.stream-block')?.getAttribute('data-flowtoken')).toBe('false');
    expect(container.querySelector('.mindmap-container')).not.toBeNull();
    expect(container.textContent).toContain('导图');
  });

  it('keeps streaming mindmap citations at the final card height instead of swapping badge to embed', () => {
    const content = '<span data-mindmap-citation="true" data-mindmap-id="mm_123" data-mindmap-title="%E5%AF%BC%E5%9B%BE">导图</span>';
    const { container } = render(<MarkdownRenderer content={content} isStreaming />);
    const embed = container.querySelector('.mindmap-container') as HTMLElement | null;

    expect(embed).not.toBeNull();
    expect(embed).toHaveAttribute('data-height', '220');
    expect(embed).toHaveAttribute('data-expand-large-maps', 'false');
  });

  it('renders completed mindmap citations with a stable fixed embed height', () => {
    const content = '<span data-mindmap-citation="true" data-mindmap-id="mm_123" data-mindmap-title="%E5%AF%BC%E5%9B%BE">导图</span>';
    const { container } = render(<MarkdownRenderer content={content} isStreaming={false} />);
    const embed = container.querySelector('.mindmap-container') as HTMLElement | null;

    expect(embed).not.toBeNull();
    expect(embed).toHaveAttribute('data-height', '220');
    expect(embed).toHaveAttribute('data-expand-large-maps', 'false');
    expect(embed).toHaveAttribute('data-zoom-on-scroll', 'false');
    expect(embed).toHaveStyle('height: 220px');
  });

  it('does not leave a blank mindmap card for empty citation ids', () => {
    const content = '<span data-mindmap-citation="true" data-mindmap-id="" data-mindmap-title="%E5%AF%BC%E5%9B%BE">导图</span>';
    const { container } = render(<MarkdownRenderer content={content} isStreaming={false} />);

    expect(container.querySelector('.mindmap-container')).toBeNull();
  });

  it('does not throw on malformed mindmap title encoding', () => {
    const content = '<span data-mindmap-citation="true" data-mindmap-id="mm_123" data-mindmap-title="bad%title">导图</span>';
    const { container } = render(<MarkdownRenderer content={content} isStreaming={false} />);

    expect(container.querySelector('.mindmap-container')).not.toBeNull();
    expect(container.textContent).toContain('bad%title');
  });

  it('keeps dangling markdown text visible in the flowtoken streaming path', () => {
    const { container } = render(
      <StreamingBlockRenderer content="看看这个半截链接 [还没补完" isStreaming />
    );

    expect(container.textContent).toContain('[还没补完');
    expect(container.querySelector('.stream-block')?.getAttribute('data-flowtoken')).toBe('true');
  });

  it('keeps bare LaTeX streaming blocks on the app markdown path', () => {
    const { container } = render(
      <StreamingBlockRenderer content={'score(Q, K) = \\\\frac{QK^T}{\\\\sqrt{d_k}}'} isStreaming />
    );

    expect(container.querySelector('.stream-block')?.getAttribute('data-flowtoken')).toBe('false');
    expect(container.querySelector('.markdown-content')).not.toBeNull();
    expect(container.textContent).not.toContain('[object Object]');
  });

  it('keeps bare LaTeX in the streaming thinking chain static', () => {
    const { container } = render(
      <StreamingBlockRenderer
        content={'<thinking>先想一想\nscore(Q, K) = \\\\frac{QK^T}{\\\\sqrt{d_k}}</thinking>\n最终答案'}
        isStreaming
      />
    );

    expect(container.querySelector('.chain-of-thought .markdown-content')).not.toBeNull();
    expect(container.querySelector('.chain-of-thought [style*="animation-name: ft-fadeIn"]')).toBeNull();
    expect(container.textContent).not.toContain('[object Object]');
  });

  it('keeps multiline thinking content static while preserving markdown rendering', () => {
    const { container } = render(
      <StreamingBlockRenderer
        content={'<thinking>先想一想\n第二行继续说明</thinking>\n最终答案'}
        isStreaming
      />
    );

    expect(container.querySelector('.chain-of-thought .markdown-content')).not.toBeNull();
    expect(container.querySelector('.chain-of-thought [style*="animation-name: ft-fadeIn"]')).toBeNull();
  });

  it('keeps multiline parsed thinking content static in StreamingMarkdownRenderer', () => {
    const { container } = render(
      <StreamingMarkdownRenderer
        content={'<thinking>先想一想\n第二行继续说明</thinking>\n最终答案'}
        isStreaming
      />
    );

    expect(container.querySelector('.thinking-content .markdown-content')).not.toBeNull();
    expect(container.querySelector('.thinking-content [style*="animation-name: ft-fadeIn"]')).toBeNull();
  });

  it('keeps parsed streaming main content static when markdown fallback is required', () => {
    const { container } = render(
      <StreamingMarkdownRenderer
        content={'<thinking>先想一想</thinking>\\nscore(Q, K) = \\\\frac{QK^T}{\\\\sqrt{d_k}}'}
        isStreaming
      />
    );

    expect(container.querySelector('.main-content .markdown-content')).not.toBeNull();
    expect(container.querySelector('.main-content [style*="animation-name: ft-fadeIn"]')).toBeNull();
    expect(container.textContent).not.toContain('[object Object]');
  });

  it('keeps bare LaTeX in the parsed thinking chain static', () => {
    const { container } = render(
      <StreamingMarkdownRenderer
        content={'<thinking>先想一想\nscore(Q, K) = \\\\frac{QK^T}{\\\\sqrt{d_k}}</thinking>\n最终答案'}
        isStreaming
      />
    );

    expect(container.querySelector('.thinking-content .markdown-content')).not.toBeNull();
    expect(container.querySelector('.thinking-content [style*="animation-name: ft-fadeIn"]')).toBeNull();
    expect(container.textContent).not.toContain('[object Object]');
  });

  it('keeps the existing animated span node stable while append-only text grows', () => {
    const { container, rerender } = render(
      <StreamingBlockRenderer content="第一句" isStreaming />
    );

    const firstAnimatedSpan = container.querySelector(FLOWTOKEN_ANIMATION_SELECTOR);
    expect(firstAnimatedSpan).not.toBeNull();

    rerender(<StreamingBlockRenderer content="第一句第二句" isStreaming />);

    const nextAnimatedSpan = container.querySelector(FLOWTOKEN_ANIMATION_SELECTOR);
    expect(nextAnimatedSpan).toBe(firstAnimatedSpan);
  });

  it('renders flowtoken spans with the slower demo-like timing', () => {
    const { container } = render(
      <FlowTokenMarkdownRenderer content="更顺滑的流式输出。" isStreaming />
    );

    const animatedSpan = container.querySelector(FLOWTOKEN_ANIMATION_SELECTOR);
    expect(animatedSpan).not.toBeNull();
    expect(animatedSpan).toHaveStyle('animation-duration: 0.35s');
    expect(animatedSpan).toHaveStyle('animation-timing-function: ease-out');
  });

  it('renders streaming updates directly without buffering them', () => {
    const { container, rerender } = render(
      <FlowTokenMarkdownRenderer
        content="Alpha"
        isStreaming
      />
    );

    rerender(
      <FlowTokenMarkdownRenderer
        content="Alpha beta gamma delta epsilon zeta eta theta."
        isStreaming
      />
    );

    expect(container.textContent).toContain('Alpha beta gamma delta epsilon zeta eta theta.');
    expect(container.querySelector(FLOWTOKEN_ANIMATION_SELECTOR)).not.toBeNull();
  });
});
