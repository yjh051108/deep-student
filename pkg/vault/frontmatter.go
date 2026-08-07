package vault

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ReadFrontMatter 解析 Markdown 文件头部的 YAML frontmatter。
// 返回 frontmatter、正文（去掉 frontmatter 后的内容）以及是否包含 frontmatter。
func ReadFrontMatter(content []byte) (FrontMatter, string, bool) {
	text := string(content)
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return FrontMatter{}, text, false
	}
	// 找第二个 --- 行
	rest := text[len("---"):]
	if strings.HasPrefix(rest, "\r\n") {
		rest = rest[2:]
	} else if strings.HasPrefix(rest, "\n") {
		rest = rest[1:]
	} else {
		return FrontMatter{}, text, false
	}
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return FrontMatter{}, text, false
	}
	fmBlock := rest[:idx]
	body := rest[idx+len("\n---"):]
	body = strings.TrimPrefix(body, "\n")
	if strings.HasPrefix(body, "\r\n") {
		body = body[2:]
	}
	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(fmBlock), &fm); err != nil {
		// frontmatter 解析失败不阻塞：当作无 frontmatter 处理
		return FrontMatter{}, text, false
	}
	return fm, body, true
}

// WriteFrontMatter 生成带 frontmatter 的 Markdown 内容。
func WriteFrontMatter(fm FrontMatter, body string) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(fm); err != nil {
		return nil, fmt.Errorf("vault: encode frontmatter: %w", err)
	}
	_ = enc.Close()
	buf.WriteString("---\n")
	body = strings.TrimPrefix(body, "\n")
	buf.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		buf.WriteString("\n")
	}
	return buf.Bytes(), nil
}

// IsMarkdownExt 判断扩展名是否为 Markdown。
func IsMarkdownExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".md", ".markdown", ".mdown", "":
		return true
	}
	return false
}

// MarshalFrontMatter 把 frontmatter 序列化为 JSON（用于 sidecar 元数据）。
func MarshalFrontMatter(fm FrontMatter) ([]byte, error) {
	return json.Marshal(fm)
}

// UnmarshalFrontMatter 解析 sidecar JSON。
func UnmarshalFrontMatter(data []byte, fm *FrontMatter) error {
	return json.Unmarshal(data, fm)
}
