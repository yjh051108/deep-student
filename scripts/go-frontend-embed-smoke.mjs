import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const desktopGoDir = path.join(repoRoot, 'desktop-go');
const embedDist = path.join(repoRoot, 'desktop-go', 'cmd', 'deep-student-go', 'frontend', 'dist');
const indexPath = path.join(embedDist, 'index.html');
const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'deep-student-go-frontend-smoke-'));

function fail(message) {
  throw new Error(`[go-frontend-embed-smoke] ${message}`);
}

function run(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: options.cwd ?? repoRoot,
    encoding: 'utf8',
    shell: false,
    env: options.env ?? process.env,
  });
  if (result.stdout) {
    process.stdout.write(result.stdout);
  }
  if (result.stderr) {
    process.stderr.write(result.stderr);
  }
  if (result.status !== 0) {
    fail(`${command} ${args.join(' ')} failed with exit code ${result.status}`);
  }
  return result.stdout.trim();
}

function assertExists(relativePath) {
  const fullPath = path.join(embedDist, relativePath);
  if (!fs.existsSync(fullPath)) {
    fail(`missing embedded frontend asset: ${relativePath}`);
  }
}

function collectHtmlAssetRefs(html) {
  const refs = new Set();
  const attrPattern = /\b(?:src|href)="([^"]+)"/g;
  let match;
  while ((match = attrPattern.exec(html)) !== null) {
    const value = match[1];
    if (/^(?:https?:|data:|#)/i.test(value)) {
      continue;
    }
    refs.add(value.replace(/^\.\//, '').replace(/^\//, ''));
  }
  return [...refs];
}

function assertRealReactBuild() {
  const html = fs.readFileSync(indexPath, 'utf8');
  if (html.includes('Deep Student Go shell') || html.includes('Wails migration shell')) {
    fail('embedded index.html is still the placeholder Go shell page');
  }
  if (!html.includes('<div id="root"></div>')) {
    fail('embedded index.html does not contain the React root');
  }

  const htmlAssetRefs = collectHtmlAssetRefs(html);
  if (!htmlAssetRefs.some(ref => /^assets\/.+\.js$/.test(ref))) {
    fail('embedded index.html does not reference a Vite JS asset');
  }
  if (!htmlAssetRefs.some(ref => /^assets\/.+\.css$/.test(ref))) {
    fail('embedded index.html does not reference a Vite CSS asset');
  }
  for (const ref of htmlAssetRefs) {
    assertExists(ref);
  }

  for (const required of [
    'assets',
    'cmaps',
    'icons',
    'standard_fonts',
    'wasm',
    'pdf.worker.wrapper.mjs',
    'pdf.worker.min.mjs',
  ]) {
    assertExists(required);
  }
  if (fs.existsSync(path.join(embedDist, 'desktop-go'))) {
    fail('embedded frontend dist contains recursive desktop-go package output');
  }
  if (fs.existsSync(path.join(embedDist, 'bundle-report.html'))) {
    fail('embedded frontend dist contains bundle-report.html');
  }
}

try {
  assertExists('index.html');
  assertRealReactBuild();

  const env = {
    ...process.env,
    DEEP_STUDENT_DATA_DIR: path.join(tmpRoot, 'data'),
  };
  const smokeOutput = run('go', ['run', './cmd/deep-student-go', '--smoke'], {
    cwd: desktopGoDir,
    env,
  });
  const parsed = JSON.parse(smokeOutput);
  if (parsed.dataDir !== env.DEEP_STUDENT_DATA_DIR) {
    fail(`Go smoke used unexpected data dir: ${smokeOutput}`);
  }

  console.log('[go-frontend-embed-smoke] ok');
} finally {
  fs.rmSync(tmpRoot, { recursive: true, force: true });
}
