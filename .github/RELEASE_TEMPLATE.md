<!-- .github/RELEASE_TEMPLATE.md — release.yml 会在创建 Release 时使用此模板 -->
<!-- 在 PR 标题 / Tag 描述里填入具体版本号 -->

# Release v1.0.0

> 首次正式发布：用 Go 语言全面重建 [helixnow/deep-student](https://github.com/helixnow/deep-student)，
> 单一二进制、13 项核心能力、9 个 LLM Provider、本地优先数据治理。

## What's New

完整变更列表见仓库根目录的 [`CHANGELOG.md`](../CHANGELOG.md)。本版本要点：

- **13 项核心能力** 全部就绪：聊天 / 学习中心 / 思维导图 / 题库 / Anki / 阅读器 / 翻译 / 作文 / 调研 / 论文 / 记忆 / Skill·MCP / 数据治理。
- **9 个 LLM Provider**：OpenAI、Anthropic、Google、DeepSeek、通义千问、Kimi、智谱、Ollama，以及任意 OpenAI 兼容端点。
- **MCP 协议双模式**：`stdio` 子进程 + `http+sse` 远程。
- **加密双槽备份** + 端到端冒烟脚本 + Tauri 旧版数据迁移工具。
- 修复了来自上游的 **BUG-001 ~ BUG-006**（见 CHANGELOG）。

## Downloads

| 文件 | 说明 |
|---|---|
| `DeepStudent-Setup-1.0.0.exe` | Windows 64 位 NSIS 安装包（推荐） |
| `deepstudent.exe` | 免安装绿色版（仅二进制） |
| `SHA256SUMS.txt` | 上述文件的 SHA-256 校验和 |

下载链接：

- 安装包：<https://github.com/helixnow/deep-student-go/releases/download/v1.0.0/DeepStudent-Setup-1.0.0.exe>
- 校验和：<https://github.com/helixnow/deep-student-go/releases/download/v1.0.0/SHA256SUMS.txt>

校验示例（PowerShell）：

```powershell
Get-FileHash .\DeepStudent-Setup-1.0.0.exe -Algorithm SHA256
# 与 SHA256SUMS.txt 中的值比对
```

## Known Issues

- macOS / Linux 平台本版本暂未提供官方安装包，参见 [issue tracker](https://github.com/helixnow/deep-student-go/issues)。
- 部分小众 LLM 端点（如自部署 vLLM）的 function-calling 兼容还在补齐。
- 首次启动若提示"未找到数据目录"，请运行一次安装包，写入注册表后再启动绿色版。

## Upgrade Notes

- **从 Tauri 旧版本（helixnow/deep-student）升级**：先安装本版本，再用迁移工具把旧数据导入新布局：

  ```bash
  deepstudent migrate --from ~/.deepstudent --to ~/Documents/deepstudent-go
  ```

  报告写入 `<to>/migrate-report.json`，按资源 / 会话 / 笔记 / 卡片 / 论文分类计数。

- **从 0.x dev 版本升级**：建议先备份 `%APPDATA%\DeepStudent` 整目录再覆盖安装。卸载时勾选「保留数据」，再装新版本即可。
- **数据目录**：`%APPDATA%\DeepStudent`（默认）——卸载时**默认保留**；如需彻底清理，卸载时取消勾选或加 `/PURGEDATA` 参数：

  ```powershell
  "$INSTDIR\Uninstall.exe" /PURGEDATA
  ```
