// Package mindmap 思维导图：生成、编辑、大纲↔视图、节点背书。
package mindmap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Node 思维导图节点。
type Node struct {
	ID       string  `json:"id"`
	Topic    string  `json:"topic"`
	Children []*Node `json:"children,omitempty"`
	Masked   bool    `json:"masked,omitempty"`
	Notes    string  `json:"notes,omitempty"`
}

// Map 完整思维导图。
type Map struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Root  *Node  `json:"root"`
}

// Service 思维导图服务。
type Service struct {
	vfs *vfs.FS
	llm *llm.Registry
	mu  sync.Mutex
}

// New 创建 Service。
func New(fs *vfs.FS, l *llm.Registry) *Service { return &Service{vfs: fs, llm: l} }

// Generate 从一句话生成思维导图。
func (s *Service) Generate(ctx context.Context, topic string) (*Map, error) {
	p, ok := s.llm.Get("openai")
	if !ok {
		return nil, fmt.Errorf("mindmap: no provider")
	}
	resp, err := p.Chat(ctx, llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a knowledge mapper. Given a topic, output a hierarchical mind map (JSON) with root → branches → leaves. Be comprehensive and well-structured."},
			{Role: llm.RoleUser, Content: "Topic: " + topic + "\n\nReturn JSON only: {\"root\":{\"id\":\"root\",\"topic\":\"...\"," +
				"\"children\":[{\"id\":\"n1\",\"topic\":\"...\",\"children\":[{\"id\":\"n1a\",\"topic\":\"...\"}]}]}}"},
		},
	})
	if err != nil {
		return nil, err
	}
	root, err := parseRoot(resp.Content)
	if err != nil {
		return nil, err
	}
	m := &Map{ID: uuid.NewString(), Title: topic, Root: root}
	return m, nil
}

// Save 保存到 VFS。
func (s *Service) Save(m *Map) (string, error) {
	uri := fmt.Sprintf("vfs://mindmap/%s", m.ID)
	data, _ := json.Marshal(m)
	_, err := s.vfs.Put(uri, data, map[string]string{"title": m.Title})
	return uri, err
}

// Load 读取。
func (s *Service) Load(uri string) (*Map, error) {
	data, _, err := s.vfs.Get(uri)
	if err != nil {
		return nil, err
	}
	m := &Map{}
	if err := json.Unmarshal(data, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Edit 多轮编辑（基于自然语言指令）。
func (s *Service) Edit(ctx context.Context, m *Map, instruction string) (*Map, error) {
	p, ok := s.llm.Get("openai")
	if !ok {
		return nil, fmt.Errorf("mindmap: no provider")
	}
	data, _ := json.Marshal(m)
	resp, err := p.Chat(ctx, llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a mind map editor. Apply the user's instruction and return updated JSON mind map."},
			{Role: llm.RoleUser, Content: "Current map:\n" + string(data) + "\n\nInstruction: " + instruction},
		},
	})
	if err != nil {
		return nil, err
	}
	root, err := parseRoot(resp.Content)
	if err != nil {
		return nil, err
	}
	m.Root = root
	return m, nil
}

// ToOutline 转大纲。
func (m *Map) ToOutline() string {
	return renderOutline(m.Root, 0)
}

// FromOutline 从大纲还原（缩进表示层级）。
func FromOutline(title, text string) *Map {
	root := &Node{ID: uuid.NewString(), Topic: title}
	lines := strings.Split(text, "\n")
	stack := []*Node{root}
	// 跳过第一行（视为标题行）
	started := false
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		if !started {
			started = true
			continue
		}
		indent := 0
		for _, c := range ln {
			if c == ' ' || c == '\t' {
				indent++
			} else {
				break
			}
		}
		indent /= 2
		topic := strings.TrimSpace(ln)
		n := &Node{ID: uuid.NewString(), Topic: topic}
		// 将 stack 顶调整到当前节点的父级
		for len(stack) > indent {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			stack = append(stack, root)
		}
		parent := stack[len(stack)-1]
		parent.Children = append(parent.Children, n)
		stack = append(stack, n)
	}
	return &Map{ID: uuid.NewString(), Title: title, Root: root}
}

// Mask 背书：随机遮罩节点。
func (m *Map) Mask(rate float64) {
	maskRecursive(m.Root, rate)
}

func maskRecursive(n *Node, rate float64) {
	if n == nil {
		return
	}
	if uuid.NewString() != "" && float64(len(n.Topic)) > 0 {
		// 简单随机遮罩
		if hash(n.ID)%100 < int(rate*100) {
			n.Masked = true
		}
	}
	for _, c := range n.Children {
		maskRecursive(c, rate)
	}
}

func hash(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

func renderOutline(n *Node, depth int) string {
	if n == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(strings.Repeat("  ", depth))
	sb.WriteString(n.Topic)
	sb.WriteString("\n")
	for _, c := range n.Children {
		sb.WriteString(renderOutline(c, depth+1))
	}
	return sb.String()
}

func parseRoot(content string) (*Node, error) {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end < 0 {
		return nil, fmt.Errorf("mindmap: no json")
	}
	body := content[start : end+1]
	var wrapper struct {
		Root *Node `json:"root"`
	}
	if err := json.Unmarshal([]byte(body), &wrapper); err != nil {
		return nil, err
	}
	if wrapper.Root == nil {
		wrapper.Root = &Node{ID: "root", Topic: "Root"}
	}
	return wrapper.Root, nil
}
