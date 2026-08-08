package deepstudent

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/helixnow/deep-student-go/internal/anki"
	"github.com/helixnow/deep-student-go/internal/chat"
	"github.com/helixnow/deep-student-go/internal/cloudstorage"
	"github.com/helixnow/deep-student-go/internal/essay"
	"github.com/helixnow/deep-student-go/internal/fsrs"
	"github.com/helixnow/deep-student-go/internal/plugins"
	"github.com/helixnow/deep-student-go/internal/quickassist"
	"github.com/helixnow/deep-student-go/internal/llmcfg"
	"github.com/helixnow/deep-student-go/internal/llmusage"
	"github.com/helixnow/deep-student-go/internal/memory"
	"github.com/helixnow/deep-student-go/internal/mindmap"
	"github.com/helixnow/deep-student-go/internal/multimodal"
	"github.com/helixnow/deep-student-go/internal/notes"
	"github.com/helixnow/deep-student-go/internal/ocr"
	"github.com/helixnow/deep-student-go/internal/paper"
	"github.com/helixnow/deep-student-go/internal/pomodoro"
	"github.com/helixnow/deep-student-go/internal/qbank"
	"github.com/helixnow/deep-student-go/internal/reader"
	"github.com/helixnow/deep-student-go/internal/research"
	"github.com/helixnow/deep-student-go/internal/skills"
	"github.com/helixnow/deep-student-go/internal/sync"
	"github.com/helixnow/deep-student-go/internal/templatemgr"
	"github.com/helixnow/deep-student-go/internal/todo"
	"github.com/helixnow/deep-student-go/internal/translate"
	"github.com/helixnow/deep-student-go/internal/voiceinput"
	"github.com/helixnow/deep-student-go/pkg/config"
	"github.com/helixnow/deep-student-go/pkg/index"
	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// ===== Hub =====

// HubImportResource imports a new resource.
func (a *App) HubImportResource(typ, title string, data []byte, tags []string) (string, error) {
	return a.Hub.ImportResource(a.Ctx, vfs.ResourceType(typ), title, data, tags)
}

// HubList lists resources by type.
func (a *App) HubList(typ string) []vfs.Entry { return a.Hub.List(vfs.ResourceType(typ)) }

// HubSearch searches by tag.
func (a *App) HubSearch(typ, tag string) []vfs.Entry { return a.Hub.Search(vfs.ResourceType(typ), tag) }

// HubGet reads a resource.
func (a *App) HubGet(uri string) (map[string]any, error) {
	d, e, err := a.Hub.Get(uri)
	if err != nil {
		return nil, err
	}
	return map[string]any{"data": d, "entry": e}, nil
}

// HubDelete removes a resource.
func (a *App) HubDelete(uri string) error { return a.Hub.Delete(uri) }

// HubContinueNote continues a note using AI.
func (a *App) HubContinueNote(uri, prompt string) (string, error) {
	ch, err := a.Hub.ContinueNote(a.Ctx, uri, prompt)
	if err != nil {
		return "", err
	}
	return collectStream(ch), nil
}

// ===== Chat =====

// ChatCreateGroup creates a session group.
func (a *App) ChatCreateGroup(name, system, skill string, tags []string) *chat.Group {
	return a.Chat.CreateGroup(name, system, skill, tags)
}

// ChatCreateSession creates a new chat session.
func (a *App) ChatCreateSession(groupID, title, model, provider string) *chat.Session {
	return a.Chat.CreateSession(groupID, title, model, provider)
}

// ChatBranch creates a branched session.
func (a *App) ChatBranch(parentID, atMsgID string) (*chat.Session, error) {
	return a.Chat.Branch(parentID, atMsgID)
}

// ChatSend sends a message and returns the full streamed reply.
func (a *App) ChatSend(sessionID, content string, refs []string, deep bool) (string, error) {
	ch, err := a.Chat.Send(a.Ctx, sessionID, content, refs, deep)
	if err != nil {
		return "", err
	}
	return collectStream(ch), nil
}

// ChatCompare compares multiple providers in parallel.
func (a *App) ChatCompare(sessionID, content string, providers []string) map[string]string {
	return a.Chat.Compare(a.Ctx, sessionID, content, providers)
}

// ===== Mindmap =====

// MindmapGenerate generates a mind map from a topic.
func (a *App) MindmapGenerate(topic string) (*mindmap.Map, error) {
	return a.Mindmap.Generate(a.Ctx, topic)
}

// MindmapSave saves a mind map to VFS.
func (a *App) MindmapSave(m *mindmap.Map) (string, error) { return a.Mindmap.Save(m) }

// MindmapLoad loads a mind map.
func (a *App) MindmapLoad(uri string) (*mindmap.Map, error) { return a.Mindmap.Load(uri) }

// MindmapEdit edits a mind map.
func (a *App) MindmapEdit(m *mindmap.Map, instruction string) (*mindmap.Map, error) {
	return a.Mindmap.Edit(a.Ctx, m, instruction)
}

// MindmapToOutline converts to outline.
func (a *App) MindmapToOutline(m *mindmap.Map) string { return m.ToOutline() }

// MindmapFromOutline builds from outline.
func (a *App) MindmapFromOutline(title, text string) *mindmap.Map {
	return mindmap.FromOutline(title, text)
}

// MindmapMask masks nodes for recitation.
func (a *App) MindmapMask(m *mindmap.Map, rate float64) { m.Mask(rate) }

// ===== QBank =====

// QBankExtract extracts a question set from a source URI.
func (a *App) QBankExtract(uri, title string) (*qbank.Set, error) {
	return a.QBank.Extract(a.Ctx, uri, title)
}

// QBankSave saves a question set.
func (a *App) QBankSave(set *qbank.Set) (string, error) { return a.QBank.Save(set) }

// QBankStartAttempt starts a new attempt.
func (a *App) QBankStartAttempt(setID string) (*qbank.Attempt, error) {
	return a.QBank.StartAttempt(setID)
}

// QBankAnswer records an answer.
func (a *App) QBankAnswer(attemptID, qID, ans string) error {
	return a.QBank.Answer(attemptID, qID, ans)
}

// QBankSubmit submits and auto-grades.
func (a *App) QBankSubmit(attemptID string) (*qbank.Attempt, error) {
	return a.QBank.Submit(attemptID)
}

// QBankAnalyze gives AI analysis.
func (a *App) QBankAnalyze(setID, qID string) (string, error) {
	return a.QBank.Analyze(a.Ctx, setID, qID)
}

// QBankMastery returns knowledge mastery.
func (a *App) QBankMastery() map[string]int { return a.QBank.Mastery() }

// ===== Anki =====

// AnkiGenerate generates Anki cards.
func (a *App) AnkiGenerate(deck, text, tplID string, batch int) (*anki.Job, error) {
	return a.Anki.GenerateFromText(a.Ctx, deck, text, tplID, batch, nil)
}

// AnkiTemplates lists templates.
func (a *App) AnkiTemplates() []*anki.Template { return a.Anki.Templates() }

