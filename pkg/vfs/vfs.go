// Package vfs 提供统一虚拟文件系统抽象，所有资源通过 vfs:// URI 寻址。
//
// 自 v2（Obsidian 式混合 VFS）起，FS 底层为真实文件系统（pkg/vault）：
//   - 内容类资源以真实 .md（YAML frontmatter）或原始格式文件落盘到用户
//     可见的 vault 目录，可直接用 Obsidian 打开编辑；
//   - 进程内 entries 索引由扫描 vault 重建（文件为准，重启不丢）；
//   - blob 存储仅保留给存量迁移与兼容旧调用方。
//
// 对外接口（Put/Get/Stat/List/Search/Delete/LockForRead/Snapshot）保持不变，
// 使用方无需改动。
package vfs

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vault"
)

// ResourceType 资源类型枚举。
type ResourceType string

// String 实现 fmt.Stringer。
func (t ResourceType) String() string { return string(t) }

const (
	TypeNote        ResourceType = "note"
	TypeTextbook    ResourceType = "textbook"
	TypeQBank       ResourceType = "qbank"
	TypeMindmap     ResourceType = "mindmap"
	TypeTranslation ResourceType = "translation"
	TypeFlashcard   ResourceType = "flashcard"
	TypePaper       ResourceType = "paper"
	TypeChat        ResourceType = "chat"
	TypeTodo        ResourceType = "todo"
	TypeSkill       ResourceType = "skill"
)

// URI 解析后的 vfs:// 资源定位符。
type URI struct {
	Type  ResourceType
	ID    string
	Raw   string
	Path  string
	Query url.Values
}

// Parse 解析 vfs:// 形式的 URI。
func Parse(raw string) (*URI, error) {
	if !strings.HasPrefix(raw, "vfs://") {
		return nil, fmt.Errorf("vfs: not a vfs URI: %s", raw)
	}
	// BUG-002: '#' 在 path 里需要正确转义；手动分离 fragment 后再交给 url.Parse
	rest := strings.TrimPrefix(raw, "vfs://")
	var frag string
	if i := strings.Index(rest, "#"); i >= 0 {
		frag = rest[i+1:]
		rest = rest[:i]
	}
	u, err := url.Parse("vfs://" + rest)
	if err != nil {
		return nil, err
	}
	t := ResourceType(strings.TrimPrefix(u.Host, ""))
	if t == "" {
		// 兼容 vfs://note/abc 形式
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) < 2 {
			return nil, fmt.Errorf("vfs: malformed uri: %s", raw)
		}
		t = ResourceType(parts[0])
		u, _ = url.Parse(raw)
	}
	id := strings.TrimPrefix(u.Path, "/")
	if id == "" {
		return nil, fmt.Errorf("vfs: empty id: %s", raw)
	}
	if frag != "" {
		// 把 fragment 还原到 ID 末尾，确保 Put/Get key 一致
		id = id + "#" + frag
	}
	q := u.Query()
	if frag != "" {
		q.Set("#", frag)
	}
	return &URI{Type: t, ID: id, Raw: raw, Path: path.Clean(id), Query: q}, nil
}

// String 返回原始形式。
func (u *URI) String() string { return u.Raw }

// Entry 资源元数据（不含 payload）。
type Entry struct {
	URI       string            `json:"uri"`
	Type      ResourceType      `json:"type"`
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Tags      []string          `json:"tags"`
	Metadata  map[string]string `json:"metadata"`
	BlobRef   string            `json:"blob_ref"` // sha256（兼容字段，vault 模式下可为空）
	Size      int64             `json:"size"`
	CreatedAt int64             `json:"created_at"`
	UpdatedAt int64             `json:"updated_at"`
	// FilePath 资源在 vault 中的绝对路径（Obsidian 式）。
	FilePath string `json:"file_path,omitempty"`
}

// FS 虚拟文件系统（vault 后端）。
type FS struct {
	mu      sync.RWMutex
	blob    *blob.Store
	vault   *vault.Vault
	entries map[string]Entry // index by vfs://...
	allowed map[ResourceType]bool
}

// NewFS 创建内存/blob 版 VFS（兼容旧调用方与测试；不落盘）。
func NewFS(blobStore *blob.Store) *FS {
	return &FS{blob: blobStore, entries: map[string]Entry{}, allowed: allowedTypes()}
}

