import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('MobileHeaderContext source contract', () => {
  it('keeps active view and config updates idempotent to avoid layout-effect loops', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src/components/layout/MobileHeaderContext.tsx'),
      'utf-8'
    );

    expect(source).toContain('function areHeaderConfigsEqual');
    expect(source).toContain('if (activeViewRef.current === viewId)');
    expect(source).toContain('areHeaderConfigsEqual(prev, newConfig) ? prev : newConfig');
    expect(source).toContain('areHeaderConfigsEqual(prev, cachedConfig) ? prev : cachedConfig');
    expect(source).toContain('const contextValue = useMemo<MobileHeaderContextValue>');
    expect(source).toContain('<MobileHeaderContext.Provider value={contextValue}>');
    expect(source).toContain('function getReactNodeSignature');
    expect(source).toContain('[activeView, setActiveView]');
    expect(source).not.toContain('Object.is(a.onMenuClick, b.onMenuClick)');
  });
});
