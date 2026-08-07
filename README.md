# DeepStudent (Go) — Local-First AI Learning Workbench

[![Release v1.0.0](https://img.shields.io/github/v/release/helixnow/deep-student-go?label=Release&style=flat-square)](https://github.com/helixnow/deep-student-go/releases/tag/v1.0.0)
[![License: AGPL-3.0](https://img.shields.io/github/license/helixnow/deep-student-go?style=flat-square)](./LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/helixnow/deep-student-go?style=flat-square)](https://goreportcard.com/report/github.com/helixnow/deep-student-go)
[![Go Version](https://img.shields.io/github/go-mod/go-version/helixnow/deep-student-go?style=flat-square)](./go.mod)
[![Platform](https://img.shields.io/badge/platform-Windows%2010%2F11-0078d4?style=flat-square)](https://github.com/helixnow/deep-student-go/releases)

> A full Go rewrite of [helixnow/deep-student](https://github.com/helixnow/deep-student).
> Single binary · 13 core capabilities · 9 LLM providers · local-first data governance.

---

## Downloads (v1.0.0)

Grab the latest build from the [GitHub Releases](https://github.com/helixnow/deep-student-go/releases) page:

| File | Description | Link |
|---|---|---|
| `DeepStudent-Setup-1.0.0.exe` | NSIS single-file installer (Windows 10/11 x64) — **recommended** | [Download](https://github.com/helixnow/deep-student-go/releases/download/v1.0.0/DeepStudent-Setup-1.0.0.exe) |
| `deepstudent.exe`           | Portable single binary (no installer)                          | [Download](https://github.com/helixnow/deep-student-go/releases/download/v1.0.0/deepstudent.exe) |
| `SHA256SUMS.txt`            | SHA-256 checksums for all release artifacts                    | [Download](https://github.com/helixnow/deep-student-go/releases/download/v1.0.0/SHA256SUMS.txt) |

> Installer size is ~115 MB; portable binary ~95 MB.

### Verify the Download

```powershell
Get-FileHash .\DeepStudent-Setup-1.0.0.exe -Algorithm SHA256
# Compare with the value listed in SHA256SUMS.txt
```

On macOS / Linux:

```bash
shasum -a 256 DeepStudent-Setup-1.0.0.exe
# or
sha256sum DeepStudent-Setup-1.0.0.exe
```

See [`CHANGELOG.md`](./CHANGELOG.md) for the full list of changes shipped in v1.0.0.

---

## Screenshots

| Chat | Mindmap | QBank |
|:---:|:---:|:---:|
| ![Chat](docs/screenshots/chat.png) | ![Mindmap](docs/screenshots/mindmap.png) | ![QBank](docs/screenshots/qbank.png) |

| Reader | Settings | Installer |
|:---:|:---:|:---:|
| ![Reader](docs/screenshots/reader.png) | ![Settings](docs/screenshots/settings.png) | ![Install](docs/screenshots/install.png) |

> Image placeholders are committed under [`docs/screenshots/`](docs/screenshots/README.md)
> (6 `.gitkeep` files). Drop the real PNGs in (same filename, no extension change) once
> the screenshots are ready — README references do not need to be updated.

---

## Quick Start

### Option 1 — Installer (recommended for most users)

1. Download `DeepStudent-Setup-1.0.0.exe` from the link above.
2. Double-click the installer. Default install location:
   `%LOCALAPPDATA%\Programs\DeepStudent`
3. Choose whether to create a Desktop shortcut and/or Start Menu entry.
4. Click **Install** and then **Launch DeepStudent**.
5. On first launch, follow the in-app onboarding wizard to:
   - Set a master password (used for the AES-256-GCM encrypted data slot).
   - Add at least one LLM provider API key in **Settings → LLM Providers**.

### Option 2 — Portable binary

1. Download `deepstudent.exe` and place it anywhere (e.g. `D:\Apps\DeepStudent\`).
2. (Optional) Create a shortcut.
3. Double-click to run. The data directory defaults to `%APPDATA%\DeepStudent`.

### Option 3 — Build from source

```bash
# 1. Install Wails CLI (v2.9.2+)
go install github.com/wailsapp/wails/v2/cmd/wails@v2.9.2

# 2. Prepare environment
cp .env.example .env
# Fill in at least one LLM provider's API key

# 3. Dev mode (hot-reload)
wails dev

# 4. Production build (current platform)
wails build -clean -o build/bin/deepstudent
```

---

## 13 Core Capabilities

| # | Capability | Package | Key Methods |
|---|---|---|---|
| 1  | **Chat / Sessions / Sub-Agents**            | `internal/chat`       | `ChatCreateSession`, `ChatSend`, `ChatCompare` |
| 2  | **Hub / Notes / VFS**                       | `internal/hub`        | `HubImportResource`, `HubContinueNote` |
| 3  | **Mindmap**                                 | `internal/mindmap`    | `MindmapGenerate`, `MindmapToOutline` |
| 4  | **QBank / Practice / Grading**              | `internal/qbank`      | `QBankExtract`, `QBankSubmit`, `QBankAnalyze` |
| 5  | **Anki Card Generation**                    | `internal/anki`       | `AnkiGenerate`, `AnkiSave` |
| 6  | **Reader** (PDF / EPUB / Markdown)          | `internal/reader`     | `ReaderOpen`, `ReaderSummarize` |
| 7  | **Translate** (with glossary support)       | `internal/translate`  | `TranslateText`, `TranslateDocument` |
| 8  | **Essay Grading**                           | `internal/essay`      | `EssayGrade` |
| 9  | **Research** (long-running, cancellable)    | `internal/research`   | `ResearchPlan`, `ResearchRun` |
| 10 | **Paper Search** (arXiv + citations)        | `internal/paper`      | `PaperSearchArXiv`, `PaperDownload`, `PaperCite` |
| 11 | **Memory** (vector-indexed user profile)    | `internal/memory`     | `MemoryIngest`, `MemoryProfile` |
| 12 | **Skills / MCP** (external tool bridge)     | `internal/skills`     | `SkillsList`, `SkillsSpawnMCP`, `SkillsCall` |
| 13 | **Governance** (encrypted dual-slot backup)  | `internal/governance` | `GovBackup`, `GovRestore`, `GovSwitchSlot` |

---

## Shared Foundations (`pkg/`)

- `pkg/config`     — Config loader (Viper + env vars)
- `pkg/logger`     — Structured logging (`log/slog` + daily rotation)
- `pkg/crypto`     — AES-256-GCM, dual-slot A/B, Argon2id
- `pkg/rpc`        — Unified RPC abstraction
- `pkg/eventbus`   — In-process pub/sub event bus
- `pkg/vfs`        — Unified virtual filesystem (`vfs://` URI)
- `pkg/store`      — Relational SQLite store (pure-Go `modernc.org/sqlite`)
- `pkg/store/blob` — Local blob storage (SHA-256 content addressing)
- `pkg/vector`     — Embedded vector index (cosine / L2)
- `pkg/llm`        — LLM adapters (9 providers)
- `pkg/mcp`        — MCP protocol (stdio + http+sse)
- `pkg/aferoext`   — afero helpers

---

## LLM Provider Support (`pkg/llm`)

All 9 providers are unit-tested against an `httptest` mock server — no network
calls in CI:

- **OpenAI** (gpt-4o / gpt-4.1 family)
- **Anthropic Claude** (claude-3.5 / claude-3.7)
- **Google Gemini** (incl. Vertex AI mode)
- **DeepSeek** (OpenAI-compatible)
- **Qwen / Tongyi** (DashScope & OpenAI-compatible base URLs)
- **Moonshot Kimi**
- **Zhipu GLM** (OpenAI-compatible)
- **Ollama** (local models, OpenAI-compatible)
- **Custom OpenAI-compatible** endpoint (free-form `base_url`)

---

## MCP — Model Context Protocol (`pkg/mcp`)

- **stdio subprocess mode**: spawn an external MCP server, JSON-RPC over
  stdin/stdout (handshake, `tools/list`, `tools/call`).
- **http + sse remote mode**: persistent connection, auto-reconnect.
- Unified `Client` API: `ListTools` / `CallTool` / `ListResources`.
- Concurrency-safe; honours context cancellation and timeouts.
- Integration test suite uses a loopback mock process — covers handshake and
  tool registration end-to-end.

---

## Data Migration (Tauri → Go)

If you previously used the Rust (Tauri) version `helixnow/deep-student`, the
migrator can import your old data into the new Go layout:

```bash
# Linux / macOS
go run ./cmd/migrate --from ~/.deepstudent --to ~/Documents/deepstudent-go

# Windows (PowerShell)
go run .\cmd/migrate --from $env:USERPROFILE\.deepstudent --to "$env:USERPROFILE\Documents\deepstudent-go"
```

The migrator writes `migrate-report.json` to the target directory with
per-type counts (resources, sessions, notes, cards, papers), failures, and
skipped rows. After migration, the new app will auto-detect the data
directory under `%APPDATA%\DeepStudent`.

---

## Uninstall

The NSIS installer ships its own uninstaller, available from:

- **Start Menu** → `DeepStudent` → `Uninstall DeepStudent`
- **Settings** → Apps → Installed apps → DeepStudent → Uninstall
- **Silent uninstall**:
  ```powershell
  "$env:LOCALAPPDATA\Programs\DeepStudent\Uninstall.exe" /S
  ```

### Data retention

The uninstaller pops up an **"Uninstall options"** page (BUG-006 fix). The
following directories are **kept by default**:

- `%APPDATA%\DeepStudent` — notes, cards, chats, settings
- `%LOCALAPPDATA%\DeepStudent` — caches, temp files

To wipe everything (**irreversible**), uncheck the box, or pass `/PURGEDATA`:

```powershell
"$env:LOCALAPPDATA\Programs\DeepStudent\Uninstall.exe" /PURGEDATA
```

See [`cmd/installer/installer.nsi`](cmd/installer/installer.nsi) for the
exact NSIS script that implements this behaviour.

---

## Development

```bash
# Run all tests (with race detector)
go test ./... -count=1 -race

# Lint
golangci-lint run

# Vulnerability scan
govulncheck ./...

# End-to-end smoke (13 capabilities, in-process)
go run ./scripts/smoke
```

CI pipelines (`.github/workflows/`):

- `test.yml`         — `go test`, `go vet`, `govulncheck` on every push / PR
- `build.yml`        — multi-platform Wails build smoke test
- `govulncheck.yml`  — scheduled vulnerability scan
- `release.yml`      — tag-triggered installer + checksum + GitHub Release

---

## License

[AGPL-3.0](./LICENSE) — same as the upstream project.
