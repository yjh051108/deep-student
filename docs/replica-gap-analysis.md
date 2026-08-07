# Go 复刻差距分析 — deep-student-go vs 云端最新（v0.9.43 + main）

> **更新（2026-08-07）**：本分析为初始基准。自分析以来，以下模块已按功能等价复刻完成：
> A1 chat_v2 会话管理/持久化/工具循环 · A2 增量同步 · A3 Memory-as-VFS · A4 OCR ·
> A5 Multimodal · A6 Todo · A7 Pomodoro · A8 Cloud storage · A9 模板管理 · A10 语音输入 ·
> A11 LLM 用量 · A13(部分) FSRS/插件/快速助手 · B1 i18n · B3 沙盒 · B5 前端导航。
> 详见 [replica-roadmap.md](replica-roadmap.md)。

> 基准：`helixnow/deep-student` v0.9.43 + main（HEAD `cabb8afe`，2026-08-04）
> 对照对象：`deep-student-go`（Wails v2，Go 1.25，纯 Go SQLite `modernc.org/sqlite`）
> 复刻口径：**功能等价**（数据模型对齐 + 后端命令齐全 + 前端页面可用 + 测试通过）

---

## 一、规模对比

| 维度 | Rust 原版（云端最新） | Go 复刻版 | 备注 |
|---|---|---|---|
| 后端 | 395+ `.rs`，约 35.6 万行 | 89 `.go`，约 1.9 万行 | Go 为精选重建 |
| 前端 | TS/TSX 约 7.8 万行（v0.9.43 再增 78.7 万行） | 80 文件，约 1.8 万行 | 16 页面 |
| RPC 命令 | 130+ 命令（`commands.rs` 5817 行）+ `cmd/` 9 域 | 117 个 Wails 方法 | — |
| 数据库 | 4 组迁移 60+ 文件（chat_v2/llm_usage/mistakes/vfs），v0.9.43 增 browser 库 | 无迁移目录，内联 `CREATE TABLE IF NOT EXISTS`（7 张表） | Rust 表数远超 Go |
| 语言 | Rust + React/TS + Tauri | Go + React/TS + Wails | — |

---

## 二、模块级差距（Go 缺失 → 需复刻）

### A. 后端能力缺失（复杂度：中/复杂）

| # | 模块 | Rust 位置 | 核心命令 | 数据表 | 复杂度 |
|---|---|---|---|---|---|
| A1 | **chat_v2 完整流水线** | `src-tauri/src/chat_v2/`（pipeline/handlers/tools/workspace） | `chat_v2_send_message` 等约 80 个 | `chat_v2_sessions/messages/blocks/attachments/...` 14+ 张 | 复杂（最大单项） |
| A2 | **data_governance 增量同步** | `data_governance/sync/`（SyncManager 5000+ 行） | `data_governance_run_sync`、`detect_conflicts` 等 50+ | `__change_log`（各库触发器）、`__blob_deletion_queue` | 复杂 |
| A3 | **Memory 记忆-as-VFS** | `src-tauri/src/memory/`（service/handlers/auto_extractor） | `memory_*` 约 35 个 | `memory_config`、`memory_audit_log`、`memory_write_idempotency` | 复杂 |
| A4 | **OCR** | `ocr_adapters/` + `pdf_ocr_service.rs` + `cmd/ocr.rs` | `get_ocr_engines`、`init_pdf_ocr_session` 等 20+ | OCR 结果入 VFS blob/segments | 复杂 |
| A5 | **Multimodal** | `multimodal/`（embedding_service/page_indexer）+ LanceDB | `vfs_multimodal_*` | `vfs_embedding_dims/segments/units`、`vectorized_data` | 复杂 |
| A6 | **Todo 待办** | `vfs/todo_handlers.rs` | `todo_*` 约 30 个 | `todo_lists`、`todo_items` | 中 |
| A7 | **Pomodoro 番茄钟** | `vfs/todo_handlers.rs`（pomodoro 段） | `pomodoro_*` 约 7 个 | `pomodoro_records` | 中 |
| A8 | **Cloud storage** | `cloud_storage/`（webdav/s3/ftp/sync_manager） | `cloud_storage_*`、`cloud_sync_*`、`secure_*` | 无专用表（配置入 settings/钥匙串） | 中 |
| A9 | **Template-management** | `commands.rs`（`*_custom_template` 段） | `get_all_custom_templates` 等 12 个 | `custom_anki_templates` | 中 |
| A10 | **Voice-input** | `voice_input.rs` | `voice_input_transcribe` | 无专用表（记入 llm_usage） | 简单 |
| A11 | **LLM 用量统计** | `llm_usage/` | `llm_usage_*` 8 个 | `llm_usage_logs`、`llm_usage_daily` | 中 |
| A12 | **Review 复习计划（SM-2）** | `review_plan_service.rs` | `review_*` 15 个 | `review_plans/sessions/history/analyses/stats` 等 10+ | 中 |
| A13 | **v0.9.43 新增后端**：browser / openai_codex / mastery / plugins / fsrs_review / quick_assistant / apkg_importer | `src-tauri/src/{browser,openai_codex,mastery,plugins}/` 等 | `cmd/{browser,openai_codex,fsrs_review,media,network}.rs` | `browser.db`（sessions/history/downloads/site_permissions/settings）、`fsrs_card_states`、`fsrs_review_logs`、`anki_decks`、`mastery_events/states` | 复杂 |

