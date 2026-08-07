# Go 复刻路线图 — flow/go-replica

> 目标：以**功能等价**为口径，将云端最新 DeepStudent（Rust/Tauri v0.9.43 + main）
> 的每一项能力在 Go 复刻版（Wails v2）中实现。
> 复刻口径：功能等价（数据模型对齐 + 后端命令齐全 + 前端页面可用 + 测试通过）。

## 总体状态

- **复刻基准**：`helixnow/deep-student` v0.9.43 + main（HEAD `cabb8afe`，2026-08-04）
- **Go 版现状**：5 个批次全部完成（见下），smoke 冒烟 25 组全绿，41 个包测试通过
- **分支**：`flow/go-replica`（fork yjh051108/deep-student）

---

## 批次 1 — 持久化地基 + 轻量模块 ✅ 已完成（f95cd1f）

- [x] **Obsidian 式混合 VFS**（`pkg/vault` + `pkg/vfs` 改造）：内容类资源以真实 .md
      （YAML frontmatter ds_id/ds_type）或原始文件落盘到用户可见 vault（默认
      `~/Documents/DeepStudent`），可直接用 Obsidian 打开；非 md 资源 sidecar 元数据；
      扫描重建索引（文件为准，重启不丢）；blob→vault 一次性迁移；`DEEPSTUDENT_VAULT` 可配
- [x] **Todo 待办**（`todo_lists` / `todo_items`）：列表 / 子任务 / 优先级 / 截止 / 重复 /
      回收站 / AI 拆解（28 RPC）
- [x] **Pomodoro 番茄钟**（`pomodoro_records`）：记录 / 今日统计 / 近 7 天（7 RPC）
- [x] **LLM 用量统计**（`llm_usage_logs` / `llm_usage_daily`）：调用日志 + 日聚合（5 RPC）
- [x] 前端：TodoPage / PomodoroPage / LLMUsagePage + 路由 + Sidebar + zustand stores

## 批次 2 — 云与治理 ✅ 已完成（e9a7c6f）

- [x] **Cloud storage**（`internal/cloudstorage`）：WebDAV + S3（SigV4 纯标准库）后端，
      加密 ZIP 版本同步（CloudManifest 版本列表 / device_id），凭据 AES 加密入库（10 RPC）
- [x] **data_governance 增量同步**（`internal/sync`）：`__change_log` 触发器（6 张表）、
      记录级增量导出/应用、LWW 冲突（本地较新进隔离区）、tombstone 软删传播、
      quarantine 重试/丢弃、SyncToCloud（11 RPC）
- [x] **Template-management 模板管理**（`internal/templatemgr`）：CRUD / 导出导入 /
      批量导入 / 内置模板 seed / 默认模板（11 RPC）
- [x] **Voice-input 语音输入**（`internal/voiceinput`）：ASR 转写（OpenAI 兼容端点，
      默认 SiliconFlow）（3 RPC）

## 批次 3 — 智能与多模态 ✅ 已完成（51e6fd3）

- [x] **Memory 记忆-as-VFS**（`internal/memory` 升级）：SQLite 持久化（重启不丢）、
      文件夹树（可嵌套）、双向关系、审计日志、批量写、标签、配置（17 RPC）
- [x] **OCR 多引擎**（`internal/ocr`）：DeepSeek-VL（多模态 API）/ 系统 OCR /
      Paddle 占位，引擎列表与切换、PDF OCR 会话流程、PDF 文本层提取（9 RPC）
- [x] **Multimodal**（`internal/multimodal`）：资源切块 + 嵌入 + 向量索引
      （复用 pkg/vector）+ 关键词/向量混合检索（4 RPC）

## 批次 4 — chat_v2 完整流水线 ✅ 已完成（098691a）

- [x] **会话持久化**（chat_v2_groups/sessions/messages/tags，对齐上游）：重启不丢
- [x] **会话管理**：列表 / 标题 / 置顶 / 软删除 / 回收站 / 恢复 / 彻底删除 / 标签 /
      搜索 / 计数 / 单条消息删除（19 RPC）
- [x] **工具循环**：RegisterTool / Tools / SendWithTools（最多 5 轮工具调用 +
      记录回灌）、LLM 工具声明、分组 CRUD v2
- [x] 多变体并行比较、会话 JSON 导出

## 批次 5 — 前端与 v0.9.43 新增面 ✅ 已完成（33312cf，待推送）

- [x] **FSRS 闪卡间隔复习**（`internal/fsrs`）：FSRS-6 调度（稳定性/难度/到期）、
      评分 1-4、复习日志、牌组统计（9 RPC）
- [x] **插件生态**（`internal/plugins`）：受管插件安装/列表/启用/卸载/扫描 vault
      （5 RPC）
- [x] **快速助手 Quick Assistant**（`internal/quickassist`）：轻量问答（4 RPC）
- [x] **Sandbox 沙盒**（前端 SandboxPage）：iframe safe-preview，HTML/CSS/JS 编辑
- [x] **i18n 双语言**（前端 lib/i18n）：en-US / zh-CN，设置页切换
- [x] 命令面板扩展（sandbox/todo/pomodoro/llm-usage 命令）

---

## 完成度对照（roadmap 原始目标）

| 原计划批次 | 内容 | 状态 |
|---|---|---|
| 批次 1 | 持久化地基 + Todo/Pomodoro/LLM用量 + Obsidian VFS | ✅ |
| 批次 2 | Cloud storage / 增量同步 / 模板管理 / 语音输入 | ✅ |
| 批次 3 | Memory-as-VFS / OCR / Multimodal | ✅ |
| 批次 4 | chat_v2 完整流水线 | ✅ |
| 批次 5 | i18n / 命令面板 / 沙盒 / FSRS / 插件 / 快速助手 | ✅ |

**工程指标**：41 个 Go 包测试全绿 · `go vet` 干净 · 前端 `tsc` + `vite build` 通过 ·
冒烟 `go run ./scripts/smoke` 25 组能力断言全过。