// NewVaultFS 创建 Obsidian 式混合 VFS：内容落盘到 vault 目录。
// 立即扫描 vault 重建索引。
func NewVaultFS(root string, blobStore *blob.Store) (*FS, error) {
	vt, err := vault.New(root)
	if err != nil {
		return nil, err
	}
	if err := vt.EnsureInternal(); err != nil {
		return nil, err
	}
	fs := &FS{blob: blobStore, vault: vt, entries: map[string]Entry{}, allowed: allowedTypes()}
	if err := fs.Reload(); err != nil {
		return nil, err
	}
	return fs, nil
}

func allowedTypes() map[ResourceType]bool {
	return map[ResourceType]bool{
		TypeNote: true, TypeTextbook: true, TypeQBank: true,
		TypeMindmap: true, TypeTranslation: true, TypeFlashcard: true,
		TypePaper: true, TypeChat: true, TypeTodo: true, TypeSkill: true,
	}
}

// VaultDir 返回 vault 根目录（vault 模式下有效）。
func (fs *FS) VaultDir() string {
	if fs.vault == nil {
		return ""
	}
	return fs.vault.Root
}

// Reload 重新扫描 vault，重建索引（外部编辑/Obsidian 修改后调用）。
func (fs *FS) Reload() error {
	if fs.vault == nil {
		return nil
	}
	scanned, _ := fs.vault.Scan()
	next := make(map[string]Entry, len(scanned))
	for _, se := range scanned {
		next[se.URI] = Entry{
			URI:       se.URI,
			Type:      ResourceType(se.Type),
			ID:        se.ID,
			Title:     se.Title,
			Tags:      se.Tags,
			Metadata:  se.Metadata,
			Size:      se.Size,
			CreatedAt: se.CreatedAt,
			UpdatedAt: se.UpdatedAt,
			FilePath:  se.FilePath,
		}
	}
	fs.mu.Lock()
	fs.entries = next
	fs.mu.Unlock()
	return nil
}

// Put 写入一个资源。
// vault 模式下：Markdown 类型写入 {type}/{title}.md（带 frontmatter），
// 原始格式（pdf/docx 等）按 meta["ext"] 落盘；同时写 blob 作兼容备份。
func (fs *FS) Put(uri string, payload []byte, meta map[string]string) (Entry, error) {
	u, err := Parse(uri)
	if err != nil {
		return Entry{}, err
	}
	if !fs.allowed[u.Type] {
		return Entry{}, fmt.Errorf("vfs: type %s not allowed", u.Type)
	}
	title := meta["title"]
	if title == "" {
		title = u.ID
	}
	ext := strings.TrimPrefix(meta["ext"], ".")
	if ext == "" {
		ext = "md"
	}

	e := Entry{
		URI:       uri,
		Type:      u.Type,
		ID:        u.ID,
		Title:     title,
		Tags:      splitTags(meta["tags"]),
		Metadata:  meta,
		CreatedAt: nowUnix(),
		UpdatedAt: nowUnix(),
	}

	if fs.vault != nil {
		// 定位已有文件（同 ds_id 重写不换名）
		var existing string
		fs.mu.RLock()
		if cur, ok := fs.entries[uri]; ok && cur.FilePath != "" {
			existing = cur.FilePath
		}
		fs.mu.RUnlock()

		md := vault.IsMarkdownExt("." + ext)
		dir := filepath.Join(fs.vault.Root, vault.TypeDir(vault.Type(u.Type)))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return Entry{}, err
		}
		var data []byte
		if md {
			fm := vault.FrontMatter{
				Title:   title,
				Tags:    e.Tags,
				Created: unixTime(e.CreatedAt),
				Updated: unixTime(e.UpdatedAt),
				DSID:    u.ID,
				DSType:  string(u.Type),
				Extra:   extraMeta(meta),
			}
			body := string(payload)
			data, err = vault.WriteFrontMatter(fm, body)
			if err != nil {
				return Entry{}, err
			}
		} else {
			data = payload
		}
		path := fs.vault.AllocatePath(vault.Type(u.Type), title, "."+ext, existing)
		if !md {
			// 非 Markdown 资源：sidecar 元数据（frontmatter 无法内嵌）
			fm := vault.FrontMatter{
				Title:   title,
				Tags:    e.Tags,
				Created: unixTime(e.CreatedAt),
				Updated: unixTime(e.UpdatedAt),
				DSID:    u.ID,
				DSType:  string(u.Type),
				Extra:   extraMeta(meta),
			}
			rel, rerr := filepath.Rel(fs.vault.Root, path)
			if rerr == nil {
				if err := fs.vault.WriteMeta(filepath.ToSlash(rel), fm); err != nil {
					return Entry{}, err
				}
			}
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return Entry{}, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return Entry{}, err
		}
		e.Size = info.Size()
		e.FilePath = path
		if cur, ok := fs.entry(uri); ok && cur.CreatedAt > 0 {
			e.CreatedAt = cur.CreatedAt // 重写保留创建时间
		}
	} else {
		// 内存版：写 blob
		if fs.blob == nil {
			return Entry{}, errors.New("vfs: blob store not configured")
		}
		ref, size, err := fs.blob.Put(payload)
		if err != nil {
			return Entry{}, err
		}
		e.BlobRef = ref
		e.Size = size
	}

	fs.mu.Lock()
	fs.entries[uri] = e
	fs.mu.Unlock()
	return e, nil
}

