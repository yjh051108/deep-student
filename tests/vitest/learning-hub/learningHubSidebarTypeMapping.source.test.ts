import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('LearningHubSidebar resource type mapping contract', () => {
  const source = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/LearningHubSidebar.tsx'),
    'utf-8'
  );

  it('uses the shared DSTU node type mapping instead of defaulting unknown resources to notes', () => {
    expect(source).toContain('nodeTypeToFolderItemType');
    expect(source).toContain('const itemType = nodeTypeToFolderItemType(item.type);');
    expect(source).not.toContain("let itemType: FolderItemType = 'note'");
    expect(source).not.toContain("default: itemType = 'note'");
  });
});
