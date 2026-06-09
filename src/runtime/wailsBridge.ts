import * as SettingsService from './wails-bindings/deep-student-go/internal/bindings/settingsservice';
import * as SystemService from './wails-bindings/deep-student-go/internal/bindings/systemservice';
import * as AnkiService from './wails-bindings/deep-student-go/internal/bindings/ankiservice';
import * as ChatService from './wails-bindings/deep-student-go/internal/bindings/chatservice';
import * as DstuService from './wails-bindings/deep-student-go/internal/bindings/dstuservice';
import * as FileService from './wails-bindings/deep-student-go/internal/bindings/fileservice';
import * as McpService from './wails-bindings/deep-student-go/internal/bindings/mcpservice';
import * as NotesService from './wails-bindings/deep-student-go/internal/bindings/notesservice';
import * as QbankService from './wails-bindings/deep-student-go/internal/bindings/qbankservice';
import * as ReviewPlanService from './wails-bindings/deep-student-go/internal/bindings/reviewplanservice';
import * as TemplateService from './wails-bindings/deep-student-go/internal/bindings/templateservice';
import * as TodoService from './wails-bindings/deep-student-go/internal/bindings/todoservice';
import * as VfsService from './wails-bindings/deep-student-go/internal/bindings/vfsservice';
import * as SkillService from './wails-bindings/deep-student-go/internal/bindings/skillservice';

type NativeArgs = Record<string, unknown> | undefined;

declare global {
  interface Window {
    _wails?: {
      environment?: unknown;
      flags?: unknown;
      invoke?: unknown;
    };
    wails?: {
      invoke?: unknown;
    };
    chrome?: {
      webview?: {
        postMessage?: unknown;
      };
    };
    webkit?: {
      messageHandlers?: {
        external?: {
          postMessage?: unknown;
        };
      };
    };
    __DEEP_STUDENT_WAILS__?: unknown;
  }
}

export const isWailsRuntime = (): boolean => {
  if (typeof window === 'undefined') return false;
  return Boolean(
    window.__DEEP_STUDENT_WAILS__ ||
      window._wails?.environment ||
      window._wails?.flags ||
      typeof window.wails?.invoke === 'function' ||
      typeof window.chrome?.webview?.postMessage === 'function' ||
      typeof window.webkit?.messageHandlers?.external?.postMessage === 'function'
  );
};

function requireStringArg(command: string, args: NativeArgs, key: string): string {
  const value = args?.[key];
  if (typeof value !== 'string') {
    throw new Error(`Wails command ${command} requires string argument "${key}"`);
  }
  return value;
}

function byteSliceToNumbers(value: unknown): number[] {
  if (Array.isArray(value)) {
    return value.map(item => Number(item) & 0xff);
  }
  if (typeof value !== 'string' || value.length === 0) {
    return [];
  }

  try {
    const binary =
      typeof atob === 'function'
        ? atob(value)
        : Buffer.from(value, 'base64').toString('binary');
    return Array.from(binary, char => char.charCodeAt(0));
  } catch {
    return Array.from(value, char => char.charCodeAt(0) & 0xff);
  }
}

function requireObjectArg(command: string, args: NativeArgs, key: string): Record<string, unknown> {
  const value = args?.[key];
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`Wails command ${command} requires object argument "${key}"`);
  }
  return value as Record<string, unknown>;
}

function optionalStringArg(args: NativeArgs, key: string): string | undefined {
  const value = args?.[key];
  return typeof value === 'string' ? value : undefined;
}

function optionalBooleanArg(args: NativeArgs, key: string): boolean {
  const value = args?.[key];
  return typeof value === 'boolean' ? value : false;
}

function optionalNumberArg(args: NativeArgs, key: string, fallback: number): number {
  const value = args?.[key];
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback;
}

function optionalNumberArrayArg(args: NativeArgs, key: string): number[] {
  const value = args?.[key];
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item): item is number => typeof item === 'number' && Number.isFinite(item));
}

function optionalStringRecordArg(args: NativeArgs, key: string): Record<string, string> {
  const value = args?.[key];
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return {};
  }
  return Object.fromEntries(
    Object.entries(value as Record<string, unknown>).filter(
      (entry): entry is [string, string] => typeof entry[1] === 'string'
    )
  );
}

function requireStringArrayArg(command: string, args: NativeArgs, key: string): string[] {
  const value = args?.[key];
  if (!Array.isArray(value) || value.some(item => typeof item !== 'string')) {
    throw new Error(`Wails command ${command} requires string[] argument "${key}"`);
  }
  return value;
}

function requireStringArgAny(command: string, args: NativeArgs, keys: string[]): string {
  for (const key of keys) {
    const value = args?.[key];
    if (typeof value === 'string') {
      return value;
    }
  }
  throw new Error(`Wails command ${command} requires one of string arguments: ${keys.join(', ')}`);
}

function optionalStringArgAny(args: NativeArgs, keys: string[]): string | undefined {
  for (const key of keys) {
    const value = args?.[key];
    if (typeof value === 'string') {
      return value;
    }
  }
  return undefined;
}

function optionalBooleanArgAny(args: NativeArgs, keys: string[]): boolean | undefined {
  for (const key of keys) {
    const value = args?.[key];
    if (typeof value === 'boolean') {
      return value;
    }
  }
  return undefined;
}

