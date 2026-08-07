// 索引系统类型与 Wails API 封装（P0-B）。
//
// 与后端 pkg/index 类型一一对应，JSON 字段使用 camelCase。

import { callWails } from "@/lib/wails";

// ===== 类型定义 =====

export interface Chunk {
  id: number;
  uri: string;
  pos: number;
  content: string;
  tokenCount: number;
  embedding?: number[];
  createdAt: string;
}

export interface IndexOptions {
  chunkSize: number;
  chunkOverlap: number;
  minChunkSize: number;
  embed: boolean;
  embedModel: string;
}

export interface SearchQuery {
  query: string;
  types?: string[];
  limit: number;
  minScore: number;
  useFts: boolean;
  useVector: boolean;
  useRerank: boolean;
}

export interface SearchResult {
  uri: string;
  type: string;
  title: string;
  snippet: string;
  score: number;
  ftsScore?: number;
  vecScore?: number;
  chunkPos?: number;
}

export interface IndexStats {
  totalResources: number;
  indexedResources: number;
  totalChunks: number;
  embeddedChunks: number;
  ftsRows: number;
  avgChunkTokens: number;
}

export interface IndexTask {
  uri: string;
  status: "pending" | "indexing" | "embedded" | "done" | "error";
  progress: number;
  error?: string;
  startedAt: number;
}

export interface BatchIndexResult {
  total: number;
  success: number;
  failed: number;
  skipped: number;
  tasks?: IndexTask[];
  errors?: IndexError[];
}

export interface IndexError {
  uri: string;
  error: string;
}

// ===== API 封装 =====

export async function fetchIndexStats(): Promise<IndexStats | null> {
  return callWails<IndexStats>("IndexStats");
}

export async function indexResource(
  uri: string,
  content: string,
  opts: IndexOptions
): Promise<boolean> {
  const err = await callWails<string>("IndexResource", uri, content, opts);
  return err === null || err === "";
}

export async function indexBatch(
  uris: string[],
  contents: string[],
  opts: IndexOptions
): Promise<BatchIndexResult | null> {
  return callWails<BatchIndexResult>("IndexBatch", uris, contents, opts);
}

export async function deleteIndex(uri: string): Promise<boolean> {
  const err = await callWails<string>("IndexDelete", uri);
  return err === null || err === "";
}

export async function rebuildAllIndexes(
  opts: IndexOptions
): Promise<BatchIndexResult | null> {
  return callWails<BatchIndexResult>("IndexRebuildAll", opts);
}

export async function indexSearch(
  q: SearchQuery
): Promise<SearchResult[] | null> {
  return callWails<SearchResult[]>("IndexSearch", q);
}

export async function indexRAGQuery(
  query: string,
  topK: number
): Promise<SearchResult[] | null> {
  return callWails<SearchResult[]>("IndexRAGQuery", query, topK);
}

export async function getIndexTask(uri: string): Promise<IndexTask | null> {
  return callWails<IndexTask>("IndexGetTask", uri);
}

export async function fetchDefaultIndexOptions(): Promise<IndexOptions | null> {
  return callWails<IndexOptions>("IndexDefaultOptions");
}

// ===== 默认选项 =====

export function defaultIndexOptions(): IndexOptions {
  return {
    chunkSize: 800,
    chunkOverlap: 100,
    minChunkSize: 50,
    embed: false,
    embedModel: "",
  };
}

export function defaultSearchQuery(query: string): SearchQuery {
  return {
    query,
    limit: 10,
    minScore: 0,
    useFts: true,
    useVector: true,
    useRerank: false,
  };
}
