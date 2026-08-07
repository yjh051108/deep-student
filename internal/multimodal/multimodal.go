// Package multimodal 提供多模态内容索引与检索：
//   - 对资源（PDF/图片/文档）生成文本描述（OCR / 文本层 / 摘要）；
//   - 用嵌入模型生成向量，存入 SQLite（vfs_multimodal_units）；
//   - 向量 + 关键词混合检索（复用 pkg/vector 与 FTS）。
//
// 对齐 Rust 原版 src-tauri/src/multimodal/ 与 vfs/multimodal_service.rs 的接口面。

package multimodal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/vector"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Unit 多模态索引单元。
type Unit struct {
	ID        string    `json:"id"`
	URI       string    `json:"uri"`
	ChunkIdx  int       `json:"chunkIdx"`
	Content   string    `json:"content"`
	Embedding []byte    `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
}

// Stats 索引统计。
type Stats struct {
	TotalUnits    int `json:"totalUnits"`
	IndexedURIs   int `json:"indexedUris"`
	EmbeddedUnits int `json:"embeddedUnits"`
	VectorDim     int `json:"vectorDim"`
}

// Service 多模态索引服务。
type Service struct {
	db     *sql.DB
	llmReg *llm.Registry
	vfs    *vfs.FS
	mu     sync.RWMutex
	// 内存向量索引（启动时从库加载）
	vec *vector.Index
	dim int
}

// New 构造服务并建表。
func New(st *store.Store, l *llm.Registry, fs *vfs.FS) *Service {
	svc := &Service{llmReg: l, vfs: fs, vec: vector.New()}
	if st != nil && st.DB != nil {
		svc.db = st.DB
		if err := svc.migrate(); err != nil {
			fmt.Printf("[multimodal] migrate: %v\n", err)
		}
		svc.loadIndex()
	}
	return svc
}

func (s *Service) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS vfs_multimodal_units (
			id TEXT PRIMARY KEY,
			uri TEXT NOT NULL,
			chunk_idx INTEGER NOT NULL,
			content TEXT NOT NULL,
			embedding BLOB,
			dim INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mm_uri ON vfs_multimodal_units(uri)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

// loadIndex 启动时加载向量。
func (s *Service) loadIndex() {
	if s.db == nil {
		return
	}
	rows, err := s.db.Query(`SELECT id, uri, chunk_idx, content, embedding, dim FROM vfs_multimodal_units`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, uri, content string
		var idx int
		var emb []byte
		var dim int
		if err := rows.Scan(&id, &uri, &idx, &content, &emb, &dim); err != nil {
			continue
		}
		if dim > 0 && len(emb) > 0 {
			s.dim = dim
			s.vec.Reset(dim)
			s.vec.Add(vector.Vector{ID: id, Dim: dim, Embed: vector.Decode(emb)})
		}
	}
}

// IndexResource 为资源建立多模态索引：读取内容 → 切块 → 嵌入 → 入库。
func (s *Service) IndexResource(ctx context.Context, uri, content string) (int, error) {
	if s.db == nil {
		return 0, errors.New("multimodal: db not ready")
	}
	// 删除旧索引
	if _, err := s.db.Exec(`DELETE FROM vfs_multimodal_units WHERE uri=?`, uri); err != nil {
		return 0, err
	}
	chunks := chunkText(content, 800)
	provider, dim, err := s.resolveEmbedder(ctx)
	if err != nil {
		// 无嵌入能力：仅存文本（关键词检索仍可用）
		dim = 0
	}
	for i, c := range chunks {
		unit := Unit{
			ID: uuid.NewString(), URI: uri, ChunkIdx: i, Content: c, CreatedAt: time.Now().UTC(),
		}
		var emb []byte
		if dim > 0 && provider != nil {
			resp, err := provider.Embed(ctx, llm.EmbedRequest{Model: "", Input: []string{c}})
			if err == nil && len(resp.Embeddings) > 0 {
				s.mu.Lock()
				if s.dim == 0 {
					s.dim = len(resp.Embeddings[0])
					s.vec.Reset(s.dim)
				}
				if len(resp.Embeddings[0]) == s.dim {
					emb = vector.Encode(resp.Embeddings[0])
					s.vec.Add(vector.Vector{ID: unit.ID, Dim: len(resp.Embeddings[0]), Embed: resp.Embeddings[0]})
				}
				s.mu.Unlock()
			}
		}
		if _, err := s.db.Exec(`INSERT INTO vfs_multimodal_units(id, uri, chunk_idx, content, embedding, dim, created_at)
			VALUES (?,?,?,?,?,?,?)`,
			unit.ID, unit.URI, unit.ChunkIdx, unit.Content, emb, dim, unit.CreatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
			return 0, err
		}
	}
	return len(chunks), nil
}

// resolveEmbedder 从 registry 找支持嵌入的 provider。
func (s *Service) resolveEmbedder(ctx context.Context) (llm.Provider, int, error) {
	if s.llmReg == nil {
		return nil, 0, errors.New("multimodal: no llm registry")
	}
	// 优先 openai（通常支持 embeddings）
	for _, name := range []string{"openai", "siliconflow", "deepseek"} {
		if p, ok := s.llmReg.Get(name); ok {
			resp, err := p.Embed(ctx, llm.EmbedRequest{Input: []string{"probe"}})
			if err == nil && len(resp.Embeddings) > 0 {
				return p, len(resp.Embeddings[0]), nil
			}
		}
	}
	return nil, 0, errors.New("multimodal: no embedder available")
}

// Search 向量 + 关键词混合检索。
func (s *Service) Search(ctx context.Context, query string, topK int) ([]Result, error) {
	if topK <= 0 {
		topK = 10
	}
	var results []Result
	// 1. 关键词（FTS 风格 LIKE 匹配 content）
	if kwRows, err := s.db.Query(`SELECT id, uri, chunk_idx, content FROM vfs_multimodal_units
		WHERE content LIKE ? LIMIT ?`, "%"+escapeLike(query)+"%", topK); err == nil {
		defer kwRows.Close()
		for kwRows.Next() {
			var r Result
			if err := kwRows.Scan(&r.ID, &r.URI, &r.ChunkIdx, &r.Content); err == nil {
				r.Score = 0.5 // 关键词命中基线分
				results = append(results, r)
			}
		}
	}
	// 2. 向量检索
	if s.vec.Size() > 0 && s.llmReg != nil {
		if p, ok := s.llmReg.Get("openai"); ok {
			if resp, err := p.Embed(ctx, llm.EmbedRequest{Input: []string{query}}); err == nil && len(resp.Embeddings) > 0 {
				hits, err := s.vec.Search(resp.Embeddings[0], topK)
				if err == nil {
					for _, h := range hits {
						// 避免与关键词结果重复
						dup := false
						for _, r := range results {
							if r.ID == h.ID {
								dup = true
								break
							}
						}
						if dup {
							continue
						}
						var r Result
						if err := s.db.QueryRow(`SELECT id, uri, chunk_idx, content FROM vfs_multimodal_units WHERE id=?`, h.ID).
							Scan(&r.ID, &r.URI, &r.ChunkIdx, &r.Content); err == nil {
							r.Score = float64(h.Score)
							results = append(results, r)
						}
					}
				}
			}
		}
	}
	// 按分数降序
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

// Delete 删除资源的索引。
func (s *Service) Delete(uri string) error {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.Query(`SELECT id FROM vfs_multimodal_units WHERE uri=?`, uri)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		s.vec.Delete(id)
	}
	_, err = s.db.Exec(`DELETE FROM vfs_multimodal_units WHERE uri=?`, uri)
	return err
}

// Stats 索引统计。
func (s *Service) Stats() (*Stats, error) {
	if s.db == nil {
		return &Stats{}, nil
	}
	st := &Stats{}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM vfs_multimodal_units`).Scan(&st.TotalUnits); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(DISTINCT uri) FROM vfs_multimodal_units`).Scan(&st.IndexedURIs); err != nil {
		return nil, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM vfs_multimodal_units WHERE embedding IS NOT NULL`).Scan(&st.EmbeddedUnits); err != nil {
		return nil, err
	}
	st.VectorDim = s.dim
	return st, nil
}

// Result 检索结果。
type Result struct {
	ID       string  `json:"id"`
	URI      string  `json:"uri"`
	ChunkIdx int     `json:"chunkIdx"`
	Content  string  `json:"content"`
	Score    float64 `json:"score"`
}

// chunkText 按约 n 字符切块（保留边界）。
func chunkText(text string, n int) []string {
	if n <= 0 {
		n = 800
	}
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}
	var out []string
	for i := 0; i < len(runes); i += n {
		end := i + n
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[i:end]))
	}
	return out
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

var _ = json.Marshal
