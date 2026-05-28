import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('FinderFileList responsive grid contract', () => {
  it('keeps real finder grid tiles from overlapping in narrow panes', () => {
    const listSource = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/components/finder/FinderFileList.tsx'),
      'utf-8'
    );
    const itemSource = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/components/finder/FinderFileItem.tsx'),
      'utf-8'
    );

    expect(listSource).toContain('const GRID_ITEM_MIN_WIDTH = 104');
    expect(listSource).toContain('minmax(min(${GRID_ITEM_MIN_WIDTH}px, 100%), 1fr)');
    expect(listSource).toContain('className="min-w-0 flex justify-center"');
    expect(itemSource).toContain('w-full max-w-[112px] min-w-0 h-[108px]');
    expect(itemSource).toContain('[overflow-wrap:anywhere]');
  });

  it('remeasures virtual rows after resize or first reveal so loaded folders do not render blank', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/components/finder/FinderFileList.tsx'),
      'utf-8'
    );

    expect(source).toContain('window.requestAnimationFrame');
    expect(source).toContain('gridVirtualizer.measure();');
    expect(source).toContain('listVirtualizer.measure();');
    expect(source).toContain('[viewMode, items.length, gridColumns, gridContainerWidth, gridVirtualizer, listVirtualizer]');
  });
});
