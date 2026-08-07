package deepstudent

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/helixnow/deep-student-go/internal/anki"
	"github.com/helixnow/deep-student-go/internal/chat"
	"github.com/helixnow/deep-student-go/internal/essay"
	"github.com/helixnow/deep-student-go/internal/llmcfg"
	"github.com/helixnow/deep-student-go/internal/memory"
	"github.com/helixnow/deep-student-go/internal/mindmap"
	"github.com/helixnow/deep-student-go/internal/notes"
	"github.com/helixnow/deep-student-go/internal/paper"
	"github.com/helixnow/deep-student-go/internal/qbank"
	"github.com/helixnow/deep-student-go/internal/reader"
	"github.com/helixnow/deep-student-go/internal/research"
	"github.com/helixnow/deep-student-go/internal/skills"
	"github.com/helixnow/deep-student-go/internal/translate"
	"github.com/helixnow/deep-student-go/pkg/config"
	"github.com/helixnow/deep-student-go/pkg/index"
	"github.com/helixnow/deep-student-go/pkg/llm"
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