// Get 读取资源。
func (fs *FS) Get(uri string) ([]byte, Entry, error) {
	e, ok := fs.entry(uri)
	if !ok {
		return nil, Entry{}, fmt.Errorf("vfs: not found: %s", uri)
	}
	if fs.vault != nil && e.FilePath != "" {
		data, err := os.ReadFile(e.FilePath)
		if err != nil {
			return nil, e, err
		}
		// Markdown 资源剥离 frontmatter 返回正文
		if vault.IsMarkdownExt(strings.ToLower(filepath.Ext(e.FilePath))) {
			_, body, _ := vault.ReadFrontMatter(data)
			return []byte(body), e, nil
		}
		return data, e, nil
	}
	if e.BlobRef != "" && fs.blob != nil {
		data, err := fs.blob.Get(e.BlobRef)
		if err != nil {
			return nil, e, err
		}
		return data, e, nil
	}
	return nil, e, fmt.Errorf("vfs: payload unavailable: %s", uri)
}

// Stat 返回元数据（不读 payload）。
func (fs *FS) Stat(uri string) (Entry, bool) {
	return fs.entry(uri)
}

// List 列出某类型的资源。
func (fs *FS) List(t ResourceType) []Entry {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	out := make([]Entry, 0)
	for _, e := range fs.entries {
		if t == "" || e.Type == t {
			out = append(out, e)
		}
	}
	return out
}

// Search 按 tag 搜索。
func (fs *FS) Search(t ResourceType, tag string) []Entry {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	out := make([]Entry, 0)
	for _, e := range fs.entries {
		if t != "" && e.Type != t {
			continue
		}
		for _, et := range e.Tags {
			if et == tag {
				out = append(out, e)
				break
			}
		}
	}
	return out
}

// Delete 删除资源（vault 模式下同时删除文件）。
func (fs *FS) Delete(uri string) error {
	e, ok := fs.entry(uri)
	if !ok {
		return fmt.Errorf("vfs: not found: %s", uri)
	}
	if fs.vault != nil && e.FilePath != "" {
		if err := os.Remove(e.FilePath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	fs.mu.Lock()
	delete(fs.entries, uri)
	fs.mu.Unlock()
	return nil
}

// LockForRead 获取读锁（用于备份等长时读操作）。
func (fs *FS) LockForRead() { fs.mu.RLock() }

// UnlockRead 释放读锁。
func (fs *FS) UnlockRead() { fs.mu.RUnlock() }

// Snapshot 在持锁期间拍下 entries 浅拷贝；调用方必须在调用前持有读锁。
func (fs *FS) Snapshot() []Entry {
	out := make([]Entry, 0, len(fs.entries))
	for _, e := range fs.entries {
		out = append(out, e)
	}
	return out
}

func (fs *FS) entry(uri string) (Entry, bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	e, ok := fs.entries[uri]
	return e, ok
}

func unixTime(unix int64) (t time.Time) {
	return time.Unix(unix, 0).UTC()
}

func extraMeta(meta map[string]string) map[string]string {
	out := map[string]string{}
	skip := map[string]bool{"title": true, "tags": true}
	for k, v := range meta {
		if skip[k] || v == "" {
			continue
		}
		out[k] = v
	}
	return out
}

func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// AllResourceTypes 返回全部资源类型（遍历顺序稳定）。
func AllResourceTypes() []ResourceType {
	return []ResourceType{
		TypeNote, TypeTextbook, TypeQBank, TypeMindmap,
		TypeTranslation, TypeFlashcard, TypePaper, TypeChat,
	}
}