### B. 前端能力缺失（复杂度：简单/中）

| # | 模块 | Rust 位置 | 说明 | 复杂度 |
|---|---|---|---|---|
| B1 | **i18n 双语言** | `src/locales/{en-US,zh-CN}/` + `i18n.ts` | i18next 懒加载，40 命名空间，fallback en-US | 中 |
| B2 | **Command-palette** | `src/command-palette/` | 全局快捷键命令面板，纯前端 | 中 |
| B3 | **Sandbox 沙盒** | `src/features/sandbox/` | 代码预览/沙盒运行（iframe），纯前端 | 简单 |
| B4 | **Pomodoro 前端** | `src/features/pomodoro/` | GlobalPomodoroWidget/PomodoroPanel/ImmersiveFocusMode + 白噪音 | 中 |
| B5 | **Todo 前端** | `src/features/todo/` | useTodoStore + quickAddParser（自然语言）+ reminderScheduler | 中 |
| B6 | **Voice-input 前端** | `src/voice-input/` | controller/audio/providerRegistry/modelSelection | 中 |
| B7 | **Template editor** | `src/components/TemplateEditor*` | 字段管理/插入栏/Lint/预览 | 中 |
| B8 | **v0.9.43 新增前端** | `src/features/{workbench,browser,flashcards,anki-tasks,data-recovery}/`、`src/quick-assistant/` | 学习工作台/内置浏览器/FSRS 闪卡/Anki 任务/数据恢复中心/快速助手 | 复杂 |

---

## 三、Go 版已有但需增强的模块

| 模块 | Go 现状 | Rust 版能力 | 需增强点 |
|---|---|---|---|
| **chat** | 单轮流式、无工具循环（9 方法） | chat_v2 完整流水线 | 见 A1（整体重写） |
| **memory** | 内存 map + Extract/Decide/Search/Decay（6 方法） | 记忆-as-VFS 35 命令 | 见 A3（整体重写） |
| **governance** | 备份/恢复/导入导出/基础 AuditLogs（9 方法） | 增量同步 + 冲突 + tombstone + 隔离区 | 见 A2（大幅扩展） |
| **qbank** | 抽题/练习/阅卷/掌握度（内存） | 题库 + Review SM-2 复习计划 + 错题追踪 | 持久化 + A12 |
| **anki** | 制卡/模板/保存（内存） | APKG 导入导出 + FSRS + 模板编辑器 | 持久化 + A13 部分 |
| **llmcfg** | 10 厂商 CRUD + assignment | llm_manager 10 vendor + 13 RequestAdapter + 路由 | adapters 层（anthropic/ernie/gemini/grok/mistral 等）|
| **skills** | 基础 skill + MCP | 技能文件系统（skill_* 命令）+ 技能市场 | 命令补全 |
| **vfs/index** | vfs:// URI + FTS5 + 向量 | VFS 统一索引 + multimodal + LanceDB | 持久化 entries + A5 |

---

## 四、Go 版独有（Rust 版没有，保留）

- **深度调研**（`internal/research`：ResearchPlan/Run，可取消/断点续跑）
- **论文检索**（`internal/paper`：arXiv/OpenAlex/DOI/BibTeX/APA/GB7714）
- **Index RAG**（`pkg/index`：FTS5 trigram 中文子串 + 向量混合检索 + 重排序）

---

## 五、基础设施差异（复刻时需对齐）

| 项 | Rust 原版 | Go 版 | 影响 |
|---|---|---|---|
| 数据库 | 4+1 个独立 SQLite（chat_v2/llm_usage/mistakes/vfs/browser）+ LanceDB | 单库 + blob 目录 | 同步/治理按库设计，Go 需重构为多库或按表建模 |
| 迁移 | Refinery + migration-lock.json | 内联 CREATE IF NOT EXISTS | 复刻 schema 时需引入迁移机制 |
| 持久化 | 全部落盘 | chat/qbank/anki/memory/vfs entries 为内存 map（**重启丢失**） | 批次 1 优先解决 |
| 事件 | 结构化事件流 | eventbus 只 publish 无 subscribe | 需接消费端 |
| 云同步 | 记录级增量 + 冲突合并 | 无 | 批次 2 |
| LLM 路由 | llm_manager 路由 + 余额徽章 | 硬编码 gpt-4o-mini | 需接 llmcfg assignment |
