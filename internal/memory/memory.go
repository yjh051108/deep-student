// Package memory 智能记忆：会话后事实抽取、向量去重、批量写入、画像聚合、衰减、隐私模式。
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Action 记忆操作决策。
type Action string

const (
	ActionAdd    Action = "ADD"
	ActionUpdate Action = "UPDATE"
	ActionAppend Action = "APPEND"
	ActionDelete Action = "DELETE"
	ActionNone   Action = "NONE"
)

// Item 单条记忆。
type Item struct {
	ID        string            `json:"id"`
	FolderID  *string           `json:"folder_id,omitempty"` // 记忆文件夹
	Category  string            `json:"category"` // identity | preference | goal | subject | other
	Content   string            `json:"content"`
	Tags      []string          `json:"tags"`
	Weight    int               `json:"weight"`
	HitCount  int               `json:"hit_count"`
	LastHit   time.Time         `json:"last_hit"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Source    string            `json:"source,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Service 记忆服务。
type Service struct {
	vfs    *vfs.FS
	store  *store.Store
	db     *Store
	llm    *llm.Registry
	mu     sync.Mutex
	items  map[string]*Item
	privacy bool
}

// New 创建 Service。自动建表；内存 map 作为读缓存，写操作落库。
func New(fs *vfs.FS, st *store.Store, l *llm.Registry) *Service {
	ms := NewStore(st)
	if err := ms.Migrate(); err != nil {
		fmt.Printf("[memory] migrate failed: %v\n", err)
	}
	svc := &Service{vfs: fs, store: st, db: ms, llm: l, items: map[string]*Item{}}
	// 启动时从库加载到内存
	if rows, err := ms.ListItems(); err == nil {
		for _, it := range rows {
			svc.items[it.ID] = it
		}
	}
	return svc
}

// ExtractFromConversation 从对话中抽取事实。
func (s *Service) ExtractFromConversation(ctx context.Context, conversation string) ([]string, error) {
	p, ok := s.llm.Get("openai")
	if !ok {
		return nil, fmt.Errorf("memory: no provider")
	}
	resp, err := p.Chat(ctx, llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "Extract durable facts about the user from the conversation. Categories: identity, preference, goal, subject status, other. Return a JSON array of strings, one fact per element. Skip transient or unclear content."},
			{Role: llm.RoleUser, Content: conversation},
		},
	})
	if err != nil {
		return nil, err
	}
	start := strings.Index(resp.Content, "[")
	end := strings.LastIndex(resp.Content, "]")
	var facts []string
	if start >= 0 && end > start {
		_ = json.Unmarshal([]byte(resp.Content[start:end+1]), &facts)
	}
	return facts, nil
}

// Decide 对每条事实给出 ADD/UPDATE/APPEND/DELETE/NONE 决策。
func (s *Service) Decide(ctx context.Context, fact string) (Action, string, error) {
	s.mu.Lock()
	existing := make([]*Item, 0, len(s.items))
	for _, it := range s.items {
		existing = append(existing, it)
	}
	s.mu.Unlock()
	summary := make([]string, 0, len(existing))
	for _, e := range existing {
		summary = append(summary, fmt.Sprintf("[%s] %s", e.ID, e.Content))
	}
	p, _ := s.llm.Get("openai")
	if p == nil {
		return ActionAdd, "", nil
	}
	resp, err := p.Chat(ctx, llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: `Decide whether the new fact should be ADD, UPDATE, APPEND, DELETE, or NONE. Return JSON: {"action":"ADD|UPDATE|APPEND|DELETE|NONE","target_id":"<id or empty>","content":"<final content>"}`},
			{Role: llm.RoleUser, Content: "Existing memories:\n" + strings.Join(summary, "\n") + "\n\nNew fact: " + fact},
		},
	})
	if err != nil {
		return ActionNone, "", err
	}
	start := strings.Index(resp.Content, "{")
	end := strings.LastIndex(resp.Content, "}")
	var dec struct {
		Action   string `json:"action"`
		TargetID string `json:"target_id"`
		Content  string `json:"content"`
	}
	if start >= 0 && end > start {
		_ = json.Unmarshal([]byte(resp.Content[start:end+1]), &dec)
	}
	return Action(dec.Action), dec.TargetID, nil
}

// ApplyFact 应用一个事实（落库 + 内存）。
func (s *Service) ApplyFact(fact string) *Item {
	now := time.Now()
	it := &Item{
		ID: uuid.NewString(), Category: "other", Content: fact,
		CreatedAt: now, UpdatedAt: now, LastHit: now, Weight: 1,
	}
	s.mu.Lock()
	s.items[it.ID] = it
	s.mu.Unlock()
	if s.db != nil {
		_ = s.db.CreateItem(it)
		_ = s.db.LogAudit("add", it.ID, fact)
	}
	return it
}

