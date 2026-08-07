package vault

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ScannedEntry 扫描得到的一条资源记录。
type ScannedEntry struct {
	URI       string
	Type      Type
	ID        string
	Title     string
	Tags      []string
	Metadata  map[string]string
	FilePath  string // 文件绝对路径
	Size      int64
	CreatedAt int64
	UpdatedAt int64
}

// Scan 递归扫描 vault，重建资源索引。
// 返回按 URI 去重后的条目；同一个文件解析失败会记入 errors。
func (v *Vault) Scan() ([]ScannedEntry, []error) {
	var entries []ScannedEntry
	var errs []error
	seen := map[string]bool{}

	visited := map[string]bool{}
	for _, dir := range scanDirs(v.Root) {
		if visited[dir] {
			continue
		}
		visited[dir] = true
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // 跳过不可读
			}
			if info.IsDir() {
				name := info.Name()
				if name == ".deepstudent" || strings.HasPrefix(name, ".") {
					return filepath.SkipDir
				}
				return nil
			}
			rel, err := filepath.Rel(v.Root, path)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			// 跳过应用内部与隐藏文件
			if strings.HasPrefix(rel, ".deepstudent/") || strings.HasPrefix(filepath.Base(path), ".") {
				return nil
			}
			entry, ok := v.entryFromFile(path, rel)
			if !ok {
				return nil
			}
			if seen[entry.URI] {
				return nil // 同 URI 已被更早的目录命中
			}
			seen[entry.URI] = true
			entries = append(entries, entry)
			return nil
		})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].URI < entries[j].URI })
	return entries, errs
}

// scanDirs 扫描起始目录：根目录（Walk 会覆盖全部类型子目录）。
func scanDirs(root string) []string { return []string{root} }

// entryFromFile 把单个文件解析为 ScannedEntry。
// 返回 ok=false 表示该文件不是 DeepStudent 资源。
func (v *Vault) entryFromFile(path, rel string) (ScannedEntry, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return ScannedEntry{}, false
	}
	ext := strings.ToLower(filepath.Ext(path))
	markdown := IsMarkdownExt(ext)

	// Markdown 文件：读全文解析 frontmatter。
	var content []byte
	var body string
	var hasFM bool
	var fm FrontMatter
	var typ Type
	if markdown {
		content, err = os.ReadFile(path)
		if err != nil {
			return ScannedEntry{}, false
		}
		fm, body, hasFM = ReadFrontMatter(content)
		typ = Type(fm.DSType)
	}

	// 非 Markdown 文件：尝试读 sidecar 元数据（.deepstudent/meta/）确定 ds 信息；
	// 没有 sidecar 的纯附件文件不纳入（用户可直接在 Obsidian 中查看原始文件）。
	if !markdown {
		if fm, ok := v.ReadMeta(rel); ok && fm.DSID != "" {
			typ = Type(fm.DSType)
			if typ == "" {
				typ = typeFromRel(rel)
			}
			if typ == "" {
				return ScannedEntry{}, false
			}
			title := fm.Title
			if title == "" {
				title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			}
			created := fm.Created
			updated := fm.Updated
			if created.IsZero() {
				created = info.ModTime()
			}
			if updated.IsZero() {
				updated = info.ModTime()
			}
			meta := map[string]string{"title": title, "ext": strings.TrimPrefix(ext, ".")}
			for k, vv := range fm.Extra {
				meta[k] = vv
			}
			return ScannedEntry{
				URI:       "vfs://" + string(typ) + "/" + fm.DSID,
				Type:      typ,
				ID:        fm.DSID,
				Title:     title,
				Tags:      fm.Tags,
				Metadata:  meta,
				FilePath:  path,
				Size:      info.Size(),
				CreatedAt: created.Unix(),
				UpdatedAt: updated.Unix(),
			}, true
		}
		return ScannedEntry{}, false
	}

	if hasFM && fm.DSID != "" {
		// 带 ds 元数据的文件：以 frontmatter 为准
		if typ == "" {
			// 从相对路径猜类型子目录
			typ = typeFromRel(rel)
		}
		if typ == "" {
			return ScannedEntry{}, false
		}
		title := fm.Title
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		meta := map[string]string{"title": title, "ext": strings.TrimPrefix(ext, ".")}
		for k, v := range fm.Extra {
			meta[k] = v
		}
		created := fm.Created
		updated := fm.Updated
		if created.IsZero() {
			created = info.ModTime()
		}
		if updated.IsZero() {
			updated = info.ModTime()
		}
		return ScannedEntry{
			URI:       "vfs://" + string(typ) + "/" + fm.DSID,
			Type:      typ,
			ID:        fm.DSID,
			Title:     title,
			Tags:      fm.Tags,
			Metadata:  meta,
			FilePath:  path,
			Size:      info.Size(),
			CreatedAt: created.Unix(),
			UpdatedAt: updated.Unix(),
		}, true
	}

	// 无 frontmatter：仅在类型子目录下、正文非空的 .md 当作资源（Obsidian 普通笔记）
	if markdown && typ == "" {
		typ = typeFromRel(rel)
		if typ == "" {
			return ScannedEntry{}, false
		}
		id := stableID(path, body)
		title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		return ScannedEntry{
			URI:       "vfs://" + string(typ) + "/" + id,
			Type:      typ,
			ID:        id,
			Title:     title,
			Metadata:  map[string]string{"title": title, "ext": "md"},
			FilePath:  path,
			Size:      info.Size(),
			CreatedAt: info.ModTime().Unix(),
			UpdatedAt: info.ModTime().Unix(),
		}, true
	}
	return ScannedEntry{}, false
}

// typeFromRel 从相对路径首段推断类型。
func typeFromRel(rel string) Type {
	first := rel
	if i := strings.Index(rel, "/"); i >= 0 {
		first = rel[:i]
	}
	switch first {
	case "notes":
		return TypeNote
	case "resources":
		return TypeTextbook
	case "mindmap":
		return TypeMindmap
	case "qbank":
		return TypeQBank
	case "translation":
		return TypeTranslation
	case "flashcard":
		return TypeFlashcard
	case "paper":
		return TypePaper
	case "chat":
		return TypeChat
	case "todo":
		return TypeTodo
	case "skills":
		return TypeSkill
	}
	return ""
}

// stableID 无 frontmatter 文件用「路径 + 首 40 字符内容」的哈希作稳定 ID，
// 文件内容变化后 ID 会变（与 Obsidian 无 ID 语义一致，仅作索引占位）。
func stableID(path, body string) string {
	h := fnv1a(path + "\x00" + firstN(body, 40))
	return fmtID(h)
}

func firstN(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// fmtID 把 uint64 哈希格式化为 16 位 hex。
func fmtID(h uint64) string {
	const hex = "0123456789abcdef"
	b := make([]byte, 16)
	for i := 15; i >= 0; i-- {
		b[i] = hex[h&0xf]
		h >>= 4
	}
	return string(b)
}

// fnv1a FNV-1a 64 位哈希（不引入额外依赖）。
func fnv1a(s string) uint64 {
	var h uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}
