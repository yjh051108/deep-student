/**
 * CardEngine - 并行制卡引擎
 *
 * 设计原则：
 * - 前端只负责事件监听和状态聚合，不控制并发
 * - 并发控制由后端 EnhancedAnkiService 负责
 * - 前端通过监听 'anki_generation_event' 收集卡片和进度
 * - 任务控制通过 Tauri 命令调用后端接口
 *
 * 职责：
 * 1. 事件监听：监听后端推送的 anki_generation_event
 * 2. 状态聚合：收集各任务的卡片和状态
 * 3. 流式输出：每张卡片生成后立即通过本地事件推送给 UI
 * 4. 任务控制：包装后端暂停/恢复/取消命令
 *
 * @see CardForge-Architecture-v2.md
 */

import {
  AnkiCardResult,
  CardForgeEvent,
  CardForgeEventListener,
  CardForgeEventType,
  CardGeneratedPayload,
  CardGenerationTask,
  ConcurrencyConfig,
  DEFAULT_CONCURRENCY_CONFIG,
  DocumentCompletePayload,
  DocumentSegment,
  TaskProgressPayload,
  TaskStatus,
  TemplateInfo,
} from '../types';

import { listen, type NativeUnlistenFn as UnlistenFn } from '@/runtime/nativeEvents';

// ============================================================================
// 类型定义
// ============================================================================

/** 卡片生成选项 */
export interface CardGenerationOptions {
  /** 牌组名称 */
  deckName?: string;
  /** 笔记类型 */
  noteType?: string;
  /** 最大卡片数 */
  maxCards?: number;
  /** 自定义要求 */
  customRequirements?: string;
  /** 任务间延迟（ms） */
  taskDelay?: number;
  /** 单任务超时（ms） */
  taskTimeout?: number;
}

/** 信号量实现 - 控制并发数 */
class Semaphore {
  private permits: number;
  private queue: Array<() => void> = [];

  constructor(permits: number) {
    this.permits = permits;
  }

  /** 获取许可 */
  async acquire(): Promise<void> {
    if (this.permits > 0) {
      this.permits--;
      return Promise.resolve();
    }

    return new Promise<void>((resolve) => {
      this.queue.push(resolve);
    });
  }

  /** 释放许可 */
  release(): void {
    this.permits++;
    const next = this.queue.shift();
    if (next) {
      this.permits--;
      next();
    }
  }

  /** 获取当前可用许可数 */
  available(): number {
    return this.permits;
  }
}

/** 任务执行上下文 */
interface TaskContext {
  taskId: string;
  segmentIndex: number;
  documentId: string;
  startTime: number;
  abortController: AbortController;
}

// ============================================================================
// CardEngine 主类
// ============================================================================

export class CardEngine {
  private config: ConcurrencyConfig;
  private semaphore: Semaphore;
  private listeners: Map<CardForgeEventType, Set<CardForgeEventListener>>;
  private isPaused: boolean = false;
  private isCancelled: boolean = false;
  private runningTasks: Map<string, TaskContext> = new Map();
  private generatedCards: Map<string, AnkiCardResult[]> = new Map();
  private eventUnlistener: UnlistenFn | null = null;
  /** 任务完成事件回调，用于解决竞态条件 */
  private taskCompletionCallbacks: Map<string, (cards: AnkiCardResult[]) => void> = new Map();
  /** 已完成的任务 ID 集合 */
  private completedTasks: Set<string> = new Set();

  constructor(config?: Partial<ConcurrencyConfig>) {
    this.config = { ...DEFAULT_CONCURRENCY_CONFIG, ...config };
    this.semaphore = new Semaphore(this.config.maxConcurrency);
    this.listeners = new Map();
  }

  // ==========================================================================
  // 公共 API
  // ==========================================================================

