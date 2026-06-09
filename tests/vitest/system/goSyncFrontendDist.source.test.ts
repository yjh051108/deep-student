import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('Go frontend dist sync package contract', () => {
  const packageJson = JSON.parse(readFileSync(resolve(process.cwd(), 'package.json'), 'utf-8')) as {
    scripts?: Record<string, string>;
  };
  const syncScript = readFileSync(resolve(process.cwd(), 'scripts/go-sync-frontend-dist.mjs'), 'utf-8');
  const embedSmokeScript = readFileSync(resolve(process.cwd(), 'scripts/go-frontend-embed-smoke.mjs'), 'utf-8');
  const packageScript = readFileSync(resolve(process.cwd(), 'scripts/go-package-windows.mjs'), 'utf-8');

  it('exposes the frontend dist sync command through package.json', () => {
    expect(packageJson.scripts?.['go:sync:frontend-dist']).toBe('node scripts/go-sync-frontend-dist.mjs');
    expect(packageJson.scripts?.['go:smoke:frontend-embed']).toBe('node scripts/go-frontend-embed-smoke.mjs');
  });

  it('keeps the frontend dist sync implementation script present', () => {
    expect(existsSync(resolve(process.cwd(), 'scripts/go-sync-frontend-dist.mjs'))).toBe(true);
    expect(existsSync(resolve(process.cwd(), 'scripts/go-frontend-embed-smoke.mjs'))).toBe(true);
  });

  it('guards the sync target and excludes recursive package output', () => {
    expect(syncScript).toContain('function safeRemoveTree');
    expect(syncScript).toContain("safeRemoveTree(targetDir, embedRoot, 'Go frontend embed dist')");
    expect(syncScript).toContain("relativePath === 'desktop-go'");
    expect(syncScript).toContain("relativePath.startsWith('desktop-go/')");
    expect(syncScript).toContain("relativePath === 'bundle-report.html'");
    expect(syncScript).toContain('assertNoReparsePoint');
    expect(syncScript).toContain('realpath');
  });

  it('prevents Windows packaging from embedding stale or placeholder assets', () => {
    expect(packageScript).toContain("run('node', ['scripts/go-sync-frontend-dist.mjs'])");
    expect(packageScript).toContain("run('node', ['scripts/go-frontend-embed-smoke.mjs'])");
    expect(packageScript).toContain('function safeRemoveTree');
    expect(packageScript).not.toMatch(/\bcargo\b/);
    expect(packageScript).not.toMatch(/\btauri\b/);
  });

  it('keeps the embed smoke focused on real React/Vite assets', () => {
    expect(embedSmokeScript).toContain('Deep Student Go shell');
    expect(embedSmokeScript).toContain('Wails migration shell');
    expect(embedSmokeScript).toContain('<div id="root"></div>');
    expect(embedSmokeScript).toContain('pdf.worker.wrapper.mjs');
    expect(embedSmokeScript).toContain('pdf.worker.min.mjs');
  });
});
