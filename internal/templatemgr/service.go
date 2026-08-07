// templatemgr 包的业务层：CRUD / 导入导出 / 内置模板 seed / 默认模板。
//
// 构造函数 New(fs, store, llmReg) 与既有 internal 包保持一致。
// 默认模板 ID 存于 settings（SQLite kv 表）。

package templatemgr

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Service 模板管理服务。
type Service struct {
	store *Store
	mu    sync.RWMutex
}

// New 构造模板服务并执行迁移 + seed 内置模板。
func New(_ *vfs.FS, s *store.Store, _ *llm.Registry) *Service {
	ts := NewStore(s)
	if err := ts.Migrate(); err != nil {
		fmt.Printf("[templatemgr] migrate failed: %v\n", err)
	}
	svc := &Service{store: ts}
	if err := svc.seedBuiltins(); err != nil {
		fmt.Printf("[templatemgr] seed builtins failed: %v\n", err)
	}
	return svc
}

// BuiltinTemplates 内置模板定义。
func BuiltinTemplates() []Template {
	return []Template{
		{
			ID:        "default",
			Name:      "Default",
			FrontTmpl: `<div class="card">{{Front}}</div>`,
			BackTmpl:  `<div class="card">{{Front}}</div><hr id=answer>{{Back}}`,
			SharedCSS: `.card { font-family: sans-serif; font-size: 18px; padding: 12px; text-align: center; }`,
			IsBuiltin: true,
		},
		{
			ID:        "basic-reversed",
			Name:      "Basic (and reversed card)",
			FrontTmpl: `<div class="card">{{Front}}</div>`,
			BackTmpl:  `<div class="card">{{Front}}</div><hr id=answer>{{Back}}`,
			SharedCSS: `.card { font-family: sans-serif; font-size: 18px; padding: 12px; }`,
			IsBuiltin: true,
		},
		{
			ID:        "cloze",
			Name:      "Cloze",
			FrontTmpl: `<div class="card">{{cloze:Text}}</div>`,
			BackTmpl:  `<div class="card">{{cloze:Text}}</div><hr id=answer>{{Extra}}`,
			SharedCSS: `.cloze { font-weight: bold; color: blue; } .card { font-family: sans-serif; font-size: 18px; padding: 12px; }`,
			IsBuiltin: true,
		},
		{
			ID:        "qa-bilingual",
			Name:      "双语问答",
			FrontTmpl: `<div class="card">{{Front}}</div>`,
			BackTmpl:  `<div class="card">{{Front}}</div><hr id=answer><div class="en">{{Back}}</div><div class="zh">{{BackExtra}}</div>`,
			SharedCSS: `.card { font-family: sans-serif; font-size: 18px; } .en { color: #333; } .zh { color: #888; margin-top: 8px; }`,
			IsBuiltin: true,
		},
	}
}

// seedBuiltins 幂等 seed 内置模板。
func (s *Service) seedBuiltins() error {
	cnt, err := s.store.Count()
	if err != nil {
		return err
	}
	if cnt > 0 {
		return nil // 已有数据（含用户模板），跳过
	}
	now := time.Now().UTC()
	for i, t := range BuiltinTemplates() {
		t.SortOrder = i
		t.CreatedAt = now
		t.UpdatedAt = now
		if err := s.store.Create(&t); err != nil {
			return err
		}
	}
	return nil
}

// List 列出全部模板。
func (s *Service) List() ([]Template, error) {
	ts, err := s.store.List()
	if err != nil {
		return nil, err
	}
	out := make([]Template, 0, len(ts))
	for _, t := range ts {
		out = append(out, *t)
	}
	return out, nil
}

// Get 按 ID 读取。
func (s *Service) Get(id string) (*Template, error) {
	return s.store.Get(id)
}

