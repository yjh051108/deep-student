/**
 * 🧪 子代理测试插件 (Subagent Test Plugin)
 * 
 * 功能：
 * 1. 自动触发子代理测试流程
 * 2. 记录 UI 渲染情况（SubagentContainer 展开/收起、消息加载等）
 * 3. 记录用户操作（点击展开、查看完整会话等）
 * 4. 记录子代理生命周期事件（创建、执行、完成/失败）
 * 5. 提供全局控制台 API 方便调试
 * 
 * 使用方式：
 * - 控制台：window.__subagentTest.startTest() 启动自动测试
 * - 控制台：window.__subagentTest.getLogs() 获取所有日志
 * - 控制台：window.__subagentTest.exportReport() 导出测试报告
 */

import { invoke } from '@/runtime/native';
import { listen, UnlistenFn } from '@tauri-apps/api/event';
import { useWorkspaceStore } from '../workspace/workspaceStore';
import { WORKSPACE_EVENTS } from '../workspace/events';

// =============================================================================
// 类型定义
// =============================================================================

/** 日志类型 */
export type SubagentLogType =
  | 'lifecycle'     // 生命周期：创建、执行、完成、失败
  | 'task'          // 🆕 任务持久化：create_task、mark_running、mark_completed
  | 'ui_render'     // UI 渲染：组件挂载、状态变化
  | 'ui_interaction'// 用户交互：展开、收起、点击
  | 'data_load'     // 数据加载：消息加载、刷新
  | 'event'         // 事件：后端事件接收
  | 'error'         // 错误
  | 'test';         // 测试流程

/** 日志条目 */
export interface SubagentLogEntry {
  id: number;
  timestamp: string;
  type: SubagentLogType;
  action: string;
  data: Record<string, unknown>;
  subagentSessionId?: string;
  workspaceId?: string;
  /** 耗时（毫秒） */
  durationMs?: number;
}

/** 测试状态 */
export type TestStatus = 'idle' | 'running' | 'completed' | 'failed';

/** 测试配置 */
export interface SubagentTestConfig {
  /** 是否自动展开 SubagentContainer */
  autoExpandContainer: boolean;
  /** 是否自动刷新间隔（毫秒），0 表示禁用 */
  autoRefreshInterval: number;
  /** 测试超时时间（毫秒） */
  testTimeout: number;
  /** 是否输出到控制台 */
  consoleEnabled: boolean;
  /** 测试任务 Prompt（发送给 LLM 的消息） */
  testPrompt: string;
}

/** 测试报告 */
export interface SubagentTestReport {
  startTime: string;
  endTime: string;
  status: TestStatus;
  totalLogs: number;
  logsByType: Record<SubagentLogType, number>;
  timeline: SubagentLogEntry[];
  summary: {
    subagentCreated: boolean;
    subagentExecuted: boolean;
    subagentCompleted: boolean;
    uiRenderedCorrectly: boolean;
    // 🆕 任务持久化状态
    taskPersisted: boolean;
    taskStarted: boolean;
    taskFinished: boolean;
    errors: string[];
  };
}

// =============================================================================
// 全局状态
// =============================================================================

const LOGS: SubagentLogEntry[] = [];
let LOG_ID = 0;
const MAX_LOGS = 1000;

let testStatus: TestStatus = 'idle';
let testStartTime: string | null = null;
let testEndTime: string | null = null;
let currentSubagentSessionId: string | null = null;
let currentWorkspaceId: string | null = null;

let eventUnlisteners: UnlistenFn[] = [];

/** 计时器映射 */
const timers: Map<string, number> = new Map();

/** 默认配置 */
const DEFAULT_CONFIG: SubagentTestConfig = {
  autoExpandContainer: true,
  autoRefreshInterval: 2000,
  testTimeout: 120000, // 2分钟超时
  consoleEnabled: true,
  testPrompt: '请创建一个子代理帮我完成任务：写一首关于科技的短诗',
};

let config: SubagentTestConfig = { ...DEFAULT_CONFIG };

// =============================================================================
// 日志函数
// =============================================================================

/**
 * 记录日志
 */
