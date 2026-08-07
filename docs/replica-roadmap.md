# Go 复刻路线图 — flow/go-replica

> 目标：以**功能等价**为口径，将云端最新 DeepStudent（Rust/Tauri v0.9.43 + main）
> 的每一项能力在 Go 复刻版（Wails v2）中实现。
> 每个批次完成标准：`go test ./...` 全绿 + 冒烟扩展 + 前端页面可用 + 提交 push 到本分支。

## 总体状态

- **复刻基准**：`helixnow/deep-student` v0.9.43 + main（HEAD `cabb8afe`，2026-08-04）
- **Go 版现状**：13 项核心能力已实现；以下路线图列出**缺失/待补齐**的模块
- **口径**：功能等价（数据模型对齐 + 后端命令齐全 + 前端页面可用 + 测试通过），非逐行翻译

---

## 批次 1 — 持久化地基 + 轻量模块

**目的**：为全项目去除"重启丢数据"风险（当前 chat/qbank/anki/memory/vfs 索引均为进程内 map）。

- [ ] 存储层改造：chat / qbank / anki / memory / vfs 索引从内存 map 迁至 SQLite（对齐 Rust 表结构）
- [ ] **Todo 待办**（`todo_lists` / `todo_items`）：列表 / 子任务 / 优先级 / 截止 / 重复 / 回收站 / AI 拆解（约 30 命令）
- [ ] **Pomodoro 番茄钟**（`pomodoro_records`）：记录 / 今日统计 / 沉浸专注模式 / 白噪音（约 7 命令）
- [ ] **LLM 用量统计**（`llm_usage_logs` / `llm_usage_daily`）：调用日志 + 每日聚合（约 8 命令）

## 批次 2 — 云与治理

- [ ] **Cloud storage**（WebDAV / S3 / FTPS）：统一存储接口 + 加密 ZIP 版本同步（含 `cloud_sync_*` 命令）
- [ ] **data_governance 增量同步**：`__change_log` 触发器、字段级冲突合并（HLClock 定序）、tombstone 删除传播、隔离区（quarantine）
- [ ] **Template-management 模板管理**：Anki 自定义模板 CRUD / 导入导出 / 内置模板
- [ ] **Voice-input 语音输入**：录音 → ASR 转写（多 provider）

## 批次 3 — 智能与多模态

- [ ] **Memory 记忆-as-VFS**：文件夹体系 / 关系 / 审计日志 / 自动提取 / 可转 Anki（约 35 命令）
- [ ] **OCR**（多引擎）：DeepSeek-VL / PaddleOCR / 系统 OCR + PDF 整卷识别流水线
- [ ] **Multimodal**：图文/视频嵌入、VL reranker、向量检索（对齐 Go 版 `pkg/vector` 扩展）

## 批次 4 — chat_v2 完整流水线（最大单项）

- [ ] 块式消息渲染（`chat_v2_messages` + `chat_v2_blocks`）
- [ ] 工具循环 + 40+ ToolPack executor（检索/会话/知识/记忆/附件/canvas/ask_user/todo/skills/子Agent/题库/Anki/论文/文档/图片生成/工作区…）
- [ ] 多变体并行（`parallel_model_ids`）
- [ ] 会话分组 / 软删除 / 回收站 / 分支 / 变体 / 标签 / 全文搜索 / 压缩摘要
- [ ] 工作区多 Agent 协作（coordinator / subagent / sleep）
- [ ] Ask-User 交互 + 敏感工具审批
- [ ] 旧 chat 迁移（`chat_v2_migrate_legacy_chat`）

## 批次 5 — 前端与 v0.9.43 新增面

- [ ] **i18n 双语言**（en-US / zh-CN，i18next 懒加载 40 命名空间）
- [ ] **Command-palette 命令面板**（全局快捷键）
- [ ] **Sandbox 沙盒工作台**（代码预览 / 沙盒运行，iframe 模式）
- [ ] **Workbench 学习工作台**（多窗口 / 壁纸 / 快捷键 / 快速助手 Quick Assistant）
- [ ] **内置浏览器**（browser.db：sessions / history / downloads / site_permissions）
- [ ] **FSRS 闪卡 / APKG**（`fsrs_card_states` / `fsrs_review_logs`、APKG 导入导出、模板编辑器、统计图表）
- [ ] **Codex 账号管理**（openai_codex：manager / protocol / store）
- [ ] **插件生态**（plugins：受管扩展 + iLink Bot）

---

## 已完成

- [x] 基建：fork 同步至云端最新（`cabb8afe`）、`flow/go-replica` 分支、Go 工程入树、gap-analysis 文档
- [x] Go 版原有 13 项核心能力（chat/hub/mindmap/qbank/anki/reader/translate/essay/research/paper/memory/skills/governance + notes/llmcfg/index）全部测试通过