// AnkiAddTemplate adds a template.
func (a *App) AnkiAddTemplate(t *anki.Template) { a.Anki.AddTemplate(t) }

// AnkiSave saves a job to VFS.
func (a *App) AnkiSave(job *anki.Job) (string, error) { return a.Anki.SaveToVFS(job) }

// AnkiExport exports cards to APKG.
func (a *App) AnkiExport(job *anki.Job) ([]byte, error) { return a.Anki.ExportAPKG(job) }

// ===== Reader =====

// ReaderOpen opens a document.
func (a *App) ReaderOpen(uri string) (*reader.Document, error) { return a.Reader.Open(uri) }

// ReaderSummarize summarizes a document.
func (a *App) ReaderSummarize(uri string, page int) (string, error) {
	doc, err := a.Reader.Open(uri)
	if err != nil {
		return "", err
	}
	return a.Reader.Summarize(a.Ctx, doc, page)
}

// ReaderInject builds a chat injection string.
func (a *App) ReaderInject(uri string, start, end int, sel string) string {
	doc, err := a.Reader.Open(uri)
	if err != nil {
		return ""
	}
	return a.Reader.InjectToChat(doc, start, end, sel)
}

// ===== Translate =====

// TranslateText translates a piece of text.
func (a *App) TranslateText(text, src, tgt, domain, custom string, glossary []map[string]string) (string, error) {
	res, err := a.Trans.Translate(a.Ctx, toTranslateRequest(text, src, tgt, domain, custom, glossary))
	if err != nil {
		return "", err
	}
	return res.Text, nil
}

// TranslateDocument translates a VFS document.
func (a *App) TranslateDocument(uri, src, tgt, domain string) (string, error) {
	return a.Trans.TranslateDocument(a.Ctx, uri, src, tgt, translate.Domain(domain))
}

// ===== Essay =====

// EssayGrade grades an essay.
func (a *App) EssayGrade(text, scenario string, dims []string) (*essay.Result, error) {
	return a.Essay.Grade(a.Ctx, text, essay.Scenario(scenario), dims)
}

// EssaySave saves a grading result.
func (a *App) EssaySave(r *essay.Result) (string, error) { return a.Essay.Save(r) }

// ===== Research =====

// ResearchPlan creates a research plan.
func (a *App) ResearchPlan(topic, depth, format string) (*research.Plan, error) {
	return a.Res.ConfirmAndPlan(a.Ctx, topic, depth, format)
}

// ResearchRun runs a full research.
func (a *App) ResearchRun(topic string, engines []string) (*research.Report, error) {
	return a.Res.Run(a.Ctx, topic, anyEngines(engines), nil)
}

// ResearchSave saves a report to VFS.
func (a *App) ResearchSave(r *research.Report) (string, error) { return a.Res.Save(r) }

// ===== Paper =====

// PaperSearchArXiv searches arXiv.
func (a *App) PaperSearchArXiv(q string, max int) ([]paper.Source, error) {
	return a.Paper.SearchArXiv(a.Ctx, q, max)
}

// PaperSearchOpenAlex searches OpenAlex.
func (a *App) PaperSearchOpenAlex(q string, max int) ([]paper.Source, error) {
	return a.Paper.SearchOpenAlex(a.Ctx, q, max)
}

// PaperDownload downloads a PDF.
func (a *App) PaperDownload(src *paper.Source) (string, error) {
	return a.Paper.Download(a.Ctx, *src)
}

// PaperCite outputs a citation.
func (a *App) PaperCite(src *paper.Source, format string) string {
	return paper.Cite(*src, format)
}

// PaperResolveDOI resolves a DOI.
func (a *App) PaperResolveDOI(doi string) (string, error) {
	return a.Paper.ResolveDOI(a.Ctx, doi)
}

// ===== Memory =====

// MemoryIngest extracts and applies facts.
func (a *App) MemoryIngest(conversation string) ([]*memory.Item, error) {
	return a.Mem.Ingest(a.Ctx, conversation)
}

// MemorySearch searches memory.
func (a *App) MemorySearch(q string) []*memory.Item { return a.Mem.Search(q) }

// MemoryProfile returns aggregated profile.
func (a *App) MemoryProfile() string { return a.Mem.Profile() }

// MemoryPrivacyMode toggles privacy mode.
func (a *App) MemoryPrivacyMode(on bool) { a.Mem.PrivacyMode(on) }

// MemoryDecay runs decay.
func (a *App) MemoryDecay() { a.Mem.Decay() }

// ===== Skills =====

// SkillsList lists skills.
func (a *App) SkillsList() []*skills.Skill { return a.Skills.Skills() }

// SkillsTools lists tools.
func (a *App) SkillsTools() []skillToolDTO { return toSkillTools(a.Skills.Tools()) }

// SkillsCall invokes a tool.
func (a *App) SkillsCall(name, args string) (any, error) {
	return a.Skills.Tool(a.Ctx, name, json.RawMessage(args))
}

// SkillsSpawnMCP spawns an MCP server.
func (a *App) SkillsSpawnMCP(name, cmd string, args, env []string) error {
	return a.Skills.SpawnMCP(a.Ctx, name, cmd, args, env)
}

// SkillsEnableServer enables an MCP server from a structured config payload.
// 前端"设置 → MCP"页直接传 JSON 字符串进来。
func (a *App) SkillsEnableServer(name, cfgJSON string) error {
	var cfg struct {
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env"`
		URL     string            `json:"url"`
		Enabled bool              `json:"enabled"`
	}
	if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil {
		return err
	}
	return a.Skills.EnableServer(a.Ctx, name, config.MCPServerConfig{
		Command: cfg.Command, Args: cfg.Args, Env: cfg.Env, URL: cfg.URL, Enabled: cfg.Enabled,
	})
}

// SkillsDisableServer disables an MCP server.
func (a *App) SkillsDisableServer(name string) { a.Skills.DisableServer(name) }

// SkillsListMCPServers lists enabled MCP servers.
func (a *App) SkillsListMCPServers() []string { return a.Skills.ListMCPServers() }

// SkillsLoadSKILLMD loads a SKILL.md.
func (a *App) SkillsLoadSKILLMD(tier, path string) error {
	return a.Skills.LoadSkillMD(skills.Tier(tier), path)
}

// skillToolDTO is the JSON shape exposed to the frontend (avoids exposing llm.Tool as raw).
type skillToolDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// toSkillTools converts llm.Tool slice to a thin DTO for the frontend.
func toSkillTools(ts []llm.Tool) []skillToolDTO {
	out := make([]skillToolDTO, 0, len(ts))
	for _, t := range ts {
		out = append(out, skillToolDTO{Name: t.Name, Description: t.Description})
	}
	return out
}

// ===== Governance =====

// GovBackup runs an encrypted backup.
func (a *App) GovBackup(target string) (string, error) { return a.Gov.Backup(target) }

// GovRestore restores a backup.
func (a *App) GovRestore(source string) error { return a.Gov.Restore(source) }

