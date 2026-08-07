// Package reader PDF/DOCX 阅读器：解析、批注、分屏引用。
package reader

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Page 文档一页/一节。
type Page struct {
	Index   int    `json:"index"`
	Content string `json:"content"`
}

// Document 解析后的文档。
type Document struct {
	URI   string `json:"uri"`
	Title string `json:"title"`
	Pages []Page `json:"pages"`
}

// Service 阅读器服务。
type Service struct {
	vfs  *vfs.FS
	llm  *llm.Registry
	mu   sync.Mutex
	docs map[string]*Document
}

// New 创建 Service。
func New(fs *vfs.FS, l *llm.Registry) *Service {
	return &Service{vfs: fs, llm: l, docs: map[string]*Document{}}
}

// Open 打开一个 VFS 资源并解析（PDF/DOCX/TXT/MD）。
func (s *Service) Open(uri string) (*Document, error) {
	data, e, err := s.vfs.Get(uri)
	if err != nil {
		return nil, err
	}
	doc := &Document{URI: uri, Title: e.Title}
	lower := strings.ToLower(uri)
	switch {
	case strings.HasSuffix(lower, ".pdf"):
		pages, err := ParsePDF(data)
		if err != nil {
			return nil, err
		}
		doc.Pages = pages
	case strings.HasSuffix(lower, ".docx"):
		pages, err := ParseDOCX(data)
		if err != nil {
			return nil, err
		}
		doc.Pages = pages
	default:
		doc.Pages = []Page{{Index: 1, Content: string(data)}}
	}
	s.mu.Lock()
	s.docs[uri] = doc
	s.mu.Unlock()
	return doc, nil
}

// InjectToChat 把选中片段注入聊天上下文（返回拼装好的 prompt）。
func (s *Service) InjectToChat(doc *Document, pageStart, pageEnd int, selection string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[From %s pages %d-%d]\n", doc.Title, pageStart, pageEnd))
	if selection != "" {
		sb.WriteString("Selected:\n" + selection + "\n")
	}
	for _, p := range doc.Pages {
		if p.Index >= pageStart && p.Index <= pageEnd {
			sb.WriteString(fmt.Sprintf("\n--- Page %d ---\n%s\n", p.Index, p.Content))
		}
	}
	return sb.String()
}

// Summarize 一页/全文摘要。
func (s *Service) Summarize(ctx context.Context, doc *Document, page int) (string, error) {
	p, ok := s.llm.Get("openai")
	if !ok {
		return "", fmt.Errorf("reader: no provider")
	}
	var content string
	if page > 0 {
		for _, p2 := range doc.Pages {
			if p2.Index == page {
				content = p2.Content
				break
			}
		}
	} else {
		for _, p2 := range doc.Pages {
			content += p2.Content + "\n"
		}
	}
	resp, err := p.Chat(ctx, llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "Summarize the following document content. Include page references where possible."},
			{Role: llm.RoleUser, Content: content},
		},
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// ReadAll 把 Reader 当 io.Reader 读取文本（用于 gpt 上传）。
func ReadAll(r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Lines 工具。
func Lines(s string) []string {
	sc := bufio.NewScanner(strings.NewReader(s))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	out := []string{}
	for sc.Scan() {
		out = append(out, sc.Text())
	}
	return out
}
