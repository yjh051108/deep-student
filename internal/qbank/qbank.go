// Package qbank 题库：解析、练习、阅卷、历史、AI 解析、掌握度。
package qbank

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

// Question 单题。
type Question struct {
	ID        string   `json:"id"`
	Stem      string   `json:"stem"`
	Options   []string `json:"options,omitempty"`
	Answer    string   `json:"answer"`
	Type      string   `json:"type"` // single | multi | fill | essay
	Points    []string `json:"points,omitempty"`
	Knowledge []string `json:"knowledge,omitempty"`
}

// Set 题目集。
type Set struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Questions []Question `json:"questions"`
	CreatedAt time.Time  `json:"created_at"`
}

// Attempt 一次练习记录。
type Attempt struct {
	ID         string            `json:"id"`
	SetID      string            `json:"set_id"`
	Answers    map[string]string `json:"answers"`
	Score      int               `json:"score"`
	Total      int               `json:"total"`
	StartedAt  time.Time         `json:"started_at"`
	FinishedAt time.Time         `json:"finished_at"`
}

// Service 题库服务。
type Service struct {
	vfs      *vfs.FS
	store    *store.Store
	llm      *llm.Registry
	mu       sync.RWMutex
	sets     map[string]*Set
	attempts map[string]*Attempt
	mastery  map[string]int // knowledge -> score 0..100
}

// New 创建 Service。
func New(fs *vfs.FS, st *store.Store, l *llm.Registry) *Service {
	return &Service{
		vfs: fs, store: st, llm: l,
		sets: map[string]*Set{}, attempts: map[string]*Attempt{},
		mastery: map[string]int{},
	}
}

// Extract 从教材/试卷 URI 提取题目。
func (s *Service) Extract(ctx context.Context, sourceURI, title string) (*Set, error) {
	data, _, err := s.vfs.Get(sourceURI)
	if err != nil {
		return nil, err
	}
	p, ok := s.llm.Get("openai")
	if !ok {
		return nil, fmt.Errorf("qbank: no provider")
	}
	resp, err := p.Chat(ctx, llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "Extract all questions from the given text. Output JSON only."},
			{Role: llm.RoleUser, Content: "Text:\n" + truncate(string(data), 12000) + "\n\nReturn JSON: {\"questions\":[{\"id\":\"q1\",\"stem\":\"...\",\"options\":[\"A\",\"B\"],\"answer\":\"A\",\"type\":\"single\",\"knowledge\":[\"...\"]}]}"},
		},
	})
	if err != nil {
		return nil, err
	}
	start := strings.Index(resp.Content, "{")
	end := strings.LastIndex(resp.Content, "}")
	if start < 0 || end < 0 {
		return nil, fmt.Errorf("qbank: no json")
	}
	var raw struct {
		Questions []Question `json:"questions"`
	}
	if err := json.Unmarshal([]byte(resp.Content[start:end+1]), &raw); err != nil {
		return nil, err
	}
	for i := range raw.Questions {
		if raw.Questions[i].ID == "" {
			raw.Questions[i].ID = fmt.Sprintf("q%d", i+1)
		}
	}
	set := &Set{ID: uuid.NewString(), Title: title, Questions: raw.Questions, CreatedAt: time.Now()}
	s.mu.Lock()
	s.sets[set.ID] = set
	s.mu.Unlock()
	return set, nil
}

// Save 保存到 VFS。
func (s *Service) Save(set *Set) (string, error) {
	uri := fmt.Sprintf("vfs://qbank/%s", set.ID)
	data, _ := json.Marshal(set)
	_, err := s.vfs.Put(uri, data, map[string]string{"title": set.Title})
	return uri, err
}

// StartAttempt 开始一次练习。
func (s *Service) StartAttempt(setID string) (*Attempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	set, ok := s.sets[setID]
	if !ok {
		return nil, fmt.Errorf("qbank: set not found")
	}
	a := &Attempt{
		ID:        uuid.NewString(),
		SetID:     setID,
		Answers:   map[string]string{},
		Total:     len(set.Questions),
		StartedAt: time.Now(),
	}
	s.attempts[a.ID] = a
	return a, nil
}

// Answer 答题。
func (s *Service) Answer(attemptID, qID, ans string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.attempts[attemptID]
	if !ok {
		return fmt.Errorf("qbank: attempt not found")
	}
	a.Answers[qID] = ans
	return nil
}

// Submit 提交并自动阅卷。
func (s *Service) Submit(attemptID string) (*Attempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.attempts[attemptID]
	if !ok {
		return nil, fmt.Errorf("qbank: attempt not found")
	}
	set, ok := s.sets[a.SetID]
	if !ok {
		return nil, fmt.Errorf("qbank: set not found")
	}
	// 复制 questions 快照，避免后续若 set 被并发修改时触发 race
	questions := append([]Question(nil), set.Questions...)
	score := 0
	for _, q := range questions {
		if a.Answers[q.ID] == q.Answer {
			score++
			for _, k := range q.Knowledge {
				s.mastery[k] = min(100, s.mastery[k]+5)
			}
		} else {
			for _, k := range q.Knowledge {
				s.mastery[k] = max(0, s.mastery[k]-3)
			}
		}
	}
	a.Score = score
	a.FinishedAt = time.Now()
	return a, nil
}

// Analyze 深度解析（基于 AI）。
func (s *Service) Analyze(ctx context.Context, setID, qID string) (string, error) {
	s.mu.RLock()
	set, ok := s.sets[setID]
	if !ok {
		s.mu.RUnlock()
		return "", fmt.Errorf("qbank: set not found")
	}
	// 在持锁期间完成 questions 复制 + 查找目标 question
	questions := append([]Question(nil), set.Questions...)
	s.mu.RUnlock()
	var q *Question
	for i := range questions {
		if questions[i].ID == qID {
			q = &questions[i]
			break
		}
	}
	if q == nil {
		return "", fmt.Errorf("qbank: question not found")
	}
	p, _ := s.llm.Get("openai")
	if p == nil {
		return "", fmt.Errorf("qbank: no provider")
	}
	resp, err := p.Chat(ctx, llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a patient tutor. Explain the problem, the knowledge point, and the solution path."},
			{Role: llm.RoleUser, Content: fmt.Sprintf("Problem: %s\nOptions: %v\nAnswer: %s", q.Stem, q.Options, q.Answer)},
		},
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// Mastery 知识点掌握度。
func (s *Service) Mastery() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]int, len(s.mastery))
	for k, v := range s.mastery {
		out[k] = v
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