export function logSubagent(
  type: SubagentLogType,
  action: string,
  data: Record<string, unknown> = {},
  context?: {
    subagentSessionId?: string;
    workspaceId?: string;
    durationMs?: number;
  }
): SubagentLogEntry {
  const entry: SubagentLogEntry = {
    id: ++LOG_ID,
    timestamp: new Date().toISOString(),
    type,
    action,
    data,
    subagentSessionId: context?.subagentSessionId || currentSubagentSessionId || undefined,
    workspaceId: context?.workspaceId || currentWorkspaceId || undefined,
    durationMs: context?.durationMs,
  };

  LOGS.push(entry);
  while (LOGS.length > MAX_LOGS) {
    LOGS.shift();
  }

  if (config.consoleEnabled) {
    const prefix = `[SubagentTest][${type}]`;
    const emoji = {
      lifecycle: '🔄',
      task: '💾',      // 🆕 任务持久化
      ui_render: '🎨',
      ui_interaction: '👆',
      data_load: '📥',
      event: '📡',
      error: '❌',
      test: '🧪',
    }[type];
    
    console.log(`${emoji} ${prefix} ${action}`, {
      ...data,
      ...(context?.durationMs ? { durationMs: context.durationMs } : {}),
    });
  }

  // 触发事件通知
  window.dispatchEvent(new CustomEvent('SUBAGENT_TEST_LOG', { detail: entry }));

  return entry;
}

/**
 * 开始计时
 */
export function startTimer(key: string): void {
  timers.set(key, Date.now());
}

/**
 * 结束计时并返回耗时
 */
export function endTimer(key: string): number {
  const start = timers.get(key);
  timers.delete(key);
  return start ? Date.now() - start : 0;
}

// =============================================================================
// UI 渲染日志（供组件调用）
// =============================================================================

/** SubagentContainer 挂载 */
export function logContainerMount(subagentSessionId: string): void {
  logSubagent('ui_render', 'container_mount', { subagentSessionId }, { subagentSessionId });
}

/** SubagentContainer 卸载 */
export function logContainerUnmount(subagentSessionId: string): void {
  logSubagent('ui_render', 'container_unmount', { subagentSessionId }, { subagentSessionId });
}

/** SubagentContainer 展开 */
export function logContainerExpand(subagentSessionId: string): void {
  startTimer(`expand_${subagentSessionId}`);
  logSubagent('ui_interaction', 'container_expand', { subagentSessionId }, { subagentSessionId });
}

/** SubagentContainer 收起 */
export function logContainerCollapse(subagentSessionId: string): void {
  logSubagent('ui_interaction', 'container_collapse', { subagentSessionId }, { subagentSessionId });
}

/** 消息加载开始 */
export function logMessagesLoadStart(subagentSessionId: string): void {
  startTimer(`load_${subagentSessionId}`);
  logSubagent('data_load', 'messages_load_start', { subagentSessionId }, { subagentSessionId });
}

/** 消息加载完成 */
export function logMessagesLoadComplete(subagentSessionId: string, messageCount: number): void {
  const duration = endTimer(`load_${subagentSessionId}`);
  const expandDuration = endTimer(`expand_${subagentSessionId}`);
  logSubagent('data_load', 'messages_load_complete', { 
    subagentSessionId, 
    messageCount,
    loadDurationMs: duration,
    expandToLoadDurationMs: expandDuration || undefined,
  }, { subagentSessionId, durationMs: duration });
}

/** 消息加载失败 */
export function logMessagesLoadError(subagentSessionId: string, error: string): void {
  const duration = endTimer(`load_${subagentSessionId}`);
  logSubagent('error', 'messages_load_error', { subagentSessionId, error }, { subagentSessionId, durationMs: duration });
}

/** 状态变化 */
export function logStatusChange(subagentSessionId: string, oldStatus: string, newStatus: string): void {
  logSubagent('ui_render', 'status_change', { subagentSessionId, oldStatus, newStatus }, { subagentSessionId });
}

/** 查看完整会话 */
export function logViewFullSession(subagentSessionId: string): void {
  logSubagent('ui_interaction', 'view_full_session', { subagentSessionId }, { subagentSessionId });
}

/** 实时刷新触发 */
export function logAutoRefresh(subagentSessionId: string, messageCount: number): void {
  logSubagent('data_load', 'auto_refresh', { subagentSessionId, messageCount }, { subagentSessionId });
}

// =============================================================================
// 🆕 任务持久化日志（供后端事件监听调用）
// =============================================================================

/** 任务创建（持久化到数据库） */
export function logTaskCreated(subagentSessionId: string, taskId: string): void {
  logSubagent('task', 'task_created', { subagentSessionId, taskId }, { subagentSessionId });
}

/** 任务开始执行（mark_running） */
export function logTaskStarted(subagentSessionId: string, taskId: string): void {
  logSubagent('task', 'task_started', { subagentSessionId, taskId }, { subagentSessionId });
}

