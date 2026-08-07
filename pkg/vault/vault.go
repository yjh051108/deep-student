// Package vault 提供 Obsidian 式混合知识库：真实文件落盘 + frontmatter 元数据。
//
// 设计目标：
//   - 内容类数据（笔记/思维导图/题库/翻译/论文/卡片）以真实 .md / 原始格式文件
//     存放到用户可见的 vault 目录，可直接用 Obsidian 打开、编辑、同步。
//   - frontmatter 使用 Obsidian 原生字段（title/tags/created/updated）+ 自定义
//     ds_* 命名空间（ds_id/ds_type），ds_id 是 URI 与文件的稳定关联键。
//   - URI 格式保持 vfs://<type>/<id>；id 与文件内 ds_id 一致，文件路径由
//     title 派生（可读性好，外部重命名不破坏关联）。
//
// 注意：本包不依赖 pkg/vfs（避免循环导入）；资源类型使用本包的 Type。
package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Type 资源类型（与 pkg/vfs.ResourceType 一一对应，可字符串互转）。
type Type string

// 资源类型常量（与 pkg/vfs 保持一致）。
const (
	TypeNote        Type = "note"
	TypeTextbook    Type = "textbook"
	TypeQBank       Type = "qbank"
	TypeMindmap     Type = "mindmap"
	TypeTranslation Type = "translation"
	TypeFlashcard   Type = "flashcard"
	TypePaper       Type = "paper"
	TypeChat        Type = "chat"
	TypeTodo        Type = "todo"
	TypeSkill       Type = "skill"
)

// Vault 知识库根目录与类型→子目录映射。
type Vault struct {
	Root string // 用户可见 vault 根目录
}

// TypeDir 返回某资源类型对应的子目录（相对 Root）。
func TypeDir(t Type) string {
	switch t {
	case TypeNote:
		return "notes"
	case TypeTextbook:
		return "resources"
	case TypeMindmap:
		return "mindmap"
	case TypeQBank:
		return "qbank"
	case TypeTranslation:
		return "translation"
	case TypeFlashcard:
		return "flashcard"
	case TypePaper:
		return "paper"
	case TypeChat:
		return "chat"
	case TypeTodo:
		return "todo"
	case TypeSkill:
		return "skills"
	default:
		return "misc"
	}
}

// New 创建 Vault（确保根目录存在）。
func New(root string) (*Vault, error) {
	if root == "" {
		return nil, fmt.Errorf("vault: empty root")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("vault: mkdir %s: %w", root, err)
	}
	return &Vault{Root: root}, nil
}

// internalDir 应用内部目录（索引缓存等，对 Obsidian 不可见友好）。
func (v *Vault) internalDir() string { return filepath.Join(v.Root, ".deepstudent") }

// EnsureInternal 确保内部目录存在。
func (v *Vault) EnsureInternal() error {
	return os.MkdirAll(v.internalDir(), 0o755)
}

// metaDir 应用侧car元数据目录（对 Obsidian 不可见）。
func (v *Vault) metaDir() string { return filepath.Join(v.internalDir(), "meta") }

// metaPath 返回某相对路径对应的 sidecar 元数据文件路径。
func (v *Vault) metaPath(rel string) string {
	h := fnv1a(filepath.ToSlash(rel))
	return filepath.Join(v.metaDir(), fmt.Sprintf("%016x.json", h))
}

// WriteMeta 为非 Markdown 资源写 sidecar 元数据（frontmatter 无法内嵌时用）。
// rel 为文件相对 vault 根的路径（斜杠分隔）。
func (v *Vault) WriteMeta(rel string, fm FrontMatter) error {
	if err := os.MkdirAll(v.metaDir(), 0o755); err != nil {
		return err
	}
	data, err := MarshalFrontMatter(fm)
	if err != nil {
		return err
	}
	return os.WriteFile(v.metaPath(rel), data, 0o644)
}

// ReadMeta 读取 sidecar 元数据；不存在返回 ok=false。
func (v *Vault) ReadMeta(rel string) (FrontMatter, bool) {
	data, err := os.ReadFile(v.metaPath(rel))
	if err != nil {
		return FrontMatter{}, false
	}
	var fm FrontMatter
	if err := UnmarshalFrontMatter(data, &fm); err != nil {
		return FrontMatter{}, false
	}
	return fm, true
}

// FilePath 计算某资源类型的文件绝对路径。
func (v *Vault) FilePath(t Type, title, ext string) string {
	name := SanitizeFilename(title)
	if name == "" {
		name = "untitled"
	}
	if ext == "" {
		ext = ".md"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return filepath.Join(v.Root, TypeDir(t), name+ext)
}

// uniquePath 若文件已存在且不是同一资源，追加序号避免覆盖。
func (v *Vault) uniquePath(p string) string {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return p
	}
	dir := filepath.Dir(p)
	base := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	ext := filepath.Ext(p)
	for i := 2; ; i++ {
		cand := filepath.Join(dir, fmt.Sprintf("%s-%d%s", base, i, ext))
		if _, err := os.Stat(cand); os.IsNotExist(err) {
			return cand
		}
	}
}

// AllocatePath 为写入分配不冲突的文件路径。existingPath 非空时优先复用
// （同一 ds_id 重写不换名）。
func (v *Vault) AllocatePath(t Type, title, ext, existingPath string) string {
	if existingPath != "" && filepath.Dir(existingPath) != "." {
		return existingPath
	}
	return v.uniquePath(v.FilePath(t, title, ext))
}

// sanitizeRe 去掉 Windows/Linux 文件名非法字符。
var sanitizeRe = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

// SanitizeFilename 清理文件名为安全片段。
func SanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	s = sanitizeRe.ReplaceAllString(s, "-")
	// 连续横线折叠 + 去首尾分隔符
	s = strings.Join(strings.FieldsFunc(s, func(r rune) bool { return r == '-' }), "-")
	s = strings.Trim(s, "-. ")
	if len(s) > 120 {
		s = s[:120]
	}
	return s
}

// FrontMatter 资源 frontmatter 元数据（YAML 结构）。
type FrontMatter struct {
	Title   string    `yaml:"title,omitempty"`
	Tags    []string  `yaml:"tags,omitempty"`
	Created time.Time `yaml:"created,omitempty"`
	Updated time.Time `yaml:"updated,omitempty"`
	DSID    string    `yaml:"ds_id,omitempty"`
	DSType  string    `yaml:"ds_type,omitempty"`
	// Extra 额外元数据（对齐 vfs.Entry.Metadata，如文件扩展名、原始 MIME）。
	Extra map[string]string `yaml:"ds_extra,omitempty"`
}

// ParseTime 兼容 Obsidian 的多种时间格式。
func ParseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	for _, layout := range []string{
		time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("vault: bad time %q", s)
}
