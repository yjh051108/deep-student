import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('LearningHub mobile back contract', () => {
  const source = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/LearningHubPage.tsx'),
    'utf-8'
  );

  it('only overrides global back on mobile while the resource screen is visible', () => {
    expect(source).toContain("const shouldOverrideBack = isSmallScreen ? screenPosition === 'right' : hasOpenApp;");
    expect(source).toContain('canGoBack: shouldOverrideBack');
    expect(source).toContain("setScreenPosition('center');");
    expect(source).toContain('[closeTabWithSplit, hasOpenApp, isSmallScreen, screenPosition]');
  });
});