// Ingest 抽取 + 决策 + 写入。
func (s *Service) Ingest(ctx context.Context, conversation string) ([]*Item, error) {
	facts, err := s.ExtractFromConversation(ctx, conversation)
	if err != nil {
		return nil, err
	}
	out := make([]*Item, 0, len(facts))
	for _, f := range facts {
		action, _, _ := s.Decide(ctx, f)
		switch action {
		case ActionAdd, ActionUpdate, ActionAppend:
			out = append(out, s.ApplyFact(f))
		}
	}
	return out, nil
}

// Search 简单关键词搜索（命中计数落库）。
func (s *Service) Search(q string) []*Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Item, 0)
	for _, it := range s.items {
		if strings.Contains(strings.ToLower(it.Content), strings.ToLower(q)) {
			it.HitCount++
			it.LastHit = time.Now()
			out = append(out, it)
			if s.db != nil {
				_ = s.db.UpdateItem(it)
			}
		}
	}
	return out
}

// Profile 汇总为用户画像。
func (s *Service) Profile() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	byCat := map[string][]*Item{}
	for _, it := range s.items {
		byCat[it.Category] = append(byCat[it.Category], it)
	}
	var sb strings.Builder
	for cat, items := range byCat {
		sb.WriteString("## " + cat + "\n")
		for _, it := range items {
			sb.WriteString("- " + it.Content + "\n")
		}
	}
	return sb.String()
}

// Decay 90 天未用降权 / 高频加权（变更落库）。
func (s *Service) Decay() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for _, it := range s.items {
		days := int(now.Sub(it.LastHit).Hours() / 24)
		if days > 90 {
			it.Weight = maxInt(0, it.Weight-1)
		}
		if it.HitCount > 5 {
			it.Weight = minInt(10, it.Weight+1)
		}
		if s.db != nil {
			_ = s.db.UpdateItem(it)
		}
	}
}

