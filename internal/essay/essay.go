// Package essay 作文批改：多场景、维度评分、润色、批注。
package essay

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Scenario 场景。
type Scenario string

const (
	ScenarioGaokao   Scenario = "gaokao"
	ScenarioIELTS    Scenario = "ielts"
	ScenarioTOEFL    Scenario = "toefl"
	ScenarioCET      Scenario = "cet"
	ScenarioPostgrad Scenario = "postgrad"
)

// Dimension 评分维度。
type Dimension struct {
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Weight float64 `json:"weight"`
	Note   string  `json:"note"`
}

// Result 批改结果。
type Result struct {
	ID          string      `json:"id"`
	Scenario    Scenario    `json:"scenario"`
	Original    string      `json:"original"`
	Polished    string      `json:"polished"`
	Dimensions  []Dimension `json:"dimensions"`
	Total       float64     `json:"total"`
	Suggestions []string    `json:"suggestions"`
	Highlights  []string    `json:"highlights"`
}

// Service 作文服务。
type Service struct {
	vfs *vfs.FS
	llm *llm.Registry
}

// New 创建 Service。
func New(fs *vfs.FS, l *llm.Registry) *Service { return &Service{vfs: fs, llm: l} }

// Grade 评分 + 润色。
func (s *Service) Grade(ctx context.Context, text string, scenario Scenario, dims []string) (*Result, error) {
	p, ok := s.llm.Get("openai")
	if !ok {
		return nil, fmt.Errorf("essay: no provider")
	}
	if len(dims) == 0 {
		dims = []string{"vocabulary", "grammar", "coherence", "structure", "content"}
	}
	sys := fmt.Sprintf("You are a strict %s examiner. Grade the essay along dimensions: %s. Return JSON only.", scenario, strings.Join(dims, ","))
	resp, err := p.Chat(ctx, llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: sys + ` Format: {"polished":"...","dimensions":[{"name":"...","score":0-100,"weight":1,"note":"..."}],"suggestions":["..."],"highlights":["..."]}`},
			{Role: llm.RoleUser, Content: "Essay:\n" + text},
		},
	})
	if err != nil {
		return nil, err
	}
	start := strings.Index(resp.Content, "{")
	end := strings.LastIndex(resp.Content, "}")
	if start < 0 || end < 0 {
		return nil, fmt.Errorf("essay: no json")
	}
	var raw struct {
		Polished    string      `json:"polished"`
		Dimensions  []Dimension `json:"dimensions"`
		Suggestions []string    `json:"suggestions"`
		Highlights  []string    `json:"highlights"`
	}
	if err := json.Unmarshal([]byte(resp.Content[start:end+1]), &raw); err != nil {
		return nil, err
	}
	r := &Result{
		ID: uuid.NewString(), Scenario: scenario, Original: text, Polished: raw.Polished,
		Dimensions: raw.Dimensions, Suggestions: raw.Suggestions, Highlights: raw.Highlights,
	}
	for _, d := range raw.Dimensions {
		r.Total += d.Score * d.Weight
	}
	return r, nil
}

// Save 保存到 VFS。
func (s *Service) Save(r *Result) (string, error) {
	uri := fmt.Sprintf("vfs://translation/%s", r.ID)
	data, _ := json.Marshal(r)
	_, err := s.vfs.Put(uri, data, map[string]string{
		"title":    string(r.Scenario) + "-essay",
		"scenario": string(r.Scenario),
	})
	return uri, err
}
