# DeepStudent (Go) — 本地优先 AI 学习工作台

[![Release v1.0.0](https://img.shields.io/github/v/release/helixnow/deep-student-go?label=Release&style=flat-square)](https://github.com/helixnow/deep-student-go/releases/tag/v1.0.0)
[![License: AGPL-3.0](https://img.shields.io/github/license/helixnow/deep-student-go?style=flat-square)](./LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/helixnow/deep-student-go?style=flat-square)](https://goreportcard.com/report/github.com/helixnow/deep-student-go)
[![Go Version](https://img.shields.io/github/go-mod/go-version/helixnow/deep-student-go?style=flat-square)](./go.mod)
[![Platform](https://img.shields.io/badge/platform-Windows%2010%2F11-0078d4?style=flat-square)](https://github.com/helixnow/deep-student-go/releases)

> [helixnow/deep-student](https://github.com/helixnow/deep-student) 的 Go 语言全面重建版。
> 单一二进制 · 13 项核心能力 · 9 个 LLM Provider · 本地优先数据治理。

---

## 下载（v1.0.0）

请到 [GitHub Releases](https://github.com/helixnow/deep-student-go/releases) 页面下载：

| 文件 | 说明 | 链接 |
|---|---|---|
| `DeepStudent-Setup-1.0.0.exe` | NSIS 单文件安装包（Windows 10/11 x64）— **推荐** | [下载](https://github.com/helixnow/deep-student-go/releases/download/v1.0.0/DeepStudent-Setup-1.0.0.exe) |
| `deepstudent.exe`           | 免安装绿色版（仅二进制）                       | [下载](https://github.com/helixnow/deep-student-go/releases/download/v1.0.0/deepstudent.exe) |
| `SHA256SUMS.txt`            | 上述所有发布产物的 SHA-256 校验和              | [下载](https://github.com/helixnow/deep-student-go/releases/download/v1.0.0/SHA256SUMS.txt) |

> 安装包约 115 MB；绿色版二进制约 95 MB。

### 校验下载

```powershell
Get-FileHash .\DeepStudent-Setup-1.0.0.exe -Algorithm SHA256
# 与 SHA256SUMS.txt 中的值比对
```

macOS / Linux：

```bash
shasum -a 256 DeepStudent-Setup-1.0.0.exe
# 或
sha256sum DeepStudent-Setup-1.0.0.exe
```

v1.0.0 的完整变更列表见 [`CHANGELOG.md`](./CHANGELOG.md)。

---

## 界面预览

| 聊天 | 思维导图 | 题库 |
|:---:|:---:|:---:|
| ![Chat](docs/screenshots/chat.png) | ![Mindmap](docs/screenshots/mindmap.png) | ![QBank](docs/screenshots/qbank.png) |

| 阅读器 | 设置 | 安装向导 |
|:---:|:---:|:---:|
| ![Reader](docs/screenshots/reader.png) | ![Settings](docs/screenshots/settings.png) | ![Install](docs/screenshots/install.png) |

> 截图占位位于 [`docs/screenshots/`](docs/screenshots/README.md)（6 个 `.gitkeep`）。
> 等真实截图准备好后，把同名 `.png` 提交进来并删掉对应的 `.gitkeep` 即可，
> README 里的图片引用路径不需要修改。

---

## 快速开始

### 方式一：安装包（推荐）

1. 下载 `DeepStudent-Setup-1.0.0.exe`。
2. 双击运行，默认安装到 `%LOCALAPPDATA%\Programs\DeepStudent`。
3. 选择是否创建桌面快捷方式 / 开始菜单项。
4. 点击 **安装**，完成后点击 **启动 DeepStudent**。
5. 首次启动按引导完成：
   - 设置主密码（用于 AES-256-GCM 加密数据槽位）。
   - 在 **设置 → LLM Provider** 中至少填入一个供应商的 API Key。

### 方式二：绿色版

1. 下载 `deepstudent.exe` 并放到任意目录（例如 `D:\Apps\DeepStudent\`）。
2. （可选）创建快捷方式。
3. 双击运行；数据目录默认为 `%APPDATA%\DeepStudent`。

### 方式三：从源码构建

```bash
# 1. 安装 Wails CLI（v2.9.2+）
go install github.com/wailsapp/wails/v2/cmd/wails@v2.9.2

# 2. 准备环境变量
cp .env.example .env
# 至少填一个 LLM 供应商的 API Key

# 3. 开发模式（前端热重载）
wails dev

# 4. 生产构建（当前平台）
wails build -clean -o build/bin/deepstudent
```

---

## 13 项核心能力

| # | 能力 | 包 | 主要方法 |
|---|---|---|---|
| 1  | **聊天 / 会话 / 子 Agent**            | `internal/chat`       | `ChatCreateSession`, `ChatSend`, `ChatCompare` |
| 2  | **学习中心 / 笔记 / VFS**            | `internal/hub`        | `HubImportResource`, `HubContinueNote` |
| 3  | **思维导图**                          | `internal/mindmap`    | `MindmapGenerate`, `MindmapToOutline` |
| 4  | **题库 / 练习 / 阅卷**                | `internal/qbank`      | `QBankExtract`, `QBankSubmit`, `QBankAnalyze` |
| 5  | **Anki 制卡**                         | `internal/anki`       | `AnkiGenerate`, `AnkiSave` |
| 6  | **阅读器**（PDF / EPUB / Markdown）   | `internal/reader`     | `ReaderOpen`, `ReaderSummarize` |
| 7  | **翻译**（支持术语表）                | `internal/translate`  | `TranslateText`, `TranslateDocument` |
| 8  | **作文批改**                          | `internal/essay`      | `EssayGrade` |
| 9  | **深度调研**（可取消、可断点续跑）    | `internal/research`   | `ResearchPlan`, `ResearchRun` |
| 10 | **论文检索**（arXiv + 引用）          | `internal/paper`      | `PaperSearchArXiv`, `PaperDownload`, `PaperCite` |
| 11 | **智能记忆**（向量索引用户画像）      | `internal/memory`     | `MemoryIngest`, `MemoryProfile` |
| 12 | **Skill / MCP**（外部工具桥接）       | `internal/skills`     | `SkillsList`, `SkillsSpawnMCP`, `SkillsCall` |
| 13 | **数据治理**（加密双槽备份）          | `internal/governance` | `GovBackup`, `GovRestore`, `GovSwitchSlot` |

---

## 共享基础（`pkg/`）

- `pkg/config`     — 配置加载（Viper + 环境变量）
- `pkg/logger`     — 结构化日志（`log/slog` + 按日滚动）
- `pkg/crypto`     — AES-256-GCM、双槽 A/B、Argon2id
- `pkg/rpc`        — 统一 RPC 抽象
- `pkg/eventbus`   — 进程内 pub/sub 事件总线
- `pkg/vfs`        — 统一虚拟文件系统（`vfs://` URI）
- `pkg/store`      — SQLite 关系存储（纯 Go `modernc.org/sqlite`）
- `pkg/store/blob` — 本地 Blob 存储（SHA-256 内容寻址）
- `pkg/vector`     — 内嵌向量索引（cosine / L2）
- `pkg/llm`        — LLM 适配（9 个 Provider）
- `pkg/mcp`        — MCP 协议（stdio + http+sse）
- `pkg/aferoext`   — afero 封装

---

## LLM Provider 支持（`pkg/llm`）

9 个 Provider 全部使用 `httptest` mock server 做单元测试，CI 完全离线：

- **OpenAI** 官方（gpt-4o / gpt-4.1 系列）
- **Anthropic Claude**（claude-3.5 / claude-3.7）
- **Google Gemini**（含 Vertex AI 模式）
- **DeepSeek**（OpenAI 兼容）
- **通义千问 Qwen**（DashScope & OpenAI 兼容两种 base_url）
- **月之暗面 Moonshot Kimi**
- **智谱 GLM**（OpenAI 兼容）
- **Ollama** 本地模型（OpenAI 兼容）
- **任意 OpenAI 兼容端点**（自定义 base_url）

---

## MCP — Model Context Protocol（`pkg/mcp`）

- **stdio 子进程模式**：spawn 外部 MCP server，JSON-RPC over stdin/stdout（handshake / `tools/list` / `tools/call`）。
- **http + sse 远程模式**：长连接 + 断线自动重连。
- 统一 `Client` API：`ListTools` / `CallTool` / `ListResources`。
- 并发安全，支持 `context` 取消与超时。
- 集成测试用 loopback mock 进程覆盖 handshake + tool registration 全流程。

---

## 数据迁移（Tauri → Go）

如果你之前用的是 Rust (Tauri) 版的 `helixnow/deep-student`，可以用迁移工具把旧数据导入到 Go 版：

```bash
# Linux / macOS
go run ./cmd/migrate --from ~/.deepstudent --to ~/Documents/deepstudent-go

# Windows（PowerShell）
go run .\cmd\migrate --from $env:USERPROFILE\.deepstudent --to "$env:USERPROFILE\Documents\deepstudent-go"
```

迁移完成后会写一份 `migrate-report.json` 到目标目录，按资源 / 会话 / 笔记 / 卡片 / 论文分类统计 `counts / failed / skipped`。
新版首次启动会自动识别 `%APPDATA%\DeepStudent` 下的数据目录。

---

## 卸载

NSIS 安装包自带卸载入口：

- **开始菜单** → `DeepStudent` → `卸载 DeepStudent`
- **设置** → 应用 → 已安装的应用 → DeepStudent → 卸载
- **静默卸载**：
  ```powershell
  "$env:LOCALAPPDATA\Programs\DeepStudent\Uninstall.exe" /S
  ```

### 数据保留

卸载流程会弹出 **「卸载选项」** 页面（BUG-006 修复），默认勾选 **保留** 以下目录：

- `%APPDATA%\DeepStudent` — 笔记、卡片、聊天记录、设置
- `%LOCALAPPDATA%\DeepStudent` — 缓存、临时文件

如需彻底清理（**不可恢复**），取消勾选，或在命令行加 `/PURGEDATA`：

```powershell
"$env:LOCALAPPDATA\Programs\DeepStudent\Uninstall.exe" /PURGEDATA
```

详见 [`cmd/installer/installer.nsi`](cmd/installer/installer.nsi)。

---

## 开发

```bash
# 跑全部测试（带 race 检测）
go test ./... -count=1 -race

# Lint
golangci-lint run

# 漏洞扫描
govulncheck ./...

# 端到端冒烟（13 项能力，in-process）
go run ./scripts/smoke
```

CI 流水线（`.github/workflows/`）：

- `test.yml`         — 每次 push / PR：`go test` + `go vet` + `govulncheck`
- `build.yml`        — 多平台 Wails 构建冒烟
- `govulncheck.yml`  — 定时漏洞扫描
- `release.yml`      — tag 触发：构建安装包 + 算校验和 + 发 GitHub Release

---

## 协议

[AGPL-3.0](./LICENSE) — 与上游项目一致。