// Create 创建模板。
func (s *Service) Create(p CreateParams) (*Template, error) {
	if strings.TrimSpace(p.Name) == "" {
		return nil, errors.New("templatemgr: name required")
	}
	if strings.TrimSpace(p.FrontTmpl) == "" || strings.TrimSpace(p.BackTmpl) == "" {
		return nil, errors.New("templatemgr: front/back required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	t := &Template{
		ID:        uuid.NewString(),
		Name:      strings.TrimSpace(p.Name),
		FrontTmpl: p.FrontTmpl,
		BackTmpl:  p.BackTmpl,
		Style:     p.Style,
		SharedCSS: p.SharedCSS,
		Preview:   p.Preview,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.Create(t); err != nil {
		return nil, err
	}
	return t, nil
}

// Update 更新模板。
func (s *Service) Update(p UpdateParams) (*Template, error) {
	if p.Name != nil && strings.TrimSpace(*p.Name) == "" {
		return nil, errors.New("templatemgr: name cannot be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.Update(p); err != nil {
		return nil, err
	}
	return s.store.Get(p.ID)
}

// Delete 删除模板（内置保护）。
func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.Delete(id); err != nil {
		return err
	}
	// 若删除了默认模板，重置为 default
	if cur, _ := s.DefaultID(); cur == id {
		_ = s.SetDefault("default")
	}
	return nil
}

// Export 导出模板为 JSON。
func (s *Service) Export(id string) ([]byte, error) {
	t, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(ImportEntry{
		Name: t.Name, FrontTmpl: t.FrontTmpl, BackTmpl: t.BackTmpl,
		Style: t.Style, SharedCSS: t.SharedCSS, Preview: t.Preview,
	}, "", "  ")
}

// Import 从 JSON 导入单个模板。
func (s *Service) Import(data []byte) (*Template, error) {
	var e ImportEntry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("templatemgr: bad template json: %w", err)
	}
	return s.Create(CreateParams{
		Name: e.Name, FrontTmpl: e.FrontTmpl, BackTmpl: e.BackTmpl,
		Style: e.Style, SharedCSS: e.SharedCSS, Preview: e.Preview,
	})
}

// ImportBulk 批量导入模板数组，返回成功/失败数。
func (s *Service) ImportBulk(data []byte) (imported, failed int, err error) {
	var entries []ImportEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return 0, 0, fmt.Errorf("templatemgr: bad bulk json: %w", err)
	}
	for _, e := range entries {
		if strings.TrimSpace(e.Name) == "" {
			failed++
			continue
		}
		if _, err := s.Create(CreateParams{
			Name: e.Name, FrontTmpl: e.FrontTmpl, BackTmpl: e.BackTmpl,
			Style: e.Style, SharedCSS: e.SharedCSS, Preview: e.Preview,
		}); err != nil {
			failed++
			continue
		}
		imported++
	}
	return imported, failed, nil
}

// ImportBuiltins 强制重新导入内置模板（已存在则跳过同名）。
func (s *Service) ImportBuiltins() (int, error) {
	existing, err := s.store.List()
	if err != nil {
		return 0, err
	}
	names := map[string]bool{}
	for _, t := range existing {
		names[t.Name] = true
	}
	now := time.Now().UTC()
	added := 0
	for i, t := range BuiltinTemplates() {
		if names[t.Name] {
			continue
		}
		t.SortOrder = i
		t.CreatedAt = now
		t.UpdatedAt = now
		if err := s.store.Create(&t); err != nil {
			return added, err
		}
		added++
	}
	return added, nil
}

// Validate 校验模板字段（前端用）。
func (s *Service) Validate(name, front, back string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("templatemgr: name required")
	}
	if strings.TrimSpace(front) == "" {
		return errors.New("templatemgr: front template required")
	}
	if strings.TrimSpace(back) == "" {
		return errors.New("templatemgr: back template required")
	}
	return nil
}

// ===================== 默认模板 =====================

const defaultTplKey = "anki.default_template_id"

// SetDefault 设置默认模板 ID。
func (s *Service) SetDefault(id string) error {
	// 校验存在
	if _, err := s.store.Get(id); err != nil {
		return err
	}
	return s.setSetting(defaultTplKey, id)
}

// DefaultID 返回默认模板 ID（不存在则 fallback default）。
func (s *Service) DefaultID() (string, error) {
	id, ok := s.getSetting(defaultTplKey)
	if !ok {
		// 兼容旧数据：尝试 default 模板存在性
		if _, err := s.store.Get("default"); err == nil {
			return "default", nil
		}
		ts, err := s.store.List()
		if err != nil || len(ts) == 0 {
			return "", errors.New("templatemgr: no templates")
		}
		return ts[0].ID, nil
	}
	return id, nil
}

// Template 按 ID 取模板（供 anki 制卡使用）；空 ID 返回默认模板。
func (s *Service) Template(id string) (*Template, error) {
	if id == "" {
		id, err := s.DefaultID()
		if err != nil {
			return nil, err
		}
		return s.store.Get(id)
	}
	return s.store.Get(id)
}

// ===================== settings kv =====================

func (s *Service) setSetting(key, value string) error {
	if s.store.db == nil {
		return errors.New("templatemgr: nil db")
	}
	_, err := s.store.db.Exec(`INSERT INTO app_settings(key, value) VALUES (?,?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

func (s *Service) getSetting(key string) (string, bool) {
	if s.store.db == nil {
		return "", false
	}
	var v string
	err := s.store.db.QueryRow(`SELECT value FROM app_settings WHERE key=?`, key).Scan(&v)
	if err != nil {
		return "", false
	}
	return v, true
}
