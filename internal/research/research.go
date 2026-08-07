// Package research 深度调研：7 引擎搜索、任务分解、报告自动入库。
package research

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

// Engine 搜索引擎枚举。
type Engine string

const (
	EngineGoogle  Engine = "google"
	EngineSerpAPI Engine = "serpapi"
	EngineTavily  Engine = "tavily"
	EngineBrave   Engine = "brave"
	EngineSearXNG Engine = "searxng"
	EngineZhipu   Engine = "zhipu"
	EngineBocha   Engine = "bocha"
)

// SearchResult 搜索结果。
type SearchResult struct {
	Engine  Engine `json:"engine"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// Report 调研报告。
type Report struct {
	ID        string          `json:"id"`
	Topic     string          `json:"topic"`
	Sections  []ReportSection `json:"sections"`
	Sources   []SearchResult  `json:"sources"`
	CreatedAt time.Time       `json:"created_at"`
}

// ReportSection 报告章节。
type ReportSection struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// Plan 调研计划。
type Plan struct {
	Steps []string `json:"steps"`
}

// Service 调研服务。
type Service struct {
	vfs *vfs.FS
	llm *llm.Registry
	bus *eventbus.Bus
	mu  sync.Mutex
}

// New 创建 Service。
func New(fs *vfs.FS, l *llm.Registry, bus *eventbus.Bus) *Service {
	return &Service{vfs: fs, llm: l, bus: bus}
}

// ConfirmAndPlan 交互式确认后返回计划。
func (s *Service) ConfirmAndPlan(ctx context.Context, topic, depth, format string) (*Plan, error) {
	p, ok := s.llm.Get("openai")
	if !ok {
		return nil, fmt.Errorf("research: no provider")
	}
	resp, err := p.Chat(ctx, llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "Decompose the user's research topic into a step-by-step plan."},
			{Role: llm.RoleUser, Content: fmt.Sprintf("Topic: %s\nDepth: %s\nFormat: %s\n\nReturn JSON: {\"steps\":[\"step 1\",\"step 2\",...]}", topic, depth, format)},
		},
	})
	if err != nil {
		return nil, err
	}
	start := strings.Index(resp.Content, "{")
	end := strings.LastIndex(resp.Content, "}")
	var plan Plan
	if start >= 0 && end > start {
		_ = json.Unmarshal([]byte(resp.Content[start:end+1]), &plan)
	}
	if len(plan.Steps) == 0 {
		plan.Steps = []string{"Define scope", "Search literature", "Analyze", "Write report"}
	}
	return &plan, nil
}

// Search 调用一个或多个引擎（占位：实际项目可对接各家 API）。
func (s *Service) Search(ctx context.Context, query string, engines []Engine) []SearchResult {
	// 真实项目应通过 HTTP 调各家；本骨架以 LLM 总结 + 占位条目返回。
	p, _ := s.llm.Get("openai")
	if p == nil {
		return nil
	}
	resp, err := p.Chat(ctx, llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "List 5 plausible web sources for the query as JSON: [{\"engine\":\"google\",\"title\":\"...\",\"url\":\"https://...\",\"snippet\":\"...\"}]"},
			{Role: llm.RoleUser, Content: "Query: " + query},
		},
	})
	if err != nil {
		return nil
	}
	start := strings.Index(resp.Content, "[")
	end := strings.LastIndex(resp.Content, "]")
	if start < 0 || end < 0 {
		return nil
	}
	var out []SearchResult
	_ = json.Unmarshal([]byte(resp.Content[start:end+1]), &out)
	return out
}

// Run 执行完整调研。
func (s *Service) Run(ctx context.Context, topic string, engines []Engine, onProgress func(stage string)) (*Report, error) {
	if onProgress != nil {
		onProgress("planning")
	}
	plan, err := s.ConfirmAndPlan(ctx, topic, "deep", "markdown")
	if err != nil {
		return nil, err
	}
	report := &Report{ID: uuid.NewString(), Topic: topic, CreatedAt: time.Now()}
	if onProgress != nil {
		onProgress("searching")
	}
	for _, step := range plan.Steps {
		results := s.Search(ctx, step, engines)
		report.Sources = append(report.Sources, results...)
		if onProgress != nil {
			onProgress("analyzing: " + step)
		}
		section, err := s.analyzeStep(ctx, step, results)
		if err != nil {
			continue
		}
		report.Sections = append(report.Sections, section)
	}
	if onProgress != nil {
		onProgress("writing")
	}
	finalSection, err := s.synthesize(ctx, topic, report.Sections)
	if err == nil {
		report.Sections = append(report.Sections, ReportSection{Title: "Conclusion", Content: finalSection})
	}
	return report, nil
}

// Save 写入 VFS。
func (s *Service) Save(r *Report) (string, error) {
	uri := fmt.Sprintf("vfs://note/%s", r.ID)
	body := renderMarkdown(r)
	_, err := s.vfs.Put(uri, []byte(body), map[string]string{
		"title": r.Topic, "tags": "research,report",
	})
	return uri, err
}

func (s *Service) analyzeStep(ctx context.Context, step string, results []SearchResult) (ReportSection, error) {
	p, _ := s.llm.Get("openai")
	if p == nil {
		return ReportSection{}, fmt.Errorf("no provider")
	}
	var sb strings.Builder
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("- [%s] %s — %s\n", r.Engine, r.Title, r.URL))
	}
	resp, err := p.Chat(ctx, llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a research analyst. Write a paragraph for the given step, grounded in the sources."},
			{Role: llm.RoleUser, Content: "Step: " + step + "\n\nSources:\n" + sb.String()},
		},
	})
	if err != nil {
		return ReportSection{}, err
	}
	return ReportSection{Title: step, Content: resp.Content}, nil
}

func (s *Service) synthesize(ctx context.Context, topic string, sections []ReportSection) (string, error) {
	p, _ := s.llm.Get("openai")
	if p == nil {
		return "", fmt.Errorf("no provider")
	}
	var sb strings.Builder
	for _, s := range sections {
		sb.WriteString("## " + s.Title + "\n" + s.Content + "\n\n")
	}
	resp, err := p.Chat(ctx, llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "Synthesize a final conclusion for the research report."},
			{Role: llm.RoleUser, Content: "Topic: " + topic + "\n\nSections:\n" + sb.String()},
		},
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func renderMarkdown(r *Report) string {
	var sb strings.Builder
	sb.WriteString("# " + r.Topic + "\n\n")
	for _, s := range r.Sections {
		sb.WriteString("## " + s.Title + "\n" + s.Content + "\n\n")
	}
	sb.WriteString("## Sources\n")
	for _, src := range r.Sources {
		sb.WriteString(fmt.Sprintf("- [%s] [%s](%s)\n", src.Engine, src.Title, src.URL))
	}
	return sb.String()
}

// Engines 列出所有引擎。
func Engines() []Engine {
	return []Engine{EngineGoogle, EngineSerpAPI, EngineTavily, EngineBrave, EngineSearXNG, EngineZhipu, EngineBocha}
}
