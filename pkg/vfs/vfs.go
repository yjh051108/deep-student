// Package vfs 提供统一虚拟文件系统抽象，所有资源通过 vfs:// URI 寻址。
package vfs

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/helixnow/deep-student-go/pkg/store/blob"
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

// FS 虚拟文件系统。
type FS struct {
	mu      sync.RWMutex
	blob    *blob.Store
	entries map[string]Entry // index by vfs://...
	allowed map[ResourceType]bool
}

// Entry 资源元数据（不含 payload）。
type Entry struct {
	URI       string            `json:"uri"`
	Type      ResourceType      `json:"type"`
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Tags      []string          `json:"tags"`
	Metadata  map[string]string `json:"metadata"`
	BlobRef   string            `json:"blob_ref"` // sha256
	Size      int64             `json:"size"`
	CreatedAt int64             `json:"created_at"`
	UpdatedAt int64             `json:"updated_at"`
}

// NewFS 创建虚拟文件系统。
func NewFS(blobStore *blob.Store) *FS {
	allowed := map[ResourceType]bool{
		TypeNote: true, TypeTextbook: true, TypeQBank: true,
		TypeMindmap: true, TypeTranslation: true, TypeFlashcard: true,
		TypePaper: true, TypeChat: true, TypeTodo: true, TypeSkill: true,
	}
	return &FS{blob: blobStore, entries: map[string]Entry{}, allowed: allowed}
}

// Put 写入一个资源。
func (fs *FS) Put(uri string, payload []byte, meta map[string]string) (Entry, error) {
	u, err := Parse(uri)
	if err != nil {
		return Entry{}, err
	}
	if !fs.allowed[u.Type] {
		return Entry{}, fmt.Errorf("vfs: type %s not allowed", u.Type)
	}
	if fs.blob == nil {
		return Entry{}, errors.New("vfs: blob store not configured")
	}
	ref, size, err := fs.blob.Put(payload)
	if err != nil {
		return Entry{}, err
	}
	e := Entry{
		URI:       uri,
		Type:      u.Type,
		ID:        u.ID,
		Title:     meta["title"],
		Tags:      splitTags(meta["tags"]),
		Metadata:  meta,
		BlobRef:   ref,
		Size:      size,
		CreatedAt: nowUnix(),
		UpdatedAt: nowUnix(),
	}
	fs.mu.Lock()
	fs.entries[uri] = e
	fs.mu.Unlock()
	return e, nil
}

// Get 读取资源。
func (fs *FS) Get(uri string) ([]byte, Entry, error) {
	fs.mu.RLock()
	e, ok := fs.entries[uri]
	fs.mu.RUnlock()
	if !ok {
		return nil, Entry{}, fmt.Errorf("vfs: not found: %s", uri)
	}
	data, err := fs.blob.Get(e.BlobRef)
	if err != nil {
		return nil, e, err
	}
	return data, e, nil
}

// Stat 返回元数据（不读 blob）。
func (fs *FS) Stat(uri string) (Entry, bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	e, ok := fs.entries[uri]
	return e, ok
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

// Delete 删除资源（不删除 blob 引用计数）。
func (fs *FS) Delete(uri string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if _, ok := fs.entries[uri]; !ok {
		return fmt.Errorf("vfs: not found: %s", uri)
	}
	delete(fs.entries, uri)
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