// GovSwitchSlot switches the active A/B slot.
func (a *App) GovSwitchSlot(to string) error { return a.Gov.SwitchSlot(to) }

// GovExport exports resources to a zip.
func (a *App) GovExport(target string, types []string) error {
	return a.Gov.Export(target, anyTypes(types))
}

// GovImport imports from a zip.
func (a *App) GovImport(source string) error { return a.Gov.Import(source) }

// GovAudit returns recent audit entries.
func (a *App) GovAudit(limit int) []auditDTO {
	rows, _ := a.Gov.AuditLogs(limit)
	out := make([]auditDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, auditDTO{Actor: r.Actor, Action: r.Action, Detail: r.Detail, TS: r.TS})
	}
	return out
}

// GovStatus returns governance status.
func (a *App) GovStatus() map[string]any { return a.Gov.Status() }

// GovIntegrityCheck reports missing files.
func (a *App) GovIntegrityCheck() []string { return a.Gov.CheckIntegrity() }

type auditDTO struct {
	Actor  string `json:"actor"`
	Action string `json:"action"`
	Detail string `json:"detail"`
	TS     int64  `json:"ts"`
}

// ===== Notes（P1 笔记系统）=====

// NotesCreate 创建新笔记。
func (a *App) NotesCreate(params notes.CreateParams) (*notes.Note, error) {
	return a.Notes.Create(params)
}

// NotesGet 读取单条笔记（含正文）。
func (a *App) NotesGet(id string) (*notes.Note, error) {
	return a.Notes.Get(id)
}

// NotesUpdate 更新笔记（支持部分字段 + 乐观锁）。
func (a *App) NotesUpdate(params notes.UpdateParams) (*notes.Note, error) {
	return a.Notes.Update(params)
}

// NotesList 列出笔记（含正文）。
func (a *App) NotesList(opts notes.ListOptions) (*notes.ListResult, error) {
	return a.Notes.List(opts)
}

// NotesListMeta 列出笔记元数据（不含正文，用于列表视图降低载荷）。
func (a *App) NotesListMeta(opts notes.ListOptions) (*notes.ListResult, error) {
	return a.Notes.ListMeta(opts)
}

// NotesMoveToTrash 将笔记移入回收站（软删除）。
func (a *App) NotesMoveToTrash(id string) error { return a.Notes.MoveToTrash(id) }

// NotesRestore 从回收站恢复笔记。
func (a *App) NotesRestore(id string) error { return a.Notes.Restore(id) }

// NotesHardDelete 永久删除笔记及其全部资产。
func (a *App) NotesHardDelete(id string) error { return a.Notes.HardDelete(id) }

// NotesEmptyTrash 清空回收站，返回被清空的笔记数量。
func (a *App) NotesEmptyTrash() (int, error) { return a.Notes.EmptyTrash() }

// NotesTrashCount 返回回收站笔记数量。
func (a *App) NotesTrashCount() (int64, error) { return a.Notes.TrashCount() }

// NotesListFolders 列出全部文件夹。
func (a *App) NotesListFolders() ([]*notes.Folder, error) { return a.Notes.ListFolders() }

// NotesCreateFolder 创建新文件夹。
func (a *App) NotesCreateFolder(name string, parentID *string) (*notes.Folder, error) {
	return a.Notes.CreateFolder(name, parentID)
}

// NotesUpdateFolder 重命名文件夹。
func (a *App) NotesUpdateFolder(id, name string) error { return a.Notes.UpdateFolder(id, name) }

// NotesDeleteFolder 删除文件夹，文件夹下的笔记移到根目录。
func (a *App) NotesDeleteFolder(id string) error { return a.Notes.DeleteFolder(id) }

// NotesAddAsset 为笔记添加附件。
func (a *App) NotesAddAsset(noteID, filename string, data []byte, mime string) (*notes.Asset, error) {
	return a.Notes.AddAsset(noteID, filename, data, mime)
}

// NotesListAssets 列出指定笔记的全部资产。
func (a *App) NotesListAssets(noteID string) ([]*notes.Asset, error) {
	return a.Notes.ListAssets(noteID)
}

// NotesGetAsset 读取资产内容与元数据。
func (a *App) NotesGetAsset(assetID string) ([]byte, *notes.Asset, error) {
	return a.Notes.GetAsset(assetID)
}

// NotesDeleteAsset 删除资产（含 blob 引用）。
func (a *App) NotesDeleteAsset(assetID string) error { return a.Notes.DeleteAsset(assetID) }

// NotesImportMarkdown 从 Markdown 字节流导入笔记，自动识别编码。
func (a *App) NotesImportMarkdown(filename string, content []byte, folderID *string) (*notes.Note, error) {
	return a.Notes.ImportMarkdown(filename, content, folderID)
}

// NotesImportBatch 批量导入 Markdown 文件。
func (a *App) NotesImportBatch(files map[string][]byte, folderID *string) ([]*notes.Note, error) {
	return a.Notes.ImportBatch(files, folderID)
}

// NotesExportNote 导出单条笔记。
func (a *App) NotesExportNote(id string, format notes.ExportFormat) ([]byte, error) {
	return a.Notes.ExportNote(id, format)
}

// NotesExportAll 导出全部笔记（未删除）。
func (a *App) NotesExportAll(format notes.ExportFormat) ([]byte, error) {
	return a.Notes.ExportAll(format)
}

// NotesSearch 简单关键字搜索（标题 + 正文）。
func (a *App) NotesSearch(keyword string, limit int) ([]notes.Note, error) {
	return a.Notes.Search(keyword, limit)
}

// NotesStats 返回笔记统计：total / trash / pinned / assets。
func (a *App) NotesStats() (map[string]int, error) { return a.Notes.Stats() }

// ===== LLMCfg（P0-A 模型厂商配置系统）=====
//
// 以下方法对前端暴露 LLMCfg Manager 的 CRUD / 测试连接 / 解析能力。
// Wails 自动把 PascalCase 方法名转成 camelCase（LLMCfgGetVendors → llmCfgGetVendors）。

// LLMCfgGetVendors 返回所有供应商。
func (a *App) LLMCfgGetVendors() []llmcfg.VendorConfig {
	if a.LLMCfg == nil {
		return []llmcfg.VendorConfig{}
	}
	return a.LLMCfg.GetVendors()
}

// LLMCfgSaveVendor 保存供应商（upsert）。
func (a *App) LLMCfgSaveVendor(v llmcfg.VendorConfig) error {
	if a.LLMCfg == nil {
		return errLLMCfgNotReady
	}
	return a.LLMCfg.SaveVendor(v)
}

// LLMCfgDeleteVendor 删除供应商（内置不可删）。
func (a *App) LLMCfgDeleteVendor(id string) error {
	if a.LLMCfg == nil {
		return errLLMCfgNotReady
	}
	return a.LLMCfg.DeleteVendor(id)
}

