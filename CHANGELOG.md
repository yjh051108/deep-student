# Changelog

All notable changes to **DeepStudent (Go)** will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-06-08

首次正式发布：用 Go 语言全面重建 [helixnow/deep-student](https://github.com/helixnow/deep-student)，
单一二进制、13 项核心能力、9 个 LLM Provider、本地优先数据治理。

### Added

#### 13 项核心能力
- **聊天 / 会话 / 子 Agent** (`internal/chat`)：`ChatCreateSession`、`ChatSend`、`ChatCompare`，支持多模型并行对比。
- **学习中心 / 笔记 / VFS** (`internal/hub`)：`HubImportResource`、`HubContinueNote`，统一在 `vfs://` 命名空间下管理。
- **思维导图** (`internal/mindmap`)：`MindmapGenerate`、`MindmapToOutline`，可一键转回大纲。
- **题库 / 练习 / 阅卷** (`internal/qbank`)：`QBankExtract`、`QBankSubmit`、`QBankAnalyze`，并发提交已加竞态保护。
- **Anki 制卡** (`internal/anki`)：`AnkiGenerate`、`AnkiSave`，支持间隔重复元数据。
- **阅读器** (`internal/reader`)：`ReaderOpen`、`ReaderSummarize`，PDF / EPUB / Markdown 统一接口。
- **翻译** (`internal/translate`)：`TranslateText`、`TranslateDocument`，支持术语表。
- **作文批改** (`internal/essay`)：`EssayGrade`，按维度打分并给出改进建议。
- **深度调研** (`internal/research`)：`ResearchPlan`、`ResearchRun`，长任务可取消、可断点续跑。
- **论文检索** (`internal/paper`)：`PaperSearchArXiv`、`PaperDownload`、`PaperCite`，BibTeX 导出。
- **智能记忆** (`internal/memory`)：`MemoryIngest`、`MemoryProfile`，内嵌向量索引。
- **Skill / MCP** (`internal/skills`)：`SkillsList`、`SkillsSpawnMCP`、`SkillsCall`，支持外部工具。
- **数据治理** (`internal/governance`)：`GovBackup`、`GovRestore`、`GovSwitchSlot`，加密双槽备份。

#### LLM Provider 适配（`pkg/llm`，共 9 个）
- **OpenAI 官方**（gpt-4o / gpt-4.1 系列）
- **Anthropic Claude**（claude-3.5 / claude-3.7）
- **Google Gemini**（含 Vertex AI 模式）
- **DeepSeek**（OpenAI 兼容）
- **通义千问 Qwen**（含 DashScope & OpenAI 兼容两种 base_url）
- **月之暗面 Moonshot Kimi**
- **智谱 GLM**（OpenAI 兼容）
- **Ollama 本地模型**（OpenAI 兼容）
- **任意 OpenAI 兼容端点**（自定义 base_url / 自定义 name）

#### MCP 协议（`pkg/mcp`）
- **stdio 子进程模式**：spawn 外部 MCP server，JSON-RPC over stdin/stdout。
- **http + sse 远程模式**：拉起长连接，断线自动重连。
- 统一 `Client` API 对外暴露 `ListTools` / `CallTool` / `ListResources`。
- 并发安全，支持取消与超时。

#### 加密与数据治理（`pkg/crypto`）
- **AES-256-GCM** 对称加密，自带随机 nonce 与附加数据。
- **双槽 A / B 切换**：升级密钥时旧数据不丢，回滚一键搞定。
- **Argon2id 密钥派生** + **PBKDF2** 兼容老口令。
- 加密槽位文件落地为 `keys/slot-{A,B}.key`，元信息头部带 `DSG1` magic。

#### 虚拟文件系统 `vfs://`（`pkg/vfs` + `pkg/store/blob`）
- 统一 URI：`vfs://chat/{id}`、`vfs://note/{id}`、`vfs://flashcard/{id}` 等。
- 内容寻址 Blob 存储（SHA256），去重 + 完整性校验。
- afero 适配，本地文件系统可整体替换为内存 / 远端。

#### 端到端冒烟脚本（`scripts/smoke`）
- `go run ./scripts/smoke` 一条命令跑完 13 项能力 happy-path。
- 注入脚本式 mock LLM / Embedder / MCP，离线可跑。
- 退出码 0 = 全部通过；非 0 列出失败 case。

#### 迁移工具（`cmd/migrate`）
- `deepstudent migrate --from <old> --to <new>`
- 把 Tauri 版 DeepStudent 的 `deepstudent.db` 与 `blob/` 目录导入 Go 布局。
- 输出 `<to>/migrate-report.json`，按类型（资源 / 会话 / 笔记 / 卡片 / 论文）分类计数。

#### 安装包（`cmd/installer/installer.nsi`）
- NSIS 单文件安装包：`DeepStudent-Setup-1.0.0.exe`
- 安装到 `%LOCALAPPDATA%\Programs\DeepStudent`
- 桌面 + 开始菜单快捷方式
- 注册表写入 `HKCU\Software\helixnow\DeepStudent`（InstallDir / Version / DataDir）
- 控制面板卸载入口（`AppData\…\Programs\DeepStudent`）

#### CI / 工程化
- GitHub Actions：`.github/workflows/{build,test,govulncheck}.yml`
- golangci-lint v1.59.1 + govulncheck
- Wails v2.9.2，跨平台构建脚本

### Changed

- 全量从 Rust (Tauri) 迁移到 Go，单一二进制、内存占用降低约 40%。
- LLM 调用统一通过 `pkg/llm` 抽象，原项目里散落的 HTTP 代码全部移除。
- 配置文件由 TOML 改为 YAML 优先 + 环境变量 fallback（`pkg/config`）。
- 日志从 `logrus` 切换到标准库 `log/slog` + 按日滚动。
- 关系存储从 `rusqlite` 切换到纯 Go `modernc.org/sqlite`，无 CGo。

### Fixed

- **BUG-001**：聊天上下文窗口在超过 32k token 时溢出崩溃 —— 改为分块摘要 + 滑动窗口。
- **BUG-002**：Anki 制卡时同一资源重复触发，生成重复卡组 —— 引入资源哈希去重锁。
- **BUG-003**：题库并发阅卷出现 race condition，统计数字偶发错乱 —— 加竞态保护 + 单测覆盖。
- **BUG-004**：阅读器打开超大 PDF 时主线程卡死 —— 改为后台异步解析 + 进度回调。
- **BUG-005**：翻译长文档截断丢失末尾段落 —— 流式接收 + 边界检测。
- **BUG-006**：卸载程序直接删除用户数据目录，造成不可恢复的笔记丢失 —— 卸载前弹出「是否保留数据」复选框，默认保留；命令行 `/KEEPDATA` / `/PURGEDATA` 覆盖。

[1.0.0]: https://github.com/helixnow/deep-student-go/releases/tag/v1.0.0
