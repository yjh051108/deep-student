import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

function readProjectFile(path: string): string {
  const absolutePath = resolve(process.cwd(), path);
  expect(existsSync(absolutePath), `${path} should exist`).toBe(true);
  return readFileSync(absolutePath, 'utf-8');
}

describe('Anki template startup native facade contract', () => {
  it('routes template startup and management paths through the native facade', () => {
    const source = readProjectFile('src/data/ankiTemplates.ts');
    const templateManager = readProjectFile('src/components/TemplateManager.tsx');
    const templateManagementPage = readProjectFile('src/components/TemplateManagementPage.tsx');
    const forceImportTemplates = readProjectFile('src/utils/forceImportTemplates.ts');

    expect(source).toContain("import { invoke as nativeInvoke } from '@/runtime/native'");
    expect(source).not.toContain("@tauri-apps/api/core");
    expect(source).not.toMatch(/const\s+\{\s*invoke\s*\}\s*=\s*await\s+import/);
    expect(source).toContain("nativeInvoke<CustomAnkiTemplate[]>('get_all_custom_templates')");
    expect(source).toContain("nativeInvoke<string | null>('get_default_template_id')");
    for (const templateSource of [templateManager, templateManagementPage, forceImportTemplates]) {
      expect(templateSource).toContain("from '@/runtime/native'");
      expect(templateSource).not.toContain("@tauri-apps/api/core");
    }
  });

  it('uses the real Go/Wails TemplateService for template commands', () => {
    const wailsBridge = readProjectFile('src/runtime/wailsBridge.ts');
    const nativeFacade = readProjectFile('src/runtime/native.ts');
    const mainGo = readProjectFile('desktop-go/cmd/deep-student-go/main.go');
    const appGo = readProjectFile('desktop-go/internal/app/app.go');
    const appPathsGo = readProjectFile('desktop-go/internal/app/paths.go');
    const bindingGo = readProjectFile('desktop-go/internal/bindings/template_service.go');
    const generatedBinding = readProjectFile('src/runtime/wails-bindings/deep-student-go/internal/bindings/templateservice.ts');

    for (const source of [wailsBridge, nativeFacade]) {
      expect(source).toContain("command === 'import_builtin_templates'");
      expect(source).toContain("command === 'get_all_custom_templates'");
      expect(source).toContain("command === 'get_default_template_id'");
      expect(source).toContain("command === 'create_custom_template'");
      expect(source).toContain("command === 'update_custom_template'");
      expect(source).toContain("command === 'delete_custom_template'");
      expect(source).toContain("command === 'set_default_template'");
    }

    expect(wailsBridge).toContain("import * as TemplateService");
    expect(wailsBridge).toContain('TemplateService.ImportBuiltinTemplates()');
    expect(wailsBridge).toContain('TemplateService.GetAllCustomTemplates()');
    expect(wailsBridge).toContain('TemplateService.GetDefaultTemplateID()');
    expect(wailsBridge).toContain('TemplateService.CreateCustomTemplate(request)');
    expect(wailsBridge).toContain('TemplateService.UpdateCustomTemplate(templateId, request)');
    expect(wailsBridge).toContain('TemplateService.DeleteCustomTemplate(templateId)');
    expect(wailsBridge).toContain('TemplateService.SetDefaultTemplate(templateId)');
    expect(wailsBridge).toContain("requireStringArgAny(command, args, ['template_data', 'templateData'])");
    expect(wailsBridge).toContain("optionalBooleanArgAny(args, ['overwrite_existing', 'overwriteExisting'])");
    expect(wailsBridge).toContain("optionalBooleanArgAny(args, ['strict_builtin', 'strictBuiltin'])");
    expect(wailsBridge).toContain('TemplateService.ImportCustomTemplatesBulk(templateData, overwriteExisting, strictBuiltin)');
    expect(wailsBridge).toContain('TemplateService.ExportTemplate(templateId)');
    expect(wailsBridge).not.toContain('Wails template command is deferred');

    expect(nativeFacade).toContain('Template command is not available in this native fallback');
    expect(mainGo).toContain('bindings.NewTemplateService(applicationState)');
    expect(appGo).toContain('Templates *templates.Service');
    expect(appGo).toContain('templates.NewServiceWithLegacyRoots(dataDir, LegacyDataDirCandidates(dataDir))');
    expect(appPathsGo).toContain('com.deepstudent.app');
    expect(appPathsGo).toContain('DEEP_STUDENT_DATA_DIR');
    expect(bindingGo).toContain('func (s *TemplateService) ImportCustomTemplatesBulk(templateData string, overwriteExisting bool, strictBuiltin bool)');
    expect(generatedBinding).toContain('export function ImportCustomTemplatesBulk');
    expect(generatedBinding).toContain('export function ExportTemplate');
  });

  it('migrates legacy Rust SQLite template data into the Go JSON store', () => {
    const templateServiceGo = readProjectFile('desktop-go/internal/templates/service.go');
    const legacySqliteGo = readProjectFile('desktop-go/internal/templates/legacy_sqlite.go');
    const templateTestsGo = readProjectFile('desktop-go/internal/templates/service_test.go');

    expect(templateServiceGo).toContain('service.migrateLegacySQLiteIfPresent(legacyRoots)');
    expect(templateServiceGo).toContain('func NewServiceWithLegacyRoots(dataDir string, legacyRoots []string)');
    expect(templateServiceGo).toMatch(/LegacyMigration\s+\*Migration `json:"legacyMigration,omitempty"`/);
    expect(legacySqliteGo).toContain('_ "modernc.org/sqlite"');
    expect(legacySqliteGo).toContain('const legacySQLiteFileName = "mistakes.db"');
    expect(legacySqliteGo).toContain('sqliteTableExists(db, "custom_anki_templates")');
    expect(legacySqliteGo).toContain('readLegacyDefaultTemplateID');
    expect(legacySqliteGo).toContain('legacySQLiteCandidatePaths');
    expect(legacySqliteGo).toContain('if isBuiltIn && !includeBuiltins');
    expect(legacySqliteGo).toContain('s.state.DefaultTemplateID == nil || s.findIndexByID(*s.state.DefaultTemplateID) < 0');
    expect(legacySqliteGo).toContain('s.state.LegacyMigration = &result');
    expect(templateTestsGo).toContain('TestNewServiceMigratesLegacySQLiteCustomTemplates');
    expect(templateTestsGo).toContain('TestLegacySQLiteMigrationPreservesExistingGoTemplates');
    expect(templateTestsGo).toContain('TestLegacySQLiteMigrationHandlesMissingColumnsAndBadJSON');
    expect(templateTestsGo).toContain('legacySQLiteFileName');
    expect(templateTestsGo).toContain('legacy migration should be idempotent');
    expect(templateTestsGo).toContain('bad fields_json should fall back to Front/Back');
  });
});
