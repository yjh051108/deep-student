import { invoke } from '@tauri-apps/api/core';
import { invoke as nativeInvoke } from '@/runtime/native';

export type AutoExtractFrequency = 'off' | 'balanced' | 'aggressive';

export interface MemoryConfig {
  memoryRootFolderId: string | null;
  memoryRootFolderTitle: string | null;
  autoCreateSubfolders: boolean;
  defaultCategory: string;
  privacyMode: boolean;
  autoExtractFrequency: AutoExtractFrequency;
}

export interface MemorySearchResult {
  noteId: string;
  noteTitle: string;
  folderPath: string;
  chunkText: string;
  score: number;
  scope?: MemoryScopeValue;
}

export interface MemoryListItem {
  id: string;
  title: string;
  folderPath: string;
  updatedAt: string;
  hits: number;
  isImportant: boolean;
  isStale: boolean;
  memoryType: string;
  memoryPurpose: string;
}

export interface MemoryReadOutput {
  noteId: string;
  title: string;
  content: string;
  folderPath: string;
  updatedAt: string;
}

export interface MemoryWriteOutput {
  noteId: string;
  isNew: boolean;
  resourceId: string;
}

export interface SmartWriteOutput {
  noteId: string;
  event: 'ADD' | 'UPDATE' | 'APPEND' | 'DELETE' | 'NONE' | 'FILTERED';
  isNew: boolean;
  confidence: number;
  reason: string;
  resourceId?: string;
  downgraded: boolean;
}

export type MemoryTypeValue = 'fact' | 'study' | 'note';
export type MemoryScopeValue = 'topic' | 'global';

export interface MemoryScopeContext {
  groupId?: string | null;
  groupName?: string | null;
  adminAll?: boolean;
}

export interface MemoryBatchWriteItemInput {
  title: string;
  content: string;
  folderPath?: string;
  scope?: MemoryScopeValue;
  memoryType?: MemoryTypeValue;
  memoryPurpose?: MemoryPurposeType;
  idempotencyKey?: string;
}

export interface MemoryBatchWriteItemResult {
  title: string;
  noteId: string;
  event: SmartWriteOutput['event'];
  isNew: boolean;
  confidence: number;
  reason: string;
  downgraded: boolean;
}

export interface MemoryBatchWriteOutput {
  total: number;
  succeeded: number;
  failed: number;
  added: number;
  updated: number;
  skipped: number;
  filtered: number;
  results: MemoryBatchWriteItemResult[];
}

export interface FolderTreeNode {
  folder: {
    id: string;
    parentId: string | null;
    title: string;
    sortOrder: number;
    isExpanded: boolean;
    createdAt: string;
    updatedAt: string;
  };
  children: FolderTreeNode[];
  items: Array<{
    id: string;
    folderId: string | null;
    itemType: string;
    itemId: string;
    sortOrder: number;
    createdAt: string;
  }>;
}

export async function getMemoryConfig(): Promise<MemoryConfig> {
  return nativeInvoke<MemoryConfig>('memory_get_config');
}

export async function setMemoryRootFolder(folderId: string): Promise<void> {
  return invoke('memory_set_root_folder', { folderId });
}

export async function setMemoryPrivacyMode(enabled: boolean): Promise<void> {
  return invoke('memory_set_privacy_mode', { enabled });
}

export async function setMemoryAutoCreateSubfolders(enabled: boolean): Promise<void> {
  return invoke('memory_set_auto_create_subfolders', { enabled });
}

export async function setMemoryDefaultCategory(category: string): Promise<void> {
  return invoke('memory_set_default_category', { category });
}

export async function setMemoryAutoExtractFrequency(frequency: AutoExtractFrequency): Promise<void> {
  return invoke('memory_set_auto_extract_frequency', { frequency });
}

export async function createMemoryRootFolder(title: string): Promise<string> {
  return invoke<string>('memory_create_root_folder', { title });
}

