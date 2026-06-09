const normalizeErrorLike = (input: unknown): { message: string; stack: string } => {
  if (input instanceof Error) {
    return {
      message: input.message || '',
      stack: input.stack || '',
    };
  }
  if (typeof input === 'string') {
    return { message: input, stack: '' };
  }
  if (input && typeof input === 'object') {
    const record = input as Record<string, unknown>;
    return {
      message: typeof record.message === 'string' ? record.message : '',
      stack: typeof record.stack === 'string' ? record.stack : '',
    };
  }
  return { message: '', stack: '' };
};

const isKnownTauriHttpNoise = (message: string, stack?: string): boolean => {
  const lcMessage = (message || '').toLowerCase();
  const lcStack = (stack || '').toLowerCase();
  if (!lcMessage && !lcStack) return false;

  const combined = `${lcMessage}\n${lcStack}`;
  const hasTauriHttpHint =
    combined.includes('http.fetch_') ||
    combined.includes('streamchannel') ||
    combined.includes('ipc custom protocol') ||
    combined.includes('@tauri-apps/plugin-http') ||
    combined.includes('tauri-plugin-http') ||
    combined.includes('tauri');

  const fetchCancelBodyNoise =
    (combined.includes('http.fetch_cancel_body') || combined.includes('fetch_cancel_body')) &&
    hasTauriHttpHint;
  const streamChannelBodyNoise =
    (combined.includes('fetch_read_body') || combined.includes('fetch_send')) &&
    combined.includes('streamchannel') &&
    hasTauriHttpHint;
  const staleResourceNoise =
    combined.includes('resource id') &&
    combined.includes('invalid') &&
    (combined.includes('http.fetch_') ||
      combined.includes('streamchannel') ||
      combined.includes('ipc custom protocol'));

  return fetchCancelBodyNoise || streamChannelBodyNoise || staleResourceNoise;
};

// ★ 2026-02-04: 最早的全局错误过滤器
// 必须在任何其他代码之前运行，以便在 tauri-plugin-mcp-bridge 之前捕获错误
// 这是一个 IIFE，在模块加载时立即执行
(() => {
  if (typeof window === 'undefined') return;

  const isGoWailsSmoke = window.location?.search?.includes('go-wails-smoke=true') === true;
  const smokeErrors: Array<Record<string, unknown>> = [];
  if (isGoWailsSmoke) {
    (window as any).__DEEP_STUDENT_WAILS_SMOKE_EARLY_ERRORS__ = smokeErrors;
  }
  
  // 过滤 Tauri HTTP 插件的已知无害错误
  // 包括：fetch_cancel_body、fetch_read_body+streamChannel、resource id invalid
  // 这些错误在连接重建或 HMR 热重载时是正常现象，不影响功能
  const earlyFilter = (event: PromiseRejectionEvent) => {
    const reason = event.reason;
    const { message, stack } = normalizeErrorLike(reason);
    if (isKnownTauriHttpNoise(message, stack)) {
      if (isGoWailsSmoke) {
        smokeErrors.push({ type: 'suppressed-tauri-http-noise', message, stack });
      }
      event.preventDefault();
      event.stopImmediatePropagation();
      return;
    }
  };

  // 使用 capture: true 确保在其他处理器之前运行
  window.addEventListener('unhandledrejection', earlyFilter, true);

  if (isGoWailsSmoke) {
    window.addEventListener('error', (event) => {
      smokeErrors.push({
        type: 'error',
        message: event.message,
        filename: event.filename,
        lineno: event.lineno,
        colno: event.colno,
      });
    }, true);
    window.addEventListener('unhandledrejection', (event) => {
      const { message, stack } = normalizeErrorLike(event.reason);
      if (!isKnownTauriHttpNoise(message, stack)) {
        smokeErrors.push({ type: 'unhandledrejection', message, stack });
      }
    }, true);
  }

  // 拦截 console.error 中的 Tauri HTTP 插件 stale resource 错误
  // 这些错误通过 Tauri IPC 同步触发 console.error，不经过 unhandledrejection
  const _origConsoleError = console.error;
  console.error = (...args: any[]) => {
    try {
      const first = normalizeErrorLike(args[0]);
      const second = normalizeErrorLike(args[1]);
      const combinedMessage = [first.message, second.message].filter(Boolean).join(' ');
      const combinedStack = [first.stack, second.stack].filter(Boolean).join('\n');
      if (isKnownTauriHttpNoise(combinedMessage, combinedStack)) {
        if (isGoWailsSmoke) {
          smokeErrors.push({
            type: 'suppressed-tauri-http-console',
            message: combinedMessage,
            stack: combinedStack,
          });
        }
        return; // 静默过滤已知无害错误
      }
    } catch { /* pass through on filter error */ }
    _origConsoleError.apply(console, args);
  };
})();

import './polyfills/promiseWithResolvers';
import React from "react";
import ReactDOM from "react-dom/client";
// 🚀 性能优化：KaTeX CSS 改为按需加载，见 src/utils/lazyStyles.ts
import App from "./App";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { OverlayCoordinatorProvider } from './components/shared/OverlayCoordinator';
// 日志与错误上报初始化（跨平台）：结合 Tauri 日志插件与自定义上报
import { disposeGlobalCacheManager } from './utils/cacheConsistencyManager';
import { DialogControlProvider } from './contexts/DialogControlContext';
import i18n from './i18n';
import { McpService, bootstrapMcpFromSettings } from './mcp/mcpService';
import { getSetting, invoke as nativeInvoke, native } from './runtime/native';
import { listen, type NativeUnlistenFn } from './runtime/nativeEvents';
// ★ DSTU Logger 初始化（依赖注入模式）
import { setDstuLogger, createLoggerFromDebugPlugin } from './dstu';
import { dstuDebugLog } from './debug-panel/plugins/DstuDebugPlugin';
import { debugMasterSwitch, debugLog } from './debug-panel/debugMasterSwitch';
// ★ 平台检测初始化（为 Android WebView 兼容性添加 CSS 类）
import { initPlatformClasses } from './utils/platform';
import { OverlayScrollbars, ClickScrollPlugin } from 'overlayscrollbars';

// 尽早初始化平台检测类，确保 CSS 规则在渲染前生效
initPlatformClasses();

// 注册 OverlayScrollbars ClickScrollPlugin — 点击轨道时平滑滚动到目标位置
OverlayScrollbars.plugin(ClickScrollPlugin);

const maybeInstallReactGrab = () => {
  try {
    const env = (import.meta as any).env ?? {};
    const isDev = env.MODE !== 'production';
    const enabled = env.VITE_ENABLE_REACT_GRAB === 'true';
    if (!isDev || !enabled) {
      return;
    }
    import('react-grab').catch((error) => {
      console.warn('[main] React Grab 加载失败', error);
    });
  } catch (error) {
    console.warn('[main] React Grab 初始化失败', error);
  }
};

maybeInstallReactGrab();

const isGoWailsSmokePage = () => {
  if (typeof window === 'undefined') return false;
  return new URLSearchParams(window.location.search).get('go-wails-smoke') === 'true';
};

type McpStdioWailsSmokeOptions = {
  command?: string;
  args?: string[];
  cwd?: string;
  env?: Record<string, string>;
  framing?: 'jsonl' | 'content_length';
  timeoutMs?: number;
};

type TextbookImportProgressSmokePayload = {
  file_name?: string;
  stage?: string;
  progress?: number;
  source?: string;
  import_id?: string;
  index?: number;
  total?: number;
  textbook_id?: string;
  resource_id?: string;
  error?: string;
};

