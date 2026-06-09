import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const desktopGoDir = path.join(repoRoot, 'desktop-go');
const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'deep-student-go-check-'));

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
}

try {
  run('go', ['test', './...'], { cwd: desktopGoDir });
  run('go', ['run', './cmd/deep-student-go', '--smoke'], {
    cwd: desktopGoDir,
    env: {
      ...process.env,
      DEEP_STUDENT_DATA_DIR: path.join(tmpRoot, 'data'),
    },
  });
} finally {
  fs.rmSync(tmpRoot, { recursive: true, force: true });
}
