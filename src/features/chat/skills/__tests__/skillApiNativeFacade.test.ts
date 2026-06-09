import { afterEach, describe, expect, it, vi } from 'vitest';

const invokeMock = vi.hoisted(() => vi.fn());

vi.mock('@/runtime/native', () => ({
  invoke: invokeMock,
}));

import {
  createSkill,
  deleteSkill,
  listSkillDirectories,
  readSkillFile,
  updateSkill,
} from '../api';

describe('skill API native facade', () => {
  afterEach(() => {
    invokeMock.mockReset();
  });

  it('routes listSkillDirectories through the native facade', async () => {
    invokeMock.mockResolvedValue([{ name: 'biology', path: 'D:/skills/biology' }]);

    await expect(listSkillDirectories('D:/skills')).resolves.toEqual([
      { name: 'biology', path: 'D:/skills/biology' },
    ]);

    expect(invokeMock).toHaveBeenCalledWith('skill_list_directories', { path: 'D:/skills' });
  });

  it('routes readSkillFile through the native facade', async () => {
    invokeMock.mockResolvedValue({ path: 'D:/skills/biology/SKILL.md', content: '# Biology' });

    await expect(readSkillFile('D:/skills/biology/SKILL.md')).resolves.toEqual({
      path: 'D:/skills/biology/SKILL.md',
      content: '# Biology',
    });

    expect(invokeMock).toHaveBeenCalledWith('skill_read_file', {
      path: 'D:/skills/biology/SKILL.md',
    });
  });

  it('routes createSkill through the native facade with legacy argument casing', async () => {
    invokeMock.mockResolvedValue({ path: 'D:/skills/biology/SKILL.md', content: '# Biology' });

    await expect(createSkill({
      basePath: 'D:/skills',
      skillId: 'biology',
      content: '# Biology',
    })).resolves.toEqual({
      path: 'D:/skills/biology/SKILL.md',
      content: '# Biology',
    });

    expect(invokeMock).toHaveBeenCalledWith('skill_create', {
      basePath: 'D:/skills',
      skillId: 'biology',
      content: '# Biology',
    });
  });

  it('routes updateSkill through the native facade', async () => {
    invokeMock.mockResolvedValue({ path: 'D:/skills/biology/SKILL.md', content: '# Updated' });

    await expect(updateSkill({
      path: 'D:/skills/biology/SKILL.md',
      content: '# Updated',
    })).resolves.toEqual({
      path: 'D:/skills/biology/SKILL.md',
      content: '# Updated',
    });

    expect(invokeMock).toHaveBeenCalledWith('skill_update', {
      path: 'D:/skills/biology/SKILL.md',
      content: '# Updated',
    });
  });

  it('routes deleteSkill through the native facade', async () => {
    invokeMock.mockResolvedValue(undefined);

    await expect(deleteSkill('D:/skills/biology')).resolves.toBeUndefined();

    expect(invokeMock).toHaveBeenCalledWith('skill_delete', { path: 'D:/skills/biology' });
  });
});
