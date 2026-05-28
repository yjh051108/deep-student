import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('ChatV2 loadSessions selection contract', () => {
  it('does not broadcast list refreshes or override an active history session with a draft', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src/features/chat/pages/useSessionLifecycle.ts'),
      'utf-8'
    );

    const loadSessionsStart = source.indexOf('const loadSessions = useCallback(async () => {');
    const loadMoreStart = source.indexOf('const loadMoreSessions = useCallback(async () => {');
    const loadSessionsSource = source.slice(loadSessionsStart, loadMoreStart);

    expect(loadSessionsSource).not.toContain('emitSessionListUpdated();');
    expect(loadSessionsSource).toContain('sessionManager.getCurrentSessionId()');
    expect(loadSessionsSource).toContain('if (!activeSessionId)');
    expect(loadSessionsSource).toContain('return stillExists ? prevId : null');
  });
});