/** 任务完成（mark_completed） */
export function logTaskCompleted(subagentSessionId: string, taskId: string, summary?: string): void {
  logSubagent('task', 'task_completed', { subagentSessionId, taskId, summary }, { subagentSessionId });
}

/** 任务失败（mark_failed） */
export function logTaskFailed(subagentSessionId: string, taskId: string, error?: string): void {
  logSubagent('task', 'task_failed', { subagentSessionId, taskId, error }, { subagentSessionId });
}

/** ChatContainer 嵌入挂载（P1 修复后使用 ChatContainer 而非 SubagentContainer） */
export function logChatContainerEmbed(subagentSessionId: string): void {
  logSubagent('ui_render', 'chat_container_embed', { subagentSessionId }, { subagentSessionId });
}

// =============================================================================
// 事件监听
// =============================================================================

/**
 * 初始化事件监听（用于捕获后端事件）
 */
async function initEventListeners(): Promise<void> {
  // 清理旧监听器
  for (const unlisten of eventUnlisteners) {
    unlisten();
  }
  eventUnlisteners = [];

  // 监听 Agent 加入事件
  const unlistenAgentJoined = await listen<any>(
    WORKSPACE_EVENTS.AGENT_JOINED,
    (event) => {
      const { workspace_id, agent } = event.payload;
      if (agent.session_id?.startsWith('subagent_')) {
        currentSubagentSessionId = agent.session_id;
        currentWorkspaceId = workspace_id;
        logSubagent('lifecycle', 'subagent_created', {
          workspaceId: workspace_id,
          sessionId: agent.session_id,
          role: agent.role,
          skillId: agent.skill_id,
        });
      }
    }
  );
  eventUnlisteners.push(unlistenAgentJoined);

  // 监听 Agent 状态变更事件
  const unlistenAgentStatus = await listen<any>(
    WORKSPACE_EVENTS.AGENT_STATUS_CHANGED,
    (event) => {
      const { workspace_id, session_id, status } = event.payload;
      if (session_id?.startsWith('subagent_')) {
        logSubagent('lifecycle', 'subagent_status_changed', {
          workspaceId: workspace_id,
          sessionId: session_id,
          status,
        }, { subagentSessionId: session_id });

        if (status === 'completed') {
          logSubagent('lifecycle', 'subagent_completed', {
            sessionId: session_id,
          }, { subagentSessionId: session_id });
        } else if (status === 'failed') {
          logSubagent('lifecycle', 'subagent_failed', {
            sessionId: session_id,
          }, { subagentSessionId: session_id });
        }
      }
    }
  );
  eventUnlisteners.push(unlistenAgentStatus);

  // 监听 Worker Ready 事件
  const unlistenWorkerReady = await listen<any>(
    WORKSPACE_EVENTS.WORKER_READY,
    (event) => {
      const { workspace_id, agent_session_id, skill_id } = event.payload;
      if (agent_session_id?.startsWith('subagent_')) {
        logSubagent('event', 'worker_ready_received', {
          workspaceId: workspace_id,
          sessionId: agent_session_id,
          skillId: skill_id,
        }, { subagentSessionId: agent_session_id });
      }
    }
  );
  eventUnlisteners.push(unlistenWorkerReady);

  // 监听消息事件
  const unlistenMessage = await listen<any>(
    WORKSPACE_EVENTS.MESSAGE_RECEIVED,
    (event) => {
      const { workspace_id, message } = event.payload;
      if (message.sender_session_id?.startsWith('subagent_') || 
          message.target_session_id?.startsWith('subagent_')) {
        logSubagent('event', 'message_received', {
          workspaceId: workspace_id,
          messageId: message.id,
          senderSessionId: message.sender_session_id,
          targetSessionId: message.target_session_id,
          messageType: message.message_type,
        });
      }
    }
  );
  eventUnlisteners.push(unlistenMessage);

  logSubagent('test', 'event_listeners_initialized', {});
}

/**
 * 清理事件监听
 */
async function cleanupEventListeners(): Promise<void> {
  for (const unlisten of eventUnlisteners) {
    unlisten();
  }
  eventUnlisteners = [];
  logSubagent('test', 'event_listeners_cleaned', {});
}

// =============================================================================
// 自动测试流程
// =============================================================================

/**
 * 启动全自动测试
 * 
 * 通过真实对话流程触发 subagent_call 工具：
 * 1. 获取当前活跃会话
 * 2. 通过 chat_v2_send_message 发送测试 Prompt
 * 3. LLM 响应并触发 subagent_call 工具
 * 4. 监听并记录所有事件
 */
