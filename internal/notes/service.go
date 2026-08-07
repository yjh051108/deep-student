// notes 包的服务层：业务逻辑（字数统计、编码识别、资产去重、导出格式化等）。

package notes

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/text/encoding/simplifiedchinese"

	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Service 笔记服务。
type Service struct {
	db   *Store
	blob *blob.Store
	vfs  *vfs.FS
	mu   sync.RWMutex
}

// New 创建笔记服务。会自动执行表迁移；迁移失败仅记录到 stderr，不阻塞构造。
func New(db *store.Store, blobStore *blob.Store, vfsFS *vfs.FS) *Service {
	ns := NewStore(db)
	if err := ns.Migrate(); err != nil {
		// 迁移失败时仍返回 Service，后续 CRUD 会暴露错误
		fmt.Printf("[notes] migrate failed: %v\n", err)
	}
	return &Service{db: ns, blob: blobStore, vfs: vfsFS}
}

// ===================== CRUD =====================

// Create 创建新笔记。
func (s *Service) Create(params CreateParams) (*Note, error) {
	if strings.TrimSpace(params.Title) == "" {
		return nil, errors.New("notes: title is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	tags := dedupTags(params.Tags)
	wordCount, charCount := countWords(params.ContentMD)
	n := &Note{
		ID:         uuid.NewString(),
		Title:      strings.TrimSpace(params.Title),
		ContentMD:  params.ContentMD,
		Tags:       tags,
		FolderID:   normalizeFolderID(params.FolderID),
		HasAssets:  false,
		AssetCount: 0,
		IsPinned:   false,
		IsDeleted:  false,
		WordCount:  wordCount,
		CharCount:  charCount,
		CreatedAt:  now,
		UpdatedAt:  now,
		Metadata:   map[string]string{},
	}
	if err := s.db.CreateNote(n); err != nil {
		return nil, err
	}
	return n, nil
}

// Get 读取单条笔记。
func (s *Service) Get(id string) (*Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db.GetNote(id)
}

// Update 更新笔记（支持部分字段 + 乐观锁）。
func (s *Service) Update(params UpdateParams) (*Note, error) {
	if params.ID == "" {
		return nil, errors.New("notes: id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, err := s.db.GetNote(params.ID)
	if err != nil {
		return nil, err
	}
	// 乐观锁校验
	if params.ExpectedUpdate != nil {
		if !existing.UpdatedAt.Equal(*params.ExpectedUpdate) {
			return nil, errors.New("notes.conflict: note has been updated elsewhere, please refresh")
		}
	}
	now := time.Now().UTC()
	// 计算新字数
	var newWordCount, newCharCount int = -1, -1
	if params.ContentMD != nil {
		newWordCount, newCharCount = countWords(*params.ContentMD)
	}
	if err := s.db.UpdateNote(
		params.ID, params.Title, params.ContentMD, params.Tags,
		params.FolderID, params.IsPinned,
		newWordCount, newCharCount, now,
	); err != nil {
		return nil, err
	}
	return s.db.GetNote(params.ID)
}

// List 列出笔记（含正文）。
func (s *Service) List(opts ListOptions) (*ListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.db.ListNotes(opts, true)
	if err != nil {
		return nil, err
	}
	total, err := s.db.CountNotes(opts)
	if err != nil {
		return nil, err
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	return &ListResult{Items: derefNotes(items), Total: total, Limit: limit, Offset: opts.Offset}, nil
}

// ListMeta 列出笔记元数据（不含正文），用于列表视图降低载荷。
func (s *Service) ListMeta(opts ListOptions) (*ListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, err := s.db.ListNotes(opts, false)
	if err != nil {
		return nil, err
	}
	total, err := s.db.CountNotes(opts)
	if err != nil {
		return nil, err
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	return &ListResult{Items: derefNotes(items), Total: total, Limit: limit, Offset: opts.Offset}, nil
}

// derefNotes 将 []*Note 转为 []Note。
func derefNotes(items []*Note) []Note {
	out := make([]Note, 0, len(items))
	for _, n := range items {
		if n != nil {
			out = append(out, *n)
		}
	}
	return out
}

// ===================== 回收站 =====================

// MoveToTrash 将笔记移入回收站（软删除）。
func (s *Service) MoveToTrash(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.SoftDelete(id, time.Now().UTC())
}

// Restore 从回收站恢复笔记。
func (s *Service) Restore(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Restore(id, time.Now().UTC())
}

// HardDelete 永久删除笔记及其全部资产。
func (s *Service) HardDelete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 1. 删除全部资产（含 blob 引用）
	assets, err := s.db.ListAssets(id)
	if err != nil {
		return err
	}
	for _, a := range assets {
		if s.blob != nil {
			_ = s.blob.Delete(a.BlobRef)
		}
		if err := s.db.DeleteAsset(a.ID); err != nil {
			return err
		}
	}
	// 2. 删除笔记行
	return s.db.HardDelete(id)
}

// EmptyTrash 清空回收站，返回被清空的笔记数量。
func (s *Service) EmptyTrash() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids, err := s.db.EmptyTrash()
	if err != nil {
		return 0, err
	}
	// 清理每条笔记的资产
	for _, id := range ids {
		assets, aerr := s.db.ListAssets(id)
		if aerr != nil {
			continue
		}
		for _, a := range assets {
			if s.blob != nil {
				_ = s.blob.Delete(a.BlobRef)
			}
			_ = s.db.DeleteAsset(a.ID)
		}
	}
	return len(ids), nil
}

// TrashCount 返回回收站笔记数量。
func (s *Service) TrashCount() (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db.GetTrashCount()
}

// ===================== 文件夹 =====================

// ListFolders 列出全部文件夹。
func (s *Service) ListFolders() ([]*Folder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db.ListFolders()
}

// CreateFolder 创建新文件夹。
func (s *Service) CreateFolder(name string, parentID *string) (*Folder, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("notes: folder name is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	f := &Folder{
		ID:        uuid.NewString(),
		Name:      strings.TrimSpace(name),
		ParentID:  normalizeFolderID(parentID),
		SortOrder: 0,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.db.CreateFolder(f); err != nil {
		return nil, err
	}
	return f, nil
}

// UpdateFolder 重命名文件夹。
func (s *Service) UpdateFolder(id, name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("notes: folder name is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.UpdateFolder(id, strings.TrimSpace(name), time.Now().UTC())
}

// DeleteFolder 删除文件夹，文件夹下的笔记移到根目录。
func (s *Service) DeleteFolder(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.DeleteFolder(id)
}

// ===================== 资产 =====================

// AddAsset 为笔记添加附件，内容通过 blob.Store 去重存储。
func (s *Service) AddAsset(noteID, filename string, data []byte, mime string) (*Asset, error) {
	if noteID == "" {
		return nil, errors.New("notes: noteID is required")
	}
	if s.blob == nil {
		return nil, errors.New("notes: blob store not configured")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// 确认笔记存在
	if _, err := s.db.GetNote(noteID); err != nil {
		return nil, err
	}
	// 写入 blob
	ref, size, err := s.blob.Put(data)
	if err != nil {
		return nil, fmt.Errorf("notes: put asset blob: %w", err)
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	a := &Asset{
		ID:        uuid.NewString(),
		NoteID:    noteID,
		Filename:  filename,
		MIMEType:  mime,
		Size:      size,
		BlobRef:   ref,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.db.AddAsset(a); err != nil {
		// 回滚 blob
		_ = s.blob.Delete(ref)
		return nil, err
	}
	// 更新笔记资产计数
	assets, _ := s.db.ListAssets(noteID)
	_ = s.db.UpdateNoteAssetStats(noteID, len(assets) > 0, len(assets))
	return a, nil
}

// ListAssets 列出指定笔记的全部资产。
func (s *Service) ListAssets(noteID string) ([]*Asset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db.ListAssets(noteID)
}

// GetAsset 读取资产内容与元数据。
func (s *Service) GetAsset(assetID string) ([]byte, *Asset, error) {
	s.mu.RLock()
	a, err := s.db.GetAsset(assetID)
	s.mu.RUnlock()
	if err != nil {
		return nil, nil, err
	}
	if s.blob == nil {
		return nil, a, errors.New("notes: blob store not configured")
	}
	data, err := s.blob.Get(a.BlobRef)
	if err != nil {
		return nil, a, err
	}
	return data, a, nil
}

// DeleteAsset 删除资产（含 blob 引用）。
func (s *Service) DeleteAsset(assetID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, err := s.db.GetAsset(assetID)
	if err != nil {
		return err
	}
	if err := s.db.DeleteAsset(assetID); err != nil {
		return err
	}
	if s.blob != nil {
		_ = s.blob.Delete(a.BlobRef)
	}
	// 更新笔记资产计数
	assets, _ := s.db.ListAssets(a.NoteID)
	_ = s.db.UpdateNoteAssetStats(a.NoteID, len(assets) > 0, len(assets))
	return nil
}

// ===================== 导入导出 =====================

// ImportMarkdown 从 Markdown 字节流导入笔记，自动识别编码（UTF-8/UTF-8 BOM/UTF-16/GBK/GB18030）。
func (s *Service) ImportMarkdown(filename string, content []byte, folderID *string) (*Note, error) {
	if len(content) == 0 {
		return nil, errors.New("notes: empty content")
	}
	decoded := decodeMarkdownBytes(content)
	title := deriveTitle(filename, decoded)
	return s.Create(CreateParams{
		Title:     title,
		ContentMD: decoded,
		Tags:      []string{},
		FolderID:  normalizeFolderID(folderID),
	})
}

// ImportBatch 批量导入 Markdown 文件。
func (s *Service) ImportBatch(files map[string][]byte, folderID *string) ([]*Note, error) {
	if len(files) == 0 {
		return nil, errors.New("notes: no files to import")
	}
	out := make([]*Note, 0, len(files))
	for name, data := range files {
		n, err := s.ImportMarkdown(name, data, folderID)
		if err != nil {
			return out, fmt.Errorf("notes: import %s: %w", name, err)
		}
		out = append(out, n)
	}
	return out, nil
}

// ExportNote 导出单条笔记。
func (s *Service) ExportNote(id string, format ExportFormat) ([]byte, error) {
	s.mu.RLock()
	n, err := s.db.GetNote(id)
	s.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	return exportNote(n, format)
}

// ExportAll 导出全部笔记（未删除）。
func (s *Service) ExportAll(format ExportFormat) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	opts := ListOptions{Limit: 10000}
	items, err := s.db.ListNotes(opts, true)
	if err != nil {
		return nil, err
	}
	return exportNotes(items, format)
}

// ===================== 工具 =====================

// Search 简单关键字搜索（标题 + 正文 LIKE）。
func (s *Service) Search(keyword string, limit int) ([]Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 {
		limit = 20
	}
	opts := ListOptions{Keyword: keyword, Limit: limit}
	items, err := s.db.ListNotes(opts, true)
	if err != nil {
		return nil, err
	}
	out := make([]Note, 0, len(items))
	for _, n := range items {
		out = append(out, *n)
	}
	return out, nil
}

// Stats 返回笔记统计：total / trash / pinned / assets。
func (s *Service) Stats() (map[string]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := map[string]int{
		"total":  0,
		"trash":  0,
		"pinned": 0,
		"assets": 0,
	}
	// total：未删除笔记数
	total, err := s.db.CountNotes(ListOptions{})
	if err != nil {
		return stats, err
	}
	stats["total"] = int(total)
	// trash：回收站数
	trash, err := s.db.GetTrashCount()
	if err != nil {
		return stats, err
	}
	stats["trash"] = int(trash)
	// pinned：置顶数（未删除）
	pinnedOpts := ListOptions{}
	pinnedItems, err := s.db.ListNotes(pinnedOpts, false)
	if err != nil {
		return stats, err
	}
	pinnedCount := 0
	for _, n := range pinnedItems {
		if n.IsPinned {
			pinnedCount++
		}
	}
	stats["pinned"] = pinnedCount
	// assets：总资产数（扫描 note_assets 表）
	// 通过 ListAssetsByNote 间接统计太重，这里直接 COUNT
	if s.db.db != nil {
		var assetCount int
		_ = s.db.db.QueryRow(`SELECT COUNT(*) FROM note_assets`).Scan(&assetCount)
		stats["assets"] = assetCount
	}
	return stats, nil
}

// ===================== 内部工具函数 =====================

// countWords 统计字数与字符数。
// 中文（CJK）按字符计数；英文/数字按空白分词计数。
// charCount 为全部 rune 数（不含换行）。
func countWords(content string) (int, int) {
	wordCount := 0
	charCount := 0
	var inWord bool
	for _, r := range content {
		if r == '\n' || r == '\r' {
			inWord = false
			continue
		}
		charCount++
		if isCJK(r) {
			// CJK 每个字符算一个词
			wordCount++
			inWord = false
			continue
		}
		if unicode.IsSpace(r) {
			inWord = false
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				wordCount++
				inWord = true
			}
		} else {
			// 标点等：单独算一个词（与常见编辑器行为接近）
			if !unicode.IsSpace(r) {
				if !inWord {
					wordCount++
					inWord = true
				}
			}
		}
	}
	return wordCount, charCount
}

// isCJK 判断 rune 是否属于 CJK 区段。
func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK 统一表意
		(r >= 0x3400 && r <= 0x4DBF) || // CJK 扩展 A
		(r >= 0x20000 && r <= 0x2A6DF) || // CJK 扩展 B
		(r >= 0x3040 && r <= 0x309F) || // 平假名
		(r >= 0x30A0 && r <= 0x30FF) // 片假名
}

// dedupTags 去重并清理标签。
func dedupTags(tags []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// normalizeFolderID 规范化 folderID：空字符串视为 nil（根目录）。
func normalizeFolderID(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// decodeMarkdownBytes 识别并解码 Markdown 字节流，参考 Rust decode_markdown_bytes。
// 顺序：UTF-8 BOM → UTF-16LE BOM → UTF-16BE BOM → UTF-8 → GBK → GB18030。
func decodeMarkdownBytes(b []byte) string {
	// UTF-8 BOM
	if bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}) {
		return stripBOM(string(b[3:]))
	}
	// UTF-16LE BOM
	if bytes.HasPrefix(b, []byte{0xFF, 0xFE}) {
		return stripBOM(decodeUTF16(b[2:], true))
	}
	// UTF-16BE BOM
	if bytes.HasPrefix(b, []byte{0xFE, 0xFF}) {
		return stripBOM(decodeUTF16(b[2:], false))
	}
	// UTF-8（无 BOM）
	if utf8.Valid(b) {
		return stripBOM(string(b))
	}
	// GBK
	if decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(b); err == nil {
		if utf8.Valid(decoded) {
			return stripBOM(string(decoded))
		}
	}
	// GB18030（兜底）
	if decoded, err := simplifiedchinese.GB18030.NewDecoder().Bytes(b); err == nil {
		return stripBOM(string(decoded))
	}
	// 最终兜底：按 UTF-8 容错解码
	return stripBOM(string(b))
}

// decodeUTF16 解码 UTF-16 字节流。littleEndian=true 表示 UTF-16LE。
func decodeUTF16(b []byte, littleEndian bool) string {
	if len(b)%2 != 0 {
		// 奇数字节，截断末尾
		b = b[:len(b)-1]
	}
	u16 := make([]uint16, len(b)/2)
	for i := 0; i < len(u16); i++ {
		if littleEndian {
			u16[i] = uint16(b[2*i]) | (uint16(b[2*i+1]) << 8)
		} else {
			u16[i] = (uint16(b[2*i]) << 8) | uint16(b[2*i+1])
		}
	}
	runes := utf16.Decode(u16)
	return string(runes)
}

// stripBOM 去除开头的 U+FEFF（UTF-8 BOM 字节序列 0xEF 0xBB 0xBF）。
func stripBOM(s string) string {
	if strings.HasPrefix(s, "\uFEFF") {
		return strings.TrimPrefix(s, "\uFEFF")
	}
	return s
}

// deriveTitle 从文件名推导笔记标题（对照 Rust derive_markdown_note_title）。
// 优先使用文件名 stem；若为通用占位符（空/"文件"/纯数字/冒号+数字）则退化到正文 H1；最终兜底时间戳。
func deriveTitle(filename, content string) string {
	// 1. 取文件名 stem
	candidate := ""
	if filename != "" {
		base := filepath.Base(filename)
		ext := filepath.Ext(base)
		if ext != "" {
			base = strings.TrimSuffix(base, ext)
		}
		candidate = strings.TrimSpace(base)
	}
	// 2. 若文件名非通用占位符，直接使用
	if !isGenericNoteTitle(candidate) {
		return candidate
	}
	// 3. 通用占位符：尝试从正文第一个 H1 提取
	if h1 := extractFirstHeading(content); h1 != "" {
		return h1
	}
	// 4. 兜底：时间戳
	return fmt.Sprintf("导入笔记_%s", time.Now().UTC().Format("20060102_150405"))
}

// isGenericNoteTitle 判断标题是否为通用占位符（对照 Rust is_generic_note_title）。
// 检测：空 / "文件" / 纯数字 / 含冒号且冒号后全是数字。
func isGenericNoteTitle(title string) bool {
	t := strings.TrimSpace(title)
	if t == "" || t == "文件" {
		return true
	}
	// 纯数字
	if t != "" {
		allDigit := true
		for _, c := range t {
			if !unicode.IsDigit(c) {
				allDigit = false
				break
			}
		}
		if allDigit {
			return true
		}
	}
	// 含冒号且冒号后全是数字
	if pos := strings.Index(t, ":"); pos >= 0 {
		after := t[pos+1:]
		if after != "" {
			allDigit := true
			for _, c := range after {
				if !unicode.IsDigit(c) {
					allDigit = false
					break
				}
			}
			if allDigit {
				return true
			}
		}
	}
	return false
}

// extractFirstHeading 从 Markdown 内容提取第一个 H1 标题（对照 Rust extract_first_heading）。
func extractFirstHeading(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			if title != "" {
				return title
			}
		}
	}
	return ""
}

// exportNote 导出单条笔记为指定格式。
func exportNote(n *Note, format ExportFormat) ([]byte, error) {
	switch format {
	case ExportMarkdown:
		return []byte(n.ContentMD), nil
	case ExportHTML:
		return []byte(markdownToHTML(n.Title, n.ContentMD)), nil
	case ExportJSON:
		return json.MarshalIndent(n, "", "  ")
	default:
		return nil, fmt.Errorf("notes: unsupported export format: %s", format)
	}
}

// exportNotes 导出多条笔记为指定格式。
func exportNotes(items []*Note, format ExportFormat) ([]byte, error) {
	switch format {
	case ExportMarkdown:
		var sb strings.Builder
		for i, n := range items {
			if i > 0 {
				sb.WriteString("\n\n---\n\n")
			}
			sb.WriteString("# ")
			sb.WriteString(n.Title)
			sb.WriteString("\n\n")
			sb.WriteString(n.ContentMD)
		}
		return []byte(sb.String()), nil
	case ExportHTML:
		var sb strings.Builder
		sb.WriteString("<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\"><title>Notes</title></head><body>\n")
		for _, n := range items {
			sb.WriteString(markdownToHTML(n.Title, n.ContentMD))
			sb.WriteString("\n")
		}
		sb.WriteString("</body></html>")
		return []byte(sb.String()), nil
	case ExportJSON:
		return json.MarshalIndent(items, "", "  ")
	default:
		return nil, fmt.Errorf("notes: unsupported export format: %s", format)
	}
}

// markdownToHTML 极简 Markdown → HTML 转换（满足导出预览需求，非完整规范）。
// 支持：# ## ### 标题、**bold**、*italic*、`code`、空行分段、无序列表 - 项。
func markdownToHTML(title, content string) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\"><title>")
	sb.WriteString(html.EscapeString(title))
	sb.WriteString("</title></head><body>\n")
	sb.WriteString("<h1>")
	sb.WriteString(html.EscapeString(title))
	sb.WriteString("</h1>\n")

	lines := strings.Split(content, "\n")
	inList := false
	inParagraph := strings.Builder{}
	flushParagraph := func() {
		if inParagraph.Len() > 0 {
			sb.WriteString("<p>")
			sb.WriteString(inlineMD(inParagraph.String()))
			sb.WriteString("</p>\n")
			inParagraph.Reset()
		}
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 标题
		if m := headingRegex.FindStringSubmatch(trimmed); m != nil {
			flushParagraph()
			if inList {
				sb.WriteString("</ul>\n")
				inList = false
			}
			level := len(m[1])
			sb.WriteString(fmt.Sprintf("<h%d>%s</h%d>\n", level, html.EscapeString(m[2]), level))
			continue
		}
		// 无序列表项
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			flushParagraph()
			if !inList {
				sb.WriteString("<ul>\n")
				inList = true
			}
			sb.WriteString("<li>")
			sb.WriteString(inlineMD(trimmed[2:]))
			sb.WriteString("</li>\n")
			continue
		}
		// 空行：分段
		if trimmed == "" {
			flushParagraph()
			if inList {
				sb.WriteString("</ul>\n")
				inList = false
			}
			continue
		}
		// 普通行：累积到段落
		if inParagraph.Len() > 0 {
			inParagraph.WriteString(" ")
		}
		inParagraph.WriteString(trimmed)
	}
	flushParagraph()
	if inList {
		sb.WriteString("</ul>\n")
	}
	sb.WriteString("</body></html>")
	return sb.String()
}

var headingRegex = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

// inlineMD 处理行内 Markdown：**bold** / *italic* / `code`。
func inlineMD(s string) string {
	// 先转义 HTML 实体
	s = html.EscapeString(s)
	// `code`
	s = codeRegex.ReplaceAllString(s, "<code>$1</code>")
	// **bold**（先于 *italic* 处理）
	s = boldRegex.ReplaceAllString(s, "<strong>$1</strong>")
	// *italic*
	s = italicRegex.ReplaceAllString(s, "<em>$1</em>")
	return s
}

var (
	boldRegex   = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	italicRegex = regexp.MustCompile(`\*([^*]+)\*`)
	codeRegex   = regexp.MustCompile("`([^`]+)`")
)
