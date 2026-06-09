import { spawn, spawnSync } from 'node:child_process';
import fs from 'node:fs';
import net from 'node:net';
import os from 'node:os';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const desktopGoDir = path.join(repoRoot, 'desktop-go');
const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'deep-student-go-live-wails-smoke-'));
const exePath = path.join(tmpRoot, process.platform === 'win32' ? 'Deep Student.exe' : 'deep-student');
const dataDir = path.join(tmpRoot, 'data');
const embeddedDistDir = path.join(desktopGoDir, 'cmd', 'deep-student-go', 'frontend', 'dist');
const includeMcpStdioSmoke = process.argv.includes('--mcp');
const includeSkillSmoke = process.argv.includes('--skills');
const includeTemplateSmoke = process.argv.includes('--templates');
const includeVfsSmoke = process.argv.includes('--vfs');
let remoteDebuggingPort = Number(process.env.DEEP_STUDENT_WAILS_REMOTE_DEBUGGING_PORT ?? '0');
let cdpURL = '';
let childExit = null;
let terminatingChild = false;

function fail(message) {
  throw new Error(`[go-live-wails-smoke] ${message}`);
}

function normalizeSmokePath(value) {
  if (typeof value !== 'string') return '';
  const normalized = path.normalize(value).replace(/[\\/]+$/, '');
  return process.platform === 'win32' ? normalized.toLowerCase() : normalized;
}

function isSameSmokePath(left, right) {
  return normalizeSmokePath(left) === normalizeSmokePath(right);
}

function isInsideSmokePath(child, parent) {
  const normalizedChild = normalizeSmokePath(child);
  const normalizedParent = normalizeSmokePath(parent);
  return normalizedChild === normalizedParent || normalizedChild.startsWith(`${normalizedParent}${path.sep}`);
}

function run(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: options.cwd ?? repoRoot,
    encoding: 'utf8',
    shell: false,
    env: options.env ?? process.env,
  });
  if (result.stdout) process.stdout.write(result.stdout);
  if (result.stderr) process.stderr.write(result.stderr);
  if (result.status !== 0) {
    const errorText = result.error ? `: ${result.error.message}` : '';
    fail(`${command} ${args.join(' ')} failed with exit code ${result.status}${errorText}`);
  }
  return result.stdout.trim();
}

function runNpm(args, options = {}) {
  if (process.platform === 'win32') {
    return run('cmd.exe', ['/d', '/s', '/c', ['npm', ...args].join(' ')], options);
  }
  return run('npm', args, options);
}

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function waitForCDP(timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  let lastError;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(`${cdpURL}/json/version`);
      if (response.ok) {
        return;
      }
      lastError = new Error(`CDP returned ${response.status}`);
    } catch (error) {
      lastError = error;
    }
    await sleep(300);
  }
  fail(`CDP endpoint did not become ready at ${cdpURL}: ${lastError?.message ?? 'timeout'}`);
}

async function fetchCDPTargets() {
  const response = await fetch(`${cdpURL}/json/list`);
  if (!response.ok) {
    fail(`CDP target list returned ${response.status}`);
  }
  return await response.json();
}

function isViteDevURL(url) {
  if (typeof url !== 'string') return false;
  try {
    const parsed = new URL(url);
    const isLoopbackHost =
      parsed.hostname === 'localhost' ||
      parsed.hostname === '127.0.0.1' ||
      parsed.hostname === '::1' ||
      parsed.hostname === '[::1]';
    const devServerHost =
      isLoopbackHost &&
      (parsed.protocol === 'http:' || parsed.protocol === 'https:') &&
      (parsed.port === '1422' || parsed.pathname.startsWith('/assets/') || parsed.pathname === '/' || parsed.pathname === '');
    return (
      devServerHost ||
      parsed.pathname.includes('/@vite/client') ||
      parsed.pathname.includes('/@react-refresh') ||
      parsed.pathname.includes('/node_modules/.vite/')
    );
  } catch {
    return url.includes('/@vite/client') || url.includes('/@react-refresh') || url.includes('/node_modules/.vite/');
  }
}

async function waitForSmokePage(browser, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const targets = await fetchCDPTargets();
    const smokeTargets = targets.filter(target =>
      target?.type === 'page' &&
      typeof target.url === 'string' &&
      target.url.includes('go-wails-smoke=true')
    );
    if (smokeTargets.length > 1) {
      fail(`expected one Wails smoke page target, found ${smokeTargets.length}: ${JSON.stringify(smokeTargets)}`);
    }
    const smokeURL = smokeTargets[0]?.url;
    if (isViteDevURL(smokeURL)) {
      fail(`live Wails smoke must use embedded assets, not Vite dev server: ${smokeURL}`);
    }

    for (const context of browser.contexts()) {
      for (const page of context.pages()) {
        const url = page.url();
        if (url.includes('go-wails-smoke=true')) {
          return page;
        }
      }
    }
    await sleep(300);
  }
  fail('could not find Wails smoke page over CDP');
}

function findFiles(root, predicate, limit = 30) {
  const matches = [];
  const stack = [root];
  while (stack.length > 0 && matches.length < limit) {
    const current = stack.pop();
    const entries = fs.readdirSync(current, { withFileTypes: true });
    for (const entry of entries) {
      const fullPath = path.join(current, entry.name);
      if (entry.isDirectory()) {
        stack.push(fullPath);
        continue;
      }
      if (entry.isFile() && predicate(fullPath)) {
        matches.push(fullPath);
        if (matches.length >= limit) break;
      }
    }
  }
  return matches;
}