// LLMCfgGetProfiles 返回所有模型。
func (a *App) LLMCfgGetProfiles() []llmcfg.ModelProfile {
	if a.LLMCfg == nil {
		return []llmcfg.ModelProfile{}
	}
	return a.LLMCfg.GetProfiles()
}

// LLMCfgGetProfilesByVendor 按供应商筛选模型。
func (a *App) LLMCfgGetProfilesByVendor(vendorID string) []llmcfg.ModelProfile {
	if a.LLMCfg == nil {
		return []llmcfg.ModelProfile{}
	}
	return a.LLMCfg.GetProfilesByVendor(vendorID)
}

// LLMCfgSaveProfile 保存模型（upsert）。
func (a *App) LLMCfgSaveProfile(p llmcfg.ModelProfile) error {
	if a.LLMCfg == nil {
		return errLLMCfgNotReady
	}
	return a.LLMCfg.SaveProfile(p)
}

// LLMCfgDeleteProfile 删除模型（内置不可删）。
func (a *App) LLMCfgDeleteProfile(id string) error {
	if a.LLMCfg == nil {
		return errLLMCfgNotReady
	}
	return a.LLMCfg.DeleteProfile(id)
}

// LLMCfgGetAssignments 获取模型分配。
func (a *App) LLMCfgGetAssignments() llmcfg.ModelAssignments {
	if a.LLMCfg == nil {
		return llmcfg.ModelAssignments{}
	}
	return a.LLMCfg.GetAssignments()
}

// LLMCfgSaveAssignments 保存模型分配。
func (a *App) LLMCfgSaveAssignments(a2 llmcfg.ModelAssignments) error {
	if a.LLMCfg == nil {
		return errLLMCfgNotReady
	}
	return a.LLMCfg.SaveAssignments(a2)
}

// LLMCfgTestConnection 测试连接。
func (a *App) LLMCfgTestConnection(profileID string) (*llmcfg.TestConnectionResult, error) {
	if a.LLMCfg == nil {
		return nil, errLLMCfgNotReady
	}
	return a.LLMCfg.TestConnection(context.Background(), profileID)
}

// LLMCfgResolveApiConfig 解析为运行时配置。
func (a *App) LLMCfgResolveApiConfig(profileID string) (*llmcfg.ApiConfig, error) {
	if a.LLMCfg == nil {
		return nil, errLLMCfgNotReady
	}
	return a.LLMCfg.ResolveApiConfig(profileID)
}

// LLMCfgReloadBuiltins 重新加载内置（用于重置）。
func (a *App) LLMCfgReloadBuiltins() error {
	if a.LLMCfg == nil {
		return errLLMCfgNotReady
	}
	return a.LLMCfg.ReloadBuiltins()
}

// errLLMCfgNotReady LLMCfg Manager 未初始化（启动早期或初始化失败时使用）。
var errLLMCfgNotReady = &llmcfgNotReadyError{}

type llmcfgNotReadyError struct{}

func (e *llmcfgNotReadyError) Error() string { return "deepstudent: llmcfg manager not ready" }

// ===== Index（P0-B 索引系统）=====
//
// 以下方法对前端暴露索引服务的全部能力：
// - 单资源 / 批量索引（FTS + 可选向量嵌入）
// - 删除索引、重建全部索引
// - 综合搜索（FTS + 向量混合）
// - RAG 检索（带重排序）
// - 索引统计与任务状态查询

// IndexStats 返回索引统计。
func (a *App) IndexStats() (*index.IndexStats, error) {
	if a.Index == nil {
		return nil, errIndexNotReady
	}
	return a.Index.Stats()
}

// IndexResource 对单个资源建立索引。
//
// content 为待索引的纯文本；opts 控制切片大小、是否生成向量嵌入等。
// 通常前端先通过 HubGet 拿到资源文本，再调用此方法。
func (a *App) IndexResource(uri, content string, opts index.IndexOptions) error {
	if a.Index == nil {
		return errIndexNotReady
	}
	return a.Index.IndexResource(a.Ctx, uri, content, opts)
}

// IndexBatch 批量索引多个资源。
//
// uris 与 contents 一一对应；opts 同 IndexResource。
func (a *App) IndexBatch(uris, contents []string, opts index.IndexOptions) (*index.BatchIndexResult, error) {
	if a.Index == nil {
		return nil, errIndexNotReady
	}
	if len(uris) != len(contents) {
		return nil, errIndexNotReady // 参数长度不匹配视为未就绪错误，简化前端处理
	}
	// 包装为 map[uri]content，由 Service 内部循环处理
	res := &index.BatchIndexResult{Total: len(uris)}
	for i := range uris {
		err := a.Index.IndexResource(a.Ctx, uris[i], contents[i], opts)
		if err != nil {
			res.Failed++
			res.Errors = append(res.Errors, index.IndexError{URI: uris[i], Error: err.Error()})
		} else {
			res.Success++
		}
	}
	return res, nil
}

// IndexDelete 删除指定 URI 的全部索引（含 FTS 与 chunks）。
func (a *App) IndexDelete(uri string) error {
	if a.Index == nil {
		return errIndexNotReady
	}
	return a.Index.DeleteIndex(uri)
}

// IndexRebuildAll 重建全部索引。
//
// 该方法会扫描 VFS 中的全部资源（笔记/教材/题库/思维导图/翻译/卡片/论文），
// 清空旧索引后重新切片、FTS 索引、可选向量嵌入。
func (a *App) IndexRebuildAll(opts index.IndexOptions) (*index.BatchIndexResult, error) {
	if a.Index == nil {
		return nil, errIndexNotReady
	}
	// 收集全部 VFS 资源（去重 type=all）
	entries := a.vfs.List("")
	uris := make([]string, 0, len(entries))
	contents := make([]string, 0, len(entries))
	for _, e := range entries {
		data, _, err := a.vfs.Get(e.URI)
		if err != nil {
			continue
		}
		uris = append(uris, e.URI)
		// 把标题拼到正文前，提升 FTS 召回
		contents = append(contents, e.Title+"\n\n"+string(data))
	}
	res := &index.BatchIndexResult{Total: len(uris)}
	for i := range uris {
		err := a.Index.IndexResource(a.Ctx, uris[i], contents[i], opts)
		if err != nil {
			res.Failed++
			res.Errors = append(res.Errors, index.IndexError{URI: uris[i], Error: err.Error()})
		} else {
			res.Success++
		}
	}
	return res, nil
}

// IndexSearch 综合搜索（FTS + 向量混合）。
func (a *App) IndexSearch(q index.SearchQuery) ([]index.SearchResult, error) {
	if a.Index == nil {
		return nil, errIndexNotReady
	}
	return a.Index.Search(a.Ctx, q)
}

// IndexRAGQuery RAG 检索（带重排序），返回 topK 个最相关切片。
func (a *App) IndexRAGQuery(query string, topK int) ([]index.SearchResult, error) {
	if a.Index == nil {
		return nil, errIndexNotReady
	}
	return a.Index.RAGQuery(a.Ctx, query, topK)
}