export async function invokeWails<T>(command: string, args?: NativeArgs): Promise<T> {
  if (command === 'check_anki_connect_status') {
    return await AnkiService.CheckConnectStatus() as T;
  }

  if (command === 'get_anki_deck_names' || command === 'anki_get_deck_names') {
    return await AnkiService.ListDeckNames() as T;
  }

  if (command === 'get_anki_model_names') {
    return await AnkiService.ListModelNames() as T;
  }

  if (command === 'save_anki_cards') {
    const request = requireObjectArg(command, args, 'request');
    const result = await AnkiService.SaveAnkiCards(request as any) as any;
    return {
      ...result,
      savedIds: Array.isArray(result?.saved_ids) ? result.saved_ids : result?.savedIds,
      taskId: typeof result?.task_id === 'string' ? result.task_id : result?.taskId,
    } as T;
  }

  if (command === 'start_enhanced_document_processing') {
    const documentContent = requireStringArgAny(command, args, ['documentContent', 'document_content']);
    const originalDocumentName =
      optionalStringArgAny(args, ['originalDocumentName', 'original_document_name']) ?? 'Document';
    const options = args?.options && typeof args.options === 'object' && !Array.isArray(args.options)
      ? args.options as Record<string, unknown>
      : {};
    return await AnkiService.StartEnhancedDocumentProcessing(documentContent, originalDocumentName, options) as T;
  }

  if (command === 'get_document_tasks') {
    const documentId = requireStringArgAny(command, args, ['documentId', 'document_id']);
    return await AnkiService.GetDocumentTasks(documentId) as T;
  }

  if (command === 'pause_document_processing') {
    const documentId = requireStringArgAny(command, args, ['documentId', 'document_id']);
    return await AnkiService.PauseDocumentProcessing(documentId) as T;
  }

  if (command === 'resume_document_processing') {
    const documentId = requireStringArgAny(command, args, ['documentId', 'document_id']);
    return await AnkiService.ResumeDocumentProcessing(documentId) as T;
  }

  if (command === 'get_document_processing_state') {
    const documentId = requireStringArgAny(command, args, ['documentId', 'document_id']);
    return await AnkiService.GetDocumentProcessingState(documentId) as T;
  }

  if (command === 'get_document_state') {
    const documentId = requireStringArgAny(command, args, ['documentId', 'document_id']);
    return await AnkiService.GetDocumentState(documentId) as T;
  }

  if (command === 'get_document_task_counts') {
    const documentId = requireStringArgAny(command, args, ['documentId', 'document_id']);
    return await AnkiService.GetDocumentTaskCounts(documentId) as T;
  }

  if (command === 'trigger_task_processing') {
    const taskId = requireStringArgAny(command, args, ['taskId', 'task_id']);
    return await AnkiService.TriggerTaskProcessing(taskId) as T;
  }

  if (command === 'delete_document_session') {
    const documentId = requireStringArgAny(command, args, ['documentId', 'document_id']);
    return await AnkiService.DeleteDocumentSession(documentId) as T;
  }

  if (command === 'get_document_cards') {
    const documentId = requireStringArgAny(command, args, ['documentId', 'document_id']);
    return await AnkiService.GetDocumentCards(documentId) as T;
  }

  if (command === 'recover_stuck_document_tasks') {
    return await AnkiService.RecoverStuckDocumentTasks() as T;
  }

  if (command === 'chat_v2_create_session') {
    const mode = requireStringArg(command, args, 'mode');
    const title = optionalStringArg(args, 'title') ?? null;
    const metadata = args?.metadata && typeof args.metadata === 'object' && !Array.isArray(args.metadata)
      ? args.metadata as Record<string, unknown>
      : null;
    const groupId = optionalStringArgAny(args, ['groupId', 'group_id']) ?? null;
    return await ChatService.CreateSession(mode, title, metadata, groupId) as T;
  }

  if (command === 'chat_v2_get_session') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    return await ChatService.GetSession(sessionId) as T;
  }

  if (command === 'chat_v2_load_session') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    return await ChatService.LoadSession(sessionId) as T;
  }

  if (command === 'chat_v2_save_session') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    return await ChatService.SaveSession(sessionId, requireObjectArg(command, args, 'sessionState') as any) as T;
  }

  if (command === 'chat_v2_update_session_settings') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    return await ChatService.UpdateSessionSettings(sessionId, requireObjectArg(command, args, 'settings') as any) as T;
  }

  if (command === 'chat_v2_archive_session') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    return await ChatService.ArchiveSession(sessionId) as T;
  }

  if (command === 'chat_v2_restore_session') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    return await ChatService.RestoreSession(sessionId) as T;
  }

  if (command === 'chat_v2_delete_session') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    return await ChatService.DeleteSession(sessionId) as T;
  }

  if (command === 'chat_v2_move_session_to_group') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    const groupId = optionalStringArgAny(args, ['groupId', 'group_id']) ?? null;
    return await ChatService.MoveSessionToGroup(sessionId, groupId) as T;
  }

  if (command === 'chat_v2_list_sessions') {
    const status = optionalStringArg(args, 'status') ?? null;
    const groupId = optionalStringArgAny(args, ['groupId', 'group_id']) ?? null;
    const limit = typeof args?.limit === 'number' ? args.limit : 50;
    const offset = typeof args?.offset === 'number' ? args.offset : 0;
    return await ChatService.ListSessions(status, groupId, limit, offset) as T;
  }

  if (command === 'chat_v2_count_sessions') {
    const status = optionalStringArg(args, 'status') ?? null;
    const groupId = optionalStringArgAny(args, ['groupId', 'group_id']) ?? null;
    return await ChatService.CountSessions(status, groupId) as T;
  }

  if (command === 'chat_v2_create_group') {
    return await ChatService.CreateGroup(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'chat_v2_get_group') {
    const groupId = requireStringArg(command, args, 'groupId');
    return await ChatService.GetGroup(groupId) as T;
  }

  if (command === 'chat_v2_update_group') {
    const groupId = requireStringArg(command, args, 'groupId');
    return await ChatService.UpdateGroup(groupId, requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'chat_v2_delete_group') {
    const groupId = requireStringArg(command, args, 'groupId');
    return await ChatService.DeleteGroup(groupId) as T;
  }

  if (command === 'chat_v2_restore_group') {
    const groupId = requireStringArg(command, args, 'groupId');
    return await ChatService.RestoreGroup(groupId) as T;
  }

  if (command === 'chat_v2_list_groups') {
    const status = optionalStringArg(args, 'status') ?? null;
    const workspaceId = optionalStringArg(args, 'workspaceId') ?? null;
    return await ChatService.ListGroups(status, workspaceId) as T;
  }

  if (command === 'chat_v2_reorder_groups') {
    const groupIds = args?.groupIds;
    if (!Array.isArray(groupIds) || groupIds.some(id => typeof id !== 'string')) {
      throw new Error('Wails command chat_v2_reorder_groups requires string[] argument "groupIds"');
    }
    return await ChatService.ReorderGroups(groupIds) as T;
  }

  if (command === 'chat_v2_get_session_tags') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    return await ChatService.GetSessionTags(sessionId) as T;
  }

  if (command === 'chat_v2_get_tags_batch') {
    const sessionIds = args?.sessionIds;
    if (!Array.isArray(sessionIds) || sessionIds.some(id => typeof id !== 'string')) {
      throw new Error('Wails command chat_v2_get_tags_batch requires string[] argument "sessionIds"');
    }
    return await ChatService.GetTagsBatch(sessionIds) as T;
  }

  if (command === 'chat_v2_add_tag') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    const tag = requireStringArg(command, args, 'tag');
    return await ChatService.AddTag(sessionId, tag) as T;
  }

  if (command === 'chat_v2_remove_tag') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    const tag = requireStringArg(command, args, 'tag');
    return await ChatService.RemoveTag(sessionId, tag) as T;
  }

  if (command === 'chat_v2_list_all_tags') {
    return await ChatService.ListAllTags() as T;
  }

  if (command === 'chat_v2_get_message_summary') {
    return await ChatService.GetMessageSummary() as T;
  }

  if (command === 'llm_usage_get_trends') {
    return await ChatService.LLMUsageGetTrends(
      optionalNumberArg(args, 'days', 30),
      optionalStringArg(args, 'granularity') ?? 'day',
    ) as T;
  }

  if (command === 'llm_usage_by_model') {
    return await ChatService.LLMUsageByModel(
      requireStringArg(command, args, 'startDate'),
      requireStringArg(command, args, 'endDate'),
    ) as T;
  }

  if (command === 'llm_usage_by_caller') {
    return await ChatService.LLMUsageByCaller(
      requireStringArg(command, args, 'startDate'),
      requireStringArg(command, args, 'endDate'),
    ) as T;
  }

  if (command === 'llm_usage_summary') {
    return await ChatService.LLMUsageSummary(
      optionalStringArg(args, 'startDate') ?? null,
      optionalStringArg(args, 'endDate') ?? null,
    ) as T;
  }

  if (command === 'llm_usage_recent') {
    return await ChatService.LLMUsageRecent(optionalNumberArg(args, 'limit', 20)) as T;
  }

  if (command === 'llm_usage_daily') {
    return await ChatService.LLMUsageDaily(
      requireStringArg(command, args, 'startDate'),
      requireStringArg(command, args, 'endDate'),
    ) as T;
  }

  if (command === 'llm_usage_cleanup') {
    return await ChatService.LLMUsageCleanup(requireStringArg(command, args, 'beforeDate')) as T;
  }

  if (command === 'chat_v2_branch_session') {
    const sourceSessionId = requireStringArg(command, args, 'sourceSessionId');
    const upToMessageId = requireStringArg(command, args, 'upToMessageId');
    return await ChatService.BranchSession(sourceSessionId, upToMessageId) as T;
  }

  if (command === 'chat_v2_send_message') {
    return await ChatService.SendMessage(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'chat_v2_continue_message') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    const messageId = requireStringArg(command, args, 'messageId');
    const options = args?.options && typeof args.options === 'object' && !Array.isArray(args.options)
      ? args.options as Record<string, unknown>
      : {};
    return await ChatService.ContinueMessage(sessionId, messageId, options) as T;
  }

  if (command === 'chat_v2_cancel_stream') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    const messageId = requireStringArg(command, args, 'messageId');
    return await ChatService.CancelStream(sessionId, messageId) as T;
  }

  if (command === 'chat_v2_retry_message') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    const messageId = requireStringArg(command, args, 'messageId');
    const options = args?.options && typeof args.options === 'object' && !Array.isArray(args.options)
      ? args.options as Record<string, unknown>
      : {};
    return await ChatService.RetryMessage(sessionId, messageId, options) as T;
  }

  if (command === 'chat_v2_edit_and_resend') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    const messageId = requireStringArg(command, args, 'messageId');
    const newContent = requireStringArg(command, args, 'newContent');
    const newContextRefs = Array.isArray(args?.newContextRefs) ? args.newContextRefs : [];
    const newPathMap = args?.newPathMap && typeof args.newPathMap === 'object' && !Array.isArray(args.newPathMap)
      ? args.newPathMap as Record<string, string>
      : {};
    const options = args?.options && typeof args.options === 'object' && !Array.isArray(args.options)
      ? args.options as Record<string, unknown>
      : {};
    return await ChatService.EditAndResend({
      sessionId,
      messageId,
      newContent,
      assistantMessageId: optionalStringArg(args, 'assistantMessageId') ?? undefined,
      newContextRefs,
      newPathMap,
      options,
    } as any) as T;
  }

  if (command === 'chat_v2_tool_approval_respond') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    const toolCallId = requireStringArg(command, args, 'toolCallId');
    const toolName = requireStringArg(command, args, 'toolName');
    const approved = args?.approved;
    if (typeof approved !== 'boolean') {
      throw new Error('Wails command chat_v2_tool_approval_respond requires boolean argument "approved"');
    }
    const reason = optionalStringArg(args, 'reason') ?? null;
    const remember = optionalBooleanArg(args, 'remember');
    const toolArguments = args?.arguments && typeof args.arguments === 'object' && !Array.isArray(args.arguments)
      ? args.arguments as Record<string, unknown>
      : {};
    return await ChatService.RespondToolApproval(sessionId, toolCallId, toolName, approved, reason, remember, toolArguments) as T;
  }

  if (command === 'chat_v2_clear_approval_history') {
    return await ChatService.ClearApprovalHistory() as T;
  }

  if (command === 'chat_v2_ask_user_respond') {
    const toolCallId = requireStringArg(command, args, 'toolCallId');
    const selectedTexts = args?.selectedTexts;
    if (!Array.isArray(selectedTexts) || selectedTexts.some(text => typeof text !== 'string')) {
      throw new Error('Wails command chat_v2_ask_user_respond requires string[] argument "selectedTexts"');
    }
    const selectedIndices = args?.selectedIndices;
    if (!Array.isArray(selectedIndices) || selectedIndices.some(index => typeof index !== 'number')) {
      throw new Error('Wails command chat_v2_ask_user_respond requires number[] argument "selectedIndices"');
    }
    const customText = optionalStringArg(args, 'customText') ?? null;
    const source = requireStringArg(command, args, 'source');
    return await ChatService.RespondAskUser(toolCallId, selectedTexts, selectedIndices, customText, source) as T;
  }

  if (command === 'chat_v2_delete_message') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    const messageId = requireStringArg(command, args, 'messageId');
    return await ChatService.DeleteMessage(sessionId, messageId) as T;
  }

  if (command === 'chat_v2_update_block_content') {
    const blockId = requireStringArg(command, args, 'blockId');
    const content = requireStringArg(command, args, 'content');
    return await ChatService.UpdateBlockContent(blockId, content) as T;
  }

  if (command === 'chat_v2_upsert_streaming_block') {
    const blockId = requireStringArg(command, args, 'blockId');
    const messageId = requireStringArg(command, args, 'messageId');
    const blockType = requireStringArg(command, args, 'blockType');
    const content = requireStringArg(command, args, 'content');
    const sessionId = optionalStringArg(args, 'sessionId') ?? null;
    return await ChatService.UpsertStreamingBlock(blockId, messageId, blockType, content, sessionId) as T;
  }

  if (command === 'dstu_list') {
    const path = requireStringArg(command, args, 'path');
    return await DstuService.List(path, args?.options as any ?? null) as T;
  }

  if (command === 'dstu_get') {
    const path = requireStringArg(command, args, 'path');
    return await DstuService.Get(path) as T;
  }

  if (command === 'dstu_create') {
    const path = requireStringArg(command, args, 'path');
    return await DstuService.Create(path, requireObjectArg(command, args, 'options') as any) as T;
  }

  if (command === 'dstu_update') {
    const path = requireStringArg(command, args, 'path');
    const content = requireStringArg(command, args, 'content');
    const resourceType = requireStringArg(command, args, 'resourceType');
    return await DstuService.Update(path, content, resourceType) as T;
  }

  if (command === 'dstu_delete') {
    const path = requireStringArg(command, args, 'path');
    return await DstuService.Delete(path) as T;
  }

  if (command === 'dstu_delete_many') {
    return await DstuService.DeleteMany(requireStringArrayArg(command, args, 'paths')) as T;
  }

  if (command === 'dstu_search') {
    const query = requireStringArg(command, args, 'query');
    return await DstuService.Search(query, args?.options as any ?? null) as T;
  }

  if (command === 'dstu_get_content') {
    const path = requireStringArg(command, args, 'path');
    return await DstuService.GetContent(path) as T;
  }

  if (command === 'notes_search') {
    const keyword = requireStringArg(command, args, 'keyword');
    return await DstuService.NotesSearch(keyword, optionalNumberArg(args, 'limit', 50)) as T;
  }

  if (command === 'notes_mentions_search') {
    const keyword = requireStringArg(command, args, 'keyword');
    const subject = optionalStringArg(args, 'subject') ?? null;
    return await QbankService.NotesMentionsSearch(keyword, subject, optionalNumberArg(args, 'limit', 8)) as T;
  }

  if (command === 'notes_list_tags') {
    return await DstuService.ListTags() as T;
  }

  if (command === 'notes_list_deleted') {
    const page = typeof args?.page === 'number' ? Math.max(0, args.page) : 0;
    const pageSize = typeof args?.page_size === 'number'
      ? Math.max(1, args.page_size)
      : typeof args?.pageSize === 'number'
        ? Math.max(1, args.pageSize)
        : 20;
    const all = await DstuService.ListDeleted('notes', null, null) as any[];
    const items = all.slice(page * pageSize, page * pageSize + pageSize).map(node => ({
      id: node.id,
      title: node.name ?? '',
      content_md: '',
      tags: Array.isArray(node.metadata?.tags) ? node.metadata.tags : [],
      created_at: String(node.createdAt ?? ''),
      updated_at: String(node.updatedAt ?? ''),
      is_favorite: Boolean(node.metadata?.isFavorite),
    }));
    return { items, total: all.length, page, page_size: pageSize } as T;
  }

  if (command === 'notes_empty_trash') {
    return await DstuService.PurgeAll('notes') as T;
  }

  if (command === 'notes_hard_delete') {
    const id = requireStringArg(command, args, 'id');
    await DstuService.Purge('/' + id);
    return true as T;
  }

  if (command === 'notes_restore') {
    const id = requireStringArg(command, args, 'id');
    await DstuService.Restore('/' + id);
    return true as T;
  }

  if (command === 'canvas_note_read') {
    const noteId = requireStringArgAny(command, args, ['noteId', 'note_id']);
    return await DstuService.CanvasReadContent(noteId, optionalStringArg(args, 'section') ?? null) as T;
  }

  if (command === 'canvas_note_append') {
    const noteId = requireStringArgAny(command, args, ['noteId', 'note_id']);
    const content = requireStringArg(command, args, 'content');
    return await DstuService.CanvasAppendContent(noteId, content, optionalStringArg(args, 'section') ?? null) as T;
  }

  if (command === 'canvas_note_replace') {
    const noteId = requireStringArgAny(command, args, ['noteId', 'note_id']);
    const search = requireStringArg(command, args, 'search');
    const replace = requireStringArg(command, args, 'replace');
    return await DstuService.CanvasReplaceContent(noteId, search, replace, optionalBooleanArg(args, 'isRegex')) as T;
  }

  if (command === 'canvas_note_set') {
    const noteId = requireStringArgAny(command, args, ['noteId', 'note_id']);
    const content = requireStringArg(command, args, 'content');
    return await DstuService.CanvasSetContent(noteId, content) as T;
  }

  if (command === 'dstu_set_metadata') {
    const path = requireStringArg(command, args, 'path');
    return await DstuService.SetMetadata(path, requireObjectArg(command, args, 'metadata') as any) as T;
  }

  if (command === 'dstu_set_favorite') {
    const path = requireStringArg(command, args, 'path');
    const favorite = args?.favorite;
    if (typeof favorite !== 'boolean') {
      throw new Error('Wails command dstu_set_favorite requires boolean argument "favorite"');
    }
    return await DstuService.SetFavorite(path, favorite) as T;
  }

  if (command === 'notes_import_markdown') {
    return await DstuService.ImportMarkdown(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'notes_import_markdown_batch') {
    return await DstuService.ImportMarkdownBatch(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'textbooks_add') {
    const sources = args?.sources;
    if (!Array.isArray(sources) || sources.some(source => typeof source !== 'string')) {
      throw new Error('Wails command textbooks_add requires string[] argument "sources"');
    }
    return await DstuService.AddTextbooks({
      sources,
      folderId: optionalStringArg(args, 'folderId') ?? null,
    } as any) as T;
  }

  if (command === 'dstu_folder_create') {
    const title = requireStringArg(command, args, 'title');
    return await DstuService.CreateFolder(
      title,
      optionalStringArg(args, 'parentId') ?? null,
      optionalStringArg(args, 'icon') ?? null,
      optionalStringArg(args, 'color') ?? null,
    ) as T;
  }

  if (command === 'dstu_folder_get') {
    const folderId = requireStringArg(command, args, 'folderId');
    return await DstuService.GetFolder(folderId) as T;
  }

  if (command === 'dstu_folder_rename') {
    const folderId = requireStringArg(command, args, 'folderId');
    const title = requireStringArg(command, args, 'title');
    return await DstuService.RenameFolder(folderId, title) as T;
  }

  if (command === 'dstu_folder_delete') {
    const folderId = requireStringArg(command, args, 'folderId');
    return await DstuService.DeleteFolder(folderId) as T;
  }

  if (command === 'dstu_folder_move') {
    const folderId = requireStringArg(command, args, 'folderId');
    return await DstuService.MoveFolder(folderId, optionalStringArg(args, 'newParentId') ?? null) as T;
  }

  if (command === 'dstu_folder_set_expanded') {
    const folderId = requireStringArg(command, args, 'folderId');
    const isExpanded = args?.isExpanded;
    if (typeof isExpanded !== 'boolean') {
      throw new Error('Wails command dstu_folder_set_expanded requires boolean argument "isExpanded"');
    }
    return await DstuService.SetFolderExpanded(folderId, isExpanded) as T;
  }

  if (command === 'dstu_folder_add_item') {
    const itemType = requireStringArg(command, args, 'itemType');
    const itemId = requireStringArg(command, args, 'itemId');
    return await DstuService.AddFolderItem(optionalStringArg(args, 'folderId') ?? null, itemType, itemId) as T;
  }

  if (command === 'dstu_folder_remove_item') {
    const itemType = requireStringArg(command, args, 'itemType');
    const itemId = requireStringArg(command, args, 'itemId');
    return await DstuService.RemoveFolderItem(itemType, itemId) as T;
  }

  if (command === 'dstu_folder_move_item') {
    const itemType = requireStringArg(command, args, 'itemType');
    const itemId = requireStringArg(command, args, 'itemId');
    return await DstuService.MoveFolderItem(itemType, itemId, optionalStringArg(args, 'newFolderId') ?? null) as T;
  }

  if (command === 'dstu_folder_list') {
    return await DstuService.ListFolders() as T;
  }

  if (command === 'dstu_folder_get_tree') {
    return await DstuService.GetFolderTree() as T;
  }

  if (command === 'dstu_folder_get_items') {
    return await DstuService.GetFolderItems(optionalStringArg(args, 'folderId') ?? null) as T;
  }

  if (command === 'dstu_folder_reorder') {
    return await DstuService.ReorderFolders(requireStringArrayArg(command, args, 'folderIds')) as T;
  }

  if (command === 'dstu_folder_reorder_items') {
    return await DstuService.ReorderFolderItems(
      optionalStringArg(args, 'folderId') ?? null,
      requireStringArrayArg(command, args, 'itemIds'),
    ) as T;
  }

  if (command === 'dstu_folder_get_breadcrumbs') {
    const folderId = requireStringArg(command, args, 'folderId');
    return await DstuService.GetFolderBreadcrumbs(folderId) as T;
  }

  if (command === 'dstu_parse_path') {
    const path = requireStringArg(command, args, 'path');
    return await DstuService.ParsePath(path) as T;
  }

  if (command === 'dstu_build_path') {
    const resourceId = requireStringArg(command, args, 'resourceId');
    return await DstuService.BuildPath(optionalStringArg(args, 'folderId') ?? null, resourceId) as T;
  }

  if (command === 'dstu_move_to_folder') {
    const resourceId = requireStringArg(command, args, 'resourceId');
    return await DstuService.MoveToFolder(resourceId, optionalStringArg(args, 'targetFolderId') ?? null) as T;
  }

  if (command === 'dstu_batch_move') {
    return await DstuService.BatchMove(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'dstu_refresh_path_cache') {
    return await DstuService.RefreshPathCache(optionalStringArg(args, 'resourceId') ?? null) as T;
  }

  if (command === 'dstu_get_path_by_id') {
    const resourceId = requireStringArg(command, args, 'resourceId');
    return await DstuService.GetPathByID(resourceId) as T;
  }

  if (command === 'dstu_folder_get_all_resources') {
    const folderId = requireStringArg(command, args, 'folderId');
    return await DstuService.GetFolderAllResources(
      folderId,
      optionalBooleanArg(args, 'includeSubfolders'),
      optionalBooleanArg(args, 'includeContent')
    ) as T;
  }

  if (command === 'dstu_get_resource_by_path') {
    const path = requireStringArg(command, args, 'path');
    return await DstuService.GetResourceByPath(path) as T;
  }

  if (command === 'dstu_get_resource_location') {
    const resourceId = requireStringArg(command, args, 'resourceId');
    return await DstuService.GetResourceLocation(resourceId) as T;
  }

  if (command === 'dstu_restore') {
    const path = requireStringArg(command, args, 'path');
    return await DstuService.Restore(path) as T;
  }

  if (command === 'dstu_restore_many') {
    return await DstuService.RestoreMany(requireStringArrayArg(command, args, 'paths')) as T;
  }

  if (command === 'dstu_purge') {
    const path = requireStringArg(command, args, 'path');
    return await DstuService.Purge(path) as T;
  }

  if (command === 'dstu_purge_all') {
    const resourceType = requireStringArg(command, args, 'resourceType');
    return await DstuService.PurgeAll(resourceType) as T;
  }

  if (command === 'dstu_list_deleted') {
    const resourceType = requireStringArg(command, args, 'resourceType');
    const limit = typeof args?.limit === 'number' ? args.limit : null;
    const offset = typeof args?.offset === 'number' ? args.offset : null;
    return await DstuService.ListDeleted(resourceType, limit, offset) as T;
  }

  if (command === 'dstu_soft_delete') {
    const id = requireStringArg(command, args, 'id');
    const itemType = requireStringArg(command, args, 'itemType');
    return await DstuService.SoftDelete(id, itemType) as T;
  }

  if (command === 'dstu_trash_restore') {
    const id = requireStringArg(command, args, 'id');
    const itemType = requireStringArg(command, args, 'itemType');
    return await DstuService.TrashRestore(id, itemType) as T;
  }

  if (command === 'dstu_list_trash') {
    const limit = typeof args?.limit === 'number' ? args.limit : null;
    const offset = typeof args?.offset === 'number' ? args.offset : null;
    return await DstuService.ListTrash(limit, offset) as T;
  }

  if (command === 'dstu_empty_trash') {
    return await DstuService.EmptyTrash() as T;
  }

  if (command === 'dstu_permanently_delete') {
    const id = requireStringArg(command, args, 'id');
    const itemType = requireStringArg(command, args, 'itemType');
    return await DstuService.PermanentlyDelete(id, itemType) as T;
  }

  if (command === 'vfs_create_or_reuse') {
    return await VfsService.CreateOrReuse(requireObjectArg(command, args, 'params') as any) as T;
  }

  if (command === 'resource_sync_note') {
    const noteId = requireStringArg(command, args, 'noteId');
    return await VfsService.ResourceSyncNote(noteId) as T;
  }

  if (command === 'resource_sync_exam') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    return await VfsService.ResourceSyncExam(sessionId) as T;
  }

  if (command === 'resource_sync_textbook_pages') {
    const textbookId = requireStringArg(command, args, 'textbookId');
    return await VfsService.ResourceSyncTextbookPages(textbookId, optionalNumberArrayArg(args, 'pageRange')) as T;
  }

  if (command === 'resource_check_sync_needed') {
    const resourceType = requireStringArg(command, args, 'resourceType');
    const sourceId = requireStringArg(command, args, 'sourceId');
    return await VfsService.ResourceCheckSyncNeeded(resourceType, sourceId, optionalStringArg(args, 'currentHash') ?? null) as T;
  }

  if (command === 'vfs_get_resource') {
    const resourceId = requireStringArg(command, args, 'resourceId');
    return await VfsService.GetResource(resourceId) as T;
  }

  if (command === 'vfs_resource_exists') {
    const resourceId = requireStringArg(command, args, 'resourceId');
    return await VfsService.ResourceExists(resourceId) as T;
  }

  if (command === 'vfs_increment_ref') {
    const resourceId = requireStringArg(command, args, 'resourceId');
    return await VfsService.IncrementRef(resourceId) as T;
  }

  if (command === 'vfs_decrement_ref') {
    const resourceId = requireStringArg(command, args, 'resourceId');
    return await VfsService.DecrementRef(resourceId) as T;
  }

  if (command === 'vfs_get_resource_path') {
    const sourceId = requireStringArg(command, args, 'sourceId');
    return await VfsService.GetResourcePath(sourceId) as T;
  }

  if (command === 'vfs_get_resource_ref_count') {
    const sourceId = requireStringArg(command, args, 'sourceId');
    return await VfsService.GetResourceRefCount(sourceId) as T;
  }

  if (command === 'vfs_update_resource_hash') {
    const sourceId = requireStringArg(command, args, 'sourceId');
    const newHash = requireStringArg(command, args, 'newHash');
    return await VfsService.UpdateResourceHash(sourceId, newHash) as T;
  }

  if (command === 'vfs_upload_attachment') {
    return await VfsService.UploadAttachment(requireObjectArg(command, args, 'params') as any) as T;
  }

  if (command === 'vfs_get_attachment') {
    const attachmentId = requireStringArg(command, args, 'attachmentId');
    return await VfsService.GetAttachment(attachmentId) as T;
  }

  if (command === 'vfs_get_attachment_content') {
    const attachmentId = requireStringArg(command, args, 'attachmentId');
    return await VfsService.GetAttachmentContent(attachmentId) as T;
  }

  if (command === 'vfs_upload_file') {
    return await VfsService.UploadFile(requireObjectArg(command, args, 'params') as any) as T;
  }

  if (command === 'vfs_get_file') {
    const fileId = requireStringArg(command, args, 'fileId');
    return await VfsService.GetFile(fileId) as T;
  }

  if (command === 'vfs_delete_file') {
    const fileId = requireStringArg(command, args, 'fileId');
    return await VfsService.DeleteFile(fileId) as T;
  }

  if (command === 'vfs_get_file_content') {
    const fileId = requireStringArg(command, args, 'fileId');
    return await VfsService.GetFileContent(fileId) as T;
  }

  if (command === 'textbooks_update_bookmarks') {
    const id = requireStringArg(command, args, 'id');
    const bookmarks = args?.bookmarks;
    if (!Array.isArray(bookmarks)) {
      throw new Error('Wails command textbooks_update_bookmarks requires array argument "bookmarks"');
    }
    return await VfsService.UpdateBookmarks(id, bookmarks) as T;
  }

  if (command === 'vfs_get_resource_refs') {
    return await VfsService.GetResourceRefs(requireObjectArg(command, args, 'params') as any) as T;
  }

  if (command === 'vfs_resolve_resource_refs') {
    const refs = args?.refs;
    if (!Array.isArray(refs)) {
      throw new Error('Wails command vfs_resolve_resource_refs requires array argument "refs"');
    }
    return await VfsService.ResolveResourceRefs(refs as any) as T;
  }

  if (command === 'vfs_update_path_cache') {
    const folderId = requireStringArg(command, args, 'folderId');
    return await VfsService.UpdatePathCache(folderId) as T;
  }

  if (command === 'vfs_get_pdf_processing_status') {
    const fileId = requireStringArg(command, args, 'fileId');
    return await VfsService.GetPdfProcessingStatus(fileId) as T;
  }

  if (command === 'vfs_get_batch_pdf_processing_status') {
    const fileIds = args?.fileIds;
    if (!Array.isArray(fileIds) || fileIds.some(fileId => typeof fileId !== 'string')) {
      throw new Error('Wails command vfs_get_batch_pdf_processing_status requires string[] argument "fileIds"');
    }
    return await VfsService.GetBatchPdfProcessingStatus(fileIds) as T;
  }

  if (command === 'vfs_cancel_pdf_processing') {
    const fileId = requireStringArg(command, args, 'fileId');
    return await VfsService.CancelPdfProcessing(fileId) as T;
  }

  if (command === 'vfs_retry_pdf_processing') {
    const fileId = requireStringArg(command, args, 'fileId');
    return await VfsService.RetryPdfProcessing(fileId) as T;
  }

  if (command === 'vfs_start_pdf_processing') {
    const fileId = requireStringArg(command, args, 'fileId');
    return await VfsService.StartPdfProcessing(fileId, optionalStringArg(args, 'startFromStage') ?? null) as T;
  }

  if (command === 'vfs_get_pdf_page_image') {
    const resourceId = requireStringArg(command, args, 'resourceId');
    const pageIndex = args?.pageIndex;
    if (typeof pageIndex !== 'number') {
      throw new Error('Wails command vfs_get_pdf_page_image requires number argument "pageIndex"');
    }
    return await VfsService.GetPdfPageImage(resourceId, pageIndex) as T;
  }

  if (command === 'vfs_get_blob_base64') {
    const blobHash = requireStringArg(command, args, 'blobHash');
    return await VfsService.GetBlobBase64(blobHash) as T;
  }

  if (command === 'vfs_unified_index_status') {
    return await VfsService.UnifiedIndexStatus() as T;
  }

  if (command === 'vfs_get_resource_units') {
    const resourceId = requireStringArg(command, args, 'resourceId');
    return await VfsService.GetResourceUnits(resourceId) as T;
  }

  if (command === 'vfs_sync_resource_units') {
    return await VfsService.SyncResourceUnits({
      resourceId: requireStringArg(command, args, 'resourceId'),
      resourceType: requireStringArg(command, args, 'resourceType'),
      data: optionalStringArg(args, 'data') ?? null,
      ocrText: optionalStringArg(args, 'ocrText') ?? null,
      ocrPagesJson: optionalStringArg(args, 'ocrPagesJson') ?? null,
      blobHash: optionalStringArg(args, 'blobHash') ?? null,
      pageCount: typeof args?.pageCount === 'number' ? args.pageCount : null,
      extractedText: optionalStringArg(args, 'extractedText') ?? null,
      previewJson: optionalStringArg(args, 'previewJson') ?? null,
    } as any) as T;
  }

  if (command === 'vfs_get_all_index_status') {
    return await VfsService.GetAllIndexStatus({
      folderId: optionalStringArg(args, 'folderId') ?? null,
      resourceType: optionalStringArg(args, 'resourceType') ?? null,
      stateFilter: optionalStringArg(args, 'stateFilter') ?? null,
      includeImageIndex: optionalBooleanArg(args, 'includeImageIndex'),
      limit: typeof args?.limit === 'number' ? args.limit : 100,
      offset: typeof args?.offset === 'number' ? args.offset : 0,
    } as any) as T;
  }

  if (command === 'vfs_reindex_resource') {
    const resourceId = requireStringArg(command, args, 'resourceId');
    return await VfsService.ReindexResource(resourceId) as T;
  }

  if (command === 'vfs_reindex_unit') {
    const unitId = requireStringArg(command, args, 'unitId');
    const mode = optionalStringArg(args, 'mode') ?? 'text';
    return await VfsService.ReindexUnit(unitId, mode) as T;
  }

  if (command === 'vfs_unified_batch_index' || command === 'vfs_batch_index_pending') {
    const batchSize = typeof args?.batchSize === 'number' ? args.batchSize : 10;
    return await VfsService.BatchIndexPending(batchSize) as T;
  }

  if (command === 'vfs_delete_resource_index') {
    const resourceId = requireStringArg(command, args, 'resourceId');
    return await VfsService.DeleteResourceIndex(resourceId) as T;
  }

  if (command === 'vfs_list_embedding_dims') {
    return await VfsService.ListEmbeddingDims() as T;
  }

  if (command === 'vfs_list_dimensions') {
    return await VfsService.ListDimensions() as T;
  }

  if (command === 'vfs_get_resource_text_chunks') {
    const resourceId = requireStringArg(command, args, 'resourceId');
    return await VfsService.GetResourceTextChunks(resourceId) as T;
  }

  if (command === 'vfs_get_resource_ocr_info') {
    const resourceId = requireStringArg(command, args, 'resourceId');
    return await VfsService.GetResourceOcrInfo(resourceId) as T;
  }

  if (command === 'vfs_clear_resource_ocr') {
    const resourceId = requireStringArg(command, args, 'resourceId');
    return await VfsService.ClearResourceOcr(resourceId) as T;
  }

  if (command === 'vfs_rag_search') {
    return await VfsService.RagSearch(requireObjectArg(command, args, 'input') as any) as T;
  }

  if (command === 'vfs_list_files') {
    return await VfsService.ListFiles({
      fileType: optionalStringArg(args, 'fileType') ?? '',
      limit: typeof args?.limit === 'number' ? args.limit : 100,
      offset: typeof args?.offset === 'number' ? args.offset : 0,
    } as any) as T;
  }

  if (command === 'vfs_create_mindmap') {
    return await VfsService.CreateMindMap(requireObjectArg(command, args, 'params') as any) as T;
  }

  if (command === 'vfs_get_mindmap') {
    const mindmapId = requireStringArg(command, args, 'mindmapId');
    return await VfsService.GetMindMap(mindmapId) as T;
  }

  if (command === 'vfs_get_mindmap_content') {
    const mindmapId = requireStringArg(command, args, 'mindmapId');
    return await VfsService.GetMindMapContent(mindmapId) as T;
  }

  if (command === 'vfs_update_mindmap') {
    const mindmapId = requireStringArg(command, args, 'mindmapId');
    return await VfsService.UpdateMindMap(mindmapId, requireObjectArg(command, args, 'params') as any) as T;
  }

  if (command === 'vfs_delete_mindmap') {
    const mindmapId = requireStringArg(command, args, 'mindmapId');
    return await VfsService.DeleteMindMap(mindmapId) as T;
  }

  if (command === 'vfs_list_mindmaps') {
    return await VfsService.ListMindMaps() as T;
  }

  if (command === 'vfs_set_mindmap_favorite') {
    const mindmapId = requireStringArg(command, args, 'mindmapId');
    const isFavorite = args?.isFavorite;
    if (typeof isFavorite !== 'boolean') {
      throw new Error('Wails command vfs_set_mindmap_favorite requires boolean argument "isFavorite"');
    }
    return await VfsService.SetMindMapFavorite(mindmapId, isFavorite) as T;
  }

  if (command === 'vfs_get_mindmap_versions') {
    const mindmapId = requireStringArg(command, args, 'mindmapId');
    return await VfsService.GetMindMapVersions(mindmapId) as T;
  }

  if (command === 'vfs_get_mindmap_version') {
    const versionId = requireStringArg(command, args, 'versionId');
    return await VfsService.GetMindMapVersion(versionId) as T;
  }

  if (command === 'vfs_get_mindmap_version_content') {
    const versionId = requireStringArg(command, args, 'versionId');
    return await VfsService.GetMindMapVersionContent(versionId) as T;
  }

  if (command === 'qbank_list_questions') {
    return await QbankService.ListQuestions(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'qbank_search_questions') {
    return await QbankService.SearchQuestions(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'qbank_rebuild_fts_index') {
    return await QbankService.RebuildFTSIndex() as T;
  }

  if (command === 'qbank_get_question') {
    const questionId = requireStringArg(command, args, 'questionId');
    return await QbankService.GetQuestion(questionId) as T;
  }

  if (command === 'qbank_create_question') {
    return await QbankService.CreateQuestion(requireObjectArg(command, args, 'params') as any) as T;
  }

  if (command === 'qbank_update_question') {
    return await QbankService.UpdateQuestion(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'qbank_delete_question') {
    const questionId = requireStringArg(command, args, 'questionId');
    return await QbankService.DeleteQuestion(questionId) as T;
  }

  if (command === 'qbank_batch_delete_questions') {
    const questionIds = args?.questionIds;
    if (!Array.isArray(questionIds) || questionIds.some(id => typeof id !== 'string')) {
      throw new Error('Wails command qbank_batch_delete_questions requires string[] argument "questionIds"');
    }
    return await QbankService.BatchDeleteQuestions(questionIds) as T;
  }

  if (command === 'qbank_submit_answer') {
    return await QbankService.SubmitAnswer(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'qbank_ai_grade') {
    return await QbankService.AIGrade(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'qbank_cancel_grading') {
    const streamEventName = requireStringArg(command, args, 'streamEventName');
    return await QbankService.CancelGrading(streamEventName) as T;
  }

  if (command === 'qbank_sync_check') {
    const examId = requireStringArg(command, args, 'examId');
    return await QbankService.CheckSyncStatus(examId) as T;
  }

  if (command === 'qbank_get_sync_conflicts') {
    const examId = requireStringArg(command, args, 'examId');
    return await QbankService.GetSyncConflicts(examId) as T;
  }

  if (command === 'qbank_resolve_sync_conflict') {
    const conflictId = requireStringArg(command, args, 'conflictId');
    const strategy = requireStringArg(command, args, 'strategy');
    return await QbankService.ResolveSyncConflict(conflictId, strategy) as T;
  }

  if (command === 'qbank_batch_resolve_conflicts') {
    const examId = requireStringArg(command, args, 'examId');
    const strategy = requireStringArg(command, args, 'strategy');
    return await QbankService.BatchResolveConflicts(examId, strategy) as T;
  }

  if (command === 'qbank_set_sync_enabled') {
    const examId = requireStringArg(command, args, 'examId');
    const enabled = args?.enabled;
    if (typeof enabled !== 'boolean') {
      throw new Error('Wails command qbank_set_sync_enabled requires boolean argument "enabled"');
    }
    return await QbankService.SetSyncEnabled(examId, enabled) as T;
  }

  if (command === 'qbank_update_sync_config') {
    const examId = requireStringArg(command, args, 'examId');
    return await QbankService.UpdateSyncConfig(examId, requireObjectArg(command, args, 'config') as any) as T;
  }

  if (command === 'qbank_toggle_favorite') {
    const questionId = requireStringArg(command, args, 'questionId');
    return await QbankService.ToggleFavorite(questionId) as T;
  }

  if (command === 'qbank_get_stats') {
    const examId = requireStringArg(command, args, 'examId');
    return await QbankService.GetStats(examId) as T;
  }

  if (command === 'qbank_refresh_stats') {
    const examId = requireStringArg(command, args, 'examId');
    return await QbankService.RefreshStats(examId) as T;
  }

  if (command === 'qbank_reset_progress') {
    const examId = requireStringArg(command, args, 'examId');
    return await QbankService.ResetProgress(examId) as T;
  }

  if (command === 'qbank_reset_questions_progress') {
    const questionIds = args?.questionIds;
    if (!Array.isArray(questionIds) || questionIds.some(id => typeof id !== 'string')) {
      throw new Error('Wails command qbank_reset_questions_progress requires string[] argument "questionIds"');
    }
    return await QbankService.ResetQuestionsProgress(questionIds) as T;
  }

  if (command === 'qbank_get_history') {
    const questionId = requireStringArg(command, args, 'questionId');
    const limit = typeof args?.limit === 'number' ? args.limit : 50;
    return await QbankService.GetHistory(questionId, limit) as T;
  }

  if (command === 'qbank_get_submissions') {
    const questionId = requireStringArg(command, args, 'questionId');
    const limit = typeof args?.limit === 'number' ? args.limit : 20;
    return await QbankService.GetSubmissions(questionId, limit) as T;
  }

  if (command === 'qbank_get_learning_trend') {
    return await QbankService.GetLearningTrend(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'qbank_get_activity_heatmap') {
    return await QbankService.GetActivityHeatmap(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'qbank_get_knowledge_stats') {
    return await QbankService.GetKnowledgeStats(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'qbank_get_knowledge_stats_with_comparison') {
    return await QbankService.GetKnowledgeStatsWithComparison(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'qbank_start_timed_practice') {
    return await QbankService.StartTimedPractice(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'qbank_generate_mock_exam') {
    return await QbankService.GenerateMockExam(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'qbank_submit_mock_exam') {
    return await QbankService.SubmitMockExam(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'qbank_get_daily_practice') {
    return await QbankService.GetDailyPractice(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'qbank_generate_paper') {
    return await QbankService.GeneratePaper(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'qbank_get_check_in_calendar') {
    return await QbankService.GetCheckInCalendar(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'get_csv_preview') {
    const filePath = requireStringArg(command, args, 'filePath');
    const rows = typeof args?.rows === 'number' ? args.rows : 5;
    return await QbankService.GetCsvPreview(filePath, rows) as T;
  }

  if (command === 'import_questions_csv') {
    return await QbankService.ImportQuestionsCsv(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'export_questions_csv') {
    return await QbankService.ExportQuestionsCsv(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'get_csv_exportable_fields') {
    return await QbankService.GetCsvExportableFields() as T;
  }

  if (command === 'review_plan_create') {
    const questionId = requireStringArg(command, args, 'questionId');
    const examId = requireStringArg(command, args, 'examId');
    return await ReviewPlanService.Create(questionId, examId) as T;
  }

  if (command === 'review_plan_process') {
    const planId = requireStringArg(command, args, 'planId');
    const quality = optionalNumberArg(args, 'quality', -1);
    const userAnswer = optionalStringArg(args, 'userAnswer') ?? null;
    const timeSpentSeconds = typeof args?.timeSpentSeconds === 'number' ? args.timeSpentSeconds : null;
    return await ReviewPlanService.Process(planId, quality, userAnswer, timeSpentSeconds) as T;
  }

  if (command === 'review_plan_get_due') {
    const examId = optionalStringArg(args, 'examId') ?? null;
    const untilDate = optionalStringArg(args, 'untilDate') ?? null;
    return await ReviewPlanService.GetDue(examId, untilDate) as T;
  }

  if (command === 'review_plan_get_due_with_filter') {
    return await ReviewPlanService.GetDueWithFilter(requireObjectArg(command, args, 'filter') as any) as T;
  }

  if (command === 'review_plan_get_stats') {
    const examId = optionalStringArg(args, 'examId') ?? null;
    return await ReviewPlanService.GetStats(examId) as T;
  }

  if (command === 'review_plan_refresh_stats') {
    const examId = optionalStringArg(args, 'examId') ?? null;
    return await ReviewPlanService.RefreshStats(examId) as T;
  }

  if (command === 'review_plan_get_by_question') {
    const questionId = requireStringArg(command, args, 'questionId');
    return await ReviewPlanService.GetByQuestion(questionId) as T;
  }

  if (command === 'review_plan_get') {
    const planId = requireStringArg(command, args, 'planId');
    return await ReviewPlanService.Get(planId) as T;
  }

  if (command === 'review_plan_suspend') {
    const planId = requireStringArg(command, args, 'planId');
    return await ReviewPlanService.Suspend(planId) as T;
  }

  if (command === 'review_plan_resume') {
    const planId = requireStringArg(command, args, 'planId');
    return await ReviewPlanService.Resume(planId) as T;
  }

  if (command === 'review_plan_delete') {
    const planId = requireStringArg(command, args, 'planId');
    return await ReviewPlanService.Delete(planId) as T;
  }

  if (command === 'review_plan_get_history') {
    const planId = requireStringArg(command, args, 'planId');
    const limit = optionalNumberArg(args, 'limit', 50);
    return await ReviewPlanService.GetHistory(planId, limit) as T;
  }

  if (command === 'review_plan_batch_create') {
    const questionIds = args?.questionIds;
    if (!Array.isArray(questionIds) || questionIds.some(id => typeof id !== 'string')) {
      throw new Error('Wails command review_plan_batch_create requires string[] argument "questionIds"');
    }
    const examId = requireStringArg(command, args, 'examId');
    return await ReviewPlanService.BatchCreate(questionIds, examId) as T;
  }

  if (command === 'review_plan_create_for_exam') {
    const examId = requireStringArg(command, args, 'examId');
    return await ReviewPlanService.CreateForExam(examId) as T;
  }

  if (command === 'review_plan_list_by_exam') {
    const examId = requireStringArg(command, args, 'examId');
    const limit = optionalNumberArg(args, 'limit', 100);
    const offset = optionalNumberArg(args, 'offset', 0);
    return await ReviewPlanService.ListByExam(examId, limit, offset) as T;
  }

  if (command === 'review_plan_get_or_create') {
    const questionId = requireStringArg(command, args, 'questionId');
    const examId = requireStringArg(command, args, 'examId');
    return await ReviewPlanService.GetOrCreate(questionId, examId) as T;
  }

  if (command === 'review_plan_get_calendar_data') {
    const startDate = optionalStringArg(args, 'startDate') ?? null;
    const endDate = optionalStringArg(args, 'endDate') ?? null;
    const examId = optionalStringArg(args, 'examId') ?? null;
    return await ReviewPlanService.GetCalendarData(startDate, endDate, examId) as T;
  }

  if (command === 'import_builtin_templates') {
    return await TemplateService.ImportBuiltinTemplates() as T;
  }

  if (command === 'get_all_custom_templates') {
    return await TemplateService.GetAllCustomTemplates() as T;
  }

  if (command === 'get_default_template_id') {
    return await TemplateService.GetDefaultTemplateID() as T;
  }

  if (command === 'create_custom_template') {
    const request = requireObjectArg(command, args, 'request');
    return await TemplateService.CreateCustomTemplate(request) as T;
  }

  if (command === 'update_custom_template') {
    const templateId = requireStringArg(command, args, 'templateId');
    const request = requireObjectArg(command, args, 'request');
    return await TemplateService.UpdateCustomTemplate(templateId, request) as T;
  }

  if (command === 'delete_custom_template') {
    const templateId = requireStringArg(command, args, 'templateId');
    return await TemplateService.DeleteCustomTemplate(templateId) as T;
  }

  if (command === 'set_default_template') {
    const templateId = requireStringArg(command, args, 'templateId');
    return await TemplateService.SetDefaultTemplate(templateId) as T;
  }

  if (command === 'import_custom_templates_bulk') {
    const templateData = requireStringArgAny(command, args, ['template_data', 'templateData']);
    const overwriteExisting = optionalBooleanArgAny(args, ['overwrite_existing', 'overwriteExisting']) ?? false;
    const strictBuiltin = optionalBooleanArgAny(args, ['strict_builtin', 'strictBuiltin']) ?? false;
    return await TemplateService.ImportCustomTemplatesBulk(templateData, overwriteExisting, strictBuiltin) as T;
  }

  if (command === 'export_template') {
    const templateId = requireStringArg(command, args, 'templateId');
    return await TemplateService.ExportTemplate(templateId) as T;
  }

  if (command === 'get_setting') {
    const key = requireStringArg(command, args, 'key');
    return await SettingsService.GetSetting(key) as T;
  }

  if (command === 'get_settings') {
    const keys = args?.keys;
    if (!Array.isArray(keys) || keys.some(key => typeof key !== 'string')) {
      throw new Error('Wails command get_settings requires string[] argument "keys"');
    }
    return await SettingsService.GetSettings(keys) as T;
  }

  if (command === 'get_settings_by_prefix') {
    const prefix = requireStringArg(command, args, 'prefix');
    return await SettingsService.GetSettingsByPrefix(prefix) as T;
  }

  if (command === 'save_setting') {
    const key = requireStringArg(command, args, 'key');
    const value = requireStringArg(command, args, 'value');
    return await SettingsService.SaveSetting(key, value) as T;
  }

  if (command === 'save_settings') {
    const values = args?.values;
    if (!values || typeof values !== 'object' || Array.isArray(values)) {
      throw new Error('Wails command save_settings requires object argument "values"');
    }
    return await SettingsService.SaveSettings(values as Record<string, string>) as T;
  }

  if (command === 'delete_setting') {
    const key = requireStringArg(command, args, 'key');
    return await SettingsService.DeleteSetting(key) as T;
  }

  if (command === 'get_backup_config') {
    return await SettingsService.GetBackupConfig() as T;
  }

  if (command === 'set_backup_config') {
    const config = requireObjectArg(command, args, 'config');
    return await SettingsService.SetBackupConfig(config as any) as T;
  }

  if (command === 'check_api_config_status') {
    return await SettingsService.CheckAPIConfigStatus() as T;
  }

  if (command === 'restore_default_api_configs') {
    return await SettingsService.RestoreDefaultAPIConfigs() as T;
  }

  if (command === 'test_api_connection') {
    const apiKey = optionalStringArgAny(args, ['apiKey', 'api_key']) ?? '';
    const apiBase = optionalStringArgAny(args, ['apiBase', 'api_base']) ?? '';
    const apiProtocol = optionalStringArgAny(args, ['apiProtocol', 'api_protocol']) ?? null;
    const supportsOpenAIResponses =
      optionalBooleanArgAny(args, ['supportsOpenAIResponses', 'supports_openai_responses']) ?? null;
    const model = optionalStringArg(args, 'model') ?? null;
    const vendorId = optionalStringArgAny(args, ['vendorId', 'vendor_id']) ?? null;
    return await SettingsService.TestAPIConnection(
      apiKey,
      apiBase,
      apiProtocol,
      supportsOpenAIResponses,
      model,
      vendorId
    ) as T;
  }

  if (command === 'test_search_engine') {
    const engine = requireStringArg(command, args, 'engine');
    return await SettingsService.TestSearchEngine(engine) as T;
  }

  if (command === 'test_web_search_connectivity') {
    const engine = optionalStringArg(args, 'engine') ?? null;
    return await SettingsService.TestWebSearchConnectivity(engine) as T;
  }

  if (command === 'test_all_search_engines') {
    return await SettingsService.TestAllSearchEngines() as T;
  }

  if (command === 'get_statistics') {
    return await SettingsService.GetStatistics() as T;
  }

  if (command === 'get_enhanced_statistics') {
    return await SettingsService.GetEnhancedStatistics() as T;
  }

  if (command === 'vfs_get_attachment_config') {
    return await SettingsService.GetAttachmentConfig() as T;
  }

  if (command === 'vfs_set_attachment_root_folder') {
    const folderId = requireStringArg(command, args, 'folderId');
    return await SettingsService.SetAttachmentRootFolder(folderId) as T;
  }

  if (command === 'vfs_create_attachment_root_folder') {
    const title = requireStringArg(command, args, 'title');
    return await SettingsService.CreateAttachmentRootFolder(title) as T;
  }

  if (command === 'memory_get_config') {
    return await SettingsService.GetMemoryConfig() as T;
  }

  if (command === 'get_model_adapter_options') {
    return await SettingsService.GetModelAdapterOptions() as T;
  }

  if (command === 'get_cn_whitelist_config') {
    return await SettingsService.GetCNWhitelistConfig() as T;
  }

  if (command === 'get_provider_strategies_config') {
    return await SettingsService.GetProviderStrategiesConfig() as T;
  }

  if (command === 'save_provider_strategies_config') {
    const strategies = requireObjectArg(command, args, 'strategies');
    return await SettingsService.SaveProviderStrategiesConfig(strategies as any) as T;
  }

  if (command === 'preheat_mcp_tools') {
    return await SettingsService.PreheatMCPTools() as T;
  }

  if (command === 'get_mcp_status') {
    return await SettingsService.GetMCPStatus() as T;
  }

  if (command === 'get_mcp_tools') {
    return await SettingsService.GetMCPTools() as T;
  }

  if (command === 'mcp_stdio_start') {
    const commandArg = requireStringArg(command, args, 'command');
    const processArgs = Array.isArray(args?.args)
      ? args.args.filter((item): item is string => typeof item === 'string')
      : [];
    const env = optionalStringRecordArg(args, 'env');
    const framing = optionalStringArg(args, 'framing') ?? null;
    const cwd = optionalStringArg(args, 'cwd') ?? null;
    return await McpService.StartStdioSession(commandArg, processArgs, env, framing, cwd) as T;
  }

  if (command === 'mcp_stdio_send') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    const payload = requireStringArg(command, args, 'payload');
    return await McpService.SendStdioMessage(sessionId, payload) as T;
  }

  if (command === 'mcp_stdio_close') {
    const sessionId = requireStringArg(command, args, 'sessionId');
    return await McpService.CloseStdioSession(sessionId) as T;
  }

  if (command === 'get_ocr_engines') {
    return await SettingsService.GetOCREngines() as T;
  }

  if (command === 'get_ocr_engine_type') {
    return await SettingsService.GetOCREngineType() as T;
  }

  if (command === 'get_ocr_thinking_enabled') {
    return await SettingsService.GetOCRThinkingEnabled() as T;
  }

  if (command === 'set_ocr_thinking_enabled') {
    return await SettingsService.SetOCRThinkingEnabled(optionalBooleanArg(args, 'enabled')) as T;
  }

  if (command === 'get_available_ocr_models') {
    return await SettingsService.GetAvailableOCRModels() as T;
  }

  if (command === 'test_ocr_engine') {
    const request = requireObjectArg(command, args, 'request');
    return await SettingsService.TestOCREngine(request as any) as T;
  }

  if (command === 'save_available_ocr_models') {
    const models = args?.models;
    if (!Array.isArray(models)) {
      throw new Error('Wails command save_available_ocr_models requires array argument "models"');
    }
    return await SettingsService.SaveAvailableOCRModels(models as any) as T;
  }

  if (command === 'update_ocr_engine_priority') {
    const engineList = args?.engineList;
    if (!Array.isArray(engineList)) {
      throw new Error('Wails command update_ocr_engine_priority requires array argument "engineList"');
    }
    return await SettingsService.UpdateOCREnginePriority(engineList as any) as T;
  }

  if (command === 'add_ocr_engine') {
    const configId = requireStringArg(command, args, 'configId');
    const model = requireStringArg(command, args, 'model');
    const name = requireStringArg(command, args, 'name');
    const engineType = optionalStringArg(args, 'engineType') ?? null;
    return await SettingsService.AddOCREngine(configId, model, name, engineType) as T;
  }

  if (command === 'remove_ocr_engine') {
    const configId = requireStringArg(command, args, 'configId');
    return await SettingsService.RemoveOCREngine(configId) as T;
  }

  if (command === 'reload_mcp_client') {
    return await SettingsService.ReloadMCPClient() as T;
  }

  if (command === 'get_api_configurations') {
    return await SettingsService.GetAPIConfigurations() as T;
  }

  if (command === 'save_api_configurations') {
    const configs = args?.configs;
    if (!Array.isArray(configs)) {
      throw new Error('Wails command save_api_configurations requires array argument "configs"');
    }
    return await SettingsService.SaveAPIConfigurations(configs as any) as T;
  }

  if (command === 'get_vendor_configs') {
    return await SettingsService.GetVendorConfigs() as T;
  }

  if (command === 'save_vendor_configs') {
    const configs = args?.configs;
    if (!Array.isArray(configs)) {
      throw new Error('Wails command save_vendor_configs requires array argument "configs"');
    }
    return await SettingsService.SaveVendorConfigs(configs as any) as T;
  }

  if (command === 'get_model_profiles') {
    return await SettingsService.GetModelProfiles() as T;
  }

  if (command === 'save_model_profiles') {
    const profiles = args?.profiles;
    if (!Array.isArray(profiles)) {
      throw new Error('Wails command save_model_profiles requires array argument "profiles"');
    }
    return await SettingsService.SaveModelProfiles(profiles as any) as T;
  }

  if (command === 'get_model_assignments') {
    return await SettingsService.GetModelAssignments() as T;
  }

  if (command === 'save_model_assignments') {
    return await SettingsService.SaveModelAssignments(requireObjectArg(command, args, 'assignments') as any) as T;
  }

  if (command === 'get_app_data_dir') {
    return await SystemService.AppDataDir() as T;
  }

  if (command === 'skill_list_directories') {
    const path = requireStringArg(command, args, 'path');
    return await SkillService.ListDirectories(path) as T;
  }

  if (command === 'skill_read_file') {
    const path = requireStringArg(command, args, 'path');
    return await SkillService.ReadFile(path) as T;
  }

  if (command === 'skill_create') {
    const basePath = requireStringArg(command, args, 'basePath');
    const skillID = requireStringArg(command, args, 'skillId');
    const content = requireStringArg(command, args, 'content');
    return await SkillService.Create(basePath, skillID, content) as T;
  }

  if (command === 'skill_update') {
    const path = requireStringArg(command, args, 'path');
    const content = requireStringArg(command, args, 'content');
    return await SkillService.Update(path, content) as T;
  }

  if (command === 'skill_delete') {
    const path = requireStringArg(command, args, 'path');
    return await SkillService.Delete(path) as T;
  }

  if (command === 'ensure_debug_log_dir') {
    return await SystemService.EnsureDebugLogDir() as T;
  }

  if (command === 'open_logs_folder') {
    const logType = requireStringArg(command, args, 'logType');
    return await SystemService.OpenLogsFolder(logType) as T;
  }

  if (command === 'report_frontend_log') {
    const payload = args?.payload;
    if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
      throw new Error('Wails command report_frontend_log requires object argument "payload"');
    }
    return await SystemService.ReportFrontendLog(payload as any) as T;
  }

  if (command === 'save_webview_settings') {
    const settings = args?.settings;
    if (!settings || typeof settings !== 'object' || Array.isArray(settings)) {
      throw new Error('Wails command save_webview_settings requires object argument "settings"');
    }
    return await SystemService.SaveWebviewSettings(settings as Record<string, unknown>) as T;
  }

  if (command === 'read_file_bytes') {
    const path = requireStringArg(command, args, 'path');
    const bytes = await FileService.ReadFileBytes(path);
    return byteSliceToNumbers(bytes) as T;
  }

  if (command === 'read_file_text') {
    const path = requireStringArg(command, args, 'path');
    return await FileService.ReadFileText(path) as T;
  }

  if (command === 'save_text_to_file') {
    const path = requireStringArg(command, args, 'path');
    const content = requireStringArg(command, args, 'content');
    return await FileService.SaveTextToFile(path, content) as T;
  }

  if (command === 'get_file_size') {
    const path = requireStringArg(command, args, 'path');
    return await FileService.GetFileSize(path) as T;
  }

  if (command === 'copy_file') {
    const sourcePath =
      typeof args?.sourcePath === 'string'
        ? args.sourcePath
        : requireStringArg(command, args, 'source_path');
    const destPath =
      typeof args?.destPath === 'string'
        ? args.destPath
        : requireStringArg(command, args, 'dest_path');
    return await FileService.CopyFile(sourcePath, destPath) as T;
  }

  if (command === 'notes_get_pref') {
    const key = requireStringArg(command, args, 'key');
    return await NotesService.GetPref(key) as T;
  }

  if (command === 'notes_set_pref') {
    const key = requireStringArg(command, args, 'key');
    const value = requireStringArg(command, args, 'value');
    return await NotesService.SetPref(key, value) as T;
  }

  if (command === 'notes_save_asset') {
    const subject = typeof args?.subject === 'string' ? args.subject : '_global';
    const noteId = requireStringArgAny(command, args, ['noteId', 'note_id']);
    const base64Data = requireStringArgAny(command, args, ['base64Data', 'base64_data']);
    const defaultExt = optionalStringArgAny(args, ['defaultExt', 'default_ext']) ?? null;
    return await NotesService.SaveAsset(subject, noteId, base64Data, defaultExt) as T;
  }

  if (command === 'notes_list_assets') {
    const subject = typeof args?.subject === 'string' ? args.subject : '_global';
    const noteId = requireStringArgAny(command, args, ['noteId', 'note_id']);
    return await NotesService.ListAssets(subject, noteId) as T;
  }

  if (command === 'notes_assets_index_scan') {
    const noteId = requireStringArgAny(command, args, ['noteId', 'note_id']);
    return await NotesService.AssetsIndexScan(noteId) as T;
  }

  if (command === 'notes_assets_scan_orphans') {
    return await NotesService.ScanOrphanAssets() as T;
  }

  if (command === 'notes_assets_bulk_delete') {
    const paths = requireStringArrayArg(command, args, 'paths');
    return await NotesService.BulkDeleteAssets(paths) as T;
  }

  if (command === 'notes_db_stats') {
    return await NotesService.DBStats() as T;
  }

  if (command === 'notes_db_vacuum') {
    return await NotesService.DBVacuum() as T;
  }

  if (command === 'notes_export') {
    return await NotesService.Export(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'notes_export_single') {
    return await NotesService.ExportSingle(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'notes_import') {
    return await NotesService.Import(requireObjectArg(command, args, 'request') as any) as T;
  }

  if (command === 'notes_delete_asset') {
    const relativePath = requireStringArgAny(command, args, ['relativePath', 'relative_path']);
    return await NotesService.DeleteAsset(relativePath) as T;
  }

  if (command === 'notes_resolve_asset_path') {
    const relativePath = requireStringArgAny(command, args, ['relativePath', 'relative_path']);
    return await NotesService.ResolveAssetPath(relativePath) as T;
  }

  if (command === 'get_image_as_base64') {
    const relativePath = requireStringArgAny(command, args, ['relativePath', 'relative_path']);
    return await NotesService.GetImageAsBase64(relativePath) as T;
  }

  if (command === 'todo_ensure_inbox') {
    return await TodoService.EnsureInbox() as T;
  }

  if (command === 'todo_create_list') {
    return await TodoService.CreateList(requireObjectArg(command, args, 'input') as any) as T;
  }

  if (command === 'todo_get_list') {
    const listId = requireStringArg(command, args, 'listId');
    return await TodoService.GetList(listId) as T;
  }

  if (command === 'todo_list_lists') {
    return await TodoService.ListLists() as T;
  }

  if (command === 'todo_update_list') {
    return await TodoService.UpdateList(requireObjectArg(command, args, 'input') as any) as T;
  }

  if (command === 'todo_delete_list') {
    const listId = requireStringArg(command, args, 'listId');
    return await TodoService.DeleteList(listId) as T;
  }

  if (command === 'todo_toggle_list_favorite') {
    const listId = requireStringArg(command, args, 'listId');
    return await TodoService.ToggleListFavorite(listId) as T;
  }

  if (command === 'todo_create_item') {
    return await TodoService.CreateItem(requireObjectArg(command, args, 'input') as any) as T;
  }

  if (command === 'todo_get_item') {
    const itemId = requireStringArg(command, args, 'itemId');
    return await TodoService.GetItem(itemId) as T;
  }

  if (command === 'todo_list_items') {
    const listId = requireStringArg(command, args, 'listId');
    return await TodoService.ListItems(listId, optionalBooleanArg(args, 'includeCompleted')) as T;
  }

  if (command === 'todo_update_item') {
    return await TodoService.UpdateItem(requireObjectArg(command, args, 'input') as any) as T;
  }

  if (command === 'todo_toggle_item') {
    const itemId = requireStringArg(command, args, 'itemId');
    return await TodoService.ToggleItem(itemId) as T;
  }

  if (command === 'todo_delete_item') {
    const itemId = requireStringArg(command, args, 'itemId');
    return await TodoService.DeleteItem(itemId) as T;
  }

  if (command === 'todo_reorder_items') {
    return await TodoService.ReorderItems(requireObjectArg(command, args, 'input') as any) as T;
  }

  if (command === 'todo_list_today') {
    return await TodoService.ListToday(optionalBooleanArg(args, 'includeCompleted')) as T;
  }

  if (command === 'todo_list_overdue') {
    return await TodoService.ListOverdue(optionalBooleanArg(args, 'includeCompleted')) as T;
  }

  if (command === 'todo_list_upcoming') {
    const days = typeof args?.days === 'number' ? args.days : 7;
    return await TodoService.ListUpcoming(days, optionalBooleanArg(args, 'includeCompleted')) as T;
  }

  if (command === 'todo_list_completed') {
    return await TodoService.ListCompleted(optionalStringArg(args, 'listId')) as T;
  }

  if (command === 'todo_search') {
    const query = requireStringArg(command, args, 'query');
    return await TodoService.Search(query) as T;
  }

  if (command === 'todo_get_active_summary') {
    return await TodoService.ActiveSummary() as T;
  }

  if (command === 'pomodoro_create_record') {
    return await TodoService.CreatePomodoroRecord(requireObjectArg(command, args, 'input') as any) as T;
  }

  if (command === 'pomodoro_get_record') {
    const recordId = requireStringArg(command, args, 'recordId');
    return await TodoService.GetPomodoroRecord(recordId) as T;
  }

  if (command === 'pomodoro_list_by_todo') {
    const todoItemId = requireStringArg(command, args, 'todoItemId');
    return await TodoService.ListPomodorosByTodo(todoItemId) as T;
  }

  if (command === 'pomodoro_today_stats') {
    return await TodoService.PomodoroTodayStats() as T;
  }

  if (command === 'pomodoro_list_today') {
    return await TodoService.ListTodayPomodoros() as T;
  }

  throw new Error(`Wails command is not implemented in the lean Go bridge: ${command}`);
}