export async function searchMemory(
  query: string,
  topK?: number,
  folderPath?: string,
  context?: MemoryScopeContext
): Promise<MemorySearchResult[]> {
  return invoke<MemorySearchResult[]>('memory_search', { query, topK, folderPath, ...context });
}

export async function searchMemoryInFolderPaths(
  query: string,
  folderPaths: string[],
  topK?: number,
  context?: MemoryScopeContext
): Promise<MemorySearchResult[]> {
  return invoke<MemorySearchResult[]>('memory_search', { query, topK, folderPaths, ...context });
}

export async function readMemory(
  noteId: string,
  context?: MemoryScopeContext
): Promise<MemoryReadOutput | null> {
  return invoke<MemoryReadOutput | null>('memory_read', { noteId, ...context });
}

export async function writeMemory(
  title: string,
  content: string,
  folderPath?: string,
  mode?: 'create' | 'update' | 'append',
  context?: MemoryScopeContext
): Promise<MemoryWriteOutput> {
  return invoke<MemoryWriteOutput>('memory_write', {
    folderPath,
    title,
    content,
    mode,
    ...context,
  });
}

export async function listMemory(
  folderPath?: string,
  limit?: number,
  offset?: number,
  context?: MemoryScopeContext
): Promise<MemoryListItem[]> {
  return invoke<MemoryListItem[]>('memory_list', {
    folderPath,
    limit,
    offset,
    ...context,
  });
}

export async function listMemoryInFolderPaths(
  folderPaths: string[],
  limit?: number,
  offset?: number,
  context?: MemoryScopeContext
): Promise<MemoryListItem[]> {
  return invoke<MemoryListItem[]>('memory_list', {
    folderPaths,
    limit,
    offset,
    ...context,
  });
}

export async function getMemoryTree(
  folderPath?: string,
  context?: MemoryScopeContext
): Promise<FolderTreeNode | null> {
  const params = { folderPath, ...context };
  const hasParams = Object.values(params).some((value) => value !== undefined);
  return hasParams
    ? invoke<FolderTreeNode | null>('memory_get_tree', params)
    : invoke<FolderTreeNode | null>('memory_get_tree');
}

export async function addMemoryRelation(
  noteIdA: string,
  noteIdB: string,
  context?: MemoryScopeContext
): Promise<void> {
  return invoke('memory_add_relation', { noteIdA, noteIdB, ...context });
}

export async function removeMemoryRelation(
  noteIdA: string,
  noteIdB: string,
  context?: MemoryScopeContext
): Promise<void> {
  return invoke('memory_remove_relation', { noteIdA, noteIdB, ...context });
}

export async function getRelatedMemories(
  noteId: string,
  context?: MemoryScopeContext
): Promise<string[]> {
  return invoke<string[]>('memory_get_related', { noteId, ...context });
}

export async function updateMemoryTags(
  noteId: string,
  tags: string[],
  context?: MemoryScopeContext
): Promise<void> {
  return invoke('memory_update_tags', { noteId, tags, ...context });
}

export async function getMemoryTags(
  noteId: string,
  context?: MemoryScopeContext
): Promise<string[]> {
  return invoke<string[]>('memory_get_tags', { noteId, ...context });
}

export interface MemoryAnkiDocument {
  documentContent: string;
  memoryCount: number;
  documentName: string;
}

export async function memoryToAnkiDocument(
  folderPath?: string,
  purposeFilter?: string,
  limit?: number,
  context?: MemoryScopeContext,
  folderPaths?: string[]
): Promise<MemoryAnkiDocument> {
  return invoke<MemoryAnkiDocument>('memory_to_anki_document', {
    folderPath,
    folderPaths,
    purposeFilter,
    limit,
    ...context,
  });
}

export interface BatchOperationResult {
  total: number;
  succeeded: number;
  failed: number;
  errors: string[];
}

