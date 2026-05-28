import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('chat mindmap embed scroll contract', () => {
  it('does not let chat-inline mindmaps consume wheel scrolling or swap from badge to card', () => {
    const cardSource = readFileSync(
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

    expect(cardSource).toContain('zoomOnScroll={false}');
    expect(cardSource).toContain('expandLargeMaps={false}');
    expect(embedSource).toContain('zoomOnScroll?: boolean');
    expect(embedSource).toContain('zoomOnScroll={zoomOnScroll}');
    expect(embedSource).not.toContain('zoomOnScroll={true}');
    expect(embedSource).not.toContain('\n        fitView\n');
    expect(rendererSource).not.toContain('<MindmapCitationBadge');
  });
});
