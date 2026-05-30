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

  it('keeps chat citation embeds fixed height while preserving default large-map expansion', () => {
    expect(citationSource).toContain('autoExpandLargeMap={false}');
    expect(embedSource).toContain('autoExpandLargeMap = true');
    expect(embedSource).toContain(
      'return autoExpandLargeMap && nodeCount > LARGE_MAP_NODE_THRESHOLD ? height * 2 : height;'
    );
  });
});
