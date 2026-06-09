import { readFileSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

const goVfsServicePath = path.join(process.cwd(), 'desktop-go/internal/vfs/service.go');

describe('vfs attachment ready modes source contract', () => {
  it('reports image mode as ready from the Go hybrid VFS upload path', () => {
    const source = readFileSync(goVfsServicePath, 'utf8');
    const uploadBody = source.slice(
      source.indexOf('func (s *Service) UploadAttachment('),
      source.indexOf('func (s *Service) GetAttachment(', source.indexOf('func (s *Service) UploadAttachment('))
    );
    const stateBody = source.slice(
      source.indexOf('func processingStateForAttachment('),
      source.indexOf('type previewPageImageRef', source.indexOf('func processingStateForAttachment('))
    );
    const imageBody = stateBody.slice(
      stateBody.indexOf('strings.HasPrefix(mimeType, "image/") || attachment.Type == "image"'),
      stateBody.indexOf('if mimeType == "application/pdf"')
    );

    expect(uploadBody.length).toBeGreaterThan(0);
    expect(stateBody.length).toBeGreaterThan(0);
    expect(uploadBody).toContain('return uploadAttachmentResult(resource, true), nil');
    expect(uploadBody).toContain('return uploadAttachmentResult(resource, false), nil');
    expect(imageBody).toContain('strings.HasPrefix(mimeType, "image/") || attachment.Type == "image"');
    expect(imageBody).toContain('status := "completed"');
    expect(imageBody).toContain('percent := 100.0');
    expect(imageBody).toContain('return &status, &percent, []string{"image"}');
    expect(imageBody).not.toContain('return &status, &percent, []string{}');
  });
});
