// Package hub 学习中心：资源/笔记/教材/题库/思维导图/翻译/卡片统一管理 + VFS + 向量化流水线。
package hub

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Service 学习中心服务。
type Service struct {
	vfs   *vfs.FS
	store *store.Store
	llm   *llm.Registry
}

// New 创建 Service。
func New(fs *vfs.FS, st *store.Store, l *llm.Registry) *Service {
	return &Service{vfs: fs, store: st, llm: l}
}

// ImportResource 导入任意资源（PDF / DOCX / Note ...）。
func (s *Service) ImportResource(ctx context.Context, typ vfs.ResourceType, title string, data []byte, tags []string) (string, error) {
	id := uuid.NewString()
	uri := fmt.Sprintf("vfs://%s/%s", typ, id)
	meta := map[string]string{
		"title": title,
		"tags":  strings.Join(tags, ","),
	}
	entry, err := s.vfs.Put(uri, data, meta)
	if err != nil {
		return "", err
	}
	if err := s.store.SaveResource(entry.URI, string(entry.Type), entry.ID, entry.Title, strings.Join(entry.Tags, ","),
		fmt.Sprintf(`{"created":%d}`, entry.CreatedAt), entry.BlobRef, entry.Size, entry.CreatedAt); err != nil {
		return "", err
	}
	return uri, nil
}

// List 列出某类型资源。
func (s *Service) List(typ vfs.ResourceType) []vfs.Entry { return s.vfs.List(typ) }

// Search 按 tag 搜索。
func (s *Service) Search(typ vfs.ResourceType, tag string) []vfs.Entry { return s.vfs.Search(typ, tag) }

// Get 读取资源。
func (s *Service) Get(uri string) ([]byte, vfs.Entry, error) { return s.vfs.Get(uri) }

// Delete 删除资源。
func (s *Service) Delete(uri string) error {
	if err := s.store.DeleteResource(uri); err != nil {
		return err
	}
	return s.vfs.Delete(uri)
}

// ContinueNote AI 续写笔记（流式）。
func (s *Service) ContinueNote(ctx context.Context, uri, prompt string) (<-chan string, error) {
	data, _, err := s.vfs.Get(uri)
	if err != nil {
		return nil, err
	}
	p, ok := s.llm.Get("openai")
	if !ok {
		return nil, fmt.Errorf("hub: no provider available")
	}
	req := llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a thoughtful writing assistant. Continue the user's note in the same style and language."},
			{Role: llm.RoleUser, Content: "Existing note:\n" + string(data) + "\n\nUser direction:\n" + prompt},
		},
	}
	ch, err := p.Stream(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make(chan string, 16)
	go func() {
		defer close(out)
		for c := range ch {
			if c.Delta != "" {
				out <- c.Delta
			}
		}
	}()
	return out, nil
}
