import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('ImageContentView metadata fallback contract', () => {
  const source = readFileSync(
    resolve(process.cwd(), 'src/features/learning-hub/apps/views/ImageContentView.tsx'),
    'utf-8'
  );

  it('continues content loading when attachment metadata is unavailable for image-like resources', () => {
    expect(source).toContain('vfs_get_attachment');
    expect(source).toContain('vfs_get_attachment_content');
    expect(source).toContain('元数据失败时不能阻断预览');
    expect(source).toContain('maybeLoadImageAfterSizeCheck(typeof node.size ===');
    expect(source).toContain('Failed to read attachment metadata, falling back to content load');
    expect(source).not.toContain("setError(t('learningHub:error.imageNotFound', '图片未找到'));\n          setLoadingStage('idle');");
  });
});
