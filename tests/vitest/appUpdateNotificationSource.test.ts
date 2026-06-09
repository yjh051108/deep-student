import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { describe, it } from 'vitest';

const appPath = path.resolve(process.cwd(), 'src/App.tsx');
const updaterHookPath = path.resolve(process.cwd(), 'src/hooks/useAppUpdater.ts');

describe('App update notification source', () => {
  it('does not render a startup update modal when an update becomes available', () => {
    const source = readFileSync(appPath, 'utf8');

    assert.doesNotMatch(source, /function StartupUpdateNotification\(/u);
    assert.doesNotMatch(source, /<StartupUpdateNotification updater=\{updater\}\s*\/>/u);
  });

  it('suppresses startup update checks only for the live Go/Wails smoke URL', () => {
    const source = readFileSync(updaterHookPath, 'utf8');

    assert.match(source, /function isGoWailsSmokeMode\(\): boolean/u);
    assert.match(source, /new URLSearchParams\(window\.location\.search\)\.get\('go-wails-smoke'\) === 'true'/u);
    assert.match(source, /useEffect\(\(\) => \{\s*if \(isGoWailsSmokeMode\(\)\) return;\s*if \(!shouldAutoCheck\(\)\) return;/u);
    assert.match(source, /checkForUpdate\(true, true\)/u);
  });
});