function assertEmbeddedDistHasSmokeHook() {
  const jsFiles = findFiles(
    embeddedDistDir,
    filePath => filePath.endsWith('.js') || filePath.endsWith('.mjs'),
    5000
  );
  const marker = '__DEEP_STUDENT_GO_WAILS_SMOKE__';
  const match = jsFiles.find(filePath => fs.readFileSync(filePath, 'utf8').includes(marker));
  if (!match) {
    fail(`embedded frontend dist does not contain ${marker}; frontend build/sync is stale`);
  }
}

function createLegacyTemplateSQLiteFixture() {
  if (!includeTemplateSmoke) return;
  fs.mkdirSync(dataDir, { recursive: true });
  const generatorPath = path.join(tmpRoot, 'create-legacy-template-db.go');
  const dbPath = path.join(dataDir, 'mistakes.db');
  fs.writeFileSync(generatorPath, `
package main

import (
  "database/sql"
  "log"
  "os"

  _ "modernc.org/sqlite"
)

func main() {
  if len(os.Args) != 2 {
    log.Fatal("usage: create-legacy-template-db <db-path>")
  }
  db, err := sql.Open("sqlite", os.Args[1])
  if err != nil {
    log.Fatal(err)
  }
  defer db.Close()
  _, err = db.Exec(\`
    CREATE TABLE custom_anki_templates (
      id TEXT PRIMARY KEY,
      name TEXT NOT NULL UNIQUE,
      description TEXT,
      author TEXT,
      version TEXT NOT NULL DEFAULT '1.0.0',
      preview_front TEXT NOT NULL,
      preview_back TEXT NOT NULL,
      note_type TEXT NOT NULL DEFAULT 'Basic',
      fields_json TEXT NOT NULL DEFAULT '[]',
      generation_prompt TEXT NOT NULL,
      front_template TEXT NOT NULL,
      back_template TEXT NOT NULL,
      css_style TEXT NOT NULL,
      field_extraction_rules_json TEXT NOT NULL DEFAULT '{}',
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL,
      is_active INTEGER NOT NULL DEFAULT 1,
      is_built_in INTEGER NOT NULL DEFAULT 0,
      preview_data_json TEXT
    );
    CREATE TABLE settings (
      key TEXT PRIMARY KEY,
      value TEXT,
      updated_at TEXT
    );
  \`)
  if err != nil {
    log.Fatal(err)
  }
  _, err = db.Exec(\`
    INSERT INTO custom_anki_templates
    (id, name, description, author, version, preview_front, preview_back, note_type,
     fields_json, generation_prompt, front_template, back_template, css_style,
     field_extraction_rules_json, created_at, updated_at, is_active, is_built_in, preview_data_json)
    VALUES
    ('legacy-wails-smoke-template', 'Legacy Wails Smoke Template', 'migrated from old mistakes.db',
     'Legacy User', '2.0.0', '{{Front}}', '{{Back}}', 'Basic', '["Front","Back"]',
     'legacy smoke prompt', '<div>{{Front}}</div>', '<div>{{Back}}</div>', '.card { color: blue; }',
     '{"Front":{"field_type":"Text","is_required":true,"description":"front"}}',
     '2024-01-02T03:04:05.000Z', '2024-01-03T03:04:05.000Z', 1, 0, '{"Front":"preview"}');
    INSERT INTO settings (key, value, updated_at)
    VALUES ('default_template_id', 'legacy-wails-smoke-template', '2024-01-04T03:04:05.000Z');
  \`)
  if err != nil {
    log.Fatal(err)
  }
}
`, 'utf8');
  run('go', ['run', generatorPath, dbPath], { cwd: desktopGoDir });
}

async function getFreePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.on('error', reject);
    server.listen(0, '127.0.0.1', () => {
      const address = server.address();
      if (!address || typeof address === 'string') {
        server.close(() => reject(new Error('could not allocate a local TCP port')));
        return;
      }
      const port = address.port;
      server.close(() => resolve(port));
    });
  });
}

