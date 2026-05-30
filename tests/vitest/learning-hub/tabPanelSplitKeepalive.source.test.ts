import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('TabPanelContainer split keepalive contract', () => {
  const source = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/apps/TabPanelContainer.tsx'),
    'utf-8'
  );

  it('keeps non-visible tabs mounted while split view renders normal panes', () => {
    expect(source).toContain('const visibleTabIds = new Set');
    expect(source).toContain('const hiddenTabs = tabs.filter');
    expect(source).toContain('{hiddenTabs.map(tab => renderTabPanel(tab, false))}');
    expect(source).toContain('style={{ display: visible ? \'flex\' : \'none\' }}');
  });
});
