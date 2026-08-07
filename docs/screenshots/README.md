# 截图占位（Screenshots Placeholders）

本目录用于存放 DeepStudent (Go) v1.0.0 的界面截图，方便 README 引用。
当前为占位阶段，**真实图片尚未生成**；README 中的 `![…](docs/screenshots/*.png)` 语法
仍能渲染（alt 文本会正常显示），不会破坏 Markdown 校验。

## 6 张截图清单

| # | 文件 | 用途 | 期望内容 |
|---|---|---|---|
| 1 | `chat.png` | 聊天 / 会话主界面 | 左侧会话列表 + 右侧消息流 + 模型选择器，展示多模型对比（ChatCompare） |
| 2 | `mindmap.png` | 思维导图 | `MindmapGenerate` 输出结果，可点击节点展开 / 折叠 |
| 3 | `qbank.png` | 题库 / 阅卷 | `QBankExtract` 抽题列表 + 提交答案 + `QBankAnalyze` 报告 |
| 4 | `reader.png` | 阅读器 | PDF / EPUB 打开 + 高亮批注 + `ReaderSummarize` 摘要侧栏 |
| 5 | `settings.png` | 设置页 | LLM Provider 配置 / MCP 端点 / 加密槽位 A/B 切换 |
| 6 | `install.png` | NSIS 安装向导 | 欢迎页 + 目录选择 + 「卸载时保留数据」选项页 |

## 文件存在性

为避免空目录被 Git 忽略，这里放 6 个 `.gitkeep` 占位文件：

- `chat.gitkeep`
- `mindmap.gitkeep`
- `qbank.gitkeep`
- `reader.gitkeep`
- `settings.gitkeep`
- `install.gitkeep`

待真实截图准备好后，**把同名 `.png` 文件 commit 进来并删除对应 `.gitkeep`** 即可，
README 中的图片引用路径不需要修改。

## 贡献指南

- 截图分辨率：建议 1920×1080 或 1440×900，深色 / 浅色主题各一张更佳。
- 文件大小：单张 < 500 KB，必要时可用 `pngquant` 压缩。
- 内容真实性：截图前请抹去真实 API Key / 邮箱 / 学号等敏感信息。