async function assertCDPClosed() {
  let response;
  try {
    response = await fetch(`${cdpURL}/json/version`, { signal: AbortSignal.timeout(500) });
  } catch {
    // Expected before the smoke app starts.
    return;
  }
  if (response.ok) {
    fail(`CDP endpoint is already active before launch: ${cdpURL}`);
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

async function terminate(child) {
  if (child.exitCode != null || child.signalCode != null) {
    return;
  }
  terminatingChild = true;
  if (process.platform === 'win32') {
    spawnSync('taskkill', ['/pid', String(child.pid), '/t', '/f'], { stdio: 'ignore' });
  } else {
    child.kill('SIGTERM');
  }
}

async function waitForExit(child, timeoutMs) {
  if (child.exitCode != null || child.signalCode != null) {
    return;
  }
  await new Promise(resolve => {
    const timer = setTimeout(resolve, timeoutMs);
    child.once('exit', () => {
      clearTimeout(timer);
      resolve();
    });
  });
}

async function safeRemoveTreeWithRetry(target, allowedRoot, label) {
  let lastError;
  for (let attempt = 0; attempt < 20; attempt += 1) {
    try {
      safeRemoveTree(target, allowedRoot, label);
      return;
    } catch (error) {
      lastError = error;
      if (!['EBUSY', 'EPERM', 'ENOTEMPTY'].includes(error?.code)) {
        throw error;
      }
      await sleep(250);
    }
  }
  throw lastError;
}

function assertChildAlive(label) {
  if (childExit) {
    fail(`${label}: app exited before smoke completed: ${JSON.stringify(childExit)}`);
  }
}

function assertSettingsPersisted(result) {
  if (typeof result.key !== 'string' || result.key.length === 0) {
    fail(`frontend smoke did not return the persisted settings key: ${JSON.stringify(result)}`);
  }
  const settingsPath = path.join(dataDir, 'settings-go.json');
  if (!fs.existsSync(settingsPath)) {
    fail(`settings roundtrip did not create ${settingsPath}`);
  }
  const settings = JSON.parse(fs.readFileSync(settingsPath, 'utf8'));
  if (settings?.[result.key] !== result.value) {
    fail(`settings roundtrip was not persisted in ${settingsPath}: ${JSON.stringify({ key: result.key, persisted: settings?.[result.key], expected: result.value })}`);
  }
}

function assertSmokeResult(result) {
  if (!result || typeof result !== 'object') {
    fail(`unexpected smoke result: ${JSON.stringify(result)}`);
  }
  if (result.ok !== true) {
    fail(`frontend smoke hook reported failure: ${JSON.stringify(result)}`);
  }
  if (result.appDataDir !== dataDir) {
    fail(`frontend smoke returned unexpected appDataDir: ${JSON.stringify(result)}`);
  }
  if (result.isWails !== true) {
    fail(`frontend smoke did not prove Wails runtime: ${JSON.stringify(result)}`);
  }
  if (result.isInjected === true) {
    fail(`frontend smoke used injected native runtime instead of Wails: ${JSON.stringify(result)}`);
  }
  if (result.smokeFlag !== true || result.hasWailsInvoke !== true || result.hasWailsEnvironment !== true) {
    fail(`frontend smoke did not prove Wails smoke flags/runtime bridge: ${JSON.stringify(result)}`);
  }
  if (result.rootMounted !== true) {
    fail(`frontend smoke did not prove React mounted into #root: ${JSON.stringify(result)}`);
  }
  if (result.smokeSentinelRendered !== true) {
    fail(`frontend smoke sentinel was not rendered: ${JSON.stringify(result)}`);
  }
  if (result.topLevelErrorBoundaryVisible === true) {
    fail(`top-level React error boundary is visible during smoke: ${JSON.stringify(result)}`);
  }
  if (Array.isArray(result.earlyErrors) && result.earlyErrors.length > 0) {
    fail(`frontend recorded early runtime errors: ${JSON.stringify(result.earlyErrors)}`);
  }
  if (result.stored !== result.value) {
    fail(`frontend smoke setting roundtrip mismatch: ${JSON.stringify(result)}`);
  }
  assertSettingsPersisted(result);
}

function assertMcpStdioSmokeResult(result) {
  if (!result || typeof result !== 'object') {
    fail(`frontend smoke did not return MCP stdio result: ${JSON.stringify(result)}`);
  }
  const mcpStdio = result.mcpStdio;
  if (!mcpStdio || typeof mcpStdio !== 'object') {
    fail(`frontend smoke did not run MCP stdio proof: ${JSON.stringify(result)}`);
  }
  if (mcpStdio.ok !== true) {
    fail(`frontend MCP stdio smoke reported failure: ${JSON.stringify(mcpStdio)}`);
  }
  if (mcpStdio.commandStarted !== true || typeof mcpStdio.sessionId !== 'string' || mcpStdio.sessionId.length === 0) {
    fail(`MCP stdio smoke did not prove mcp_stdio_start: ${JSON.stringify(mcpStdio)}`);
  }
  if (mcpStdio.messageEventReceived !== true) {
    fail(`MCP stdio smoke did not receive a Wails message event: ${JSON.stringify(mcpStdio)}`);
  }
  if (mcpStdio.closedEventReceived !== true) {
    fail(`MCP stdio smoke did not receive a Wails closed event: ${JSON.stringify(mcpStdio)}`);
  }
  if (mcpStdio.errorEventReceived !== false) {
    fail(`MCP stdio smoke saw a Wails error event: ${JSON.stringify(mcpStdio)}`);
  }
  if (mcpStdio.closeCommandAttempted !== true || mcpStdio.closeCommandSucceeded !== true) {
    fail(`MCP stdio smoke did not prove mcp_stdio_close cleanup: ${JSON.stringify(mcpStdio)}`);
  }
  if (mcpStdio.activeCloseSessionStarted !== true || mcpStdio.activeCloseCommandSucceeded !== true) {
    fail(`MCP stdio smoke did not prove mcp_stdio_close on an active session: ${JSON.stringify(mcpStdio)}`);
  }
  if (mcpStdio.activeCloseSendRejected !== true) {
    fail(`MCP stdio smoke did not prove closed sessions reject later send: ${JSON.stringify(mcpStdio)}`);
  }
  if (mcpStdio.browserFallbackRejected !== true || mcpStdio.tauriFallbackRejected !== true) {
    fail(`MCP stdio smoke did not reject browser/Tauri fallback paths: ${JSON.stringify(mcpStdio)}`);
  }
  if (mcpStdio.initializeServerName !== 'dstu-mcp-smoke') {
    fail(`MCP stdio smoke initialize response came from an unexpected server: ${JSON.stringify(mcpStdio)}`);
  }
  if (mcpStdio.toolName !== 'smoke_echo' || mcpStdio.toolCallText !== 'echo: wails') {
    fail(`MCP stdio smoke tool roundtrip failed: ${JSON.stringify(mcpStdio)}`);
  }
  for (const route of [
    'mcp_stdio_start -> McpService.StartStdioSession',
    'mcp_stdio_send -> McpService.SendStdioMessage',
    'mcp_stdio_close -> McpService.CloseStdioSession',
  ]) {
    if (!Object.values(mcpStdio).includes(route)) {
      fail(`MCP stdio smoke result is missing route evidence ${route}: ${JSON.stringify(mcpStdio)}`);
    }
  }
}

function assertSkillSmokeResult(result) {
  if (!result || typeof result !== 'object') {
    fail(`frontend smoke did not return Skill result: ${JSON.stringify(result)}`);
  }
  const skills = result.skills;
  if (!skills || typeof skills !== 'object') {
    fail(`frontend smoke did not run Skill proof: ${JSON.stringify(result)}`);
  }
  if (skills.ok !== true) {
    fail(`frontend Skill smoke reported failure: ${JSON.stringify(skills)}`);
  }
  const expectedSkillsBasePath = path.join(dataDir, 'skills');
  if (!isSameSmokePath(skills.basePath, expectedSkillsBasePath)) {
    fail(`Skill smoke used an unexpected base path: ${JSON.stringify(skills)}`);
  }
  if (typeof skills.skillPath !== 'string' || !isInsideSmokePath(skills.skillPath, expectedSkillsBasePath)) {
    fail(`Skill smoke created outside the temp data dir: ${JSON.stringify(skills)}`);
  }
  if (skills.createdContentIncludesName !== true || skills.readContentIncludesName !== true || skills.updatedContentIncludesName !== true) {
    fail(`Skill smoke did not prove create/read/update content: ${JSON.stringify(skills)}`);
  }
  if (skills.listBeforeIncludesSkill !== true || skills.listAfterDeleteIncludesSkill !== false) {
    fail(`Skill smoke did not prove list/delete behavior: ${JSON.stringify(skills)}`);
  }
  if (skills.deleteSucceeded !== true || skills.readAfterDeleteRejected !== true) {
    fail(`Skill smoke did not prove delete/read-after-delete behavior: ${JSON.stringify(skills)}`);
  }
  for (const route of [
    'skill_create -> SkillService.Create',
    'skill_read_file -> SkillService.ReadFile',
    'skill_update -> SkillService.Update',
    'skill_list_directories -> SkillService.ListDirectories',
    'skill_delete -> SkillService.Delete',
  ]) {
    if (!Object.values(skills).includes(route)) {
      fail(`Skill smoke result is missing route evidence ${route}: ${JSON.stringify(skills)}`);
    }
  }
}

function assertTemplateSmokeResult(result) {
  if (!result || typeof result !== 'object') {
    fail(`frontend smoke did not return Template result: ${JSON.stringify(result)}`);
  }
  const templates = result.templates;
  if (!templates || typeof templates !== 'object') {
    fail(`frontend smoke did not run Template proof: ${JSON.stringify(result)}`);
  }
  if (templates.ok !== true) {
    fail(`frontend Template smoke reported failure: ${JSON.stringify(templates)}`);
  }
  if (templates.builtinCount < 6 || typeof templates.importBuiltinResult !== 'string' || !templates.importBuiltinResult.includes('导入完成')) {
    fail(`Template smoke did not prove builtin import/list behavior: ${JSON.stringify(templates)}`);
  }
  if (templates.legacyMigratedName !== 'Legacy Wails Smoke Template') {
    fail(`Template smoke did not prove legacy mistakes.db migration: ${JSON.stringify(templates)}`);
  }
  if (!Array.isArray(templates.legacyMigratedFields) || !templates.legacyMigratedFields.includes('Front') || !templates.legacyMigratedFields.includes('Back')) {
    fail(`Template smoke did not normalize migrated legacy template fields: ${JSON.stringify(templates)}`);
  }
  if (typeof templates.createdId !== 'string' || templates.createdId.length === 0) {
    fail(`Template smoke did not return created template id: ${JSON.stringify(templates)}`);
  }
  if (templates.defaultTemplateId !== templates.createdId) {
    fail(`Template smoke did not prove set/get default template: ${JSON.stringify(templates)}`);
  }
  if (templates.updatedDescription !== templates.exportedDescription || templates.exportedId !== templates.createdId) {
    fail(`Template smoke did not prove update/export parity: ${JSON.stringify(templates)}`);
  }
  if (typeof templates.importedId !== 'string' || templates.importedId.length === 0 || templates.importedIsBuiltIn !== false) {
    fail(`Template smoke did not prove custom bulk import identity: ${JSON.stringify(templates)}`);
  }
  if (!Array.isArray(templates.importedFields) || !templates.importedFields.includes('Front') || !templates.importedFields.includes('Back')) {
    fail(`Template smoke did not normalize legacy fields_json import payloads: ${JSON.stringify(templates)}`);
  }
  if (templates.importedHasLegacyFieldsJson === true) {
    fail(`Template smoke persisted legacy fields_json instead of normalized fields: ${JSON.stringify(templates)}`);
  }
  for (const route of [
    'import_builtin_templates -> TemplateService.ImportBuiltinTemplates',
    'get_all_custom_templates -> TemplateService.GetAllCustomTemplates',
    'get_default_template_id -> TemplateService.GetDefaultTemplateID',
    'create_custom_template -> TemplateService.CreateCustomTemplate',
    'update_custom_template -> TemplateService.UpdateCustomTemplate',
    'set_default_template -> TemplateService.SetDefaultTemplate',
    'import_custom_templates_bulk -> TemplateService.ImportCustomTemplatesBulk',
    'export_template -> TemplateService.ExportTemplate',
  ]) {
    if (!Object.values(templates).includes(route)) {
      fail(`Template smoke result is missing route evidence ${route}: ${JSON.stringify(templates)}`);
    }
  }

  const templatesPath = path.join(dataDir, 'templates-go.json');
  if (!fs.existsSync(templatesPath)) {
    fail(`Template smoke did not create ${templatesPath}`);
  }
  const store = JSON.parse(fs.readFileSync(templatesPath, 'utf8'));
  const storedTemplates = Array.isArray(store?.templates) ? store.templates : [];
  const storedCreated = storedTemplates.find(template => template?.id === templates.createdId);
  const storedImported = storedTemplates.find(template => template?.id === templates.importedId);
  const storedLegacy = storedTemplates.find(template => template?.id === 'legacy-wails-smoke-template');
  if (!storedLegacy || storedLegacy.name !== 'Legacy Wails Smoke Template') {
    fail(`Template smoke did not persist migrated legacy template: ${JSON.stringify({ templates, storedLegacy })}`);
  }
  if (!store.legacyMigration || store.legacyMigration.imported !== 1 || store.legacyMigration.defaultTemplateId !== 'legacy-wails-smoke-template') {
    fail(`Template smoke did not persist legacy migration metadata in ${templatesPath}: ${JSON.stringify(store.legacyMigration)}`);
  }
  if (!storedCreated || storedCreated.description !== templates.updatedDescription) {
    fail(`Template smoke did not persist the updated created template: ${JSON.stringify({ templates, storedCreated })}`);
  }
  if (!storedImported || storedImported.name !== templates.importedName || storedImported.is_built_in !== false) {
    fail(`Template smoke did not persist the imported custom template: ${JSON.stringify({ templates, storedImported })}`);
  }
  if (Object.prototype.hasOwnProperty.call(storedImported, 'fields_json')) {
    fail(`Template smoke persisted legacy fields_json in ${templatesPath}: ${JSON.stringify(storedImported)}`);
  }
  if (store.defaultTemplateId !== templates.createdId) {
    fail(`Template smoke did not persist defaultTemplateId in ${templatesPath}: ${JSON.stringify({ templates, defaultTemplateId: store.defaultTemplateId })}`);
  }
}

function assertVfsSmokeResult(result) {
  if (!result || typeof result !== 'object') {
    fail(`frontend smoke did not return VFS result: ${JSON.stringify(result)}`);
  }
  const vfs = result.vfs;
  if (!vfs || typeof vfs !== 'object') {
    fail(`frontend smoke did not run VFS proof: ${JSON.stringify(result)}`);
  }
  if (vfs.ok !== true) {
    fail(`frontend VFS smoke reported failure: ${JSON.stringify(vfs)}`);
  }
  const expectedInputDir = path.join(dataDir, 'smoke-fixtures');
  if (typeof vfs.sourcePath !== 'string' || !isInsideSmokePath(vfs.sourcePath, expectedInputDir)) {
    fail(`VFS smoke source fixture escaped the temp input dir: ${JSON.stringify(vfs)}`);
  }
  if (typeof vfs.resourceId !== 'string' || vfs.resourceId.length === 0) {
    fail(`VFS smoke did not return a resource id: ${JSON.stringify(vfs)}`);
  }
  if (typeof vfs.resourceHash !== 'string' || vfs.resourceHash.length === 0) {
    fail(`VFS smoke did not return a resource hash: ${JSON.stringify(vfs)}`);
  }
  if (typeof vfs.fileId !== 'string' || vfs.fileId.length === 0) {
    fail(`VFS smoke did not return a file/source id: ${JSON.stringify(vfs)}`);
  }
  if (vfs.textbookType !== 'textbook' || vfs.previewType !== 'pdf') {
    fail(`VFS smoke did not import a PDF textbook node: ${JSON.stringify(vfs)}`);
  }
  if (vfs.resourceType !== 'textbook' || vfs.resourceStorageMode !== 'external') {
    fail(`VFS smoke did not prove an external Go hybrid VFS textbook resource: ${JSON.stringify(vfs)}`);
  }
  if (typeof vfs.resourceOriginalPath !== 'string' || !isInsideSmokePath(vfs.resourceOriginalPath, expectedInputDir)) {
    fail(`VFS smoke resource original path is not the smoke fixture: ${JSON.stringify(vfs)}`);
  }
  if (typeof vfs.resourceExternalPath !== 'string' || vfs.resourceExternalPath.length === 0) {
    fail(`VFS smoke resource external path is empty: ${JSON.stringify(vfs)}`);
  }
  if (path.isAbsolute(vfs.resourceExternalPath)) {
    fail(`VFS smoke resource external path should be a library-relative path: ${JSON.stringify(vfs)}`);
  }
  if (typeof vfs.resourceResolvedPath !== 'string' || !isInsideSmokePath(vfs.resourceResolvedPath, dataDir)) {
    fail(`VFS smoke resolved resource path is not inside temp data dir: ${JSON.stringify(vfs)}`);
  }
  if (vfs.fileContentFound !== true || vfs.rawContentContainsSentinel !== true) {
    fail(`VFS smoke did not prove retrievable imported PDF bytes/content: ${JSON.stringify(vfs)}`);
  }
  if (vfs.extractedTextContainsSentinel !== true) {
    fail(`VFS smoke did not prove extracted text contains the fixture sentinel: ${JSON.stringify(vfs)}`);
  }
  if (vfs.fileLookupMatchesResource !== true || vfs.fileLookupMatchesHash !== true) {
    fail(`VFS smoke file compatibility lookup did not match resource identity: ${JSON.stringify(vfs)}`);
  }
  if (vfs.pdfMediaType !== 'pdf') {
    fail(`VFS smoke PDF status did not identify a PDF resource: ${JSON.stringify(vfs)}`);
  }
  if (!['completed', 'completed_with_issues'].includes(vfs.pdfStage)) {
    fail(`VFS smoke PDF status did not reach a terminal non-error state: ${JSON.stringify(vfs)}`);
  }
  if (vfs.pdfReadyText !== true || vfs.pdfPageCount !== 1) {
    fail(`VFS smoke did not prove text-layer readiness for the fixture PDF: ${JSON.stringify(vfs)}`);
  }
  if (!['completed', 'completed_with_issues'].includes(vfs.batchPdfStage)) {
    fail(`VFS smoke batch PDF status did not reach a terminal non-error state: ${JSON.stringify(vfs)}`);
  }
  if (vfs.batchPdfReadyText !== true || vfs.batchPdfPageCount !== 1) {
    fail(`VFS smoke batch PDF status did not prove text-layer readiness for the fixture PDF: ${JSON.stringify(vfs)}`);
  }
  if (!Array.isArray(vfs.progressStages) || !['hashing', 'copying', 'saving', 'done'].every(stage => vfs.progressStages.includes(stage))) {
    fail(`VFS smoke did not receive the expected textbook import progress stages: ${JSON.stringify(vfs)}`);
  }
  if (vfs.doneEventMatchesIdentity !== true) {
    fail(`VFS smoke done event did not match imported textbook/resource identity: ${JSON.stringify(vfs)}`);
  }
  for (const route of [
    'textbooks_add -> DstuService.AddTextbooks',
    'textbook-import-progress -> Wails EventBus',
    'vfs_get_resource -> VfsService.GetResource',
    'vfs_get_file -> VfsService.GetFile',
    'vfs_get_file_content -> VfsService.GetFileContent',
    'vfs_get_pdf_processing_status -> VfsService.GetPdfProcessingStatus',
    'vfs_get_batch_pdf_processing_status -> VfsService.GetBatchPdfProcessingStatus',
  ]) {
    if (!Object.values(vfs).includes(route)) {
      fail(`VFS smoke result is missing route evidence ${route}: ${JSON.stringify(vfs)}`);
    }
  }
}

function formatConsoleLocation(message) {
  const location = message.location();
  if (!location?.url) {
    return '';
  }
  const line = Number.isFinite(location.lineNumber) ? `:${location.lineNumber}` : '';
  const column = Number.isFinite(location.columnNumber) ? `:${location.columnNumber}` : '';
  return ` (${location.url}${line}${column})`;
}

function isBenignWailsCustomScriptProbe(response) {
  if (response.status() !== 404 || response.request().method() !== 'HEAD') {
    return false;
  }
  try {
    const parsed = new URL(response.url());
    return parsed.hostname === 'wails.localhost' && parsed.pathname === '/wails/custom.js';
  } catch {
    return false;
  }
}

function isBenignWailsCustomScriptConsole(message) {
  const lower = message.toLowerCase();
  return lower.includes('failed to load resource') && lower.includes('http://wails.localhost/wails/custom.js');
}

function isExpectedWailsRuntime422Failure(message) {
  return message === '422 POST http://wails.localhost/wails/runtime';
}

function expectedWailsRuntime422FailureCount(result) {
  let count = 0;
  if (result?.mcpStdio?.activeCloseSendRejected === true) {
    count += 1;
  }
  if (result?.skills?.readAfterDeleteRejected === true) {
    count += 1;
  }
  return count;
}

function filterExpectedWailsRuntimeHTTPFailures(httpFailures, result) {
  let remainingExpectedRuntimeFailures = expectedWailsRuntime422FailureCount(result);
  if (remainingExpectedRuntimeFailures <= 0) {
    return httpFailures;
  }
  return httpFailures.filter(message => {
    if (remainingExpectedRuntimeFailures > 0 && isExpectedWailsRuntime422Failure(message)) {
      remainingExpectedRuntimeFailures -= 1;
      return false;
    }
    return true;
  });
}

function isExpectedWailsRuntime422ConsoleError(message) {
  const lower = message.toLowerCase();
  return (
    lower.includes('failed to load resource') &&
    lower.includes('status of 422') &&
    lower.includes('http://wails.localhost/wails/runtime')
  );
}

function filterExpectedWailsRuntimeConsoleErrors(consoleErrors, result) {
  let remainingExpectedRuntimeFailures = expectedWailsRuntime422FailureCount(result);
  if (remainingExpectedRuntimeFailures <= 0) {
    return consoleErrors;
  }
  return consoleErrors.filter(message => {
    if (remainingExpectedRuntimeFailures > 0 && isExpectedWailsRuntime422ConsoleError(message)) {
      remainingExpectedRuntimeFailures -= 1;
      return false;
    }
    return true;
  });
}

function attachPageDiagnostics(page, consoleErrors, requestFailures, httpFailures, pageErrors) {
  if (page.__deepStudentSmokeDiagnosticsAttached) return;
  page.__deepStudentSmokeDiagnosticsAttached = true;
  page.on('console', message => {
    if (message.type() === 'error') {
      consoleErrors.push(`${message.text()}${formatConsoleLocation(message)}`);
    }
  });
  page.on('pageerror', error => {
    pageErrors.push(error?.stack || error?.message || String(error));
  });
  page.on('requestfailed', request => {
    requestFailures.push(`${request.url()} ${request.failure()?.errorText ?? ''}`.trim());
  });
  page.on('response', response => {
    if (response.status() >= 400 && !isBenignWailsCustomScriptProbe(response)) {
      httpFailures.push(`${response.status()} ${response.request().method()} ${response.url()}`);
    }
  });
}

async function collectPageDiagnostics(page) {
  return await page.evaluate(() => {
    const resourceUrls = performance
      .getEntriesByType('resource')
      .map(entry => entry.name)
      .filter(name => typeof name === 'string');
    return {
      href: window.location.href,
      hookType: typeof window.__DEEP_STUDENT_GO_WAILS_SMOKE__,
      smokeSentinelRendered: Boolean(document.querySelector('[data-deep-student-smoke-rendered="true"]')),
      rootChildren: document.querySelector('#root')?.childElementCount ?? null,
      topLevelErrorBoundaryVisible: Boolean(document.querySelector('[data-deep-student-error-boundary="top-level"]')),
      resourceUrls,
      resourceUrlsTotal: resourceUrls.length,
      smokeEarlyErrors: Array.isArray(window.__DEEP_STUDENT_WAILS_SMOKE_EARLY_ERRORS__)
        ? window.__DEEP_STUDENT_WAILS_SMOKE_EARLY_ERRORS__.slice(0, 10)
        : [],
      wailsEnvironment: window._wails?.environment ?? null,
      wailsFlags: window._wails?.flags ?? null,
      wailsInvokeType: typeof window._wails?.invoke,
    };
  });
}

async function assertNoViteDevResources(page) {
  const diagnostics = await collectPageDiagnostics(page);
  const viteResources = diagnostics.resourceUrls.filter(isViteDevURL);
  if (isViteDevURL(diagnostics.href) || viteResources.length > 0) {
    fail(`live Wails smoke loaded Vite/dev-server resources: ${JSON.stringify({
      href: diagnostics.href,
      viteResources: viteResources.slice(0, 20),
    })}`);
  }
  return diagnostics;
}

let child;
let browser;
try {
  if (remoteDebuggingPort === 0) {
    remoteDebuggingPort = await getFreePort();
  }
  if (!Number.isInteger(remoteDebuggingPort) || remoteDebuggingPort < 1024 || remoteDebuggingPort > 65535) {
    fail('DEEP_STUDENT_WAILS_REMOTE_DEBUGGING_PORT must be a local TCP port from 1024 to 65535');
  }
  cdpURL = `http://127.0.0.1:${remoteDebuggingPort}`;
  await assertCDPClosed();

  runNpm(['run', 'build']);
  run('node', ['scripts/go-sync-frontend-dist.mjs']);
  run('node', ['scripts/go-frontend-embed-smoke.mjs']);
  assertEmbeddedDistHasSmokeHook();
  run('go', ['build', '-o', exePath, './cmd/deep-student-go'], { cwd: desktopGoDir });
  createLegacyTemplateSQLiteFixture();

  const env = {
    ...process.env,
    DEEP_STUDENT_DATA_DIR: dataDir,
    DEEP_STUDENT_WAILS_UI_SMOKE: '1',
    DEEP_STUDENT_WAILS_REMOTE_DEBUGGING_PORT: String(remoteDebuggingPort),
  };
  child = spawn(exePath, [], {
    cwd: tmpRoot,
    env,
    stdio: ['ignore', 'pipe', 'pipe'],
    windowsHide: true,
  });
  child.stdout.on('data', chunk => process.stdout.write(chunk));
  child.stderr.on('data', chunk => process.stderr.write(chunk));
  child.on('exit', (code, signal) => {
    childExit = { code, signal };
    if (!terminatingChild && code != null && code !== 0) {
      process.stderr.write(`[go-live-wails-smoke] app exited with code ${code}\n`);
    }
  });

  await waitForCDP(20000);
  browser = await chromium.connectOverCDP(cdpURL);
  const consoleErrors = [];
  const requestFailures = [];
  const httpFailures = [];
  const pageErrors = [];
  const attachContextDiagnostics = context => {
    for (const existingPage of context.pages()) {
      attachPageDiagnostics(existingPage, consoleErrors, requestFailures, httpFailures, pageErrors);
    }
    context.on('page', newPage => attachPageDiagnostics(newPage, consoleErrors, requestFailures, httpFailures, pageErrors));
  };
  for (const context of browser.contexts()) {
    attachContextDiagnostics(context);
  }

  assertChildAlive('after CDP attach');
  const page = await waitForSmokePage(browser, 20000);
  attachPageDiagnostics(page, consoleErrors, requestFailures, httpFailures, pageErrors);

  await page.waitForLoadState('domcontentloaded', { timeout: 20000 });
  await page.waitForFunction(() => Boolean(document.querySelector('#root')), null, { timeout: 20000 });
  try {
    await page.waitForFunction(() => {
      const root = document.querySelector('#root');
      const smokeSentinel = document.querySelector('[data-deep-student-smoke-rendered="true"]');
      const errorBoundary = document.querySelector('[data-deep-student-error-boundary="top-level"]');
      return Boolean(errorBoundary || smokeSentinel || (root && root.childElementCount > 0));
    }, null, { timeout: 30000 });
  } catch (error) {
    const diagnostics = await collectPageDiagnostics(page).catch(diagnosticError => ({
      diagnosticError: diagnosticError?.message ?? String(diagnosticError),
    }));
    fail(`timed out waiting for React to mount into #root: ${error?.message ?? error}\n${JSON.stringify(diagnostics, null, 2)}`);
  }
  try {
    await page.waitForFunction(() => typeof window.__DEEP_STUDENT_GO_WAILS_SMOKE__ === 'function', null, {
      timeout: 20000,
    });
  } catch (error) {
    const diagnostics = await collectPageDiagnostics(page).catch(diagnosticError => ({
      diagnosticError: diagnosticError?.message ?? String(diagnosticError),
    }));
    fail(`timed out waiting for frontend smoke hook: ${error?.message ?? error}\n${JSON.stringify(diagnostics, null, 2)}`);
  }
  await assertNoViteDevResources(page);
  assertChildAlive('before frontend smoke hook');
  const smokeOptions = includeMcpStdioSmoke || includeSkillSmoke || includeTemplateSmoke || includeVfsSmoke
    ? {
      mcpStdio: includeMcpStdioSmoke ? {
        command: exePath,
        args: ['--mcp-stdio-smoke-child'],
        cwd: tmpRoot,
        framing: 'content_length',
        timeoutMs: 12000,
      } : undefined,
      skills: includeSkillSmoke,
      templates: includeTemplateSmoke,
      vfs: includeVfsSmoke,
    }
    : undefined;
  const result = await page.evaluate(options => window.__DEEP_STUDENT_GO_WAILS_SMOKE__(options), smokeOptions);
  assertSmokeResult(result);
  if (includeMcpStdioSmoke) {
    assertMcpStdioSmokeResult(result);
  }
  if (includeSkillSmoke) {
    assertSkillSmokeResult(result);
  }
  if (includeTemplateSmoke) {
    assertTemplateSmokeResult(result);
  }
  if (includeVfsSmoke) {
    assertVfsSmokeResult(result);
  }
  const diagnostics = await assertNoViteDevResources(page);

  const fatalConsoleErrors = filterExpectedWailsRuntimeConsoleErrors(consoleErrors, result).filter(message => {
    const lower = message.toLowerCase();
    return (
      !lower.includes('favicon') &&
      !lower.includes('sentry') &&
      !lower.includes('resizeobserver loop') &&
      !lower.includes('download the react devtools') &&
      !isBenignWailsCustomScriptConsole(message)
    );
  });
  const unexpectedHTTPFailures = filterExpectedWailsRuntimeHTTPFailures(httpFailures, result);
  if (unexpectedHTTPFailures.length > 0) {
    fail(`resource HTTP failures during smoke: ${unexpectedHTTPFailures.join('\n')}`);
  }
  if (fatalConsoleErrors.length > 0) {
    fail(`console errors during smoke: ${fatalConsoleErrors.join('\n')}`);
  }
  if (pageErrors.length > 0) {
    fail(`page errors during smoke: ${pageErrors.join('\n')}`);
  }
  if (requestFailures.length > 0) {
    fail(`resource request failures during smoke: ${requestFailures.join('\n')}`);
  }
  assertChildAlive('before reporting smoke success');

  console.log(JSON.stringify({
    appDataDir: result.appDataDir,
    isWails: result.isWails,
    mcpStdio: includeMcpStdioSmoke ? result.mcpStdio : undefined,
    skills: includeSkillSmoke ? result.skills : undefined,
    templates: includeTemplateSmoke ? result.templates : undefined,
    vfs: includeVfsSmoke ? result.vfs : undefined,
    rootChildren: diagnostics.rootChildren,
    smokePort: remoteDebuggingPort,
    title: await page.title(),
  }, null, 2));
  console.log('[go-live-wails-smoke] ok');
} finally {
  if (browser) {
    await browser.close().catch(() => undefined);
  }
  if (child) {
    await terminate(child);
    await waitForExit(child, 5000);
  }
  await safeRemoveTreeWithRetry(tmpRoot, os.tmpdir(), 'live Wails smoke temp root');
}
