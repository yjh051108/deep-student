import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const sourceDir = path.join(repoRoot, 'dist');
const embedRoot = path.join(repoRoot, 'desktop-go', 'cmd', 'deep-student-go', 'frontend');
const targetDir = path.join(embedRoot, 'dist');
const requiredTopLevelEntries = ['index.html', 'assets', 'cmaps', 'icons', 'standard_fonts', 'wasm'];

function relativePosix(from, to) {
  return path.relative(from, to).split(path.sep).join('/');
}

function assertInside(parent, child, label) {
  const relative = path.relative(parent, child);
  if (relative === '' || (!relative.startsWith('..') && !path.isAbsolute(relative))) {
    return;
  }
  throw new Error(`${label} is outside ${parent}: ${child}`);
}

function assertDirectory(target, label) {
  const stat = fs.statSync(target, { throwIfNoEntry: false });
  if (!stat?.isDirectory()) {
    throw new Error(`${label} does not exist or is not a directory: ${target}`);
  }
}

function assertNoReparsePoint(target, label) {
  const stat = fs.lstatSync(target, { throwIfNoEntry: false });
  if (!stat) {
    return;
  }
  if (stat.isSymbolicLink()) {
    throw new Error(`${label} must not be a symlink: ${target}`);
  }
  if ((stat.mode & fs.constants.S_IFMT) === fs.constants.S_IFLNK) {
    throw new Error(`${label} must not be a reparse-point link: ${target}`);
  }
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

function shouldSkip(relativePath) {
  return relativePath === 'bundle-report.html' || relativePath === 'desktop-go' || relativePath.startsWith('desktop-go/');
}

function copyTree(source, target) {
  const entries = fs.readdirSync(source, { withFileTypes: true });
  fs.mkdirSync(target, { recursive: true });

  for (const entry of entries) {
    const sourcePath = path.join(source, entry.name);
    const targetPath = path.join(target, entry.name);
    const relativePath = relativePosix(sourceDir, sourcePath);

    if (shouldSkip(relativePath)) {
      continue;
    }
    assertNoReparsePoint(sourcePath, `source entry ${relativePath}`);

    if (entry.isDirectory()) {
      copyTree(sourcePath, targetPath);
    } else if (entry.isFile()) {
      fs.copyFileSync(sourcePath, targetPath);
    }
  }
}

function countFiles(target) {
  let count = 0;
  for (const entry of fs.readdirSync(target, { withFileTypes: true })) {
    const entryPath = path.join(target, entry.name);
    if (entry.isDirectory()) {
      count += countFiles(entryPath);
    } else if (entry.isFile()) {
      count += 1;
    }
  }
  return count;
}

assertInside(repoRoot, sourceDir, 'source dist');
assertInside(repoRoot, targetDir, 'Go frontend embed dist');
assertInside(embedRoot, targetDir, 'Go frontend embed dist');
if (path.basename(targetDir) !== 'dist') {
  throw new Error(`Go frontend embed target must be a dist directory: ${targetDir}`);
}
if (path.resolve(sourceDir) === path.resolve(targetDir)) {
  throw new Error('source dist and Go frontend embed dist must not be the same directory');
}
assertDirectory(sourceDir, 'source dist');
assertNoReparsePoint(sourceDir, 'source dist');
fs.mkdirSync(embedRoot, { recursive: true });
assertNoReparsePoint(embedRoot, 'Go frontend embed root');

for (const entry of requiredTopLevelEntries) {
  if (!fs.existsSync(path.join(sourceDir, entry))) {
    throw new Error(`source dist is missing required build output: ${entry}`);
  }
}

safeRemoveTree(targetDir, embedRoot, 'Go frontend embed dist');
copyTree(sourceDir, targetDir);

for (const entry of requiredTopLevelEntries) {
  if (!fs.existsSync(path.join(targetDir, entry))) {
    throw new Error(`Go frontend embed dist is missing synced entry: ${entry}`);
  }
}

if (fs.existsSync(path.join(targetDir, 'desktop-go'))) {
  throw new Error('Go frontend embed dist unexpectedly contains recursive desktop-go package output');
}
if (fs.existsSync(path.join(targetDir, 'bundle-report.html'))) {
  throw new Error('Go frontend embed dist unexpectedly contains bundle-report.html');
}

const fileCount = countFiles(targetDir);
console.log(`[go-sync-frontend-dist] copied ${fileCount} files from ${sourceDir} -> ${targetDir}`);