export async function startTest(customConfig?: Partial<SubagentTestConfig>): Promise<void> {
  if (testStatus === 'running') {
    console.warn('[SubagentTest] Test already running');
    return;
  }

  // 合并配置
  config = { ...DEFAULT_CONFIG, ...customConfig };
  
  // 重置状态
  LOGS.length = 0;
  LOG_ID = 0;
  testStatus = 'running';
  testStartTime = new Date().toISOString();
  testEndTime = null;
  currentSubagentSessionId = null;
  currentWorkspaceId = null;

  logSubagent('test', 'test_started', { 
    config,
    mode: 'auto',
  });

  try {
    // 初始化事件监听
    await initEventListeners();

    // 检查是否有活跃的工作区
    const workspaceId = useWorkspaceStore.getState().currentWorkspaceId;
    if (workspaceId) {
      currentWorkspaceId = workspaceId;
      logSubagent('test', 'found_existing_workspace', { workspaceId });
    }

    // 获取当前会话 ID
    // 使用 sessionManager 获取当前活跃会话
    const { sessionManager } = await import('../core/session/sessionManager');
    const currentSessionId = sessionManager.getCurrentSessionId();
    
    if (!currentSessionId) {
      throw new Error('没有活跃的会话，请先打开一个对话');
    }

    const store = sessionManager.get(currentSessionId);
    if (!store) {
      throw new Error(`无法获取会话 Store: ${currentSessionId}`);
    }

    // 检查会话状态
    const sessionStatus = store.getState().sessionStatus;
    if (sessionStatus === 'streaming') {
      throw new Error('当前会话正在流式传输中，请稍后重试');
    }

    logSubagent('test', 'found_session', { sessionId: currentSessionId });

    // 通过真实对话流程发送消息
    logSubagent('test', 'sending_test_prompt', { 
      sessionId: currentSessionId,
      prompt: config.testPrompt,
    });

    startTimer('llm_response');

    // 使用 store.sendMessage() 发送消息（复用 MultiAgentDebugPlugin 的发送消息逻辑）
    await store.getState().sendMessage(config.testPrompt);

    logSubagent('test', 'prompt_sent', { 
      sessionId: currentSessionId,
    });

    logSubagent('test', 'waiting_for_subagent', {
      hint: '等待 LLM 触发 subagent_call 工具...',
    });

    // 设置超时
    setTimeout(() => {
      if (testStatus === 'running') {
        const responseDuration = endTimer('llm_response');
        // 如果有子代理被创建，则认为测试成功
        if (currentSubagentSessionId) {
          testStatus = 'completed';
          testEndTime = new Date().toISOString();
          logSubagent('test', 'test_completed', { 
            subagentSessionId: currentSubagentSessionId,
            responseDurationMs: responseDuration,
          }, { durationMs: responseDuration });
        } else {
          testStatus = 'failed';
          testEndTime = new Date().toISOString();
          logSubagent('error', 'test_timeout', { 
            timeoutMs: config.testTimeout,
            message: 'LLM 没有触发 subagent_call 工具，可能原因：1) 工作区未创建 2) 模型不支持 3) Prompt 不够明确',
          });
        }
        cleanupEventListeners();
      }
    }, config.testTimeout);

  } catch (error: unknown) {
    testStatus = 'failed';
    testEndTime = new Date().toISOString();
    logSubagent('error', 'test_failed', { 
      error: error instanceof Error ? error.message : String(error) 
    });
    await cleanupEventListeners();
  }
}

/**
 * 停止测试
 */
export async function stopTest(): Promise<void> {
  if (testStatus !== 'running') {
    console.warn('[SubagentTest] No test running');
    return;
  }

  testStatus = 'completed';
  testEndTime = new Date().toISOString();
  logSubagent('test', 'test_stopped', {});
  await cleanupEventListeners();
}

/**
 * 获取所有日志
 */
export function getLogs(): SubagentLogEntry[] {
  return [...LOGS];
}

/**
 * 获取过滤后的日志
 */
export function getLogsByType(type: SubagentLogType): SubagentLogEntry[] {
  return LOGS.filter(log => log.type === type);
}

/**
 * 清空日志
 */
export function clearLogs(): void {
  LOGS.length = 0;
  LOG_ID = 0;
}

/**
 * 获取测试状态
 */
export function getTestStatus(): TestStatus {
  return testStatus;
}

