import { afterEach, describe, expect, it, vi } from 'vitest';

const mockListDirectories = vi.hoisted(() => vi.fn());
const mockReadFile = vi.hoisted(() => vi.fn());
const mockCreate = vi.hoisted(() => vi.fn());
const mockUpdate = vi.hoisted(() => vi.fn());
const mockDelete = vi.hoisted(() => vi.fn());

vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/skillservice', () => ({
  ListDirectories: mockListDirectories,
  ReadFile: mockReadFile,
  Create: mockCreate,
  Update: mockUpdate,
  Delete: mockDelete,
}));

vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/ankiservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/chatservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/dstuservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/fileservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/mcpservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/notesservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/qbankservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/reviewplanservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/settingsservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/systemservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/todoservice', () => ({}));
vi.mock('@/runtime/wails-bindings/deep-student-go/internal/bindings/vfsservice', () => ({}));

describe('wails bridge skill service payload forwarding', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('routes skill_list_directories to SkillService.ListDirectories', async () => {
    const { invokeWails } = await import('@/runtime/wailsBridge');
    mockListDirectories.mockResolvedValue([{ name: 'biology', path: 'D:/skills/biology' }]);

    await expect(invokeWails('skill_list_directories', { path: 'D:/skills' })).resolves.toEqual([
      { name: 'biology', path: 'D:/skills/biology' },
    ]);

    expect(mockListDirectories).toHaveBeenCalledWith('D:/skills');
  });

  it('routes skill_read_file to SkillService.ReadFile', async () => {
    const { invokeWails } = await import('@/runtime/wailsBridge');
    mockReadFile.mockResolvedValue({ path: 'D:/skills/biology/SKILL.md', content: '# Biology' });

    await expect(invokeWails('skill_read_file', {
      path: 'D:/skills/biology/SKILL.md',
    })).resolves.toEqual({
      path: 'D:/skills/biology/SKILL.md',
      content: '# Biology',
    });

    expect(mockReadFile).toHaveBeenCalledWith('D:/skills/biology/SKILL.md');
  });

  it('routes skill_create to SkillService.Create with legacy skillId input', async () => {
    const { invokeWails } = await import('@/runtime/wailsBridge');
    mockCreate.mockResolvedValue({ path: 'D:/skills/biology/SKILL.md', content: '# Biology' });

    await expect(invokeWails('skill_create', {
      basePath: 'D:/skills',
      skillId: 'biology',
      content: '# Biology',
    })).resolves.toEqual({
      path: 'D:/skills/biology/SKILL.md',
      content: '# Biology',
    });

    expect(mockCreate).toHaveBeenCalledWith('D:/skills', 'biology', '# Biology');
  });

  it('routes skill_update to SkillService.Update', async () => {
    const { invokeWails } = await import('@/runtime/wailsBridge');
    mockUpdate.mockResolvedValue({ path: 'D:/skills/biology/SKILL.md', content: '# Updated' });

    await expect(invokeWails('skill_update', {
      path: 'D:/skills/biology/SKILL.md',
      content: '# Updated',
    })).resolves.toEqual({
      path: 'D:/skills/biology/SKILL.md',
      content: '# Updated',
    });

    expect(mockUpdate).toHaveBeenCalledWith('D:/skills/biology/SKILL.md', '# Updated');
  });

  it('routes skill_delete to SkillService.Delete', async () => {
    const { invokeWails } = await import('@/runtime/wailsBridge');
    mockDelete.mockResolvedValue(undefined);

    await expect(invokeWails('skill_delete', { path: 'D:/skills/biology' })).resolves.toBeUndefined();

    expect(mockDelete).toHaveBeenCalledWith('D:/skills/biology');
  });
});
