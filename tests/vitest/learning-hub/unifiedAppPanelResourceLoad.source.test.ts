import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('UnifiedAppPanel resource loading contract', () => {
  const source = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/apps/UnifiedAppPanel.tsx'),
    'utf-8'
  );

  it('loads by stable resource id instead of refetching on display path metadata changes', () => {
    expect(source).toContain('const path = resourceId.startsWith');
    expect(source).toContain('const result = await dstu.get(path);');
    expect(source).toContain('}, [resourceId, t, type]);');
    expect(source).not.toContain('}, [dstuPath, resourceId, t, type]);');
  });
});