  /**
   * 处理所有分段，并行生成卡片
   *
   * @param documentId 文档 ID
   * @param segments 文档分段列表
   * @param templates 可用模板列表
   * @param options 生成选项
   * @returns 生成的所有卡片
   */
  async processSegments(
    documentId: string,
    segments: DocumentSegment[],
    templates: TemplateInfo[],
    options?: CardGenerationOptions
  ): Promise<AnkiCardResult[]> {
    // 重置状态
    this.reset();

    const startTime = Date.now();
    const allCards: AnkiCardResult[] = [];

    try {
      // 发送文档开始事件
      this.emitEvent('document:start', documentId, {
        totalSegments: segments.length,
        templates: templates.map(t => t.id),
      });

      // 监听后端事件流 - 必须等待完成，避免事件丢失
      await this.setupBackendEventListener(documentId);

      // 构建任务列表
      const tasks = segments.map((segment, index) => this.createTask(
        documentId,
        segment,
        index,
        templates
      ));

      // 并行处理所有任务
      const results = await this.executeTasksInParallel(tasks, options);

      // 合并所有结果
      for (const cards of results) {
        if (cards && cards.length > 0) {
          allCards.push(...cards);
        }
      }

      // 检查是否被取消或暂停
      if (this.isCancelled) {
        this.emitEvent('document:cancelled', documentId, {
          cardsGenerated: allCards.length,
          reason: 'User cancelled',
        });
      } else if (this.isPaused) {
        this.emitEvent('document:paused', documentId, {
          cardsGenerated: allCards.length,
          completedSegments: results.filter(r => r !== null).length,
          totalSegments: segments.length,
        });
      } else {
        // 发送文档完成事件
        const payload: DocumentCompletePayload = {
          totalCards: allCards.length,
          totalSegments: segments.length,
          successfulTasks: results.filter(r => r !== null && r.length > 0).length,
          failedTasks: results.filter(r => r === null || r.length === 0).length,
          durationMs: Date.now() - startTime,
        };
        this.emitEvent('document:complete', documentId, payload);
      }

      return allCards;
    } catch (error: unknown) {
      console.error('❌ CardEngine.processSegments 失败:', error);
      this.emitEvent('document:complete', documentId, {
        totalCards: allCards.length,
        totalSegments: segments.length,
        successfulTasks: 0,
        failedTasks: segments.length,
        durationMs: Date.now() - startTime,
        error: error instanceof Error ? error.message : String(error),
      });
      throw error;
    } finally {
      // 清理事件监听器
      this.cleanupBackendEventListener();
    }
  }

  /**
   * 暂停任务执行
   */
  pause(): void {
    if (this.isPaused) {
      console.warn('⚠️ CardEngine 已处于暂停状态');
      return;
    }

    console.log('⏸️ CardEngine 暂停中...');
    this.isPaused = true;

    // 中止所有正在运行的任务
    for (const [taskId, context] of this.runningTasks.entries()) {
      console.log(`  ⏸️ 中止任务: ${taskId}`);
      context.abortController.abort();

      this.emitEvent('task:paused', context.documentId, {
        taskId,
        segmentIndex: context.segmentIndex,
      });
    }
  }

  /**
   * 恢复任务执行
   * 注意：这里只是将状态标记为未暂停，实际恢复由后端控制
   */
  resume(): void {
    if (!this.isPaused) {
      console.warn('⚠️ CardEngine 未处于暂停状态');
      return;
    }

    console.log('▶️ CardEngine 恢复中...');
    this.isPaused = false;

    // 发送恢复事件
    for (const [taskId, context] of this.runningTasks.entries()) {
      this.emitEvent('task:resumed', context.documentId, {
        taskId,
        segmentIndex: context.segmentIndex,
      });
    }
  }

  /**
   * 取消所有任务
   */
  cancel(): void {
    if (this.isCancelled) {
      console.warn('⚠️ CardEngine 已被取消');
      return;
    }

    console.log('🛑 CardEngine 取消中...');
    this.isCancelled = true;
    this.isPaused = false; // 取消时清除暂停状态

    // 中止所有正在运行的任务
    for (const [taskId, context] of this.runningTasks.entries()) {
      console.log(`  🛑 取消任务: ${taskId}`);
      context.abortController.abort();
    }

    // 清空任务队列
    this.runningTasks.clear();
  }

