import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('IndexStatusView visible row contract', () => {
  const source = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/views/IndexStatusView.tsx'),
    'utf-8'
  );

  it('renders the selected state rows from the same display-state source as the badges', () => {
    expect(source).toContain("const state = resolveResourceDisplayState(resource, imageIndexCapability === 'ready');");
    expect(source).toContain('{displayedResources.map(renderResourceRow)}');
    expect(source).not.toContain('{summary.resources.map(renderResourceRow)}');
  });
});
