import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('mindmap citation embed height contract', () => {
  const citationSource = readFileSync(
    resolve(process.cwd(), 'src/features/chat/components/MindmapCitationCard.tsx'),
    'utf-8'
  );
  const embedSource = readFileSync(
    resolve(process.cwd(), 'src/features/mindmap/components/mindmap/MindMapEmbed.tsx'),
    'utf-8'
  );
  const rendererSource = readFileSync(
    resolve(process.cwd(), 'src/features/chat/components/renderers/MarkdownRenderer.tsx'),
    'utf-8'
  );

  it('keeps chat citation embeds fixed height while preserving default large-map expansion', () => {
    expect(citationSource).toContain('autoExpandLargeMap={false}');
    expect(citationSource).toContain('compactErrorFallback');
    expect(embedSource).toContain('autoExpandLargeMap = true');
    expect(embedSource).toContain('compactErrorFallback = false');
    expect(embedSource).toContain(
      'return autoExpandLargeMap && nodeCount > LARGE_MAP_NODE_THRESHOLD ? height * 2 : height;'
    );
  });

  it('keeps streaming mindmap citations compact until the message is complete', () => {
    expect(rendererSource).toContain('deferRichEmbeds?: boolean;');
    expect(rendererSource).toContain('if (isStreaming || deferRichEmbeds)');
    expect(rendererSource).toContain('className="mindmap-citation-placeholder"');
    expect(rendererSource).toContain('<MindmapCitationCard');
  });

  it('defers rich mindmap embeds across all blocked markdown chunks while streaming', () => {
    const blockedSource = readFileSync(
      resolve(process.cwd(), 'src/features/chat/components/renderers/BlockedMarkdownRenderer.tsx'),
      'utf-8'
    );

    expect(blockedSource).toContain('isStreaming={isStreamingBlock}');
    expect(blockedSource).toContain('deferRichEmbeds={deferRichEmbeds}');
    expect(blockedSource).toContain('deferRichEmbeds={isStreaming}');
  });

  it('defers rich mindmap embeds across streaming block renderer chunks while streaming', () => {
    const streamingBlockSource = readFileSync(
      resolve(process.cwd(), 'src/features/chat/components/renderers/StreamingBlockRenderer.tsx'),
      'utf-8'
    );

    expect(streamingBlockSource).toContain('isStreaming={isActive && isStreaming}');
    expect(streamingBlockSource).toContain('deferRichEmbeds={isStreaming}');
    expect(streamingBlockSource).toContain('prev.isStreaming === next.isStreaming');
  });
});
