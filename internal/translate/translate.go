// Package translate 翻译：全文、段落、7 预设、术语表。
package translate

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Domain 领域预设。
type Domain string

const (
	DomainAcademic Domain = "academic"
	DomainTech     Domain = "tech"
	DomainLiterary Domain = "literary"
	DomainLegal    Domain = "legal"
	DomainMedical  Domain = "medical"
	DomainBusiness Domain = "business"
	DomainGeneral  Domain = "general"
)

// Request 翻译请求。
type Request struct {
	Text         string          `json:"text"`
	Source       string          `json:"source"`
	Target       string          `json:"target"`
	Domain       Domain          `json:"domain"`
	CustomPrompt string          `json:"custom_prompt,omitempty"`
	Glossary     []GlossaryEntry `json:"glossary,omitempty"`
}

// GlossaryEntry 术语表。
type GlossaryEntry struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// Result 翻译结果。
type Result struct {
	Text  string `json:"text"`
	Notes string `json:"notes,omitempty"`
}

// Service 翻译服务。
type Service struct {
	vfs *vfs.FS
	llm *llm.Registry
	mu  sync.Mutex
}

// New 创建 Service。
func New(fs *vfs.FS, l *llm.Registry) *Service { return &Service{vfs: fs, llm: l} }

// Translate 单段翻译。
func (s *Service) Translate(ctx context.Context, req Request) (*Result, error) {
	p, ok := s.llm.Get("openai")
	if !ok {
		return nil, fmt.Errorf("translate: no provider")
	}
	sysPrompt := domainPrompt(req.Domain)
	if req.CustomPrompt != "" {
		sysPrompt += "\n\n" + req.CustomPrompt
	}
	if len(req.Glossary) > 0 {
		var sb strings.Builder
		sb.WriteString("Glossary (must follow):\n")
		for _, g := range req.Glossary {
			sb.WriteString(fmt.Sprintf("- %s → %s\n", g.Source, g.Target))
		}
		sysPrompt += "\n\n" + sb.String()
	}
	resp, err := p.Chat(ctx, llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: sysPrompt},
			{Role: llm.RoleUser, Content: fmt.Sprintf("Translate from %s to %s. Output only the translation.\n\n%s", req.Source, req.Target, req.Text)},
		},
	})
	if err != nil {
		return nil, err
	}
	return &Result{Text: resp.Content}, nil
}

// TranslateDocument 翻译 VFS 文档并保存。
func (s *Service) TranslateDocument(ctx context.Context, uri, source, target string, domain Domain) (string, error) {
	data, _, err := s.vfs.Get(uri)
	if err != nil {
		return "", err
	}
	res, err := s.Translate(ctx, Request{Text: string(data), Source: source, Target: target, Domain: domain})
	if err != nil {
		return "", err
	}
	out := fmt.Sprintf("vfs://translation/%s", uri[len("vfs://"):])
	_, err = s.vfs.Put(out, []byte(res.Text), map[string]string{
		"source": uri, "lang": fmt.Sprintf("%s→%s", source, target),
	})
	return out, err
}

func domainPrompt(d Domain) string {
	switch d {
	case DomainAcademic:
		return "You are an academic translator. Preserve technical terms, citation style, and formal register."
	case DomainTech:
		return "You are a software/IT translator. Use standard industry terms; keep code blocks, identifiers, file paths unchanged."
	case DomainLiterary:
		return "You are a literary translator. Preserve imagery, rhythm, and stylistic devices; do not over-literal."
	case DomainLegal:
		return "You are a legal translator. Preserve clause structure, defined terms; output must be enforceable in target jurisdiction."
	case DomainMedical:
		return "You are a medical translator. Use ICD/WHO terminology; do not paraphrase clinical findings."
	case DomainBusiness:
		return "You are a business translator. Use formal business register; preserve numbers, dates, party names."
	default:
		return "You are a professional translator. Output a natural, accurate translation."
	}
}

// Domains 列出所有领域。
func Domains() []Domain {
	return []Domain{DomainAcademic, DomainTech, DomainLiterary, DomainLegal, DomainMedical, DomainBusiness, DomainGeneral}
}
