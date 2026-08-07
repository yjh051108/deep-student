# flow/go-replica — Go 语言复刻分支

本分支承载 **DeepStudent 的 Go 语言复刻工程**（Wails v2 + Go + React/TS）。

## 本分支与 main 的关系

- **`main`**：Rust (Tauri) 原版 [helixnow/deep-student](https://github.com/helixnow/deep-student) 的云端最新代码（v0.9.43 + main）。
- **`flow/go-replica`**：从 `main` 切出的个人分支，**分支树整体替换为 Go 复刻项目**（删除 Rust 树，放入 Go 工程）。

> 分支根部不是 Rust 代码，而是 Go 复刻项目自身。两者通过本说明文件与
> [`docs/replica-gap-analysis.md`](docs/replica-gap-analysis.md)、
> [`docs/replica-roadmap.md`](docs/replica-roadmap.md) 建立对照。

## 复刻基准

| 项目 | 版本 | 说明 |
|---|---|---|
| 上游原版 | `v0.9.43` + main（2026-08-04） | 复刻目标基准 |
| Go 复刻版 | 1.0.0 起 | 本分支 |

## 构建方式（Go 复刻版）

```bash
# 1. 安装 Wails CLI（v2.9.2+）
go install github.com/wailsapp/wails/v2/cmd/wails@v2.9.2

# 2. 准备环境变量（至少一个 LLM 供应商 API Key）
cp .env.example .env

# 3. 开发模式（前端热重载）
wails dev

# 4. 生产构建（当前平台）
wails build -clean -o build/bin/deepstudent
```

## 测试

```bash
# 全部单元测试（带 race）
go test ./... -count=1 -race

# 端到端冒烟（13 项核心能力，in-process）
go run ./scripts/smoke
```

## 复刻进度

详见 [`docs/replica-roadmap.md`](docs/replica-roadmap.md)（分 5 个批次推进，每批完成后提交到本分支并 push）。
