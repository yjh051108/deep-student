// 此文件由 Go 重构版适配：原版 scripts/generate-version.mjs 自动生成
//（原版依赖 src-tauri 配置，Go 版直接维护本文件）
export const VERSION_INFO = {
  APP_VERSION: '0.9.43', // 应用主版本号（对齐原版云端最新版）
  BUILD_NUMBER: '0.9.43.0', // 内部版本号
  GIT_HASH: 'go-replica', // Git commit short hash
  FULL_VERSION: '0.9.43 (go-replica)', // 完整版本号
  SENTRY_RELEASE: 'deepstudent@0.9.43', // Sentry release 标识
} as const;

export default VERSION_INFO;
