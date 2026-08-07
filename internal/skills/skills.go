// Package skills Skill 与 MCP：12 个内置 Skill、SKILL.md 三级加载、MCP 注册、Tool 注册。
package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/helixnow/deep-student-go/pkg/config"
	"github.com/helixnow/deep-student-go/pkg/eventbus"
	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/mcp"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Tier 加载层级。
type Tier string

const (
	TierBuiltin Tier = "builtin"
	TierGlobal  Tier = "global"
	TierProject Tier = "project"
)

// Skill 技能定义。
type Skill struct {
	Name        string     `json:"name"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Tier        Tier       `json:"tier"`
	Tools       []llm.Tool `json:"tools"`
	Prompt      string     `json:"prompt"`
	Source      string     `json:"source"`
}

// Service Skill 服务。
type Service struct {
	vfs    *vfs.FS
	store  *store.Store
	llm    *llm.Registry
	bus    *eventbus.Bus
	mu     sync.Mutex
	tools  map[string]ToolBinding // name -> binding
	mcps   map[string]*mcp.Client
	skills map[string]*Skill
}

// ToolBinding 工具绑定。
type ToolBinding struct {
	Name    string
	Desc    string
	Schema  json.RawMessage
	Handler func(ctx context.Context, args json.RawMessage) (any, error)
}

// New 创建 Service。
func New(fs *vfs.FS, st *store.Store, l *llm.Registry, bus *eventbus.Bus) *Service {
	s := &Service{
		vfs: fs, store: st, llm: l, bus: bus,
		tools:  map[string]ToolBinding{},
		mcps:   map[string]*mcp.Client{},
		skills: map[string]*Skill{},
	}
	s.registerBuiltinSkills()
	return s
}

// RegisterTool 注册一个工具绑定。
func (s *Service) RegisterTool(t ToolBinding) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[t.Name] = t
}

// Tool 调用工具。
func (s *Service) Tool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	s.mu.Lock()
	t, ok := s.tools[name]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("skills: tool not found: %s", name)
	}
	return t.Handler(ctx, args)
}

// Tools 列出工具。
func (s *Service) Tools() []llm.Tool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]llm.Tool, 0, len(s.tools))
	for _, t := range s.tools {
		out = append(out, llm.Tool{Name: t.Name, Description: t.Desc, Parameters: t.Schema})
	}
	return out
}

// SpawnMCP 启动一个 MCP 客户端并注册其工具。
func (s *Service) SpawnMCP(ctx context.Context, name, cmd string, args, env []string) error {
	c := mcp.NewStdio(name, cmd, args, env)
	if err := c.Start(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	s.mcps[name] = c
	s.mu.Unlock()
	for _, t := range c.Tools() {
		toolName := name + "." + t.Name
		client := c
		s.RegisterTool(ToolBinding{
			Name:   toolName,
			Desc:   t.Description,
			Schema: t.InputSchema,
			Handler: func(ctx context.Context, args json.RawMessage) (any, error) {
				return client.CallTool(ctx, t.Name, args)
			},
		})
	}
	return nil
}

// EnableServer 是 SpawnMCP 的设置入口封装：从 config 里的 MCPServerConfig
// 取出 command / args / env / url 字段，对 stdio 走 SpawnMCP，对 http+sse
// 直接 NewSSE + Start；如果已存在同名 server 则先停掉旧的。
func (s *Service) EnableServer(ctx context.Context, name string, cfg config.MCPServerConfig) error {
	s.mu.Lock()
	if old, ok := s.mcps[name]; ok {
		s.mu.Unlock()
		_ = old.Close()
		// 清掉旧注册的工具
		s.mu.Lock()
		delete(s.mcps, name)
		s.mu.Unlock()
	} else {
		s.mu.Unlock()
	}
	if cfg.URL != "" {
		// http+sse 传输
		c := mcp.NewSSE(name, cfg.URL, nil)
		if err := c.Start(ctx); err != nil {
			return err
		}
		s.mu.Lock()
		s.mcps[name] = c
		s.mu.Unlock()
		for _, t := range c.Tools() {
			toolName := name + "." + t.Name
			client := c
			s.RegisterTool(ToolBinding{
				Name:   toolName,
				Desc:   t.Description,
				Schema: t.InputSchema,
				Handler: func(ctx context.Context, args json.RawMessage) (any, error) {
					return client.CallTool(ctx, t.Name, args)
				},
			})
		}
		return nil
	}
	return s.SpawnMCP(ctx, name, cfg.Command, cfg.Args, envToSlice(cfg.Env))
}

// DisableServer 停止一个 MCP server 并注销其工具。
func (s *Service) DisableServer(name string) {
	s.mu.Lock()
	c, ok := s.mcps[name]
	if ok {
		delete(s.mcps, name)
	}
	s.mu.Unlock()
	if c != nil {
		_ = c.Close()
	}
	// 清掉由此 server 注册的工具（命名空间 = "name."）
	s.mu.Lock()
	defer s.mu.Unlock()
	prefix := name + "."
	for tn, tb := range s.tools {
		if strings.HasPrefix(tn, prefix) {
			delete(s.tools, tn)
			_ = tb
		}
	}
}

// ListMCPServers 列出已启用的 server 名称。
func (s *Service) ListMCPServers() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.mcps))
	for n := range s.mcps {
		out = append(out, n)
	}
	return out
}

func envToSlice(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}

// LoadSkillMD 从 SKILL.md 加载 Skill。
func (s *Service) LoadSkillMD(tier Tier, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	// 解析 frontmatter
	name := ""
	if strings.HasPrefix(content, "---") {
		end := strings.Index(content[3:], "---")
		if end > 0 {
			meta := content[3 : 3+end]
			for _, line := range strings.Split(meta, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "name:") {
					name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
				}
			}
			content = content[3+end+3:]
		}
	}
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	sk := &Skill{
		Name:        name,
		Description: firstLine(content),
		Tier:        tier,
		Prompt:      content,
		Source:      path,
	}
	s.mu.Lock()
	s.skills[sk.Name] = sk
	s.mu.Unlock()
	return nil
}

// LoadBuiltinSkillsDir 加载内置 Skill 目录。
func (s *Service) LoadBuiltinSkillsDir(dir string) error {
	if _, err := os.Stat(dir); err != nil {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			p := filepath.Join(dir, e.Name(), "SKILL.md")
			if _, err := os.Stat(p); err == nil {
				_ = s.LoadSkillMD(TierBuiltin, p)
			}
		} else if strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			_ = s.LoadSkillMD(TierBuiltin, filepath.Join(dir, e.Name()))
		}
	}
	return nil
}

// Skill 读取。
func (s *Service) Skill(name string) *Skill {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.skills[name]
}

// Skills 列出。
func (s *Service) Skills() []*Skill {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Skill, 0, len(s.skills))
	for _, sk := range s.skills {
		out = append(out, sk)
	}
	return out
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "\n"); i > 0 {
		return s[:i]
	}
	return s
}

func (s *Service) registerBuiltinSkills() {
	// 12 内置 Skill 的轻量定义
	builtin := []Skill{
		{Name: "cards", Title: "Anki Cards", Description: "Generate Anki flashcards from documents.", Tier: TierBuiltin},
		{Name: "research", Title: "Deep Research", Description: "Multi-step research with web search and report.", Tier: TierBuiltin},
		{Name: "paper", Title: "Paper Search", Description: "Search and download academic papers.", Tier: TierBuiltin},
		{Name: "mindmap", Title: "Mind Map", Description: "Generate knowledge mind map from topic.", Tier: TierBuiltin},
		{Name: "qbank", Title: "Question Bank", Description: "Generate question sets from textbooks.", Tier: TierBuiltin},
		{Name: "memory", Title: "Smart Memory", Description: "Extract and apply long-term memory.", Tier: TierBuiltin},
		{Name: "tutor", Title: "AI Tutor", Description: "Tutor mode with step-by-step guidance.", Tier: TierBuiltin},
		{Name: "lit-review", Title: "Literature Review", Description: "Compose a literature review.", Tier: TierBuiltin},
		{Name: "exam-analysis", Title: "Exam Analysis", Description: "Analyze exam papers and knowledge points.", Tier: TierBuiltin},
		{Name: "session-mgr", Title: "Session Manager", Description: "Group and tag chat sessions.", Tier: TierBuiltin},
		{Name: "office", Title: "Office Suite", Description: "Generate slides, docs, sheets.", Tier: TierBuiltin},
		{Name: "todo", Title: "Todo", Description: "Capture and track todos.", Tier: TierBuiltin},
	}
	for i := range builtin {
		s.skills[builtin[i].Name] = &builtin[i]
	}
}
