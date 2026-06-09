import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const desktopGoDir = path.join(repoRoot, 'desktop-go');
const outputDir = path.join(repoRoot, 'dist', 'desktop-go', 'windows');
const exePath = path.join(outputDir, 'Deep Student.exe');
const bundledDllPath = path.join(outputDir, 'pdfium.dll');
const sourceDll = path.join(repoRoot, 'pdfium.dll');

if (!fs.existsSync(sourceDll)) {
  console.error(`Go-owned pdfium.dll was not found at ${sourceDll}`);
  process.exit(1);
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
    throw new Error(`${command} ${args.join(' ')} failed with exit code ${result.status}`);
  }
  return result.stdout.trim();
}

function assertPDFiumSmoke(output) {
  const parsed = JSON.parse(output);
  if (parsed.totalPages !== 1 || parsed.renderedPages !== 1 || parsed.firstPageBytes <= 0) {
    throw new Error(`unexpected PDFium smoke output: ${output}`);
  }
}

function assertInside(parent, child, label) {
  const relative = path.relative(parent, child);
  if (relative === '' || (!relative.startsWith('..') && !path.isAbsolute(relative))) {
    return;
  }
  throw new Error(`${label} is outside ${parent}: ${child}`);
}

function realpath(target) {
  return fs.realpathSync.native ? fs.realpathSync.native(target) : fs.realpathSync(target);
}

function safeRemoveTree(target, allowedRoot, label) {
  const resolvedRoot = path.resolve(allowedRoot);
  const resolvedTarget = path.resolve(target);
  assertInside(resolvedRoot, resolvedTarget, label);
  if (resolvedRoot === resolvedTarget) {
    throw new Error(`${label} must not delete its allowed root: ${resolvedRoot}`);
  }

  const rootReal = realpath(resolvedRoot);
  const targetStat = fs.lstatSync(resolvedTarget, { throwIfNoEntry: false });
  if (!targetStat) {
    const parentReal = realpath(path.dirname(resolvedTarget));
    assertInside(rootReal, parentReal, `${label} parent realpath`);
    return;
  }
  if (targetStat.isSymbolicLink()) {
    throw new Error(`${label} must not delete a symlink or junction: ${resolvedTarget}`);
  }

  const targetReal = realpath(resolvedTarget);
  assertInside(rootReal, targetReal, `${label} realpath`);
  if (rootReal === targetReal) {
    throw new Error(`${label} must not delete its allowed root: ${rootReal}`);
  }
  fs.rmSync(resolvedTarget, { recursive: true, force: true });
}

run('node', ['scripts/go-sync-frontend-dist.mjs']);
run('node', ['scripts/go-frontend-embed-smoke.mjs']);

fs.mkdirSync(outputDir, { recursive: true });
console.log(`[go-package-windows] building ${exePath}`);
run('go', ['build', '-o', exePath, './cmd/deep-student-go'], { cwd: desktopGoDir });
fs.copyFileSync(sourceDll, bundledDllPath);
console.log(`[go-package-windows] copied ${sourceDll} -> ${bundledDllPath}`);

const smokeEnv = { ...process.env };
delete smokeEnv.DEEP_STUDENT_PDFIUM_PATH;
delete smokeEnv.DEEP_STUDENT_ENABLE_DEV_PDFIUM_PATHS;
smokeEnv.DEEP_STUDENT_DATA_DIR = path.join(outputDir, '.smoke-data');

console.log('[go-package-windows] app smoke');
try {
  run(exePath, ['--smoke'], { env: smokeEnv });

  console.log('[go-package-windows] bundled PDFium smoke');
  assertPDFiumSmoke(run(exePath, ['--smoke-pdfium'], { env: smokeEnv }));
} finally {
  safeRemoveTree(smokeEnv.DEEP_STUDENT_DATA_DIR, outputDir, 'package smoke data dir');
}

console.log(`[go-package-windows] ok: ${outputDir}`);