type GoWailsSmokeOptions = {
  mcpStdio?: McpStdioWailsSmokeOptions;
  skills?: boolean;
  templates?: boolean;
  vfs?: boolean;
};

const runMcpStdioWailsSmoke = async (options: McpStdioWailsSmokeOptions = {}) => {
  const command = typeof options.command === 'string' ? options.command : '';
  if (!command.trim()) {
    throw new Error('MCP stdio Wails smoke requires a command');
  }

  const timeoutMs = Math.max(1_000, Math.min(options.timeoutMs ?? 8_000, 30_000));
  const eventNames: string[] = [];
  const rpcMessages: Array<Record<string, unknown>> = [];
  const unlistenFns: NativeUnlistenFn[] = [];
  const waiters = new Map<number, {
    resolve: (message: Record<string, unknown>) => void;
    reject: (error: Error) => void;
    timer: number;
  }>();

  let sessionId = '';
  let messageEventReceived = false;
  let closedEventReceived = false;
  let errorEventReceived = false;
  let closeCommandSucceeded = false;
  let closeCommandAttempted = false;
  let invalidMessageReceived = false;
  let invalidMessageError = '';
  let activeCloseSessionStarted = false;
  let activeCloseCommandSucceeded = false;
  let activeCloseSendRejected = false;

  const waitForRpcResponse = (id: number, label: string): Promise<Record<string, unknown>> => {
    const existing = rpcMessages.find(message => message.id === id);
    if (existing) {
      return Promise.resolve(existing);
    }
    return new Promise((resolve, reject) => {
      const timer = window.setTimeout(() => {
        waiters.delete(id);
        reject(new Error(`Timed out waiting for MCP stdio ${label}`));
      }, timeoutMs);
      waiters.set(id, { resolve, reject, timer });
    });
  };

  const waitForClosed = (): Promise<void> => {
    if (closedEventReceived) return Promise.resolve();
    return new Promise((resolve, reject) => {
      let settled = false;
      const poll = window.setInterval(() => {
        if (settled) return;
        if (!closedEventReceived) return;
        settled = true;
        window.clearInterval(poll);
        window.clearTimeout(timer);
        resolve();
      }, 25);
      const timer = window.setTimeout(() => {
        if (settled) return;
        settled = true;
        window.clearInterval(poll);
        reject(new Error('Timed out waiting for MCP stdio closed event'));
      }, timeoutMs);
    });
  };

  const sendRpc = async (payload: Record<string, unknown>) => {
    await nativeInvoke('mcp_stdio_send', {
      sessionId,
      payload: JSON.stringify(payload),
    });
  };

  try {
    sessionId = await nativeInvoke<string>('mcp_stdio_start', {
      command,
      args: Array.isArray(options.args) ? options.args.filter((item): item is string => typeof item === 'string') : [],
      cwd: typeof options.cwd === 'string' ? options.cwd : null,
      env: options.env && typeof options.env === 'object' ? options.env : {},
      framing: options.framing ?? 'content_length',
    });

    const messageEventName = `mcp-stdio-${sessionId}-message`;
    const closedEventName = `mcp-stdio-${sessionId}-closed`;
    const errorEventName = `mcp-stdio-${sessionId}-error`;

    unlistenFns.push(await listen(messageEventName, event => {
      eventNames.push(messageEventName);
      messageEventReceived = true;
      const payload = event.payload as any;
      if (!payload || typeof payload.message !== 'string') return;
      let parsed: Record<string, unknown>;
      try {
        parsed = JSON.parse(payload.message) as Record<string, unknown>;
      } catch (error) {
        invalidMessageReceived = true;
        invalidMessageError = error instanceof Error ? error.message : String(error);
        const waitingIds = Array.from(waiters.keys());
        for (const id of waitingIds) {
          const waiter = waiters.get(id);
          if (!waiter) continue;
          window.clearTimeout(waiter.timer);
          waiters.delete(id);
          waiter.reject(new Error(`Invalid MCP stdio JSON message: ${invalidMessageError}`));
        }
        return;
      }
      rpcMessages.push(parsed);
      const id = typeof parsed.id === 'number' ? parsed.id : Number(parsed.id);
      const waiter = Number.isFinite(id) ? waiters.get(id) : undefined;
      if (waiter) {
        window.clearTimeout(waiter.timer);
        waiters.delete(id);
        waiter.resolve(parsed);
      }
    }));
    unlistenFns.push(await listen(closedEventName, () => {
      eventNames.push(closedEventName);
      closedEventReceived = true;
    }));
    unlistenFns.push(await listen(errorEventName, event => {
      eventNames.push(errorEventName);
      errorEventReceived = true;
      const payload = event.payload;
      console.warn('[MCP][stdio smoke] error event', payload);
    }));

    await sendRpc({
      jsonrpc: '2.0',
      id: 1,
      method: 'initialize',
      params: {
        protocolVersion: '2024-11-05',
        capabilities: {},
        clientInfo: { name: 'deep-student-wails-smoke', version: '1.0.0' },
      },
    });
    const initialize = await waitForRpcResponse(1, 'initialize response');

    await sendRpc({ jsonrpc: '2.0', method: 'notifications/initialized' });

    await sendRpc({ jsonrpc: '2.0', id: 2, method: 'tools/list' });
    const tools = await waitForRpcResponse(2, 'tools/list response');

    await sendRpc({
      jsonrpc: '2.0',
      id: 3,
      method: 'tools/call',
      params: {
        name: 'smoke_echo',
        arguments: { message: 'wails' },
      },
    });
    const call = await waitForRpcResponse(3, 'tools/call response');

    await sendRpc({ jsonrpc: '2.0', id: 4, method: 'shutdown' });
    await waitForRpcResponse(4, 'shutdown response');
    await waitForClosed();

    closeCommandAttempted = true;
    await nativeInvoke('mcp_stdio_close', { sessionId });
    closeCommandSucceeded = true;

    const activeCloseSessionId = await nativeInvoke<string>('mcp_stdio_start', {
      command,
      args: Array.isArray(options.args) ? options.args.filter((item): item is string => typeof item === 'string') : [],
      cwd: typeof options.cwd === 'string' ? options.cwd : null,
      env: options.env && typeof options.env === 'object' ? options.env : {},
      framing: options.framing ?? 'content_length',
    });
    activeCloseSessionStarted = Boolean(activeCloseSessionId);
    await nativeInvoke('mcp_stdio_close', { sessionId: activeCloseSessionId });
    activeCloseCommandSucceeded = true;
    try {
      await nativeInvoke('mcp_stdio_send', {
        sessionId: activeCloseSessionId,
        payload: JSON.stringify({ jsonrpc: '2.0', id: 99, method: 'tools/list' }),
      });
    } catch {
      activeCloseSendRejected = true;
    }

    const initializeResult = initialize.result as any;
    const toolsResult = tools.result as any;
    const callResult = call.result as any;
    const toolName = toolsResult?.tools?.[0]?.name ?? '';
    const toolCallText = callResult?.content?.[0]?.text ?? '';

    return {
      browserFallbackRejected: native.runtime.isWails() && !native.runtime.isInjected(),
      activeCloseCommandSucceeded,
      activeCloseSendRejected,
      activeCloseSessionStarted,
      closeCommandAttempted,
      closeCommandSucceeded,
      closedEventReceived,
      commandStarted: Boolean(sessionId),
      errorEventReceived,
      eventNames,
      framing: options.framing ?? 'content_length',
      initializeServerName: initializeResult?.serverInfo?.name ?? '',
      invalidMessageError,
      invalidMessageReceived,
      messageEventReceived,
      ok: Boolean(
        sessionId &&
        messageEventReceived &&
        closedEventReceived &&
        !errorEventReceived &&
        !invalidMessageReceived &&
        initializeResult?.serverInfo?.name === 'dstu-mcp-smoke' &&
        toolName === 'smoke_echo' &&
        toolCallText === 'echo: wails' &&
        closeCommandSucceeded &&
        activeCloseSessionStarted &&
        activeCloseCommandSucceeded &&
        activeCloseSendRejected
      ),
      routeClose: 'mcp_stdio_close -> McpService.CloseStdioSession',
      routeSend: 'mcp_stdio_send -> McpService.SendStdioMessage',
      routeStart: 'mcp_stdio_start -> McpService.StartStdioSession',
      sessionId,
      tauriFallbackRejected: !native.runtime.isTauri(),
      toolCallText,
      toolName,
    };
  } finally {
    for (const waiter of waiters.values()) {
      window.clearTimeout(waiter.timer);
      waiter.reject(new Error('MCP stdio smoke finished before response arrived'));
    }
    waiters.clear();
    while (unlistenFns.length) {
      const unlisten = unlistenFns.pop();
      try {
        unlisten?.();
      } catch {
        // ignore cleanup failures in smoke path
      }
    }
    if (sessionId && !closeCommandSucceeded) {
      try {
        closeCommandAttempted = true;
        await nativeInvoke('mcp_stdio_close', { sessionId });
      } catch {
        // best-effort cleanup; assertion happens through returned fields/failures
      }
    }
  }
};

