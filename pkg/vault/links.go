package vault

import (
	"regexp"
	"strings"
)

// wikilinkRe 匹配 Obsidian [[wikilink]]（支持别名 | 与块锚 #^）。
var wikilinkRe = regexp.MustCompile(`\[\[([^\[\]]+?)\]\]`)

// ExtractLinks 从 Markdown 正文提取所有 [[wikilink]] 目标。
// 返回的目标为链接本体（去掉别名与锚点）。
func ExtractLinks(body string) []string {
	matches := wikilinkRe.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]string, 0, len(matches))
	seen := map[string]bool{}
	for _, m := range matches {
		target := normalizeLinkTarget(m[1])
		if target == "" || seen[target] {
			continue
		}
		seen[target] = true
		out = append(out, target)
	}
	return out
}

// normalizeLinkTarget 解析 [[a|b]] / [[a#^block]] → 目标名 a。
func normalizeLinkTarget(raw string) string {
	s := strings.TrimSpace(raw)
	if i := strings.Index(s, "|"); i >= 0 {
		s = s[:i]
	}
	if i := strings.Index(s, "#"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// LinkEntry 一条双链关系。
type LinkEntry struct {
	SourceURI   string `json:"sourceUri"`   // 发出链接的资源
	SourceTitle string `json:"sourceTitle"` // 发出链接的资源标题
	Target      string `json:"target"`      // 链接目标（标题或 ds_id）
	TargetURI   string `json:"targetUri,omitempty"` // 命中资源 URI（若可解析）
	TargetTitle string `json:"targetTitle,omitempty"`
}

// ResolveLinks 扫描全部条目并解析双链，返回出链（source→target）与
// 每个资源可解析的入链。target 优先按标题匹配，其次按 ds_id 匹配。
func ResolveLinks(entries []ScannedEntry, contents map[string]string) []LinkEntry {
	// 建立标题/ds_id → URI 与 URI → 标题 的解析表
	byTitle := map[string]string{}
	byID := map[string]string{}
	titleOf := map[string]string{}
	for _, e := range entries {
		if e.Title != "" {
			byTitle[e.Title] = e.URI
			titleOf[e.URI] = e.Title
		}
		if e.ID != "" {
			byID[e.ID] = e.URI
		}
	}
	var out []LinkEntry
	for _, e := range entries {
		body := contents[e.URI]
		for _, target := range ExtractLinks(body) {
			le := LinkEntry{
				SourceURI:   e.URI,
				SourceTitle: e.Title,
				Target:      target,
			}
			if uri, ok := byTitle[target]; ok {
				le.TargetURI = uri
			} else if uri, ok := byID[target]; ok {
				le.TargetURI = uri
			}
			if le.TargetURI != "" {
				le.TargetTitle = titleOf[le.TargetURI]
			}
			out = append(out, le)
		}
	}
	return out
}
