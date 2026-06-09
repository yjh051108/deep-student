import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('API connection native facade contract', () => {
  it('routes settings API connection tests through native and Wails', () => {
    const settingsApi = readFileSync(resolve(process.cwd(), 'src/utils/settingsApi.ts'), 'utf-8');
    const vendorState = readFileSync(resolve(process.cwd(), 'src/features/settings/components/useSettingsVendorState.tsx'), 'utf-8');
    const wailsBridge = readFileSync(resolve(process.cwd(), 'src/runtime/wailsBridge.ts'), 'utf-8');

    expect(settingsApi).toContain("nativeInvoke<boolean>('test_api_connection'");
    expect(settingsApi).not.toContain("invoke<boolean>('test_api_connection'");
    expect(vendorState).toContain("nativeInvoke<boolean>('test_api_connection'");
    expect(vendorState).not.toContain("from '@tauri-apps/api/core'");
    expect(wailsBridge).toContain("command === 'test_api_connection'");
    expect(wailsBridge).toContain('SettingsService.TestAPIConnection');
    expect(wailsBridge).toContain("['apiKey', 'api_key']");
    expect(wailsBridge).toContain("['apiBase', 'api_base']");
    expect(wailsBridge).toContain("['apiProtocol', 'api_protocol']");
    expect(wailsBridge).toContain("['supportsOpenAIResponses', 'supports_openai_responses']");
    expect(wailsBridge).toContain("['vendorId', 'vendor_id']");
    expect(settingsApi).toContain('api_protocol: options.apiProtocol ?? null');
    expect(settingsApi).toContain('supports_openai_responses: options.supportsOpenAIResponses ?? null');
    expect(settingsApi).toContain('vendor_id: options.vendorId ?? null');
    expect(vendorState).toContain('api_protocol: api.apiProtocol');
    expect(vendorState).toContain('apiProtocol: api.apiProtocol');
    expect(vendorState).toContain('supports_openai_responses: api.supportsOpenAIResponses');
    expect(vendorState).toContain('supportsOpenAIResponses: api.supportsOpenAIResponses');
    expect(vendorState).toContain('vendor_id: vendorId');
    expect(vendorState).toContain('vendorId: vendorId');
  });
});