const normalizePathForSmoke = (path: string): string => path.replace(/\\/g, '/');
const isPathInsideForSmoke = (childPath: string, parentPath: string): boolean => {
  const normalizedChild = normalizePathForSmoke(childPath).replace(/\/+$/, '');
  const normalizedParent = normalizePathForSmoke(parentPath).replace(/\/+$/, '');
  return normalizedChild === normalizedParent || normalizedChild.startsWith(`${normalizedParent}/`);
};

const decodeBase64ForSmoke = (value: string | null | undefined): string => {
  if (!value) return '';
  try {
    const binary = window.atob(value);
    const bytes = Uint8Array.from(binary, char => char.charCodeAt(0));
    return new TextDecoder().decode(bytes);
  } catch {
    return '';
  }
};

const buildMinimalTextPdfForSmoke = (sentinel: string): string => {
  const stream = `BT\n/F1 12 Tf\n72 720 Td\n(${sentinel}) Tj\nET`;
  const objects = [
    '1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj',
    '2 0 obj\n<< /Type /Pages /Count 1 /Kids [3 0 R] >>\nendobj',
    '3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>\nendobj',
    '4 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj',
    `5 0 obj\n<< /Length ${stream.length} >>\nstream\n${stream}\nendstream\nendobj`,
  ];
  return `%PDF-1.4\n${objects.join('\n')}\n%%EOF`;
};

const isTerminalPdfSmokeStage = (stage: string | undefined): boolean => (
  stage === 'completed' || stage === 'completed_with_issues'
);

