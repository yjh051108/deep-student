import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('FolderTreeView responsive grid contract', () => {
  it('keeps grid items wide and tall enough for two-line labels in narrow panels', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/components/FolderTreeView.tsx'),
      'utf-8'
    );

    expect(source).toContain("gridTemplateColumns: 'repeat(auto-fill, minmax(84px, 1fr))'");
    expect(source).toContain('h-[82px] min-w-0 overflow-hidden');
    expect(source).toContain('max-w-[84px] min-w-0');
    expect(source).toContain('[overflow-wrap:anywhere]');
  });
});