// IndexGetTask 查询单个 URI 的索引任务状态。
func (a *App) IndexGetTask(uri string) *index.IndexTask {
	if a.Index == nil {
		return nil
	}
	return a.Index.GetTask(uri)
}

// IndexDefaultOptions 返回默认索引选项，供前端 UI 初始化表单。
func (a *App) IndexDefaultOptions() index.IndexOptions { return index.DefaultOptions() }

// errIndexNotReady 索引服务未初始化。
var errIndexNotReady = errors.New("deepstudent: index service not ready")

// ===== Todo 待办 =====

// TodoEnsureInbox 确保内置收件箱存在。
func (a *App) TodoEnsureInbox() (*todo.List, error) { return a.Todo.EnsureInbox() }

// TodoCreateList 创建列表。
func (a *App) TodoCreateList(p todo.CreateListParams) (*todo.List, error) { return a.Todo.CreateList(p) }

// TodoGetList 读取列表（含统计）。
func (a *App) TodoGetList(id string) (*todo.List, error) { return a.Todo.GetList(id) }

// TodoListLists 列出列表（含统计）。
func (a *App) TodoListLists(includeDeleted bool) ([]todo.List, error) { return a.Todo.ListLists(includeDeleted) }

// TodoUpdateList 更新列表。
func (a *App) TodoUpdateList(p todo.UpdateListParams) (*todo.List, error) { return a.Todo.UpdateList(p) }

// TodoDeleteList 软删除列表（回收站）。
func (a *App) TodoDeleteList(id string) error { return a.Todo.DeleteList(id) }

// TodoRestoreList 恢复列表。
func (a *App) TodoRestoreList(id string) error { return a.Todo.RestoreList(id) }

// TodoPurgeList 彻底删除列表。
func (a *App) TodoPurgeList(id string) error { return a.Todo.PurgeList(id) }

// TodoPurgeDeletedLists 清空回收站列表，返回数量。
func (a *App) TodoPurgeDeletedLists() (int64, error) { return a.Todo.PurgeDeletedLists() }

// TodoListDeletedLists 列出回收站列表。
func (a *App) TodoListDeletedLists() ([]todo.List, error) { return a.Todo.ListDeletedLists() }

// TodoCreateItem 创建条目。
func (a *App) TodoCreateItem(p todo.CreateItemParams) (*todo.Item, error) { return a.Todo.CreateItem(p) }

// TodoGetItem 读取条目。
func (a *App) TodoGetItem(id string) (*todo.Item, error) { return a.Todo.GetItem(id) }

// TodoListItems 列出条目（listID 为空跨列表；filter 见 todo.ItemFilter）。
func (a *App) TodoListItems(listID, filter string) ([]todo.Item, error) {
	return a.Todo.ListItems(listID, todo.ItemFilter(filter))
}

// TodoUpdateItem 更新条目。
func (a *App) TodoUpdateItem(p todo.UpdateItemParams) (*todo.Item, error) { return a.Todo.UpdateItem(p) }

// TodoToggleItem 切换完成状态。
func (a *App) TodoToggleItem(id string) (*todo.Item, error) { return a.Todo.ToggleItem(id) }

// TodoDeleteItem 软删除条目（回收站）。
func (a *App) TodoDeleteItem(id string) error { return a.Todo.DeleteItem(id) }

// TodoRestoreItem 恢复条目。
func (a *App) TodoRestoreItem(id string) error { return a.Todo.RestoreItem(id) }

// TodoPurgeItem 彻底删除条目。
func (a *App) TodoPurgeItem(id string) error { return a.Todo.PurgeItem(id) }

// TodoPurgeDeletedItems 清空回收站条目，返回数量。
func (a *App) TodoPurgeDeletedItems() (int64, error) { return a.Todo.PurgeDeletedItems() }

// TodoListDeletedItems 列出回收站条目。
func (a *App) TodoListDeletedItems() ([]todo.Item, error) { return a.Todo.ListDeletedItems() }

// TodoReorderItems 重排条目顺序。
func (a *App) TodoReorderItems(listID string, ids []string) error { return a.Todo.ReorderItems(listID, ids) }

// TodoListToday 今日到期待办。
func (a *App) TodoListToday() ([]todo.Item, error) { return a.Todo.ListToday() }

// TodoListOverdue 逾期未办。
func (a *App) TodoListOverdue() ([]todo.Item, error) { return a.Todo.ListOverdue() }

// TodoListUpcoming 未来 7 天待办。
func (a *App) TodoListUpcoming() ([]todo.Item, error) { return a.Todo.ListUpcoming() }

// TodoListReminders 最近到提醒时间的条目。
func (a *App) TodoListReminders(limit int) ([]todo.Item, error) { return a.Todo.ListReminders(limit) }

// TodoSearch 搜索待办。
func (a *App) TodoSearch(keyword string, limit int) ([]todo.Item, error) { return a.Todo.Search(keyword, limit) }

// TodoSummary 活跃待办总览。
func (a *App) TodoSummary() (*todo.Summary, error) { return a.Todo.Summary() }

// TodoAIBreakdown 用 LLM 拆解任务为子任务列表（未入库，由前端批量创建）。
func (a *App) TodoAIBreakdown(title, notes string) ([]todo.Item, error) {
	return a.Todo.AIBreakdown(a.Ctx, title, notes)
}

// ===== Pomodoro 番茄钟 =====

// PomodoroCreate 创建一条番茄钟记录。
func (a *App) PomodoroCreate(p pomodoro.CreateParams) (*pomodoro.Record, error) {
	return a.Pomodoro.Create(p)
}

// PomodoroGet 读取记录。
func (a *App) PomodoroGet(id string) (*pomodoro.Record, error) { return a.Pomodoro.Get(id) }

// PomodoroListByTodo 列出关联某待办的记录。
func (a *App) PomodoroListByTodo(todoItemID string) ([]pomodoro.Record, error) {
	return a.Pomodoro.ListByTodo(todoItemID)
}

// PomodoroListToday 今日记录。
func (a *App) PomodoroListToday() ([]pomodoro.Record, error) { return a.Pomodoro.ListToday() }

// PomodoroTodayStats 今日统计。
func (a *App) PomodoroTodayStats() (*pomodoro.Stats, error) { return a.Pomodoro.TodayStats() }

// PomodoroDailyStats 最近 N 天每日专注统计。
func (a *App) PomodoroDailyStats(days int) ([]pomodoro.DailyStat, error) {
	return a.Pomodoro.DailyStats(days)
}

// ===== LLM 用量统计 =====

// LLMUsageRecord 记录一次 LLM 调用。
func (a *App) LLMUsageRecord(e llmusage.LogEntry) (*llmusage.Log, error) {
	return a.LLMUsage.Record(e)
}

// LLMUsageQuery 查询调用日志。
func (a *App) LLMUsageQuery(filter llmusage.LogFilter) ([]llmusage.Log, error) {
	return a.LLMUsage.Query(filter)
}

