package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
)

// MigrateResult 迁移结果统计。
type MigrateResult struct {
	Total      int      `json:"total"`
	Migrated   int      `json:"migrated"`
	Skipped    int      `json:"skipped"`
	Failed     int      `json:"failed"`
	Errors     []string `json:"errors"`
	DidRun     bool     `json:"didRun"` // 是否执行了迁移（vault 非空则跳过）
}

// MigrateFromBlob 把存量 blob 资源一次性导出到 vault（幂等）。
//
// 触发条件：vault 根目录尚无任何资源文件（.md/.pdf/.docx 等），
// 且 resources 表 / blob 有数据。迁移完成后 blob 保持只读。
func (v *Vault) MigrateFromBlob(st *store.Store, bs *blob.Store) (*MigrateResult, error) {
	res := &MigrateResult{}
	if st == nil || st.DB == nil {
		return res, nil
	}
	// vault 已有资源则不迁移（避免重复导出）
	if v.hasResourceFiles() {
		return res, nil
	}
	rows, err := st.ListResources("")
	if err != nil {
		return nil, fmt.Errorf("vault: list resources: %w", err)
	}
	res.Total = len(rows)
	if res.Total == 0 {
		return res, nil
	}
	res.DidRun = true

	for _, row := range rows {
		if err := v.exportRow(bs, row); err != nil {
			res.Failed++
			res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", row.URI, err))
			continue
		}
		res.Migrated++
	}
	return res, nil
}

// exportRow 把一条 resources 记录导出为 vault 文件。
func (v *Vault) exportRow(bs *blob.Store, row store.ResourceRow) error {
	if bs == nil || row.BlobRef == "" {
		return fmt.Errorf("no blob ref")
	}
	data, err := bs.Get(row.BlobRef)
	if err != nil {
		return err
	}
	typ := Type(row.Type)
	title := row.Title
	if title == "" {
		title = row.ID
	}
	ext := "md"
	if meta, ok := parseMeta(row.Metadata); ok {
		if e := meta["ext"]; e != "" {
			ext = e
		}
	}
	dir := filepath.Join(v.Root, TypeDir(typ))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := v.AllocatePath(typ, title, "."+ext, "")
	fm := FrontMatter{
		Title:   title,
		Tags:    splitTagsCSV(row.Tags),
		Created: time.Unix(row.CreatedAt, 0).UTC(),
		Updated: time.Unix(row.UpdatedAt, 0).UTC(),
		DSID:    row.ID,
		DSType:  string(typ),
	}
	var payload []byte
	if IsMarkdownExt("." + ext) {
		body := string(data)
		payload, err = WriteFrontMatter(fm, body)
		if err != nil {
			return err
		}
	} else {
		// 非 Markdown（PDF/DOCX…）无法内嵌 frontmatter：落盘原始文件 + sidecar
		payload = data
		fm.Extra = map[string]string{"ext": ext}
		if err := v.WriteMeta(relPathFromRoot(v, path), fm); err != nil {
			return err
		}
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return err
	}
	return nil
}

// relPathFromRoot 计算文件相对 vault 根的斜杠路径。
func relPathFromRoot(v *Vault, p string) string {
	rel, err := filepath.Rel(v.Root, p)
	if err != nil {
		return filepath.ToSlash(p)
	}
	return filepath.ToSlash(rel)
}

// hasResourceFiles vault 根下是否存在资源文件（递归，排除 .deepstudent）。
func (v *Vault) hasResourceFiles() bool {
	found := false
	_ = filepath.Walk(v.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if info.Name() == ".deepstudent" || strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(info.Name()))
		switch ext {
		case ".md", ".markdown", ".pdf", ".docx", ".txt":
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func parseMeta(s string) (map[string]string, bool) {
	out := map[string]string{}
	// metadata 形如 {"title":"x","ext":"pdf",...}（JSON 或简易 key:value）
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "{") {
		for _, part := range strings.Split(strings.Trim(s, "{}"), ",") {
			kv := strings.SplitN(part, ":", 2)
			if len(kv) == 2 {
				k := strings.Trim(strings.TrimSpace(kv[0]), `"`)
				val := strings.Trim(strings.TrimSpace(kv[1]), `"`)
				if k != "" {
					out[k] = val
				}
			}
		}
	}
	return out, len(out) > 0
}

func splitTagsCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
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