const runSkillWailsSmoke = async () => {
  const {
    createSkill,
    deleteSkill,
    listSkillDirectories,
    readSkillFile,
    updateSkill,
  } = await import('./features/chat/skills/api');
  const appDataDir = await native.system.getAppDataDir();
  const basePath = `${appDataDir.replace(/[\\/]+$/, '')}/skills`;
  const skillId = `wails-smoke-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
  const createdContent = [
    '# Wails Smoke Skill',
    '',
    `Skill ID: ${skillId}`,
    '',
    'This skill is created by the live Go/Wails smoke and must be deleted by the same smoke run.',
  ].join('\n');
  const updatedContent = `${createdContent}\n\nUpdated by SkillService smoke.`;

  let skillPath = '';
  let skillDir = '';
  let createdContentIncludesName = false;
  let readContentIncludesName = false;
  let updatedContentIncludesName = false;
  let listBeforeIncludesSkill = false;
  let listAfterDeleteIncludesSkill = false;
  let deleteSucceeded = false;
  let readAfterDeleteRejected = false;

  try {
    const created = await createSkill({
      basePath,
      skillId,
      content: createdContent,
    });
    skillPath = created.path;
    skillDir = skillPath.replace(/[\\/]+SKILL\.md$/i, '');
    createdContentIncludesName = created.content.includes(skillId);

    const read = await readSkillFile(skillPath);
    readContentIncludesName =
      read.content.includes(skillId) &&
      normalizePathForSmoke(read.path) === normalizePathForSmoke(skillPath);

    const updated = await updateSkill({
      path: skillPath,
      content: updatedContent,
    });
    updatedContentIncludesName =
      updated.content.includes('Updated by SkillService smoke') &&
      normalizePathForSmoke(updated.path) === normalizePathForSmoke(skillPath);

    const listBeforeDelete = await listSkillDirectories(basePath);
    listBeforeIncludesSkill = listBeforeDelete.some(
      entry => entry.name === skillId && normalizePathForSmoke(entry.path) === normalizePathForSmoke(skillDir),
    );

    await deleteSkill(skillDir);
    deleteSucceeded = true;

    const listAfterDelete = await listSkillDirectories(basePath);
    listAfterDeleteIncludesSkill = listAfterDelete.some(entry => entry.name === skillId);

    try {
      await readSkillFile(skillPath);
    } catch {
      readAfterDeleteRejected = true;
    }

    const normalizedBasePath = normalizePathForSmoke(basePath);
    const normalizedSkillPath = normalizePathForSmoke(skillPath);
    return {
      basePath,
      createdContentIncludesName,
      deleteSucceeded,
      listAfterDeleteIncludesSkill,
      listBeforeIncludesSkill,
      ok: Boolean(
        createdContentIncludesName &&
        readContentIncludesName &&
        updatedContentIncludesName &&
        listBeforeIncludesSkill &&
        !listAfterDeleteIncludesSkill &&
        deleteSucceeded &&
        readAfterDeleteRejected &&
        isPathInsideForSmoke(normalizedSkillPath, normalizedBasePath)
      ),
      readAfterDeleteRejected,
      readContentIncludesName,
      routeCreate: 'skill_create -> SkillService.Create',
      routeDelete: 'skill_delete -> SkillService.Delete',
      routeList: 'skill_list_directories -> SkillService.ListDirectories',
      routeRead: 'skill_read_file -> SkillService.ReadFile',
      routeUpdate: 'skill_update -> SkillService.Update',
      skillDir,
      skillId,
      skillPath,
      updatedContentIncludesName,
    };
  } finally {
    if (skillDir && !deleteSucceeded) {
      try {
        await deleteSkill(skillDir);
      } catch {
        // best-effort cleanup; returned proof records the failed delete path
      }
    }
  }
};

const runVfsWailsSmoke = async () => {
  const [{ textbookDstuAdapter }, { vfsFileApi }, { vfsPdfProcessingApi }, { getAllIndexStatus }] = await Promise.all([
    import('./dstu/adapters/textbookDstuAdapter'),
    import('./api/vfsFileApi'),
    import('./api/vfsPdfProcessingApi'),
    import('./api/vfsUnifiedIndexApi'),
  ]);

  const appDataDir = await native.system.getAppDataDir();
  const inputDir = `${appDataDir.replace(/[\\/]+$/, '')}/smoke-fixtures`;
  const unique = `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
  const fileName = `wails-smoke-textbook-${unique}.pdf`;
  const sourcePath = `${inputDir}/${fileName}`;
  const sentinel = `Live Wails textbook hybrid VFS smoke ${unique}`;
  const pdfContent = buildMinimalTextPdfForSmoke(sentinel);
  const progressEvents: TextbookImportProgressSmokePayload[] = [];
  let unlisten: NativeUnlistenFn | undefined;

  try {
    unlisten = await listen<TextbookImportProgressSmokePayload>('textbook-import-progress', event => {
      const payload = event.payload;
      if (!payload || typeof payload !== 'object') return;
      const payloadSource = typeof payload.source === 'string' ? normalizePathForSmoke(payload.source) : '';
      if (payload.file_name === fileName || payloadSource === normalizePathForSmoke(sourcePath)) {
        progressEvents.push(payload);
      }
    });

    await native.files.saveText(sourcePath, pdfContent);
    const imported = await textbookDstuAdapter.addTextbooks([sourcePath], null);
    if (!imported.ok) {
      throw new Error(`textbookDstuAdapter.addTextbooks failed: ${imported.error.message}`);
    }
    if (imported.value.length !== 1) {
      throw new Error(`Expected one imported textbook, got ${imported.value.length}`);
    }

    const textbook = imported.value[0];
    const fileId = textbook.sourceId || textbook.id;
    const resourceId = textbook.resourceId || String(textbook.metadata?.resourceId || '');
    const resourceHash = textbook.resourceHash || String(textbook.metadata?.resourceHash || textbook.metadata?.sha256 || '');
    if (!fileId || !resourceId || !resourceHash) {
      throw new Error(`Textbook import did not expose hybrid VFS identity: ${JSON.stringify(textbook)}`);
    }

    const listResult = await textbookDstuAdapter.listTextbooks();
    if (!listResult.ok) {
      throw new Error(`textbookDstuAdapter.listTextbooks failed: ${listResult.error.message}`);
    }
    const listNode = listResult.value.find(node => node.id === fileId || node.resourceId === resourceId);
    const getResult = await textbookDstuAdapter.getTextbook(fileId);
    if (!getResult.ok) {
      throw new Error(`textbookDstuAdapter.getTextbook failed: ${getResult.error.message}`);
    }

    const resource = await nativeInvoke<any>('vfs_get_resource', { resourceId });
    const resolvedPath = await nativeInvoke<string | null>('vfs_get_resource_path', { sourceId: resourceId });
    const file = await vfsFileApi.get(fileId);
    const fileContent = await vfsFileApi.getContent(fileId);
    const decodedContent = decodeBase64ForSmoke(fileContent.content);

    const statusDeadline = Date.now() + 5_000;
    let pdfStatus = await vfsPdfProcessingApi.getStatus(fileId);
    while (!isTerminalPdfSmokeStage(pdfStatus.stage) && Date.now() < statusDeadline) {
      await new Promise(resolve => window.setTimeout(resolve, 100));
      pdfStatus = await vfsPdfProcessingApi.getStatus(fileId);
    }
    const batchStatus = await vfsPdfProcessingApi.getBatchStatus([fileId]);
    const indexStatus = await getAllIndexStatus({ resourceType: 'textbook', limit: 50 });
    const indexEntry = indexStatus.resources.find(entry => entry.resourceId === resourceId || entry.sourceId === fileId);

    const progressStages = progressEvents.map(event => event.stage || '').filter(Boolean);
    const doneEvent = progressEvents.find(event => event.stage === 'done');
    const doneEventMatchesIdentity = Boolean(
      doneEvent &&
      doneEvent.textbook_id === fileId &&
      doneEvent.resource_id === resourceId,
    );

    const normalizedSourcePath = normalizePathForSmoke(sourcePath);
    const resourceOriginalPath = typeof resource?.originalPath === 'string'
      ? resource.originalPath
      : String(resource?.metadata?.sourcePath || resource?.metadata?.importPath || '');
    const resourceExternalPath = typeof resource?.externalPath === 'string' ? resource.externalPath : '';
    const resourceResolvedPath = typeof resolvedPath === 'string' ? resolvedPath : '';
    const fileLookupMatchesResource = Boolean(file?.resourceId === resourceId);
    const fileLookupMatchesHash = Boolean(file?.sha256 === resourceHash || file?.blobHash === resourceHash);
    const batchPdfStatus = batchStatus.statuses[fileId];
    const rawContentContainsSentinel = decodedContent.includes(sentinel);
    const fileExtractedText = typeof file?.extractedText === 'string' ? file.extractedText : '';
    const resourceMetadata = resource?.metadata && typeof resource.metadata === 'object' ? resource.metadata : {};
    const resourceExtractedText = typeof resourceMetadata.extractedText === 'string'
      ? resourceMetadata.extractedText
      : typeof resourceMetadata.extracted_text === 'string'
        ? resourceMetadata.extracted_text
        : '';
    const extractedTextContainsSentinel = fileExtractedText.includes(sentinel) || resourceExtractedText.includes(sentinel);
    const batchPdfReadyText = batchPdfStatus?.readyModes.includes('text') === true;
    const batchPdfPageCount = batchPdfStatus?.totalPages ?? 0;
    const batchPdfTerminal = isTerminalPdfSmokeStage(batchPdfStatus?.stage);
    const indexEntryMatches = Boolean(indexEntry && indexEntry.name === fileName);

    return {
      batchPdfPageCount,
      batchPdfReadyText,
      batchPdfStage: batchPdfStatus?.stage ?? '',
      doneEventMatchesIdentity,
      extractedTextContainsSentinel,
      fileContentFound: fileContent.found === true,
      fileId,
      fileLookupMatchesHash,
      fileLookupMatchesResource,
      indexEntryMatches,
      ok: Boolean(
        native.runtime.isWails() &&
        !native.runtime.isInjected() &&
        textbook.type === 'textbook' &&
        textbook.previewType === 'pdf' &&
        listNode &&
        getResult.value?.id === fileId &&
        resource?.id === resourceId &&
        resource?.hash === resourceHash &&
        resource?.type === 'textbook' &&
        resource?.storageMode === 'external' &&
        isPathInsideForSmoke(normalizePathForSmoke(resourceOriginalPath), normalizePathForSmoke(inputDir)) &&
        resourceExternalPath &&
        resourceResolvedPath &&
        isPathInsideForSmoke(normalizePathForSmoke(resourceResolvedPath), normalizePathForSmoke(appDataDir)) &&
        fileContent.found === true &&
        rawContentContainsSentinel &&
        extractedTextContainsSentinel &&
        fileLookupMatchesResource &&
        fileLookupMatchesHash &&
        pdfStatus.mediaType === 'pdf' &&
        isTerminalPdfSmokeStage(pdfStatus.stage) &&
        pdfStatus.readyModes.includes('text') &&
        pdfStatus.totalPages === 1 &&
        batchPdfStatus?.mediaType === 'pdf' &&
        batchPdfTerminal &&
        batchPdfReadyText &&
        batchPdfPageCount === 1 &&
        ['hashing', 'copying', 'saving', 'done'].every(stage => progressStages.includes(stage)) &&
        doneEventMatchesIdentity &&
        indexEntryMatches &&
        normalizedSourcePath.includes('/smoke-fixtures/')
      ),
      pdfMediaType: pdfStatus.mediaType ?? '',
      pdfPageCount: pdfStatus.totalPages ?? 0,
      pdfReadyText: pdfStatus.readyModes.includes('text'),
      pdfStage: pdfStatus.stage,
      previewType: textbook.previewType,
      progressStages,
      rawContentContainsSentinel,
      resourceExternalPath,
      resourceHash,
      resourceId,
      resourceOriginalPath,
      resourceResolvedPath,
      resourceStorageMode: resource?.storageMode ?? '',
      resourceType: resource?.type ?? '',
      routeBatchPdfStatus: 'vfs_get_batch_pdf_processing_status -> VfsService.GetBatchPdfProcessingStatus',
      routeEvent: 'textbook-import-progress -> Wails EventBus',
      routeFile: 'vfs_get_file -> VfsService.GetFile',
      routeFileContent: 'vfs_get_file_content -> VfsService.GetFileContent',
      routePdfStatus: 'vfs_get_pdf_processing_status -> VfsService.GetPdfProcessingStatus',
      routeResource: 'vfs_get_resource -> VfsService.GetResource',
      routeTextbooksAdd: 'textbooks_add -> DstuService.AddTextbooks',
      sourcePath,
      sentinel,
      textbookName: textbook.name,
      textbookType: textbook.type,
    };
  } finally {
    try {
      unlisten?.();
    } catch {
      // ignore cleanup failures in smoke path
    }
  }
};