// PrivacyMode 隐私模式：阻止所有外部 API。
func (s *Service) PrivacyMode(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if on {
		s.llm = llm.NewRegistry() // 清空所有 provider
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ===================== Memory-as-VFS 扩展 =====================

// Write 写入一条记忆（显式分类/文件夹/来源），返回条目。
func (s *Service) Write(fact, category, folderID, source string, tags []string) (*Item, error) {
	now := time.Now().UTC()
	it := &Item{
		ID: uuid.NewString(), Category: category, Content: fact,
		Tags: tags, Weight: 1, CreatedAt: now, UpdatedAt: now, LastHit: now,
		Source: source, Metadata: map[string]string{},
	}
	if folderID != "" {
		it.FolderID = &folderID
	}
	if it.Category == "" {
		it.Category = "other"
	}
	s.mu.Lock()
	s.items[it.ID] = it
	s.mu.Unlock()
	if s.db != nil {
		if err := s.db.CreateItem(it); err != nil {
			return nil, err
		}
		_ = s.db.LogAudit("write", it.ID, fact)
	}
	return it, nil
}

// WriteBatch 批量写入。
func (s *Service) WriteBatch(entries []WriteEntry) ([]*Item, error) {
	var out []*Item
	for _, e := range entries {
		it, err := s.Write(e.Content, e.Category, e.FolderID, e.Source, e.Tags)
		if err != nil {
			return out, err
		}
		out = append(out, it)
	}
	return out, nil
}

// Get 读取单条记忆。
func (s *Service) Get(id string) (*Item, error) {
	if s.db != nil {
		return s.db.GetItem(id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[id]
	if !ok {
		return nil, fmt.Errorf("memory: not found: %s", id)
	}
	return it, nil
}

// List 列出全部记忆。
func (s *Service) List() []*Item {
	if s.db != nil {
		if rows, err := s.db.ListItems(); err == nil {
			return rows
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Item, 0, len(s.items))
	for _, it := range s.items {
		out = append(out, it)
	}
	return out
}

// UpdateContent 更新记忆内容/分类/权重。
func (s *Service) UpdateContent(id string, content, category *string, weight *int) (*Item, error) {
	it, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if content != nil {
		it.Content = *content
	}
	if category != nil {
		it.Category = *category
	}
	if weight != nil {
		it.Weight = *weight
	}
	it.UpdatedAt = time.Now().UTC()
	if s.db != nil {
		if err := s.db.UpdateItem(it); err != nil {
			return nil, err
		}
		_ = s.db.LogAudit("update", id, it.Content)
	}
	s.mu.Lock()
	s.items[id] = it
	s.mu.Unlock()
	return it, nil
}

// Delete 删除记忆。
func (s *Service) Delete(id string) error {
	if s.db != nil {
		if err := s.db.DeleteItem(id); err != nil {
			return err
		}
		_ = s.db.LogAudit("delete", id, "")
	}
	s.mu.Lock()
	delete(s.items, id)
	s.mu.Unlock()
	return nil
}

// UpdateTags 更新记忆标签。
func (s *Service) UpdateTags(id string, tags []string) (*Item, error) {
	if s.db != nil {
		if err := s.db.UpdateItemTags(id, tags); err != nil {
			return nil, err
		}
	}
	if it, err := s.Get(id); err == nil {
		it.Tags = tags
		s.mu.Lock()
		s.items[id] = it
		s.mu.Unlock()
		return it, nil
	}
	return s.Get(id)
}

// ===================== 文件夹 =====================

// CreateFolder 创建记忆文件夹。
func (s *Service) CreateFolder(name string, parentID *string) (*Folder, error) {
	now := time.Now().UTC()
	f := &Folder{ID: uuid.NewString(), Name: name, ParentID: parentID, CreatedAt: now, UpdatedAt: now}
	if s.db != nil {
		if err := s.db.CreateFolder(f); err != nil {
			return nil, err
		}
		_ = s.db.LogAudit("create-folder", f.ID, name)
	}
	return f, nil
}

// ListFolders 列出文件夹。
func (s *Service) ListFolders() ([]*Folder, error) {
	if s.db != nil {
		return s.db.ListFolders()
	}
	return nil, nil
}

// DeleteFolder 删除文件夹（其下记忆移到根）。
func (s *Service) DeleteFolder(id string) error {
	if s.db != nil {
		if err := s.db.DeleteFolder(id); err != nil {
			return err
		}
		_ = s.db.LogAudit("delete-folder", id, "")
	}
	return nil
}

// MoveToFolder 移动记忆到文件夹。
func (s *Service) MoveToFolder(id, folderID string) (*Item, error) {
	if s.db != nil {
		if err := s.db.MoveItemToFolder(id, folderID); err != nil {
			return nil, err
		}
	}
	it, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	fid := folderID
	it.FolderID = &fid
	it.UpdatedAt = time.Now().UTC()
	s.mu.Lock()
	s.items[id] = it
	s.mu.Unlock()
	return it, nil
}

// GetTree 返回文件夹树（含各文件夹记忆数）。
func (s *Service) GetTree() ([]FolderNode, error) {
	folders, err := s.ListFolders()
	if err != nil {
		return nil, err
	}
	items := s.List()
	counts := map[string]int{}
	for _, it := range items {
		if it.FolderID != nil {
			counts[*it.FolderID]++
		}
	}
	nodeOf := func(f *Folder) FolderNode {
		return FolderNode{ID: f.ID, Name: f.Name, ParentID: f.ParentID, Count: counts[f.ID], Children: []FolderNode{}}
	}
	byID := map[string]*FolderNode{}
	var roots []FolderNode
	for _, f := range folders {
		n := nodeOf(f)
		byID[f.ID] = &n
		if f.ParentID == nil {
			roots = append(roots, n)
		}
	}
	// 二次构建父子关系
	built := map[string]bool{}
	var build func(id string) FolderNode
	build = func(id string) FolderNode {
		if built[id] {
			return *byID[id]
		}
		built[id] = true
		var children []FolderNode
		for _, f := range folders {
			if f.ParentID != nil && *f.ParentID == id {
				children = append(children, build(f.ID))
			}
		}
		n := *byID[id]
		n.Children = children
		return n
	}
	// 根 = 无父节点
	var out []FolderNode
	for _, f := range folders {
		if f.ParentID == nil {
			out = append(out, build(f.ID))
		}
	}
	return out, nil
}

// ===================== 关系 =====================

// AddRelation 建立两条记忆间的双向关联。
func (s *Service) AddRelation(sourceID, targetID, relType string) error {
	if s.db != nil {
		if err := s.db.AddRelation(sourceID, targetID, relType); err != nil {
			return err
		}
	}
	return nil
}

// GetRelated 返回与某记忆关联的记忆。
func (s *Service) GetRelated(id string) ([]*Item, error) {
	if s.db == nil {
		return nil, nil
	}
	ids, err := s.db.ListRelations(id)
	if err != nil {
		return nil, err
	}
	var out []*Item
	for _, rid := range ids {
		if it, err := s.db.GetItem(rid); err == nil {
			out = append(out, it)
		}
	}
	return out, nil
}

// ===================== 配置 / 审计 =====================

// SetAutoExtractFrequency 设置自动提取频率（manual|hourly|daily）。
func (s *Service) SetAutoExtractFrequency(freq string) error {
	if s.db != nil {
		return s.db.SetConfig("auto_extract_frequency", freq)
	}
	return nil
}

// SetDefaultCategory 设置默认分类。
func (s *Service) SetDefaultCategory(cat string) error {
	if s.db != nil {
		return s.db.SetConfig("default_category", cat)
	}
	return nil
}

// GetAuditLogs 读取审计日志。
func (s *Service) GetAuditLogs(limit int) ([]AuditEntry, error) {
	if s.db != nil {
		return s.db.AuditLogs(limit)
	}
	return nil, nil
}

// FolderNode 文件夹树节点。
type FolderNode struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	ParentID *string      `json:"parentId,omitempty"`
	Count    int          `json:"count"`
	Children []FolderNode `json:"children"`
}

// WriteEntry 批量写入条目。
type WriteEntry struct {
	Content  string   `json:"content"`
	Category string   `json:"category"`
	FolderID string   `json:"folderId"`
	Source   string   `json:"source"`
	Tags     []string `json:"tags"`
}
