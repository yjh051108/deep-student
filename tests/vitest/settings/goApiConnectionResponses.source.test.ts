import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('Go API connection responses contract', () => {
  it('keeps protocol hints active in the Go settings service', () => {
    const source = readFileSync(resolve(process.cwd(), 'desktop-go/internal/settings/service.go'), 'utf-8');

    expect(source).toContain('func (s *Service) TestAPIConnection(apiKey string, apiBase string, apiProtocol *string, supportsOpenAIResponses *bool');
    expect(source).not.toContain('func (s *Service) TestAPIConnection(apiKey string, apiBase string, _ *string, _ *bool');
    expect(source).toContain('resolveTestAPIProtocol(baseURL, apiProtocol, supportsOpenAIResponses)');
    expect(source).toContain('appendAPIEndpoint(baseURL, protocol)');
    expect(source).toContain('apiTestRequestBody(modelID, protocol)');
  });

  it('keeps chat-completions and responses probes structurally separate', () => {
    const source = readFileSync(resolve(process.cwd(), 'desktop-go/internal/settings/service.go'), 'utf-8');

    expect(source).toContain('"openai_responses"');
    expect(source).toContain('endpoint = "/responses"');
    expect(source).toContain('"input":             "Hi"');
    expect(source).toContain('"max_output_tokens": 1');
    expect(source).toContain('"messages":   []map[string]string{{"role": "user", "content": "Hi"}}');
    expect(source).toContain('"max_tokens": 1');
  });
});