// LLMUsageQueryDaily 查询按日聚合。
func (a *App) LLMUsageQueryDaily(filter llmusage.DailyFilter) ([]llmusage.DailyAggregate, error) {
	return a.LLMUsage.QueryDaily(filter)
}

// LLMUsageSummary 用量总览。
func (a *App) LLMUsageSummary() (*llmusage.Summary, error) { return a.LLMUsage.Summary() }

// LLMUsageCleanup 清理指定日期之前的日志，返回删除数量。
func (a *App) LLMUsageCleanup(before time.Time) (int64, error) {
	return a.LLMUsage.CleanupOlderThan(before)
}

// ===== Obsidian 双链 / 图谱 =====

// VFSLinks 返回指定资源的出链（[[wikilink]]）。
func (a *App) VFSLinks(uri string) []vfs.LinkEntry { return a.vfs.Links(uri) }

// VFSBacklinks 返回指向指定资源的入链。
func (a *App) VFSBacklinks(uri string) []vfs.LinkEntry { return a.vfs.Backlinks(uri) }

// VFSGraph 返回全库双链图。
func (a *App) VFSGraph() []vfs.LinkEntry { return a.vfs.Graph() }

// VFSReload 重新扫描 vault（Obsidian 外部编辑后同步）。
func (a *App) VFSReload() error { return a.vfs.Reload() }

// ===== Template-management 模板管理 =====

// TemplateList 列出全部模板。
func (a *App) TemplateList() ([]templatemgr.Template, error) { return a.Templates.List() }

// TemplateGet 按 ID 读取模板。
func (a *App) TemplateGet(id string) (*templatemgr.Template, error) { return a.Templates.Get(id) }

// TemplateCreate 创建模板。
func (a *App) TemplateCreate(p templatemgr.CreateParams) (*templatemgr.Template, error) {
	return a.Templates.Create(p)
}

// TemplateUpdate 更新模板。
func (a *App) TemplateUpdate(p templatemgr.UpdateParams) (*templatemgr.Template, error) {
	return a.Templates.Update(p)
}

// TemplateDelete 删除模板。
func (a *App) TemplateDelete(id string) error { return a.Templates.Delete(id) }

// TemplateExport 导出模板 JSON。
func (a *App) TemplateExport(id string) ([]byte, error) { return a.Templates.Export(id) }

// TemplateImport 导入单个模板 JSON。
func (a *App) TemplateImport(data []byte) (*templatemgr.Template, error) {
	return a.Templates.Import(data)
}

// TemplateImportBulk 批量导入模板 JSON 数组。
func (a *App) TemplateImportBulk(data []byte) (imported, failed int, err error) {
	return a.Templates.ImportBulk(data)
}

// TemplateImportBuiltins 强制导入内置模板。
func (a *App) TemplateImportBuiltins() (int, error) { return a.Templates.ImportBuiltins() }

// TemplateSetDefault 设置默认模板。
func (a *App) TemplateSetDefault(id string) error { return a.Templates.SetDefault(id) }

// TemplateGetDefaultID 获取默认模板 ID。
func (a *App) TemplateGetDefaultID() (string, error) { return a.Templates.DefaultID() }

// TemplateValidate 校验模板字段。
func (a *App) TemplateValidate(name, front, back string) error {
	return a.Templates.Validate(name, front, back)
}

// ===== Voice-input 语音输入 =====

// VoiceTranscribe 转写音频（audioData 为原始音频字节，mime 如 audio/wav）。
func (a *App) VoiceTranscribe(audioData []byte, mime string) (*voiceinput.TranscribeResult, error) {
	return a.Voice.Transcribe(a.Ctx, audioData, mime)
}

// VoiceSetProvider 配置 ASR provider。
func (a *App) VoiceSetProvider(p voiceinput.Provider) { a.Voice.SetProvider(p) }

// VoiceProvider 返回当前 provider 配置。
func (a *App) VoiceProvider() voiceinput.Provider { return a.Voice.Provider() }

// ===== Cloud storage 云存储 =====

// CloudSaveConfig 保存云存储配置（凭据加密入库）。
func (a *App) CloudSaveConfig(cfg cloudstorage.Config) error { return a.Cloud.SaveConfig(cfg) }

// CloudLoadConfig 读取云存储配置。
func (a *App) CloudLoadConfig() (cloudstorage.Config, bool, error) { return a.Cloud.LoadConfig() }

// CloudClearConfig 清除配置。
func (a *App) CloudClearConfig() error { return a.Cloud.ClearConfig() }

// CloudCheckConnection 测试连接。
func (a *App) CloudCheckConnection() error { return a.Cloud.CheckConnection(a.Ctx) }

// CloudUploadBackup 上传本地备份到云端。
func (a *App) CloudUploadBackup(localPath, note string) (*cloudstorage.Version, error) {
	return a.Cloud.UploadBackup(a.Ctx, localPath, note)
}

// CloudDownloadLatest 下载最新备份到数据目录，返回本地路径。
func (a *App) CloudDownloadLatest() (string, error) {
	_, dest, err := a.Cloud.DownloadLatest(a.Ctx, cloudstorage.RestoreDir(a.cfg.DataDir))
	return dest, err
}

// CloudDownloadVersion 下载指定版本。
func (a *App) CloudDownloadVersion(key string) (string, error) {
	_, dest, err := a.Cloud.DownloadVersion(a.Ctx, key, cloudstorage.RestoreDir(a.cfg.DataDir))
	return dest, err
}

// CloudListVersions 列出云端版本。
func (a *App) CloudListVersions() ([]cloudstorage.Version, error) { return a.Cloud.ListVersions(a.Ctx) }

// CloudGetStatus 同步状态。
func (a *App) CloudGetStatus() (map[string]any, error) { return a.Cloud.GetStatus(a.Ctx) }

// CloudDeleteVersion 删除远端版本。
func (a *App) CloudDeleteVersion(key string) error { return a.Cloud.DeleteVersion(a.Ctx, key) }

// ===== 增量同步（data_governance sync）=====

// SyncRun 与云端执行一次增量同步（需先配置 CloudSaveConfig）。
func (a *App) SyncRun() (*sync.SyncOutcome, error) {
	cfg, ok, err := a.Cloud.LoadConfig()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("cloud not configured")
	}
	be, err := cloudstorage.NewBackend(cfg)
	if err != nil {
		return nil, err
	}
	deviceID, err := a.Cloud.DeviceID()
	if err != nil {
		return nil, err
	}
	remoteDir := cfg.RemoteDir
	if remoteDir == "" {
		remoteDir = "deepstudent-backups"
	}
	return a.Sync.SyncToCloud(a.Ctx, be, remoteDir, deviceID)
}

// SyncPending 返回待同步变更数。
func (a *App) SyncPending() (int64, error) {
	cur, err := a.Sync.Cursor()
	if err != nil {
		return 0, err
	}
	max, err := a.Sync.MaxSeq()
	if err != nil {
		return 0, err
	}
	return max - cur, nil
}

