import { spawn } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import process from 'node:process';

const commandName = process.platform === 'win32' ? 'wails3.exe' : 'wails3';
const pathDirs = (process.env.PATH ?? '').split(path.delimiter).filter(Boolean);
const candidates = [
  ...pathDirs.map(dir => path.join(dir, commandName)),
  path.join(os.homedir(), 'go', 'bin', commandName),
];

function commandExists(candidate) {
  return fs.existsSync(candidate);
}

const command = candidates.find(commandExists);
if (!command) {
  console.error('wails3 was not found. Install it with: go install github.com/wailsapp/wails/v3/cmd/wails3@latest');
  process.exit(1);
}

const child = spawn(command, process.argv.slice(2), {
  stdio: 'inherit',
  shell: false,
});

child.on('exit', code => {
  process.exit(code ?? 1);
});
