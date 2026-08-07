// Cloud Storage / 同步前端类型与 Wails 封装
// ------------------------------------------------------------
// 对接后端 cloudstorage（Cloud* 方法）+ sync（Sync* 方法）。

import { callWails } from "@/lib/wails";

/** 云存储配置 —— 与后端 cloudstorage.Config 对齐 */
export interface CloudConfig {
  provider: "webdav" | "s3";
  webdavUrl?: string;
  webdavUser?: string;
  webdavPass?: string;
  s3Endpoint?: string;
  s3Region?: string;
  s3Bucket?: string;
  s3AccessKey?: string;
  s3SecretKey?: string;
  remoteDir?: string;
}

/** 云端备份版本 —— 与后端 cloudstorage.Version 对齐 */
export interface CloudVersion {
  key: string;
  size: number;
  created: string;
  checksum: string;
  note?: string;
}

/** 同步状态 —— 与后端 SyncGetStatus 返回对齐 */
export interface SyncStatus {
  cursor: number;
  maxSeq: number;
  pending: number;
  quarantine: number;
  cloud: {
    connected: boolean;
    deviceId?: string;
    remoteDir?: string;
    versionCount?: number;
    error?: string;
  };
}

/** 隔离区记录 —— 与后端 sync.QuarantineEntry 对齐 */
export interface QuarantineEntry {
  id: number;
  table: string;
  recordId: string;
  reason: string;
  payload: string;
  createdAt: string;
}

export const cloudApi = {
  saveConfig: (cfg: CloudConfig) => callWails<void>("CloudSaveConfig", cfg),
  loadConfig: () => callWails<[CloudConfig, boolean]>("CloudLoadConfig"),
  clearConfig: () => callWails<void>("CloudClearConfig"),
  checkConnection: () => callWails<void>("CloudCheckConnection"),
  uploadBackup: (localPath: string, note: string) =>
    callWails<CloudVersion>("CloudUploadBackup", localPath, note),
  downloadLatest: () => callWails<string>("CloudDownloadLatest"),
  downloadVersion: (key: string) => callWails<string>("CloudDownloadVersion", key),
  listVersions: () => callWails<CloudVersion[]>("CloudListVersions"),
  getStatus: () => callWails<Record<string, unknown>>("CloudGetStatus"),
  deleteVersion: (key: string) => callWails<void>("CloudDeleteVersion", key),
};

export const syncApi = {
  run: () => callWails<{ uploaded: number; downloaded: number; quarantined: number }>("SyncRun"),
  pending: () => callWails<number>("SyncPending"),
  status: () => callWails<SyncStatus>("SyncGetStatus"),
  listQuarantine: (limit = 100) =>
    callWails<QuarantineEntry[]>("SyncListQuarantine", limit),
  retryQuarantine: (id: number) => callWails<void>("SyncRetryQuarantine", id),
  discardQuarantine: (id: number) => callWails<void>("SyncDiscardQuarantine", id),
  discardAllQuarantine: () => callWails<number>("SyncDiscardAllQuarantine"),
};