  /**
   * 订阅事件
   *
   * @param event 事件类型
   * @param listener 事件监听器
   * @returns 取消订阅函数
   */
  on(event: CardForgeEventType, listener: CardForgeEventListener): () => void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }

    const listeners = this.listeners.get(event)!;
    listeners.add(listener);

    // 返回取消订阅函数
    return () => {
      listeners.delete(listener);
      if (listeners.size === 0) {
        this.listeners.delete(event);
      }
    };
  }

  /**
   * 获取当前状态
   */
  getState() {
    return {
      isPaused: this.isPaused,
      isCancelled: this.isCancelled,
      runningTasksCount: this.runningTasks.size,
      availablePermits: this.semaphore.available(),
      totalCardsGenerated: Array.from(this.generatedCards.values())
        .reduce((sum, cards) => sum + cards.length, 0),
    };
  }

  // ==========================================================================
  // 私有方法
  // ==========================================================================

  /**
   * 重置引擎状态
   */
  private reset(): void {
    this.isPaused = false;
    this.isCancelled = false;
    this.runningTasks.clear();
    this.generatedCards.clear();
    this.taskCompletionCallbacks.clear();
    this.completedTasks.clear();
    this.cleanupBackendEventListener();
  }

  /**
   * 创建制卡任务
   */
  private createTask(
    documentId: string,
    segment: DocumentSegment,
    index: number,
    templates: TemplateInfo[]
  ): CardGenerationTask {
    const taskId = `${documentId}_segment_${index}`;

    return {
      taskId,
      documentId,
      segmentIndex: index,
      content: segment.content,
      status: 'pending',
      availableTemplates: templates,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      retryCount: 0,
    };
  }

  /**
   * 并行执行任务列表
   */
  private async executeTasksInParallel(
    tasks: CardGenerationTask[],
    options?: CardGenerationOptions
  ): Promise<(AnkiCardResult[] | null)[]> {
    const promises = tasks.map(task =>
      this.executeTaskWithSemaphore(task, options)
    );

    return Promise.all(promises);
  }

  /**
   * 使用信号量执行单个任务
   */
  private async executeTaskWithSemaphore(
    task: CardGenerationTask,
    options?: CardGenerationOptions
  ): Promise<AnkiCardResult[] | null> {
    // 等待信号量许可
    await this.semaphore.acquire();

    try {
      // 检查是否暂停或取消
      if (this.isPaused || this.isCancelled) {
        console.log(`⏸️ 任务 ${task.taskId} 跳过（引擎已${this.isPaused ? '暂停' : '取消'}）`);
        return null;
      }

      // 任务间延迟
      if (this.config.taskDelay > 0) {
        await this.delay(this.config.taskDelay);
      }

      // 执行任务
      const result = await this.processSegment(task, options);
      return result;
    } finally {
      // 释放信号量
      this.semaphore.release();
    }
  }

  /**
   * 处理单个分段
   *
   * 注意：实际的制卡逻辑由后端执行，前端只负责调度和状态管理
   */
  private async processSegment(
    task: CardGenerationTask,
    options?: CardGenerationOptions
  ): Promise<AnkiCardResult[] | null> {
    const { taskId, documentId, segmentIndex, content } = task;

    // 创建任务上下文
    const context: TaskContext = {
      taskId,
      segmentIndex,
      documentId,
      startTime: Date.now(),
      abortController: new AbortController(),
    };

    this.runningTasks.set(taskId, context);

    try {
      // 发送任务开始事件
      this.emitEvent('task:start', documentId, {
        taskId,
        segmentIndex,
        content: content.substring(0, 100) + '...',
      });

      // 检查是否暂停或取消
      if (this.isPaused || this.isCancelled) {
        return null;
      }

      // 注意：这里不直接调用后端生成卡片
      // 而是等待后端通过 anki_generation_event 推送卡片
      // 前端的职责是监听事件并收集卡片

      // 等待后端处理完成（通过事件监听）
      const cards = await this.waitForTaskCompletion(
        taskId,
        context.abortController.signal,
        options?.taskTimeout || this.config.taskTimeout
      );

      // 发送任务完成事件
      this.emitEvent('task:complete', documentId, {
        taskId,
        segmentIndex,
        cardsGenerated: cards?.length || 0,
        durationMs: Date.now() - context.startTime,
      });

      return cards;
    } catch (error: unknown) {
      // 检查是否是中止错误
      if (error instanceof Error && error.name === 'AbortError') {
        console.log(`⏸️ 任务 ${taskId} 被中止`);
        return null;
      }

      // 其他错误
      console.error(`❌ 任务 ${taskId} 失败:`, error);
      this.emitEvent('task:error', documentId, {
        taskId,
        segmentIndex,
        error: error instanceof Error ? error.message : String(error),
      });

      return null;
    } finally {
      // 清理任务上下文
      this.runningTasks.delete(taskId);
    }
  }

  /**
   * 等待任务完成
   *
   * 使用 Promise-based 事件等待机制，解决竞态条件问题：
   * 1. 先检查任务是否已完成（事件可能在 Promise 创建前到达）
   * 2. 注册回调等待任务完成事件
   * 3. 支持超时和中止
   */
  private async waitForTaskCompletion(
    taskId: string,
    signal: AbortSignal,
    timeout: number
  ): Promise<AnkiCardResult[] | null> {
    return new Promise((resolve, reject) => {
      let resolved = false;
      let timeoutId: ReturnType<typeof setTimeout> | null = null;
      let abortHandler: (() => void) | null = null;

      const cleanup = () => {
        if (timeoutId) {
          clearTimeout(timeoutId);
          timeoutId = null;
        }
        // 移除 abort 监听器，防止内存泄漏
        if (abortHandler) {
          signal.removeEventListener('abort', abortHandler);
          abortHandler = null;
        }
        this.taskCompletionCallbacks.delete(taskId);
      };

      const resolveWith = (cards: AnkiCardResult[]) => {
        if (resolved) return;
        resolved = true;
        cleanup();
        resolve(cards);
      };

      const rejectWith = (error: Error) => {
        if (resolved) return;
        resolved = true;
        cleanup();
        reject(error);
      };

      // 1. 检查是否已中止
      if (signal.aborted) {
        rejectWith(new Error('Task aborted'));
        return;
      }

      // 2. 检查任务是否已完成（解决竞态条件：事件在 Promise 创建前到达）
      if (this.completedTasks.has(taskId)) {
        const cards = this.generatedCards.get(taskId) || [];
        resolveWith(cards);
        return;
      }

      // 3. 检查是否已有卡片且任务已完成
      const existingCards = this.generatedCards.get(taskId);
      if (existingCards && existingCards.length > 0 && this.completedTasks.has(taskId)) {
        resolveWith(existingCards);
        return;
      }

      // 4. 注册完成回调（事件驱动，无需轮询）
      this.taskCompletionCallbacks.set(taskId, (cards: AnkiCardResult[]) => {
        resolveWith(cards);
      });

      // 5. 设置超时
      timeoutId = setTimeout(() => {
        // 超时时返回已收集的卡片（可能部分完成）
        const partialCards = this.generatedCards.get(taskId) || [];
        console.warn(`⚠️ 任务 ${taskId} 超时 (${timeout}ms)，已收集 ${partialCards.length} 张卡片`);
        resolveWith(partialCards);
      }, timeout);

      // 6. 监听中止信号（保存引用以便 cleanup 时移除）
      abortHandler = () => {
        const abortError = new Error('Task aborted');
        abortError.name = 'AbortError';
        rejectWith(abortError);
      };
      signal.addEventListener('abort', abortHandler);
    });
  }

  /**
   * 标记任务完成并触发回调
   */
  private markTaskCompleted(taskId: string): void {
    this.completedTasks.add(taskId);

    // 触发等待中的回调
    const callback = this.taskCompletionCallbacks.get(taskId);
    if (callback) {
      const cards = this.generatedCards.get(taskId) || [];
      callback(cards);
      this.taskCompletionCallbacks.delete(taskId);
    }
  }

  /**
   * 监听后端事件流
   * 使用 nativeEvents 的 listen 监听 'anki_generation_event'
   *
   * 注意：此方法现在是异步的，必须等待 listen() 完成后才能开始处理任务
   * 否则可能丢失后端在监听器设置期间发送的事件
   */
  private async setupBackendEventListener(documentId: string): Promise<void> {
    try {
      // 使用统一 native event facade，等待监听器设置完成
      this.eventUnlistener = await listen<unknown>('anki_generation_event', (tauriEvent) => {
        try {
          const rawPayload = (tauriEvent as { payload?: unknown }).payload;
          const payload =
            rawPayload && typeof rawPayload === 'object' && 'payload' in (rawPayload as Record<string, unknown>)
              ? (rawPayload as { payload: unknown }).payload
              : rawPayload;
          if (!payload || typeof payload !== 'object') {
            return;
          }

          const streamPayload = payload as Record<string, unknown>;

          // 处理不同类型的事件
          if ('NewCard' in streamPayload) {
            this.handleNewCardEvent(documentId, streamPayload.NewCard);
          } else if ('NewErrorCard' in streamPayload) {
            this.handleNewCardEvent(documentId, streamPayload.NewErrorCard, true);
          } else if ('TaskStatusUpdate' in streamPayload) {
            this.handleTaskStatusEvent(documentId, streamPayload.TaskStatusUpdate);
          } else if ('TaskProcessingError' in streamPayload) {
            this.handleTaskProcessingError(documentId, streamPayload.TaskProcessingError);
          } else if ('TaskCompleted' in streamPayload) {
            // 任务完成事件 - 标记任务完成，触发等待回调
            const taskCompleted = streamPayload.TaskCompleted as { task_id?: string } | undefined;
            if (taskCompleted?.task_id) {
              this.markTaskCompleted(taskCompleted.task_id);
            }
          } else if ('DocumentProcessingCompleted' in streamPayload) {
            this.handleDocumentCompleteEvent(documentId, streamPayload.DocumentProcessingCompleted);
          } else if ('DocumentProcessingPaused' in streamPayload) {
            // ★ 2026-01 修复：处理文档暂停事件
            this.handleDocumentPausedEvent(documentId);
          } else if ('DocumentProcessingStarted' in streamPayload) {
            // 文档开始处理事件 - 记录日志
            const startEvent = streamPayload.DocumentProcessingStarted as { total_segments?: number } | undefined;
            console.log('📄 文档开始处理:', { documentId, totalSegments: startEvent?.total_segments });
          }
        } catch (error: unknown) {
          console.error('❌ 处理后端事件失败:', error);
        }
      });
    } catch (error: unknown) {
      console.error('❌ 监听后端事件失败:', error);
      throw error; // 传播错误，让调用方知道监听失败
    }
  }

  /**
   * 清理后端事件监听器
   */
  private cleanupBackendEventListener(): void {
    if (this.eventUnlistener) {
      this.eventUnlistener();
      this.eventUnlistener = null;
    }
  }

  /**
   * 处理新卡片事件
   */
  private handleNewCardEvent(documentId: string, event: any, isErrorEvent = false): void {
    const rawCard = event?.card ?? event;
    const taskId = event?.task_id ?? rawCard?.task_id;

    if (!rawCard || !taskId) {
      return;
    }

    const extraFields = rawCard.extra_fields || rawCard.extraFields || rawCard.fields || {};

    // 转换后端卡片格式为前端格式
    const ankiCard: AnkiCardResult = {
      id: rawCard.id || `${taskId}_${Date.now()}`,
      taskId,
      templateId: rawCard.template_id || rawCard.templateId || 'unknown',
      front: rawCard.front || '',
      back: rawCard.back || '',
      text: rawCard.text,
      tags: rawCard.tags || [],
      fields: extraFields,
      images: rawCard.images || [],
      isErrorCard: rawCard.is_error_card ?? rawCard.isErrorCard ?? isErrorEvent,
      errorContent: rawCard.error_content ?? rawCard.errorContent,
      createdAt: rawCard.created_at || rawCard.createdAt || new Date().toISOString(),
      confidence: rawCard.confidence,
      metadata: rawCard.metadata,
    };

    // 存储卡片
    if (!this.generatedCards.has(taskId)) {
      this.generatedCards.set(taskId, []);
    }
    this.generatedCards.get(taskId)!.push(ankiCard);

    // 发送卡片生成事件
    const payload: CardGeneratedPayload = {
      card: ankiCard,
      taskId,
      segmentIndex: this.getSegmentIndexFromTaskId(taskId),
    };
    this.emitEvent('card:generated', documentId, payload);
  }

  /**
   * 处理任务状态更新事件
   */
  private handleTaskStatusEvent(documentId: string, event: any): void {
    const { task_id, status, error_message, message } = event || {};

    const segmentIndex = this.getSegmentIndexFromTaskId(task_id);
    const mappedStatus = this.mapBackendStatus(status);
    const errorMsg = error_message || message;

    if (errorMsg) {
      // 发送错误事件
      this.emitEvent('task:error', documentId, {
        taskId: task_id,
        segmentIndex,
        error: errorMsg,
      });
    } else {
      // 发送进度事件
      const payload: TaskProgressPayload = {
        taskId: task_id,
        segmentIndex,
        status: mappedStatus,
        progress: 0, // 后端未提供进度信息
        cardsGenerated: this.generatedCards.get(task_id)?.length || 0,
      };
      this.emitEvent('task:progress', documentId, payload);
    }

    if (['failed', 'truncated', 'cancelled'].includes(mappedStatus)) {
      this.markTaskCompleted(task_id);
    }
  }

  private handleTaskProcessingError(documentId: string, event: any): void {
    const { task_id, error_message } = event || {};
    if (!task_id) return;

    const segmentIndex = this.getSegmentIndexFromTaskId(task_id);
    this.emitEvent('task:error', documentId, {
      taskId: task_id,
      segmentIndex,
      error: error_message || '任务处理失败',
    });
    this.markTaskCompleted(task_id);
  }

  /**
   * 处理文档完成事件
   */
  private handleDocumentCompleteEvent(documentId: string, event: any): void {
    // 文档完成事件由主流程处理，这里只记录日志
    console.log('📄 文档处理完成:', event);
  }

  /**
   * ★ 2026-01 修复：处理文档暂停事件
   */
  private handleDocumentPausedEvent(documentId: string): void {
    console.log('⏸️ 文档处理已暂停:', documentId);
    this.isPaused = true;
    // 计算已生成卡片总数
    const totalCards = Array.from(this.generatedCards.values())
      .reduce((sum, cards) => sum + cards.length, 0);
    // 发送暂停事件，通知上层组件
    this.emitEvent('document:paused', documentId, {
      cardsGenerated: totalCards,
    });
  }

  /**
   * 从 task_id 提取 segment_index
   */
  private getSegmentIndexFromTaskId(taskId: string): number {
    const match = taskId.match(/segment_(\d+)/);
    return match ? parseInt(match[1], 10) : 0;
  }

  /**
   * 映射后端状态到前端状态
   */
  private mapBackendStatus(backendStatus: string): TaskStatus {
    const statusMap: Record<string, TaskStatus> = {
      'Pending': 'pending',
      'Processing': 'processing',
      'Streaming': 'streaming',
      'Paused': 'paused',
      'Completed': 'completed',
      'Failed': 'failed',
      'Truncated': 'truncated',
      'Cancelled': 'cancelled',
    };

    return statusMap[backendStatus] || 'pending';
  }

  /**
   * 发送事件
   */
  private emitEvent<T = unknown>(
    type: CardForgeEventType,
    documentId: string,
    payload: T
  ): void {
    const event: CardForgeEvent<T> = {
      type,
      documentId,
      timestamp: new Date().toISOString(),
      payload,
    };

    const listeners = this.listeners.get(type);
    if (listeners) {
      for (const listener of listeners) {
        try {
          listener(event);
        } catch (error: unknown) {
          console.error(`❌ 事件监听器执行失败 (${type}):`, error);
        }
      }
    }
  }

  /**
   * 延迟函数
   */
  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}