/**
 * 生成测试报告
 */
export function generateReport(): SubagentTestReport {
  const logsByType: Record<SubagentLogType, number> = {
    lifecycle: 0,
    task: 0,        // 🆕
    ui_render: 0,
    ui_interaction: 0,
    data_load: 0,
    event: 0,
    error: 0,
    test: 0,
  };

  for (const log of LOGS) {
    logsByType[log.type]++;
  }

  // 分析摘要
  const subagentCreated = LOGS.some(l => l.action === 'subagent_created');
  const subagentExecuted = LOGS.some(l => l.action === 'worker_ready_received');
  const subagentCompleted = LOGS.some(l => l.action === 'subagent_completed');
  // 🆕 P1 修复后使用 ChatContainer 嵌入，或者回退到旧的 container_mount
  const uiRenderedCorrectly = (
    LOGS.some(l => l.action === 'chat_container_embed') ||
    LOGS.some(l => l.action === 'container_mount')
  ) && LOGS.some(l => l.action === 'messages_load_complete');
  
  // 🆕 任务持久化状态
  const taskPersisted = LOGS.some(l => l.action === 'task_created');
  const taskStarted = LOGS.some(l => l.action === 'task_started');
  const taskFinished = LOGS.some(l => l.action === 'task_completed' || l.action === 'task_failed');
  
  const errors = LOGS.filter(l => l.type === 'error').map(l => l.action);

  return {
    startTime: testStartTime || '',
    endTime: testEndTime || new Date().toISOString(),
    status: testStatus,
    totalLogs: LOGS.length,
    logsByType,
    timeline: [...LOGS],
    summary: {
      subagentCreated,
      subagentExecuted,
      subagentCompleted,
      uiRenderedCorrectly,
      taskPersisted,
      taskStarted,
      taskFinished,
      errors,
    },
  };
}

/**
 * 导出测试报告为 JSON
 */
export function exportReport(): string {
  const report = generateReport();
  return JSON.stringify(report, null, 2);
}

/**
 * 下载测试报告
 */
export function downloadReport(): void {
  const report = exportReport();
  const blob = new Blob([report], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `subagent-test-report-${new Date().toISOString().replace(/[:.]/g, '-')}.json`;
  a.click();
  URL.revokeObjectURL(url);
}

/**
 * 获取配置
 */
export function getConfig(): SubagentTestConfig {
  return { ...config };
}

/**
 * 更新配置
 */
export function updateConfig(newConfig: Partial<SubagentTestConfig>): void {
  config = { ...config, ...newConfig };
  logSubagent('test', 'config_updated', { config });
}

// =============================================================================
// 全局注入
// =============================================================================

function injectSubagentTest(): void {
  (window as any).__subagentTest = {
    // 测试控制
    startTest,
    stopTest,
    getTestStatus,
    
    // 日志管理
    getLogs,
    getLogsByType,
    clearLogs,
    
    // 报告
    generateReport,
    exportReport,
    downloadReport,
    
    // 配置
    getConfig,
    updateConfig,
    
    // 手动日志（供组件调用）
    log: logSubagent,
    logContainerMount,
    logContainerUnmount,
    logContainerExpand,
    logContainerCollapse,
    logMessagesLoadStart,
    logMessagesLoadComplete,
    logMessagesLoadError,
    logStatusChange,
    logViewFullSession,
    logAutoRefresh,
    // 🆕 任务持久化日志
    logTaskCreated,
    logTaskStarted,
    logTaskCompleted,
    logTaskFailed,
    logChatContainerEmbed,
  };

  console.log('🧪 [SubagentTest] Plugin loaded. Use window.__subagentTest to access.');
  console.log('   - startTest(): Start automated test');
  console.log('   - getLogs(): Get all logs');
  console.log('   - generateReport(): Generate test report');
  console.log('   - downloadReport(): Download report as JSON');
}

// 立即注入
injectSubagentTest();

export default {
  startTest,
  stopTest,
  getLogs,
  generateReport,
  exportReport,
  downloadReport,
  logSubagent,
  logContainerMount,
  logContainerUnmount,
  logContainerExpand,
  logContainerCollapse,
  logMessagesLoadStart,
  logMessagesLoadComplete,
  logMessagesLoadError,
  logStatusChange,
  logViewFullSession,
  logAutoRefresh,
  // 🆕 任务持久化日志
  logTaskCreated,
  logTaskStarted,
  logTaskCompleted,
  logTaskFailed,
  logChatContainerEmbed,
};
