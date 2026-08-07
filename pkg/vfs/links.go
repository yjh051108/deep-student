package vfs

import (
	"os"

	"github.com/helixnow/deep-student-go/pkg/vault"
)

// LinkEntry 一条双链关系（对齐 vault.LinkEntry 的 JSON 结构）。
type LinkEntry = vault.LinkEntry

// Links 返回指定资源的出链（[[wikilink]]）。
// 仅在 vault 模式下有意义；内存版返回空。
func (fs *FS) Links(uri string) []LinkEntry {
	if fs.vault == nil {
		return nil
	}
	e, ok := fs.entry(uri)
	if !ok {
		return nil
	}
	body, err := readFileBody(fs, e)
	if err != nil {
		return nil
	}
	all := fs.allLinks()
	out := make([]LinkEntry, 0, len(all))
	for _, le := range all {
		if le.SourceURI == uri {
			out = append(out, le)
		}
	}
	_ = body
	return out
}

// Backlinks 返回指向指定资源的入链。
func (fs *FS) Backlinks(uri string) []LinkEntry {
	if fs.vault == nil {
		return nil
	}
	e, ok := fs.entry(uri)
	if !ok {
		return nil
	}
	all := fs.allLinks()
	out := make([]LinkEntry, 0, len(all))
	for _, le := range all {
		if le.TargetURI == uri {
			out = append(out, le)
		}
	}
	_ = e
	return out
}

// Graph 返回全库双链图（所有出链）。
func (fs *FS) Graph() []LinkEntry {
	if fs.vault == nil {
		return nil
	}
	return fs.allLinks()
}

// allLinks 扫描全部资源正文并解析双链。
func (fs *FS) allLinks() []LinkEntry {
	fs.mu.RLock()
	entries := make([]Entry, 0, len(fs.entries))
	for _, e := range fs.entries {
		entries = append(entries, e)
	}
	fs.mu.RUnlock()

	scanned := make([]vault.ScannedEntry, 0, len(entries))
	contents := map[string]string{}
	for _, e := range entries {
		se := vault.ScannedEntry{
			URI:       e.URI,
			Type:      vault.Type(e.Type),
			ID:        e.ID,
			Title:     e.Title,
			Tags:      e.Tags,
			Metadata:  e.Metadata,
			FilePath:  e.FilePath,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
		}
		scanned = append(scanned, se)
		if e.FilePath != "" {
			if data, err := readFileBody(fs, e); err == nil {
				contents[e.URI] = data
			}
		}
	}
	return vault.ResolveLinks(scanned, contents)
}

// readFileBody 读取资源正文（Markdown 剥离 frontmatter）。
func readFileBody(fs *FS, e Entry) (string, error) {
	data, err := os.ReadFile(e.FilePath)
	if err != nil {
		return "", err
	}
	if vault.IsMarkdownExt(extOf(e.FilePath)) {
		_, body, _ := vault.ReadFrontMatter(data)
		return body, nil
	}
	return string(data), nil
}

func extOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '.' {
			return p[i:]
		}
		if p[i] == '/' || p[i] == '\\' {
			break
		}
	}
	return ""
}