// SyncGetStatus 返回同步状态（游标/待同步/隔离区）。
func (a *App) SyncGetStatus() (map[string]any, error) {
	cur, _ := a.Sync.Cursor()
	max, _ := a.Sync.MaxSeq()
	qc, _ := a.Sync.QuarantineCount()
	cloudStatus, _ := a.Cloud.GetStatus(a.Ctx)
	return map[string]any{
		"cursor":     cur,
		"maxSeq":     max,
		"pending":    max - cur,
		"quarantine": qc,
		"cloud":      cloudStatus,
	}, nil
}

// SyncListQuarantine 列出隔离区。
func (a *App) SyncListQuarantine(limit int) ([]sync.QuarantineEntry, error) {
	return a.Sync.QuarantineList(limit)
}

// SyncRetryQuarantine 重试隔离记录。
func (a *App) SyncRetryQuarantine(id int64) error { return a.Sync.RetryQuarantine(id) }

// SyncDiscardQuarantine 丢弃隔离记录。
func (a *App) SyncDiscardQuarantine(id int64) error { return a.Sync.DiscardQuarantine(id) }

// SyncDiscardAllQuarantine 清空隔离区。
func (a *App) SyncDiscardAllQuarantine() (int64, error) { return a.Sync.DiscardAllQuarantine() }

// ===== Memory-as-VFS 扩展 =====

// MemoryWrite 写入一条记忆（显式分类/文件夹）。
func (a *App) MemoryWrite(fact, category, folderID, source string, tags []string) (*memory.Item, error) {
	return a.Mem.Write(fact, category, folderID, source, tags)
}

// MemoryWriteBatch 批量写入记忆。
func (a *App) MemoryWriteBatch(entries []memory.WriteEntry) ([]*memory.Item, error) {
	return a.Mem.WriteBatch(entries)
}

// MemoryList 列出全部记忆。
func (a *App) MemoryList() []*memory.Item { return a.Mem.List() }

// MemoryUpdateContent 更新记忆内容/分类/权重。
func (a *App) MemoryUpdateContent(id string, content, category *string, weight *int) (*memory.Item, error) {
	return a.Mem.UpdateContent(id, content, category, weight)
}

// MemoryDelete 删除记忆。
func (a *App) MemoryDelete(id string) error { return a.Mem.Delete(id) }

// MemoryUpdateTags 更新记忆标签。
func (a *App) MemoryUpdateTags(id string, tags []string) (*memory.Item, error) {
	return a.Mem.UpdateTags(id, tags)
}

// MemoryCreateFolder 创建记忆文件夹。
func (a *App) MemoryCreateFolder(name string, parentID *string) (*memory.Folder, error) {
	return a.Mem.CreateFolder(name, parentID)
}

// MemoryListFolders 列出记忆文件夹。
func (a *App) MemoryListFolders() ([]*memory.Folder, error) { return a.Mem.ListFolders() }

// MemoryGetTree 返回记忆文件夹树。
func (a *App) MemoryGetTree() ([]memory.FolderNode, error) { return a.Mem.GetTree() }

// MemoryDeleteFolder 删除记忆文件夹。
func (a *App) MemoryDeleteFolder(id string) error { return a.Mem.DeleteFolder(id) }

// MemoryMoveToFolder 移动记忆到文件夹。
func (a *App) MemoryMoveToFolder(id, folderID string) (*memory.Item, error) {
	return a.Mem.MoveToFolder(id, folderID)
}

// MemoryAddRelation 建立记忆关联。
func (a *App) MemoryAddRelation(sourceID, targetID, relType string) error {
	return a.Mem.AddRelation(sourceID, targetID, relType)
}

// MemoryGetRelated 返回关联记忆。
func (a *App) MemoryGetRelated(id string) ([]*memory.Item, error) { return a.Mem.GetRelated(id) }

// MemoryGetAuditLogs 读取记忆审计日志。
func (a *App) MemoryGetAuditLogs(limit int) ([]memory.AuditEntry, error) {
	return a.Mem.GetAuditLogs(limit)
}

// ===== OCR 多引擎 =====

// OCRRecognize 识别图片文字。
func (a *App) OCRRecognize(imageData []byte, mime string) (*ocr.OcrResult, error) {
	return a.OCR.Recognize(a.Ctx, imageData, mime)
}

// OCRListEngines 列出 OCR 引擎。
func (a *App) OCRListEngines() []ocr.EngineInfo { return a.OCR.ListEngines() }

// OCRSetEngineType 切换 OCR 引擎。
func (a *App) OCRSetEngineType(t string) error { return a.OCR.SetEngineType(ocr.EngineType(t)) }

// OCRGetEngineType 当前引擎。
func (a *App) OCRGetEngineType() string { return string(a.OCR.EngineType()) }

// OCRSetThinking 开关 VL 推理。
func (a *App) OCRSetThinking(on bool) { a.OCR.SetThinking(on) }

// OCRStartPDFSession 启动 PDF OCR 会话。
func (a *App) OCRStartPDFSession(pdfName string, pageCount int) (string, error) {
	return a.OCR.StartPDFSession(pdfName, pageCount)
}

// OCRUploadPage 上传一页识别。
func (a *App) OCRUploadPage(sessionID string, pageIndex int, imageData []byte, mime string) (string, error) {
	return a.OCR.UploadPage(a.Ctx, sessionID, pageIndex, imageData, mime)
}

// OCRCancelPDFSession 取消会话。
func (a *App) OCRCancelPDFSession(sessionID string) { a.OCR.CancelPDFSession(sessionID) }

// OCRExtractTextFromPDF 提取 PDF 文本层。
func (a *App) OCRExtractTextFromPDF(data []byte) (string, error) {
	return a.OCR.ExtractTextFromPDFBytes(data)
}

// ===== Multimodal 多模态索引 =====

// MultiIndexResource 为资源建立多模态索引。
func (a *App) MultiIndexResource(uri, content string) (int, error) {
	return a.Multi.IndexResource(a.Ctx, uri, content)
}

// MultiSearch 混合检索。
func (a *App) MultiSearch(query string, topK int) ([]multimodal.Result, error) {
	return a.Multi.Search(a.Ctx, query, topK)
}

// MultiDelete 删除资源索引。
func (a *App) MultiDelete(uri string) error { return a.Multi.Delete(uri) }

// MultiStats 索引统计。
func (a *App) MultiStats() (*multimodal.Stats, error) { return a.Multi.Stats() }

// ===== chat_v2 会话管理 =====

// ChatV2ListGroups 列出分组。
func (a *App) ChatV2ListGroups(includeDeleted bool) []*chat.Group { return a.Chat.ListGroups(includeDeleted) }

// ChatV2UpdateGroup 更新分组。
func (a *App) ChatV2UpdateGroup(g *chat.Group) error { return a.Chat.UpdateGroup(g) }

// ChatV2DeleteGroup 软删除分组。
func (a *App) ChatV2DeleteGroup(id string) error { return a.Chat.DeleteGroup(id) }