export async function batchDeleteMemories(
  noteIds: string[],
  context?: MemoryScopeContext
): Promise<BatchOperationResult> {
  return invoke<BatchOperationResult>('memory_batch_delete', { noteIds, ...context });
}

export async function batchMoveMemories(
  noteIds: string[],
  targetFolderPath: string,
  context?: MemoryScopeContext
): Promise<BatchOperationResult> {
  return invoke<BatchOperationResult>('memory_batch_move', { noteIds, targetFolderPath, ...context });
}

export async function moveMemoryToFolder(
  noteId: string,
  targetFolderPath: string,
  context?: MemoryScopeContext
): Promise<void> {
  return invoke('memory_move_to_folder', { noteId, targetFolderPath, ...context });
}

// ★ 修复风险2：按 note_id 更新记忆
export async function updateMemoryById(
  noteId: string,
  title?: string,
  content?: string,
  context?: MemoryScopeContext
): Promise<MemoryWriteOutput> {
  return invoke<MemoryWriteOutput>('memory_update_by_id', {
    noteId,
    title,
    content,
    ...context,
  });
}

// ★ 修复风险3：删除记忆
export async function deleteMemory(noteId: string, context?: MemoryScopeContext): Promise<void> {
  return invoke('memory_delete', { noteId, ...context });
}

export interface MemoryExportItem {
  title: string;
  content: string;
  folder: string;
  updatedAt: string;
}

export interface MemoryProfileSection {
  category: string;
  content: string;
}

export async function getMemoryProfile(
  folderPaths?: string[],
  context?: MemoryScopeContext
): Promise<MemoryProfileSection[]> {
  return invoke<MemoryProfileSection[]>('memory_get_profile', { folderPaths, ...context });
}

export async function exportAllMemories(
  folderPaths?: string[],
  context?: MemoryScopeContext
): Promise<MemoryExportItem[]> {
  return invoke<MemoryExportItem[]>('memory_export_all', { folderPaths, ...context });
}

export type MemoryPurposeType = 'internalized' | 'memorized' | 'supplementary' | 'systemic';

export async function writeMemorySmart(
  title: string,
  content: string,
  folderPath?: string,
  memoryType?: MemoryTypeValue,
  memoryPurpose?: MemoryPurposeType,
  scope?: MemoryScopeValue,
  idempotencyKey?: string,
  groupId?: string | null,
  groupName?: string | null
): Promise<SmartWriteOutput> {
  return invoke<SmartWriteOutput>('memory_write_smart', {
    folderPath,
    scope,
    title,
    content,
    memoryType,
    memoryPurpose,
    idempotencyKey,
    groupId,
    groupName,
  });
}

export async function writeMemoryBatch(
  items: MemoryBatchWriteItemInput[],
  defaultFolderPath?: string,
  defaultMemoryType?: MemoryTypeValue,
  defaultMemoryPurpose?: MemoryPurposeType,
  defaultScope?: MemoryScopeValue,
  groupId?: string | null,
  groupName?: string | null
): Promise<MemoryBatchWriteOutput> {
  return invoke<MemoryBatchWriteOutput>('memory_write_batch', {
    items,
    defaultFolderPath,
    defaultScope,
    defaultMemoryType,
    defaultMemoryPurpose,
    groupId,
    groupName,
  });
}

export interface MemoryAuditLogItem {
  id: number;
  timestamp: string;
  source: string;
  operation: string;
  success: boolean;
  noteId: string | null;
  title: string | null;
  contentPreview: string | null;
  folder: string | null;
  event: string | null;
  confidence: number | null;
  reason: string | null;
  sessionId: string | null;
  durationMs: number | null;
}

export async function getMemoryAuditLogs(params?: {
  limit?: number;
  offset?: number;
  sourceFilter?: string;
  operationFilter?: string;
  successFilter?: boolean;
}): Promise<MemoryAuditLogItem[]> {
  return invoke<MemoryAuditLogItem[]>('memory_get_audit_logs', params ?? {});
}
