import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

function readSource(path: string): string {
  return readFileSync(resolve(process.cwd(), path), 'utf-8');
}

function functionSource(source: string, signature: string): string {
  const start = source.indexOf(signature);
  expect(start).toBeGreaterThanOrEqual(0);

  const bodyStart = source.indexOf('{', start);
  expect(bodyStart).toBeGreaterThanOrEqual(0);

  let depth = 0;
  for (let index = bodyStart; index < source.length; index += 1) {
    const char = source[index];
    if (char === '{') depth += 1;
    if (char === '}') depth -= 1;
    if (depth === 0) {
      return source.slice(start, index + 1);
    }
  }

  throw new Error(`Could not find function body for ${signature}`);
}

describe('native app data directory Wails source contract', () => {
  const bridgeSource = readSource('src/runtime/wailsBridge.ts');
  const nativeSource = readSource('src/runtime/native.ts');

  it('maps get_app_data_dir to SystemService.AppDataDir in the Wails bridge', () => {
    expect(bridgeSource).toContain("bindings/systemservice");
    expect(bridgeSource).toMatch(
      /if \(command === 'get_app_data_dir'\) {\s*return await SystemService\.AppDataDir\(\) as T;\s*}/
    );
  });

  it('keeps image base64 reads behind the native facade and Wails bridge', () => {
    const configApiSource = readSource('src/utils/configApi.ts');
    const crepeImageUploadSource = readSource('src/components/crepe/features/imageUpload.ts');

    expect(nativeSource).toContain('export async function getImageAsBase64');
    expect(nativeSource).toContain("return invoke<string>('get_image_as_base64'");
    expect(nativeSource).toContain('getImageAsBase64,');
    expect(bridgeSource).toContain("if (command === 'get_image_as_base64')");
    expect(bridgeSource).toContain('NotesService.GetImageAsBase64(relativePath)');
    expect(configApiSource).toContain('getNativeImageAsBase64(relativePath)');
    expect(crepeImageUploadSource).toContain("import { getImageAsBase64, invoke } from '@/runtime/native'");
    expect(configApiSource).not.toContain("invoke<string>('get_image_as_base64'");
    expect(crepeImageUploadSource).not.toContain("invoke<string>('get_image_as_base64'");
    expect(configApiSource).not.toContain("@tauri-apps/api/core");
  });

  it('returns through Wails before any browser fallback path can handle the command', () => {
    const invokeSource = functionSource(nativeSource, 'export async function invoke');
    const injectedIndex = invokeSource.indexOf('if (isInjectedNativeRuntime())');
    const wailsIndex = invokeSource.indexOf('if (isWailsRuntime())');
    const fallbackBeforeTauriIndex = invokeSource.indexOf('if (shouldUseFallbackBeforeTauri(command))');
    const tauriIndex = invokeSource.indexOf('if (isTauriRuntime())');
    const finalFallbackIndex = invokeSource.lastIndexOf('return fallbackInvoke<T>(command, args);');

    expect(injectedIndex).toBeGreaterThanOrEqual(0);
    expect(wailsIndex).toBeGreaterThan(injectedIndex);
    expect(wailsIndex).toBeLessThan(fallbackBeforeTauriIndex);
    expect(wailsIndex).toBeLessThan(tauriIndex);
    expect(wailsIndex).toBeLessThan(finalFallbackIndex);
    expect(invokeSource.slice(0, wailsIndex)).not.toContain('fallbackInvoke');
    expect(invokeSource).toContain(
      `if (isWailsRuntime()) {
    return invokeWails<T>(command, args);
  }`
    );
  });
});