// ChatV2RestoreGroup 恢复分组。
func (a *App) ChatV2RestoreGroup(id string) error { return a.Chat.RestoreGroup(id) }

// ChatV2PurgeGroup 彻底删除分组。
func (a *App) ChatV2PurgeGroup(id string) error { return a.Chat.PurgeGroup(id) }

// ChatV2ListSessions 列出会话。
func (a *App) ChatV2ListSessions(filter chat.SessionFilter) []*chat.Session {
	return a.Chat.ListSessions(filter)
}

// ChatV2GetSession 读取会话。
func (a *App) ChatV2GetSession(id string) (*chat.Session, error) { return a.Chat.GetSession(id) }

// ChatV2UpdateTitle 修改会话标题。
func (a *App) ChatV2UpdateTitle(id, title string) error { return a.Chat.UpdateSessionTitle(id, title) }

// ChatV2Pin 置顶会话。
func (a *App) ChatV2Pin(id string, pinned bool) error { return a.Chat.PinSession(id, pinned) }

// ChatV2SoftDelete 软删除会话。
func (a *App) ChatV2SoftDelete(id string) error { return a.Chat.SoftDeleteSession(id) }

// ChatV2Restore 恢复会话。
func (a *App) ChatV2Restore(id string) error { return a.Chat.RestoreSession(id) }

// ChatV2Purge 彻底删除会话。
func (a *App) ChatV2Purge(id string) error { return a.Chat.PurgeSession(id) }

// ChatV2UpdateTags 更新会话标签。
func (a *App) ChatV2UpdateTags(id string, tags []string) error { return a.Chat.UpdateSessionTags(id, tags) }

// ChatV2Search 搜索会话消息。
func (a *App) ChatV2Search(keyword string, limit int) ([]chat.SearchHit, error) {
	return a.Chat.SearchContent(keyword, limit)
}

// ChatV2Count 会话总数。
func (a *App) ChatV2Count() (int64, error) { return a.Chat.CountSessions() }

// ChatV2DeleteMessage 删除单条消息。
func (a *App) ChatV2DeleteMessage(sessionID, messageID string) error {
	return a.Chat.DeleteMessage(sessionID, messageID)
}

// ChatV2RegisterTool 注册工具。
func (a *App) ChatV2RegisterTool(name string, fn chat.ToolFunc) { a.Chat.RegisterTool(name, fn) }

// ChatV2Tools 列出工具。
func (a *App) ChatV2Tools() []string { return a.Chat.Tools() }

// ChatV2Send 带工具循环发送消息。
func (a *App) ChatV2Send(sessionID, content string, refs []string) (string, []chat.ToolCallRecord, error) {
	return a.Chat.SendWithTools(a.Ctx, sessionID, content, refs, nil)
}

// ChatV2Export 导出全部会话 JSON。
func (a *App) ChatV2Export() ([]byte, error) { return a.Chat.ExportJSON() }

// ===== FSRS 间隔复习 =====

// FSRSAddCards 批量添加闪卡。
func (a *App) FSRSAddCards(deck string, cards []fsrs.CardInput) ([]*fsrs.CardState, error) {
	return a.FSRS.AddCards(deck, cards)
}

// FSRSDue 到期卡片。
func (a *App) FSRSDue(deck string, limit int) ([]*fsrs.CardState, error) {
	return a.FSRS.DueCards(deck, limit)
}

// FSRSAll 列出卡片。
func (a *App) FSRSAll(deck string, limit int) ([]*fsrs.CardState, error) {
	return a.FSRS.AllCards(deck, limit)
}

// FSRSReview 复习评分。
func (a *App) FSRSReview(cardID string, rating int) (*fsrs.CardState, error) {
	return a.FSRS.Review(cardID, fsrs.Rating(rating))
}

// FSRSGet 读取卡片。
func (a *App) FSRSGet(cardID string) (*fsrs.CardState, error) { return a.FSRS.Get(cardID) }

// FSRSReviewLogs 复习记录。
func (a *App) FSRSReviewLogs(cardID string, limit int) ([]fsrs.ReviewLog, error) {
	return a.FSRS.ReviewLogs(cardID, limit)
}

// FSRSDelete 删除卡片。
func (a *App) FSRSDelete(cardID string) error { return a.FSRS.Delete(cardID) }

// FSRSDueCount 到期数量。
func (a *App) FSRSDueCount() (int64, error) { return a.FSRS.DueCount() }

// FSRSDeckStats 牌组统计。
func (a *App) FSRSDeckStats() ([]fsrs.DeckStat, error) { return a.FSRS.DeckStats() }

// ===== 插件生态 =====

// PluginsInstall 安装插件。
func (a *App) PluginsInstall(name string, manifestJSON []byte, files map[string][]byte) (*plugins.Plugin, error) {
	return a.Plugins.Install(name, manifestJSON, files)
}

// PluginsList 列出插件。
func (a *App) PluginsList() ([]*plugins.Plugin, error) { return a.Plugins.List() }

// PluginsSetEnabled 启用/禁用。
func (a *App) PluginsSetEnabled(id string, enabled bool) error { return a.Plugins.SetEnabled(id, enabled) }

// PluginsUninstall 卸载。
func (a *App) PluginsUninstall(id string) error { return a.Plugins.Uninstall(id) }

// PluginsScanVault 扫描外部安装插件。
func (a *App) PluginsScanVault() ([]string, error) { return a.Plugins.ScanVault() }

// ===== 快速助手 =====

// QuickAsk 快速提问。
func (a *App) QuickAsk(question string) (string, error) { return a.Quick.Ask(a.Ctx, question) }

// QuickHistory 历史。
func (a *App) QuickHistory(limit int) []quickassist.Message { return a.Quick.History(limit) }

// QuickClear 清空。
func (a *App) QuickClear() { a.Quick.Clear() }

// QuickSummary 摘要。
func (a *App) QuickSummary() map[string]any { return a.Quick.Summary() }

// ===== 设置 KV（对齐原版 get_setting / save_setting / delete_setting）=====

// GetSetting 读取设置；不存在返回空串。
func (a *App) GetSetting(key string) (string, error) {
	v, _, err := a.store.GetSetting(key)
	return v, err
}

// SaveSetting 保存设置。
func (a *App) SaveSetting(key, value string) error {
	return a.store.SaveSetting(key, value)
}

// DeleteSetting 删除设置。
func (a *App) DeleteSetting(key string) (bool, error) {
	return a.store.DeleteSetting(key)
}

// GetSettingsByPrefix 按前缀查询设置。
func (a *App) GetSettingsByPrefix(prefix string) ([]store.SettingRow, error) {
	return a.store.GetSettingsByPrefix(prefix)
}

// LLMCfgListApiConfigurations 列出全部 API 配置（对齐原版 get_api_configurations）。
func (a *App) LLMCfgListApiConfigurations() []llmcfg.ApiConfig {
	return a.LLMCfg.ListApiConfigurations()
}
