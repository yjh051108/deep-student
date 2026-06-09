import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

function readProjectFile(path: string): string {
  const absolutePath = resolve(process.cwd(), path);
  expect(existsSync(absolutePath), `${path} should exist`).toBe(true);
  return readFileSync(absolutePath, 'utf-8');
}

describe('live Go/Wails smoke source contract', () => {
  it('exposes the live Wails smoke as an npm command', () => {
    const packageJson = JSON.parse(readProjectFile('package.json')) as {
      scripts?: Record<string, string>;
    };

    expect(packageJson.scripts?.['go:smoke:live-wails']).toBe('node scripts/go-live-wails-smoke.mjs');
    expect(packageJson.scripts?.['go:smoke:live-wails-mcp']).toBe('node scripts/go-live-wails-smoke.mjs --mcp');
    expect(packageJson.scripts?.['go:smoke:live-wails-skills']).toBe('node scripts/go-live-wails-smoke.mjs --skills');
    expect(packageJson.scripts?.['go:smoke:live-wails-templates']).toBe('node scripts/go-live-wails-smoke.mjs --templates');
    expect(packageJson.scripts?.['go:smoke:live-wails-vfs']).toBe('node scripts/go-live-wails-smoke.mjs --vfs');
    expect(packageJson.scripts?.['go:smoke:live-wails-core']).toBe('node scripts/go-live-wails-smoke.mjs --mcp --skills --templates --vfs');
  });

  it('adds a dedicated live Wails smoke runner that drives WebView remote debugging', () => {
    const smokeScript = readProjectFile('scripts/go-live-wails-smoke.mjs');

    expect(smokeScript).toContain('chromium');
    expect(smokeScript).toContain('connectOverCDP');
    expect(smokeScript).toContain("['run', 'build']");
    expect(smokeScript).toContain('assertEmbeddedDistHasSmokeHook');
    expect(smokeScript).toContain('DEEP_STUDENT_WAILS_REMOTE_DEBUGGING_PORT');
    expect(smokeScript).toContain('process.env.DEEP_STUDENT_WAILS_REMOTE_DEBUGGING_PORT');
    expect(smokeScript).toContain('DEEP_STUDENT_WAILS_UI_SMOKE');
    expect(smokeScript).toContain('DEEP_STUDENT_DATA_DIR');
    expect(smokeScript).not.toContain('DEEP_STUDENT_WAILS_SMOKE_PORT');
    expect(smokeScript).toContain('__DEEP_STUDENT_GO_WAILS_SMOKE__');
    expect(smokeScript).toContain('window.__DEEP_STUDENT_GO_WAILS_SMOKE__');
    expect(smokeScript).toContain("parsed.hostname === '127.0.0.1'");
    expect(smokeScript).toContain("parsed.hostname === 'localhost'");
    expect(smokeScript).toContain('assertNoViteDevResources');
    expect(smokeScript).toContain('page.on(\'response\'');
    expect(smokeScript).toContain('response.status() >= 400');
    expect(smokeScript).toContain('response.url()');
    expect(smokeScript).toContain('httpFailures');
    expect(smokeScript).toContain('isBenignWailsCustomScriptProbe');
    expect(smokeScript).toContain("parsed.hostname === 'wails.localhost'");
    expect(smokeScript).toContain("parsed.pathname === '/wails/custom.js'");
    expect(smokeScript).toContain("response.request().method() !== 'HEAD'");
    expect(smokeScript).toContain('resource HTTP failures during smoke');
    expect(smokeScript).toContain('resourceUrlsTotal');
    expect(smokeScript).toContain('result.isWails !== true');
    expect(smokeScript).toContain('result.smokeFlag !== true');
    expect(smokeScript).toContain('result.rootMounted !== true');
    expect(smokeScript).toContain('result.smokeSentinelRendered !== true');
    expect(smokeScript).toContain('result.topLevelErrorBoundaryVisible === true');
    expect(smokeScript).toContain('settings-go.json');
    expect(smokeScript).toContain('assertChildAlive');
    expect(smokeScript).not.toMatch(/\bcargo\b/i);
    expect(smokeScript).not.toContain('@tauri-apps');
    expect(smokeScript).not.toContain('src-tauri');
    expect(smokeScript).not.toMatch(/\btauri\s+dev\b/i);
    expect(smokeScript).not.toMatch(/\btauri\s+build\b/i);
  });

  it('asserts the live Wails smoke uses a temp data directory', () => {
    const smokeScript = readProjectFile('scripts/go-live-wails-smoke.mjs');

    expect(smokeScript).toContain("fs.mkdtempSync(path.join(os.tmpdir(), 'deep-student-go-live-wails-smoke-'))");
    expect(smokeScript).toContain("const dataDir = path.join(tmpRoot, 'data')");
    expect(smokeScript).toContain('DEEP_STUDENT_DATA_DIR: dataDir');
    expect(smokeScript).toContain('result.appDataDir !== dataDir');
    expect(smokeScript).toContain('assertSettingsPersisted(result)');
    expect(smokeScript).toContain("await safeRemoveTreeWithRetry(tmpRoot, os.tmpdir(), 'live Wails smoke temp root')");
  });

  it('asserts the live Wails smoke proves MCP stdio over the real Wails bridge', () => {
    const smokeScript = readProjectFile('scripts/go-live-wails-smoke.mjs');

    expect(smokeScript).toContain('assertMcpStdioSmokeResult');
    expect(smokeScript).toContain('includeMcpStdioSmoke');
    expect(smokeScript).toContain("'--mcp-stdio-smoke-child'");
    expect(smokeScript).toContain('result.mcpStdio');
    expect(smokeScript).toContain('mcp_stdio_start');
    expect(smokeScript).toContain('mcp_stdio_send');
    expect(smokeScript).toContain('mcp_stdio_close');
    expect(smokeScript).toContain('McpService.StartStdioSession');
    expect(smokeScript).toContain('McpService.SendStdioMessage');
    expect(smokeScript).toContain('McpService.CloseStdioSession');
    expect(smokeScript).toContain('messageEventReceived');
    expect(smokeScript).toContain('closedEventReceived');
    expect(smokeScript).toContain('errorEventReceived');
    expect(smokeScript).toContain('browserFallbackRejected');
    expect(smokeScript).toContain('tauriFallbackRejected');
  });

  it('asserts the live Wails smoke proves SkillService file CRUD over the real Wails bridge', () => {
    const smokeScript = readProjectFile('scripts/go-live-wails-smoke.mjs');

    expect(smokeScript).toContain('includeSkillSmoke');
    expect(smokeScript).toContain('assertSkillSmokeResult');
    expect(smokeScript).toContain('result.skills');
    expect(smokeScript).toContain('skill_create');
    expect(smokeScript).toContain('skill_read_file');
    expect(smokeScript).toContain('skill_update');
    expect(smokeScript).toContain('skill_list_directories');
    expect(smokeScript).toContain('skill_delete');
    expect(smokeScript).toContain('SkillService.Create');
    expect(smokeScript).toContain('SkillService.ReadFile');
    expect(smokeScript).toContain('SkillService.Update');
    expect(smokeScript).toContain('SkillService.ListDirectories');
    expect(smokeScript).toContain('SkillService.Delete');
    expect(smokeScript).toContain("path.join(dataDir, 'skills')");
    expect(smokeScript).toContain('normalizeSmokePath');
    expect(smokeScript).toContain('isInsideSmokePath');
    expect(smokeScript).not.toContain("skills.basePath !== path.join(dataDir, 'skills')");
    expect(smokeScript).toContain('expectedWailsRuntime422FailureCount');
    expect(smokeScript).toContain('filterExpectedWailsRuntimeHTTPFailures');
    expect(smokeScript).toContain('result?.skills?.readAfterDeleteRejected === true');
    expect(smokeScript).toContain('readAfterDeleteRejected');
  });

  it('asserts the live Wails smoke proves TemplateService CRUD/import/export over the real Wails bridge', () => {
    const smokeScript = readProjectFile('scripts/go-live-wails-smoke.mjs');

    expect(smokeScript).toContain('includeTemplateSmoke');
    expect(smokeScript).toContain('assertTemplateSmokeResult');
    expect(smokeScript).toContain('result.templates');
    expect(smokeScript).toContain('templates-go.json');
    expect(smokeScript).toContain('createLegacyTemplateSQLiteFixture');
    expect(smokeScript).toContain('create-legacy-template-db.go');
    expect(smokeScript).toContain('legacy-wails-smoke-template');
    expect(smokeScript).toContain('legacyMigration.imported !== 1');
    expect(smokeScript).toContain('import_builtin_templates -> TemplateService.ImportBuiltinTemplates');
    expect(smokeScript).toContain('get_all_custom_templates -> TemplateService.GetAllCustomTemplates');
    expect(smokeScript).toContain('get_default_template_id -> TemplateService.GetDefaultTemplateID');
    expect(smokeScript).toContain('create_custom_template -> TemplateService.CreateCustomTemplate');
    expect(smokeScript).toContain('update_custom_template -> TemplateService.UpdateCustomTemplate');
    expect(smokeScript).toContain('set_default_template -> TemplateService.SetDefaultTemplate');
    expect(smokeScript).toContain('import_custom_templates_bulk -> TemplateService.ImportCustomTemplatesBulk');
    expect(smokeScript).toContain('export_template -> TemplateService.ExportTemplate');
    expect(smokeScript).toContain('importedHasLegacyFieldsJson');
    expect(smokeScript).toContain('legacyMigratedName');
    expect(smokeScript).toContain('storedCreated.description !== templates.updatedDescription');
    expect(smokeScript).toContain('store.defaultTemplateId !== templates.createdId');
    expect(smokeScript).toContain('templates: includeTemplateSmoke');
  });

  it('asserts the live Wails smoke proves PDF textbook import over Go hybrid VFS', () => {
    const smokeScript = readProjectFile('scripts/go-live-wails-smoke.mjs');

    expect(smokeScript).toContain('includeVfsSmoke');
    expect(smokeScript).toContain('assertVfsSmokeResult');
    expect(smokeScript).toContain('result.vfs');
    expect(smokeScript).toContain('vfs: includeVfsSmoke');
    expect(smokeScript).toContain('assertVfsSmokeResult(result)');
    expect(smokeScript).toContain('vfs.ok');
    expect(smokeScript).toContain('vfs.sourcePath');
    expect(smokeScript).toContain('vfs.resourceId');
    expect(smokeScript).toContain('vfs.resourceHash');
    expect(smokeScript).toContain('vfs.fileId');
    expect(smokeScript).toContain('vfs.textbookType');
    expect(smokeScript).toContain('vfs.previewType');
    expect(smokeScript).toContain('vfs.resourceType');
    expect(smokeScript).toContain('vfs.resourceStorageMode');
    expect(smokeScript).toContain('vfs.resourceOriginalPath');
    expect(smokeScript).toContain('vfs.resourceExternalPath');
    expect(smokeScript).toContain('vfs.resourceResolvedPath');
    expect(smokeScript).toContain('vfs.fileContentFound');
    expect(smokeScript).toContain('vfs.rawContentContainsSentinel');
    expect(smokeScript).toContain('vfs.extractedTextContainsSentinel');
    expect(smokeScript).toContain('vfs.fileLookupMatchesResource');
    expect(smokeScript).toContain('vfs.fileLookupMatchesHash');
    expect(smokeScript).toContain('vfs.pdfMediaType');
    expect(smokeScript).toContain('vfs.pdfStage');
    expect(smokeScript).toContain('vfs.pdfReadyText');
    expect(smokeScript).toContain('vfs.pdfPageCount');
    expect(smokeScript).toContain('vfs.batchPdfStage');
    expect(smokeScript).toContain('vfs.batchPdfReadyText');
    expect(smokeScript).toContain('vfs.batchPdfPageCount');
    expect(smokeScript).toContain('vfs.progressStages');
    expect(smokeScript).toContain('vfs.doneEventMatchesIdentity');
    expect(smokeScript).toContain('textbooks_add -> DstuService.AddTextbooks');
    expect(smokeScript).toContain('textbook-import-progress -> Wails EventBus');
    expect(smokeScript).toContain('vfs_get_resource -> VfsService.GetResource');
    expect(smokeScript).toContain('vfs_get_file -> VfsService.GetFile');
    expect(smokeScript).toContain('vfs_get_file_content -> VfsService.GetFileContent');
    expect(smokeScript).toContain('vfs_get_pdf_processing_status -> VfsService.GetPdfProcessingStatus');
    expect(smokeScript).toContain('vfs_get_batch_pdf_processing_status -> VfsService.GetBatchPdfProcessingStatus');
    expect(smokeScript).toContain("path.join(dataDir, 'smoke-fixtures')");
    expect(smokeScript).not.toContain('vfs_get_pdf_page_image -> VfsService.GetPdfPageImage');
  });

  it('exposes a frontend smoke hook for the live Go/Wails runner', () => {
    const mainSource = readProjectFile('src/main.tsx');

    expect(mainSource).toContain('__DEEP_STUDENT_GO_WAILS_SMOKE__');
    expect(mainSource).toContain('__DEEP_STUDENT_WAILS_SMOKE_EARLY_ERRORS__');
    expect(mainSource).toContain('suppressed-tauri-http-noise');
    expect(mainSource).toContain('suppressed-tauri-http-console');
    expect(mainSource).toContain('go-wails-smoke');
    expect(mainSource).toContain('native.runtime.isWails()');
    expect(mainSource).toContain('native.runtime.isInjected()');
    expect(mainSource).toContain('deepStudentWailsSmoke');
    expect(mainSource).toContain('window.setInterval');
    expect(mainSource).toContain('data-deep-student-error-boundary="top-level"');
    expect(mainSource).toContain('data-deep-student-smoke-rendered="true"');
    expect(mainSource).toContain('rootMounted');
    expect(mainSource).toContain('smokeSentinelRendered');
    expect(mainSource).toContain('hasWailsInvoke');
    expect(mainSource).toContain('renderApp();');
    expect(mainSource).toContain('initSentryIfConfigured().catch');
    expect(mainSource).toMatch(/\(window as any\)\.__DEEP_STUDENT_GO_WAILS_SMOKE__\s*=\s*async/);
  });

  it('exposes frontend MCP stdio assertions in the live Wails smoke hook', () => {
    const mainSource = readProjectFile('src/main.tsx');

    expect(mainSource).toContain('runMcpStdioWailsSmoke');
    expect(mainSource).toContain('mcp_stdio_start');
    expect(mainSource).toContain('mcp_stdio_send');
    expect(mainSource).toContain('mcp_stdio_close');
    expect(mainSource).toContain('mcp-stdio-${sessionId}-message');
    expect(mainSource).toContain('mcp-stdio-${sessionId}-closed');
    expect(mainSource).toContain('mcp-stdio-${sessionId}-error');
    expect(mainSource).toContain('messageEventReceived');
    expect(mainSource).toContain('closedEventReceived');
    expect(mainSource).toContain('errorEventReceived');
    expect(mainSource).toContain('browserFallbackRejected');
    expect(mainSource).toContain('tauriFallbackRejected');
    expect(mainSource).toContain('mcpStdio');
  });

  it('exposes frontend SkillService assertions in the live Wails smoke hook', () => {
    const mainSource = readProjectFile('src/main.tsx');

    expect(mainSource).toContain('runSkillWailsSmoke');
    expect(mainSource).toContain('skill_create');
    expect(mainSource).toContain('skill_read_file');
    expect(mainSource).toContain('skill_update');
    expect(mainSource).toContain('skill_list_directories');
    expect(mainSource).toContain('skill_delete');
    expect(mainSource).toContain('Wails Smoke Skill');
    expect(mainSource).toContain('readAfterDeleteRejected');
    expect(mainSource).toContain('isPathInsideForSmoke');
    expect(mainSource).toContain("normalizedChild === normalizedParent || normalizedChild.startsWith(`${normalizedParent}/`)");
    expect(mainSource).toContain('skills');

    const skillSmokeStart = mainSource.indexOf('const runSkillWailsSmoke');
    const skillSmokeEnd = mainSource.indexOf('const installGoWailsSmokeHook');
    const skillSmokeBody = mainSource.slice(skillSmokeStart, skillSmokeEnd);
    expect(skillSmokeBody).toContain("await import('./features/chat/skills/api')");
    expect(skillSmokeBody).toContain('createSkill');
    expect(skillSmokeBody).toContain('readSkillFile');
    expect(skillSmokeBody).toContain('updateSkill');
    expect(skillSmokeBody).toContain('listSkillDirectories');
    expect(skillSmokeBody).toContain('deleteSkill');
    expect(skillSmokeBody).not.toContain("nativeInvoke('skill_");
    expect(skillSmokeBody).not.toContain('nativeInvoke("skill_');
  });

  it('exposes frontend VFS textbook assertions in the live Wails smoke hook', () => {
    const mainSource = readProjectFile('src/main.tsx');

    expect(mainSource).toContain('runVfsWailsSmoke');
    expect(mainSource).toContain('options?: GoWailsSmokeOptions');
    expect(mainSource).toContain('vfs?: boolean');
    expect(mainSource).toContain('templates?: boolean');
    expect(mainSource).toContain('options?.vfs');
    expect(mainSource).toContain('options?.templates');
    expect(mainSource).toContain('const templates = options?.templates ? await runTemplateWailsSmoke() : undefined');
    expect(mainSource).toContain('const vfs = options?.vfs ? await runVfsWailsSmoke() : undefined');
    expect(mainSource).toContain('templates,');
    expect(mainSource).toContain('vfs,');
    expect(mainSource).toContain('buildMinimalTextPdfForSmoke');
    expect(mainSource).toContain('Live Wails textbook hybrid VFS smoke');
    expect(mainSource).toContain("import('./dstu/adapters/textbookDstuAdapter')");
    expect(mainSource).toContain("import('./api/vfsFileApi')");
    expect(mainSource).toContain("import('./api/vfsPdfProcessingApi')");
    expect(mainSource).toContain("import('./api/vfsUnifiedIndexApi')");
    expect(mainSource).toContain("listen<TextbookImportProgressSmokePayload>('textbook-import-progress'");
    expect(mainSource).toContain('native.files.saveText(sourcePath, pdfContent)');
    expect(mainSource).toContain('textbookDstuAdapter.addTextbooks([sourcePath], null)');
    expect(mainSource).toContain("nativeInvoke<any>('vfs_get_resource', { resourceId })");
    expect(mainSource).toContain("nativeInvoke<string | null>('vfs_get_resource_path', { sourceId: resourceId })");
    expect(mainSource).toContain('vfsFileApi.get(fileId)');
    expect(mainSource).toContain('vfsFileApi.getContent(fileId)');
    expect(mainSource).toContain('vfsPdfProcessingApi.getStatus(fileId)');
    expect(mainSource).toContain('vfsPdfProcessingApi.getBatchStatus([fileId])');
    expect(mainSource).toContain('getAllIndexStatus({ resourceType: \'textbook\', limit: 50 })');
    expect(mainSource).toContain('doneEventMatchesIdentity');
    expect(mainSource).toContain('rawContentContainsSentinel');
    expect(mainSource).toContain('extractedTextContainsSentinel');
    expect(mainSource).toContain('batchPdfReadyText');
    expect(mainSource).toContain('batchPdfPageCount');
    expect(mainSource).toContain('batchPdfTerminal');
    expect(mainSource).toContain('fileLookupMatchesResource');
    expect(mainSource).toContain('fileLookupMatchesHash');
    expect(mainSource).not.toContain('getFileLikeContent(fileId)');
    expect(mainSource).not.toContain('vfs_get_pdf_page_image');

    const vfsSmokeStart = mainSource.indexOf('const runVfsWailsSmoke');
    const vfsSmokeEnd = mainSource.indexOf('const installGoWailsSmokeHook');
    const vfsSmokeBody = mainSource.slice(vfsSmokeStart, vfsSmokeEnd);
    const extractedTextStart = vfsSmokeBody.indexOf('const extractedTextContainsSentinel');
    const extractedTextEnd = vfsSmokeBody.indexOf(';', extractedTextStart);
    const extractedTextExpression = vfsSmokeBody.slice(extractedTextStart, extractedTextEnd);
    expect(extractedTextExpression).toContain('fileExtractedText.includes(sentinel)');
    expect(extractedTextExpression).toContain('resourceExtractedText.includes(sentinel)');
    expect(extractedTextExpression).not.toContain('decodedContent');
  });

  it('exposes frontend TemplateService assertions in the live Wails smoke hook', () => {
    const mainSource = readProjectFile('src/main.tsx');

    expect(mainSource).toContain('runTemplateWailsSmoke');
    expect(mainSource).toContain("nativeInvoke<string>('import_builtin_templates')");
    expect(mainSource).toContain("nativeInvoke<any[]>('get_all_custom_templates')");
    expect(mainSource).toContain("template?.id === 'legacy-wails-smoke-template'");
    expect(mainSource).toContain('legacyMigratedName');
    expect(mainSource).toContain("nativeInvoke<string>('create_custom_template'");
    expect(mainSource).toContain("nativeInvoke('set_default_template', { templateId })");
    expect(mainSource).toContain("nativeInvoke<string | null>('get_default_template_id')");
    expect(mainSource).toContain("nativeInvoke('update_custom_template'");
    expect(mainSource).toContain("nativeInvoke<{ template_data: string }>('export_template'");
    expect(mainSource).toContain("nativeInvoke<string>('import_custom_templates_bulk'");
    expect(mainSource).toContain('fields_json');
    expect(mainSource).toContain('field_extraction_rules_json');
    expect(mainSource).toContain('importedHasLegacyFieldsJson');
    expect(mainSource).toContain('routeImportBuiltin');
    expect(mainSource).toContain('routeImportBulk');
    expect(mainSource).toContain('routeExport');
  });
});
