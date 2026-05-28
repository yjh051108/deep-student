import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('learning hub memory entry scope contract', () => {
  it('routes memory quick access to MemoryView instead of the raw memory root folder', () => {
    const sidebar = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/LearningHubSidebar.tsx'),
      'utf-8'
    );
    const page = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/LearningHubPage.tsx'),
      'utf-8'
    );
    const sidebarV2 = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/LearningHubSidebarV2.tsx'),
      'utf-8'
    );

    for (const source of [sidebar, page, sidebarV2]) {
      expect(source).not.toContain('enterFolder(config.memoryRootFolderId');
      expect(source).not.toContain('finderEnterFolder(config.memoryRootFolderId');
    }
    for (const source of [sidebar, sidebarV2]) {
      expect(source).toContain("quickAccessNavigate(type)");
    }

    expect(page).toContain("finderQuickAccessNavigate('memory')");
  });

  it('does not expose global write/export/profile controls from raw memory folders', () => {
    const banner = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/components/MemoryFolderBanner.tsx'),
      'utf-8'
    );

    expect(banner).not.toContain('writeMemorySmart');
    expect(banner).not.toContain('writeMemoryBatch');
    expect(banner).not.toContain('getMemoryProfile');
    expect(banner).not.toContain('exportAllMemories');
    expect(banner).toContain('getMemoryAuditLogs');
    expect(banner).toContain('setMemoryAutoExtractFrequency');
  });

  it('does not hardcode the memory tree preview to the global folder', () => {
    const preview = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/components/MemoryTreePreview.tsx'),
      'utf-8'
    );
    const sidebar = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/LearningHubSidebar.tsx'),
      'utf-8'
    );

    expect(preview).toContain('rootPath?: string');
    expect(preview).toContain('getMemoryTree(rootPath)');
    expect(preview).not.toContain("getMemoryTree('全局')");
    expect(sidebar).toContain('memoryTreeRootPath');
    expect(sidebar).toContain('rootPath={memoryTreeRootPath}');
  });

  it('keeps all-memory admin access behind an explicit MemoryView management mode', () => {
    const memoryView = readFileSync(
      resolve(process.cwd(), 'src/features/learning-hub/views/MemoryView.tsx'),
      'utf-8'
    );
    const memoryApi = readFileSync(resolve(process.cwd(), 'src/api/memoryApi.ts'), 'utf-8');
    const memoryExecutor = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/chat_v2/tools/memory_executor.rs'),
      'utf-8'
    );
    const pipelinePrompt = readFileSync(
      resolve(process.cwd(), 'src-tauri/src/chat_v2/pipeline/prompt.rs'),
      'utf-8'
    );

    expect(memoryView).toContain("useState<MemoryManagementScope>('topicAndGlobal')");
    expect(memoryView).toContain("adminAll: memoryScopeFilter === 'all'");
    expect(memoryView).toContain('memory.scope_all_confirm');
    expect(memoryView).toContain('Agent 默认仍只能看到当前课题 + 全局');
    expect(memoryView).toContain('memory.scope_all_warning');

    expect(memoryApi).toContain('context?: MemoryScopeContext');
    expect(memoryApi).not.toContain('adminAll: true');
    expect(memoryExecutor).not.toContain('admin_all');
    expect(memoryExecutor).not.toContain('adminAll');
    expect(pipelinePrompt).not.toContain('admin_all');
    expect(pipelinePrompt).not.toContain('adminAll');
  });
});
