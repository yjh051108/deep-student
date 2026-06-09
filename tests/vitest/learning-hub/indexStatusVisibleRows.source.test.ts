import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('IndexStatusView visible row contract', () => {
  const source = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/views/IndexStatusView.tsx'),
    'utf-8'
  );

  it('renders the selected state rows from the same display-state source as the badges', () => {
    expect(source).toContain('displayState: normalizeIndexState(resource.displayIndexState)');
    expect(source).toContain('indexed: normalizedCounts?.display.indexed ?? 0');
    expect(source).toContain('const displayedRows = selectedState ===');
    expect(source).toContain('displayedRows.length === 0');
    expect(source).toContain('{displayedRows.map(renderResourceRow)}');
    expect(source).toContain('const state = row.displayState;');
    expect(source).toContain("resource.mmIndexState === 'pending' || resource.mmIndexState === 'failed'");
    expect(source).not.toContain('resolveResourceDisplayState');
    expect(source).not.toContain('{summary.resources.map(renderResourceRow)}');
  });
});
