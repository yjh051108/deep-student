import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('retired deep research report chat API contract', () => {
  it('does not expose retired research report wrappers or direct Tauri report invokes', () => {
    const chatApi = readFileSync(resolve(process.cwd(), 'src/utils/chatApi.ts'), 'utf-8');

    expect(chatApi).not.toMatch(/export\s+async\s+function\s+research(?:List|Get|Delete|Export)/);
    expect(chatApi).not.toContain("invoke('research_list_reports'");
    expect(chatApi).not.toContain("invoke('research_get_report'");
    expect(chatApi).not.toContain("invoke('research_delete_report'");
    expect(chatApi).not.toContain("invoke('research_export_all_reports_zip'");
  });
});
