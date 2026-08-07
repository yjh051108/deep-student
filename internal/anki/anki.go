// Package anki Anki 制卡：聊天触发、模板、任务板、3D 预览、同步。
package anki

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/helixnow/deep-student-go/pkg/eventbus"
	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Card Anki 卡片。
type Card struct {
	ID       string   `json:"id"`
	Deck     string   `json:"deck"`
	Front    string   `json:"front"`
	Back     string   `json:"back"`
	Tags     []string `json:"tags,omitempty"`
	Source   string   `json:"source,omitempty"` // vfs:// 来源
	Template string   `json:"template,omitempty"`
}

// Template 模板。
type Template struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	FrontTmpl string `json:"front"`
	BackTmpl  string `json:"back"`
	Style     string `json:"style"`
	SharedCSS string `json:"css"`
}

// Job 制卡任务。
type Job struct {
	ID         string    `json:"id"`
	Deck       string    `json:"deck"`
	SourceURI  string    `json:"source_uri"`
	Total      int       `json:"total"`
	Done       int       `json:"done"`
	Cards      []Card    `json:"cards"`
	Status     string    `json:"status"` // pending | running | done | failed
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
}

// Service Anki 服务。
type Service struct {
	vfs       *vfs.FS
	llm       *llm.Registry
	bus       *eventbus.Bus
	mu        sync.Mutex
	templates map[string]*Template
	jobs      map[string]*Job
}

// New 创建 Service。
func New(fs *vfs.FS, l *llm.Registry, bus *eventbus.Bus) *Service {
	s := &Service{vfs: fs, llm: l, bus: bus, templates: map[string]*Template{}, jobs: map[string]*Job{}}
	// 预置 1 套模板
	s.templates["default"] = &Template{
		ID: "default", Name: "Default",
		FrontTmpl: `<div class="card">{{Front}}</div>`,
		BackTmpl:  `<div class="card">{{Front}}</div><hr><div>{{Back}}</div>`,
		SharedCSS: `.card { font-family: sans-serif; font-size: 18px; padding: 12px; }`,
	}
	return s
}

// GenerateFromText 文本 → 卡片（流式回调进度）。
func (s *Service) GenerateFromText(ctx context.Context, deck, text, templateID string, batch int, onProgress func(*Job)) (*Job, error) {
	p, ok := s.llm.Get("openai")
	if !ok {
		return nil, fmt.Errorf("anki: no provider")
	}
	job := &Job{ID: uuid.NewString(), Deck: deck, Status: "running", StartedAt: time.Now()}
	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()
	tpl := s.templates["default"]
	if templateID != "" {
		if t, ok := s.templates[templateID]; ok {
			tpl = t
		}
	}

	// 分批生成
	chunks := splitText(text, 1500)
	job.Total = len(chunks) * batch
	for _, chunk := range chunks {
		resp, err := p.Chat(ctx, llm.ChatRequest{
			Model: "gpt-4o-mini",
			Messages: []llm.Message{
				{Role: llm.RoleSystem, Content: "You are an Anki card generator. Produce " + fmt.Sprintf("%d", batch) +
					" high-quality cloze/QA flashcards in JSON array: [{\"front\":\"...\",\"back\":\"...\",\"tags\":[\"...\"]}]"},
				{Role: llm.RoleUser, Content: "Source:\n" + chunk},
			},
		})
		if err != nil {
			job.Status = "failed"
			job.FinishedAt = time.Now()
			return job, err
		}
		var batchCards []struct {
			Front string   `json:"front"`
			Back  string   `json:"back"`
			Tags  []string `json:"tags"`
		}
		start := strings.Index(resp.Content, "[")
		end := strings.LastIndex(resp.Content, "]")
		if start >= 0 && end > start {
			_ = json.Unmarshal([]byte(resp.Content[start:end+1]), &batchCards)
		}
		for _, c := range batchCards {
			card := Card{
				ID: uuid.NewString(), Deck: deck, Front: c.Front, Back: c.Back, Tags: c.Tags,
				Template: tpl.Name, Source: job.SourceURI,
			}
			job.Cards = append(job.Cards, card)
			job.Done++
		}
		if onProgress != nil {
			onProgress(job)
		}
		s.bus.PublishAsync(ctx, "anki.progress", job)
	}
	job.Status = "done"
	job.FinishedAt = time.Now()
	return job, nil
}

// SaveToVFS 写入 VFS。
func (s *Service) SaveToVFS(job *Job) (string, error) {
	uri := fmt.Sprintf("vfs://flashcard/%s", job.ID)
	data, _ := json.Marshal(job)
	_, err := s.vfs.Put(uri, data, map[string]string{"title": job.Deck, "tags": strings.Join([]string{"anki", job.Deck}, ",")})
	return uri, err
}

// ExportAPKG 导出 .apkg 格式（最小化骨架，实际为 JSON collection）。
func (s *Service) ExportAPKG(job *Job) ([]byte, error) {
	return json.MarshalIndent(job.Cards, "", "  ")
}

// AddTemplate 添加模板。
func (s *Service) AddTemplate(t *Template) { s.mu.Lock(); s.templates[t.ID] = t; s.mu.Unlock() }

// Templates 列出模板。
func (s *Service) Templates() []*Template {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Template, 0, len(s.templates))
	for _, t := range s.templates {
		out = append(out, t)
	}
	return out
}

// Job 读取。
func (s *Service) Job(id string) *Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.jobs[id]
}

func splitText(s string, max int) []string {
	if len(s) <= max {
		return []string{s}
	}
	var out []string
	for i := 0; i < len(s); i += max {
		end := i + max
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
	}
	return out
}
