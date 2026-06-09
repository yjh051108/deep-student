import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const desktopGoDir = path.join(repoRoot, 'desktop-go');
const sourceDll = path.join(repoRoot, 'pdfium.dll');

if (!fs.existsSync(sourceDll)) {
  console.error(`Go-owned pdfium.dll was not found at ${sourceDll}`);
  process.exit(1);
}

const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'deep-student-go-pdfium-smoke-'));
const appDir = path.join(tmpRoot, 'app');
const resourcesDir = path.join(tmpRoot, 'Resources');
const exePath = path.join(appDir, process.platform === 'win32' ? 'Deep Student.exe' : 'deep-student');

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

function smokeLayout(label, dllPath) {
  fs.rmSync(path.join(appDir, 'pdfium.dll'), { force: true });
  fs.rmSync(path.join(resourcesDir, 'pdfium.dll'), { force: true });
  fs.mkdirSync(path.dirname(dllPath), { recursive: true });
  fs.copyFileSync(sourceDll, dllPath);

  const env = { ...process.env };
  delete env.DEEP_STUDENT_PDFIUM_PATH;
  delete env.DEEP_STUDENT_ENABLE_DEV_PDFIUM_PATHS;
  env.DEEP_STUDENT_DATA_DIR = path.join(tmpRoot, 'data', label.replaceAll(/[^a-zA-Z0-9_-]/g, '_'));

  console.log(`[go-pdfium-smoke] ${label}`);
  const output = run(exePath, ['--smoke-pdfium'], { env });
  const parsed = JSON.parse(output);
  if (parsed.totalPages !== 1 || parsed.renderedPages !== 1 || parsed.firstPageBytes <= 0) {
    throw new Error(`unexpected PDFium smoke output for ${label}: ${output}`);
  }
}

try {
  fs.mkdirSync(appDir, { recursive: true });
  fs.mkdirSync(resourcesDir, { recursive: true });
  console.log(`[go-pdfium-smoke] building ${exePath}`);
  run('go', ['build', '-o', exePath, './cmd/deep-student-go'], { cwd: desktopGoDir });
  smokeLayout('exe-adjacent pdfium.dll', path.join(appDir, 'pdfium.dll'));
  smokeLayout('..\\Resources\\pdfium.dll', path.join(resourcesDir, 'pdfium.dll'));
  console.log('[go-pdfium-smoke] ok');
} finally {
  fs.rmSync(tmpRoot, { recursive: true, force: true });
}