const runTemplateWailsSmoke = async () => {
  const unique = `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
  const createdName = `Wails Smoke Template ${unique}`;
  const importedName = `Wails Smoke Imported Template ${unique}`;
  const updatedDescription = `updated by live Wails smoke ${unique}`;

  const importBuiltinResult = await nativeInvoke<string>('import_builtin_templates');
  const builtinTemplates = await nativeInvoke<any[]>('get_all_custom_templates');
  const builtinCount = builtinTemplates.filter(template => template?.is_built_in === true).length;
  const legacyMigrated = builtinTemplates.find(template => template?.id === 'legacy-wails-smoke-template');
  const legacyMigratedFields = Array.isArray(legacyMigrated?.fields) ? legacyMigrated.fields : [];

  const createRequest = {
    name: createdName,
    description: 'created by live Wails smoke',
    author: 'Deep Student Smoke',
    fields: ['Front', 'Back'],
    field_extraction_rules: {
      Front: {
        field_type: 'Text',
        is_required: true,
        default_value: '',
        description: 'front field',
      },
      Back: {
        field_type: 'Text',
        is_required: true,
        default_value: '',
        description: 'back field',
      },
    },
    front_template: '<div>{{Front}}</div>',
    back_template: '<div>{{Back}}</div>',
    css_style: '.card { color: #123456; }',
    generation_prompt: 'Generate one Front and Back pair.',
    is_active: true,
    note_type: 'Basic',
    preview_back: '{{Back}}',
    preview_front: '{{Front}}',
  };

  const templateId = await nativeInvoke<string>('create_custom_template', { request: createRequest });
  const afterCreate = await nativeInvoke<any[]>('get_all_custom_templates');
  const created = afterCreate.find(template => template?.id === templateId);

  await nativeInvoke('set_default_template', { templateId });
  const defaultTemplateId = await nativeInvoke<string | null>('get_default_template_id');

  await nativeInvoke('update_custom_template', {
    templateId,
    request: {
      description: updatedDescription,
      expected_version: created?.version,
      preview_back: `{{Back}} ${unique}`,
    },
  });
  const afterUpdate = await nativeInvoke<any[]>('get_all_custom_templates');
  const updated = afterUpdate.find(template => template?.id === templateId);

  const exported = await nativeInvoke<{ template_data: string }>('export_template', { templateId });
  const exportedTemplate = JSON.parse(exported.template_data);

  const importedId = `wails-smoke-import-${unique}`;
  const bulkPayload = JSON.stringify([{
    id: importedId,
    name: importedName,
    description: 'legacy JSON import shape for live Wails smoke',
    fields_json: JSON.stringify(['Front', 'Back']),
    field_extraction_rules_json: JSON.stringify({
      Front: {
        field_type: 'Text',
        is_required: true,
        default_value: '',
        description: 'front field',
      },
      Back: {
        field_type: 'Text',
        is_required: true,
        default_value: '',
        description: 'back field',
      },
    }),
    front_template: '{{Front}}',
    back_template: '{{Back}}',
    css_style: '.card { color: green; }',
    generation_prompt: 'Generate imported card fields.',
    is_active: true,
    is_built_in: false,
    note_type: 'Basic',
    preview_back: '{{Back}}',
    preview_front: '{{Front}}',
    version: '1.0.0',
  }]);
  const bulkImportResult = await nativeInvoke<string>('import_custom_templates_bulk', {
    overwrite_existing: true,
    overwriteExisting: true,
    strict_builtin: false,
    strictBuiltin: false,
    template_data: bulkPayload,
    templateData: bulkPayload,
  });
  const afterBulkImport = await nativeInvoke<any[]>('get_all_custom_templates');
  const imported = afterBulkImport.find(template => template?.id === importedId);
  const importedFields = Array.isArray(imported?.fields) ? imported.fields : [];

  return {
    bulkImportResult,
    builtinCount,
    createdDescription: created?.description ?? '',
    createdId: templateId,
    defaultTemplateId,
    exportedDescription: exportedTemplate?.description ?? '',
    exportedId: exportedTemplate?.id ?? '',
    importBuiltinResult,
    importedFields,
    importedHasLegacyFieldsJson: Object.prototype.hasOwnProperty.call(imported ?? {}, 'fields_json'),
    importedId,
    importedIsBuiltIn: imported?.is_built_in,
    importedName: imported?.name ?? '',
    legacyMigratedFields,
    legacyMigratedName: legacyMigrated?.name ?? '',
    ok: Boolean(
      native.runtime.isWails() &&
      !native.runtime.isInjected() &&
      typeof importBuiltinResult === 'string' &&
      importBuiltinResult.includes('导入完成') &&
      builtinCount >= 6 &&
      legacyMigrated?.name === 'Legacy Wails Smoke Template' &&
      legacyMigratedFields.includes('Front') &&
      legacyMigratedFields.includes('Back') &&
      created?.id === templateId &&
      created?.name === createdName &&
      defaultTemplateId === templateId &&
      updated?.description === updatedDescription &&
      exportedTemplate?.id === templateId &&
      exportedTemplate?.description === updatedDescription &&
      typeof bulkImportResult === 'string' &&
      bulkImportResult.includes('导入完成') &&
      imported?.id === importedId &&
      imported?.name === importedName &&
      imported?.is_built_in === false &&
      importedFields.includes('Front') &&
      importedFields.includes('Back') &&
      !Object.prototype.hasOwnProperty.call(imported ?? {}, 'fields_json')
    ),
    routeCreate: 'create_custom_template -> TemplateService.CreateCustomTemplate',
    routeDefaultGet: 'get_default_template_id -> TemplateService.GetDefaultTemplateID',
    routeDefaultSet: 'set_default_template -> TemplateService.SetDefaultTemplate',
    routeExport: 'export_template -> TemplateService.ExportTemplate',
    routeImportBuiltin: 'import_builtin_templates -> TemplateService.ImportBuiltinTemplates',
    routeImportBulk: 'import_custom_templates_bulk -> TemplateService.ImportCustomTemplatesBulk',
    routeList: 'get_all_custom_templates -> TemplateService.GetAllCustomTemplates',
    routeUpdate: 'update_custom_template -> TemplateService.UpdateCustomTemplate',
    updatedDescription: updated?.description ?? '',
    updatedVersion: updated?.version ?? '',
  };
};

const installGoWailsSmokeHook = () => {
  if (typeof window === 'undefined') return;
  if (!isGoWailsSmokePage()) return;

  const tryInstall = (): boolean => {
    if (typeof (window as any).__DEEP_STUDENT_GO_WAILS_SMOKE__ === 'function') {
      return true;
    }
    if (!native.runtime.isWails() || native.runtime.isInjected()) return false;
    const wails = (window as any)._wails as Record<string, unknown> | undefined;
    const wailsFlags = wails?.flags as Record<string, unknown> | undefined;
    if (wailsFlags?.deepStudentWailsSmoke !== true) return false;

    (window as any).__DEEP_STUDENT_GO_WAILS_SMOKE__ = async (options?: GoWailsSmokeOptions) => {
      const key = `go.wails.smoke.${Date.now()}`;
      const value = `ok-${Math.random().toString(36).slice(2)}`;
      await nativeInvoke('save_setting', { key, value });
      const stored = await nativeInvoke<string | null>('get_setting', { key });
      const root = document.querySelector('#root');
      const smokeSentinel = document.querySelector('[data-deep-student-smoke-rendered="true"]');
      const errorBoundary = document.querySelector('[data-deep-student-error-boundary="top-level"]');
      const earlyErrors = (window as any).__DEEP_STUDENT_WAILS_SMOKE_EARLY_ERRORS__;
      const isWails = native.runtime.isWails();
      const smokeFlag = ((window as any)._wails?.flags as Record<string, unknown> | undefined)?.deepStudentWailsSmoke === true;
      const mcpStdio = options?.mcpStdio ? await runMcpStdioWailsSmoke(options.mcpStdio) : undefined;
      const skills = options?.skills ? await runSkillWailsSmoke() : undefined;
      const templates = options?.templates ? await runTemplateWailsSmoke() : undefined;
      const vfs = options?.vfs ? await runVfsWailsSmoke() : undefined;
      return {
        appDataDir: await native.system.getAppDataDir(),
        earlyErrors: Array.isArray(earlyErrors) ? earlyErrors.slice(0, 10) : [],
        hasWailsEnvironment: Boolean((window as any)._wails?.environment),
        hasWailsInvoke: typeof (window as any)._wails?.invoke === 'function',
        href: window.location.href,
        isInjected: native.runtime.isInjected(),
        isWails,
        key,
        mcpStdio,
        skills,
        templates,
        vfs,
        rootMounted: Boolean(smokeSentinel || (root && root.childElementCount > 0)),
        smokeSentinelRendered: Boolean(smokeSentinel),
        smokeFlag,
        stored,
        topLevelErrorBoundaryVisible: Boolean(errorBoundary),
        value,
        ok: Boolean(isWails && !native.runtime.isInjected() && smokeFlag && stored === value),
      };
    };
    return true;
  };

  if (tryInstall()) return;

  const startedAt = Date.now();
  const timer = window.setInterval(() => {
    if (tryInstall() || Date.now() - startedAt > 10_000) {
      window.clearInterval(timer);
    }
  }, 100);
};

installGoWailsSmokeHook();

// ★ 注入 DSTU Logger（连接到调试面板）
setDstuLogger(createLoggerFromDebugPlugin(dstuDebugLog));

type CleanupFn = () => void;

const GLOBAL_MAIN_CLEANUP_KEY = '__DSTU_MAIN_EVENT_CLEANUPS__';
const cleanupRegistry: CleanupFn[] = [];

if (typeof window !== 'undefined') {
  const previousCleanups = (window as any)[GLOBAL_MAIN_CLEANUP_KEY] as CleanupFn[] | undefined;
  if (Array.isArray(previousCleanups)) {
    previousCleanups.forEach(fn => {
      try {
        fn();
      } catch (error) {
        console.warn('[main] 旧事件清理失败', error);
      }
    });
  }
  (window as any)[GLOBAL_MAIN_CLEANUP_KEY] = cleanupRegistry;
}

const registerCleanup = (fn: CleanupFn) => {
  cleanupRegistry.push(() => {
    try {
      fn();
    } catch (error) {
      console.warn('[main] 事件注销失败', error);
    }
  });
};

// 过滤特定 Tauri 警告（调试开关关闭时）
const installConsoleWarningFilter = () => {
  const originalWarn = console.warn;
  const tauriCallbackWarn = "[TAURI] Couldn't find callback id";
  console.warn = (...args: unknown[]) => {
    const first = args[0];
    const shouldSuppress =
      !debugMasterSwitch.isEnabled() &&
      typeof first === 'string' &&
      first.includes(tauriCallbackWarn);
    if (!shouldSuppress) {
      originalWarn.apply(console, args as any);
    }
  };
  registerCleanup(() => {
    console.warn = originalWarn;
  });
};

installConsoleWarningFilter();
// 动态初始化 Sentry（仅当配置存在且用户已同意）
// 🆕 合规要求：Sentry 默认关闭，需用户在设置中主动开启
const SENTRY_CONSENT_KEY = 'sentry_error_reporting_enabled';
let __sentryInit = false as boolean;
async function initSentryIfConfigured() {
  try {
    const dsn = (import.meta as any).env?.VITE_SENTRY_DSN;
    if (!dsn || __sentryInit) return;

    // 检查用户是否同意了错误报告
    try {
      const consent = await getSetting(SENTRY_CONSENT_KEY);
      if (consent !== 'true') return; // 默认不开启
    } catch {
      return; // 数据库未就绪或读取失败，不初始化
    }

    const Sentry: any = await import('@sentry/browser');
    const { VERSION_INFO: vi } = await import('./version');
    Sentry.init({
      dsn,
      integrations: [
        Sentry.browserTracingIntegration?.() || undefined,
      ].filter(Boolean),
      tracesSampleRate: Number((import.meta as any).env?.VITE_SENTRY_TRACES_SAMPLE_RATE ?? 0.1),
      environment: (import.meta as any).env?.MODE || 'production',
      release: vi.SENTRY_RELEASE || (window as any).__APP_VERSION__ || '0.0.0',
    });
    __sentryInit = true;
  } catch {}
}

/** 导出 Sentry 同意 key，供设置页面使用 */
export { SENTRY_CONSENT_KEY };

const root = ReactDOM.createRoot(document.getElementById("root") as HTMLElement);

/** Safe i18n accessor for contexts where hooks are unavailable (e.g. error boundary fallback).
 *  Falls back to the provided default string if i18n is not yet initialised or throws. */
const safeT = (key: string, fallback: string, options?: Record<string, unknown>): string => {
  try { return i18n.t(key, { defaultValue: fallback, ...options }) as string; } catch { return fallback; }
};

const TopLevelFallback: React.FC<{ error?: any; componentStack?: string }> = ({ error, componentStack }) => {
  const errorMessage = error instanceof Error ? error.message : String(error ?? 'Unknown error');
  const errorStack = error instanceof Error ? error.stack : undefined;
  const fullLog = [
    `Error: ${errorMessage}`,
    errorStack ? `\nStack:\n${errorStack}` : '',
    componentStack ? `\nComponent Stack:\n${componentStack}` : '',
    `\nTimestamp: ${new Date().toISOString()}`,
    `\nUserAgent: ${navigator.userAgent}`,
  ].filter(Boolean).join('');

  const [showDetails, setShowDetails] = React.useState(false);
  const [copied, setCopied] = React.useState(false);

  const handleCopy = () => {
    try {
      navigator.clipboard.writeText(fullLog).then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      });
    } catch {
      // fallback: select text for manual copy
      const el = document.getElementById('error-log-content');
      if (el) {
        const range = document.createRange();
        range.selectNodeContents(el);
        const sel = window.getSelection();
        sel?.removeAllRanges();
        sel?.addRange(range);
      }
    }
  };

  return (
    <div
      data-deep-student-error-boundary="top-level"
      style={{
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      height: '100vh',
      width: '100vw',
      fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
      backgroundColor: '#fafafa',
      color: '#1a1a1a',
    }}>
      <div style={{ fontSize: 48, marginBottom: 16 }}>⚠️</div>
      <h1 style={{ fontSize: 20, fontWeight: 600, marginBottom: 8 }}>
        {safeT('common:error_boundary.title', '应用遇到严重错误')}
      </h1>
      <p style={{ fontSize: 14, color: '#666', marginBottom: 24, maxWidth: 400, textAlign: 'center' }}>
        {safeT('common:error_boundary.description', '应用发生了无法恢复的错误。请尝试刷新页面，如果问题持续请联系支持。')}
      </p>
      <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
        <button
          onClick={() => window.location.reload()}
          style={{
            padding: '10px 24px',
            fontSize: 14,
            fontWeight: 500,
            color: '#fff',
            backgroundColor: '#2563eb',
            border: 'none',
            borderRadius: 8,
            cursor: 'pointer',
          }}
        >
          {safeT('common:error_boundary.refresh', '刷新页面')}
        </button>
        <button
          onClick={() => setShowDetails(v => !v)}
          style={{
            padding: '10px 24px',
            fontSize: 14,
            fontWeight: 500,
            color: '#333',
            backgroundColor: '#fff',
            border: '1px solid #ddd',
            borderRadius: 8,
            cursor: 'pointer',
          }}
        >
          {showDetails
            ? safeT('common:error_boundary.hide_details', '隐藏详情')
            : safeT('common:error_boundary.show_details', '查看错误详情')}
        </button>
      </div>
      {showDetails && (
        <div style={{ width: '100%', maxWidth: 640, padding: '0 24px' }}>
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
            <button
              onClick={handleCopy}
              style={{
                padding: '6px 16px',
                fontSize: 13,
                color: copied ? '#16a34a' : '#555',
                backgroundColor: '#fff',
                border: '1px solid #ddd',
                borderRadius: 6,
                cursor: 'pointer',
              }}
            >
              {copied
                ? safeT('common:error_boundary.copied', '已复制')
                : safeT('common:error_boundary.copy_error', '复制错误日志')}
            </button>
          </div>
          <pre
            id="error-log-content"
            style={{
              padding: 16,
              fontSize: 12,
              lineHeight: 1.6,
              backgroundColor: '#f5f5f5',
              border: '1px solid #e5e5e5',
              borderRadius: 8,
              overflow: 'auto',
              maxHeight: 300,
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-all',
              color: '#d32f2f',
              userSelect: 'text',
            }}
          >
            {fullLog}
          </pre>
        </div>
      )}
    </div>
  );
};

const appTree = (
  <ErrorBoundary name="TopLevel" fallback={(error, componentStack) => <TopLevelFallback error={error} componentStack={componentStack} />}>
    <OverlayCoordinatorProvider>
      <DialogControlProvider>
        {isGoWailsSmokePage() ? <span data-deep-student-smoke-rendered="true" hidden /> : null}
        <App />
      </DialogControlProvider>
    </OverlayCoordinatorProvider>
  </ErrorBoundary>
);

const renderApp = () => {
  // 在开发态移除 StrictMode，避免 effect/事件监听的二次执行造成噪声与性能影响；
  // 生产环境仍保留 StrictMode 以捕获潜在问题。
  if ((import.meta as any).env?.MODE === 'production') {
    root.render(<React.StrictMode>{appTree}</React.StrictMode>);
  } else {
    root.render(appTree);
  }
};

renderApp();
initSentryIfConfigured().catch(() => {});


// Initialize Frontend MCP Service from saved settings (best-effort)
bootstrapMcpFromSettings({ preheat: true }).catch((err) => {
  debugLog.warn('[MCP] Bootstrap failed:', err);
});

// Respond to settings change to reload MCP servers from DB
const handleSystemSettingsChanged = async (event?: Event) => {
  const detail = (event as CustomEvent<any> | undefined)?.detail;
  const shouldReloadMcp = Boolean(
    detail?.mcpReloaded ||
    detail?.mcpChanged ||
    (typeof detail?.settingKey === 'string' && detail.settingKey.startsWith('mcp.'))
  );
  if (!shouldReloadMcp) return;
  bootstrapMcpFromSettings({ preheat: true }).catch((err) => {
    debugLog.warn('[MCP] Bootstrap (settings reload) failed:', err);
  });
};
window.addEventListener('systemSettingsChanged', handleSystemSettingsChanged);
registerCleanup(() => window.removeEventListener('systemSettingsChanged', handleSystemSettingsChanged));

if ((window as any).__TAURI_INTERNALS__ || native.runtime.isWails() || native.runtime.isInjected()) {
  (async () => {
    try {
      const baseWarn = console.warn.bind(console) as (...args: unknown[]) => void;
      // 安全加载日志插件（可选）。使用 vite-ignore 避免 Vite 预打包时强制解析依赖。
      const safeLoadLogPlugin = async () => {
        if (!native.runtime.isTauri()) {
          return null;
        }
        try {
          const PKG = '@tauri-apps/plugin-log';
          const mod = await import(/* @vite-ignore */ PKG);
          return mod as any;
        } catch {
          return null;
        }
      };

      const logPlugin = await safeLoadLogPlugin();
      if (logPlugin && typeof logPlugin.attachConsole === 'function') {
        try { await logPlugin.attachConsole(); } catch {}
        const safeFallbackWarn = (...warnArgs: unknown[]) => {
          try {
            baseWarn?.(...warnArgs);
          } catch {
            // ignore fallback logging failures
          }
        };
        const forwardConsole = (
          fnName: 'log' | 'debug' | 'info' | 'warn' | 'error',
          logger: (message: string) => Promise<void>
        ) => {
          const original = (console as any)[fnName]?.bind(console) as (...args: any[]) => void;
          (console as any)[fnName] = (...args: any[]) => {
            try { original?.(...args); } catch {}
            try {
              const msg = args.map(a => {
                if (a instanceof Error) return `${a.name}: ${a.message}`;
                if (typeof a === 'string') return a;
                try { return JSON.stringify(a); } catch { return String(a); }
              }).join(' ');
              logger?.(msg).catch((err) => {
                // 不能再走被代理 console.warn，否则 warn 通道失败时会递归。
                safeFallbackWarn('[Main] console forward failed:', err);
              });
            } catch {
              // ignore serialization/logging errors
            }
          };
        };
        forwardConsole('log', logPlugin.trace ?? logPlugin.info);
        forwardConsole('debug', logPlugin.debug ?? logPlugin.info);
        forwardConsole('info', logPlugin.info);
        forwardConsole('warn', logPlugin.warn ?? logPlugin.info);
        forwardConsole('error', logPlugin.error ?? logPlugin.info);
      }

      const recent = new Map<string, number>();
      const throttleMs = 10_000;

      const serializeUnknown = (value: unknown) => {
        if (value === undefined || value === null) {
          return null;
        }
        if (value instanceof Error) {
          return {
            message: value.message,
            name: value.name,
            stack: value.stack ?? null,
          };
        }
        const valueType = typeof value;
        if (valueType === 'string' || valueType === 'number' || valueType === 'boolean') {
          return value;
        }
        try {
          return JSON.parse(JSON.stringify(value));
        } catch {
          return String(value);
        }
      };

      const emitLog = (payload: any) => {
        const key = JSON.stringify({
          message: payload?.message,
          stack: payload?.stack,
          kind: payload?.kind,
        });
        const now = Date.now();
        for (const [storedKey, storedAt] of recent) {
          if (now - storedAt > throttleMs) {
            recent.delete(storedKey);
          }
        }
        const last = recent.get(key);
        if (last && now - last < throttleMs) {
          return;
        }
        recent.set(key, now);
        nativeInvoke('report_frontend_log', { payload }).catch((err) => {
          baseWarn?.('[Main] report_frontend_log failed:', err);
        });
      };

      const handleWindowError = (event: ErrorEvent) => {
        if (!event.message && !(event.error instanceof Error)) {
          return;
        }
        const stack = event.error instanceof Error ? event.error.stack ?? null : null;
        emitLog({
          level: 'ERROR',
          kind: 'WINDOW_ERROR',
          message: event.message || (event.error && String(event.error)) || safeT('common:frontend_errors.window_error', 'Window Error'),
          stack,
          url: event.filename || window.location.href,
          line: event.lineno ?? null,
          column: event.colno ?? null,
          route: window.location.hash || window.location.pathname,
          user_agent: navigator.userAgent,
          extra: serializeUnknown(event.error),
        });
        // 同步写入日志插件（若可用）
        (async () => {
          const lp = await safeLoadLogPlugin();
          try { await lp?.error?.(`[WINDOW_ERROR] ${event.message}`); } catch {}
        })();
      };
      window.addEventListener('error', handleWindowError);
      registerCleanup(() => window.removeEventListener('error', handleWindowError));

      const handleUnhandledRejection = (event: PromiseRejectionEvent) => {
        const reason = event.reason;
        let message = safeT('common:frontend_errors.unhandled_promise_rejection', 'Unhandled Promise Rejection');
        let stack: string | null = null;
        if (reason instanceof Error) {
          message = reason.message || message;
          stack = reason.stack ?? null;
        } else if (typeof reason === 'string') {
          message = reason;
        } else if (reason && typeof reason === 'object' && 'message' in reason) {
          message = String((reason as { message?: unknown }).message ?? message);
        }

        // ★ 2026-02-04: 过滤 Tauri HTTP 插件的已知 bug
        // 当请求被取消时，插件内部会尝试调用 fetch_cancel_body 命令
        // 但该命令在某些情况下未正确注册，导致大量无害的错误日志
        // 参考: https://github.com/tauri-apps/plugins-workspace/issues/2557
        if (isKnownTauriHttpNoise(message, stack || undefined)) {
          event.preventDefault(); // 阻止默认的错误输出
          return; // 静默忽略此错误
        }

        emitLog({
          level: 'ERROR',
          kind: 'UNHANDLED_REJECTION',
          message,
          stack,
          url: window.location.href,
          route: window.location.hash || window.location.pathname,
          user_agent: navigator.userAgent,
          extra: serializeUnknown(reason),
        });
        (async () => {
          const lp = await safeLoadLogPlugin();
          try { await lp?.error?.(`[UNHANDLED_REJECTION] ${message}`); } catch {}
        })();
      };

      window.addEventListener('unhandledrejection', handleUnhandledRejection);
      registerCleanup(() => window.removeEventListener('unhandledrejection', handleUnhandledRejection));
      
      // 🔧 MCP Debug Enhancement Module - 全自动调试支持
      // 仅在开发模式 + 调试总开关开启时初始化（或通过 env 强制启用）
      const env = (import.meta as any).env ?? {};
      const isDev = env.MODE !== 'production';
      const forceEnableMcpDebug = env.VITE_ENABLE_MCP_DEBUG === 'true';
      let mcpDebugInitialized = false;
      let mcpDebugDestroy: (() => void) | null = null;

      const initMcpDebug = async () => {
        if (mcpDebugInitialized) return;
        try {
          const { initMCPDebug, registerAllStores, destroyMCPDebug } = await import('./mcp-debug');
          mcpDebugDestroy = destroyMCPDebug;
          await initMCPDebug({
            autoStartErrorCapture: true,
            autoStartNetworkMonitor: false, // 按需启动，避免性能开销
            autoStartPerformanceMonitor: false,
          });
          mcpDebugInitialized = true;
          console.log('[main] MCP Debug module initialized');
          // 延迟注册 stores，确保应用已完全加载
          setTimeout(() => {
            registerAllStores().catch((err) => {
              console.warn('[main] Store registration failed:', err);
            });
          }, 2000);
        } catch (err) {
          console.warn('[main] MCP Debug initialization failed:', err);
        }
      };

      const teardownMcpDebug = () => {
        if (!mcpDebugInitialized) return;
        try { mcpDebugDestroy?.(); } catch {}
        mcpDebugInitialized = false;
      };

      const shouldEnableMcpDebug = () => forceEnableMcpDebug || (isDev && debugMasterSwitch.isEnabled());

      if (shouldEnableMcpDebug()) {
        void initMcpDebug();
      }

      const unsubscribeDebugSwitch = debugMasterSwitch.addListener((enabled) => {
        if (forceEnableMcpDebug || !isDev) return;
        if (enabled) {
          void initMcpDebug();
        } else {
          teardownMcpDebug();
        }
      });
      registerCleanup(() => unsubscribeDebugSwitch());
    } catch {
      // ignore initialization errors
    }
  })();
}

// 🆕 P1防闪退：Chat V2 会话保存（应用生命周期）
// 动态导入避免循环依赖，使用同步方式触发保存
const triggerChatV2EmergencySave = () => {
  try {
    // 动态获取 sessionManager 和 autoSave（避免启动时循环依赖）
    const chatV2Module = (window as any).__CHAT_V2_EMERGENCY_SAVE__;
    if (chatV2Module && typeof chatV2Module.emergencySave === 'function') {
      chatV2Module.emergencySave();
    }
  } catch (e) {
    console.warn('[main] Chat V2 emergency save failed:', e);
  }
};

// 确保在页面关闭时保存MCP缓存和Chat V2会话
const handleBeforeUnload = () => {
  // 🆕 P1: 触发 Chat V2 紧急保存
  triggerChatV2EmergencySave();
  
  try {
    McpService.dispose();
  } catch {}
  // 🔧 清理全局缓存管理器（停止 cleanup 定时器、释放缓存）
  try {
    disposeGlobalCacheManager();
  } catch {}
};
window.addEventListener('beforeunload', handleBeforeUnload);
registerCleanup(() => window.removeEventListener('beforeunload', handleBeforeUnload));

// 🆕 P1防闪退：移动端 visibilitychange 监听
// 当应用进入后台时触发保存（移动端常见场景）
const handleVisibilityChange = () => {
  if (document.visibilityState === 'hidden') {
    triggerChatV2EmergencySave();
  }
};
document.addEventListener('visibilitychange', handleVisibilityChange);
registerCleanup(() => document.removeEventListener('visibilitychange', handleVisibilityChange));

if ((import.meta as any)?.hot) {
  (import.meta as any).hot.dispose(() => {
    cleanupRegistry.forEach(fn => fn());
    cleanupRegistry.length = 0;
    if (typeof window !== 'undefined' && (window as any)[GLOBAL_MAIN_CLEANUP_KEY] === cleanupRegistry) {
      delete (window as any)[GLOBAL_MAIN_CLEANUP_KEY];
    }
  });
}
