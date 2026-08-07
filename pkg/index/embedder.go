// 向量嵌入服务，调用 llm.Provider Embed 接口生成向量。
//
// 参考 Rust embedding_service.rs：
// - 批量嵌入（默认 32 条一批）
// - 嵌入结果缓存到 store.Chunks 表的 embedding BLOB 字段
// - BLOB 编码：每个 float32 4 字节，little-endian（复用 vector.Encode/Decode）

package index

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/helixnow/deep-student-go/pkg/llm"
)

// EmbedBatchSize 默认批量嵌入大小。
const EmbedBatchSize = 32

// Embedder 向量嵌入器，持有 llm.Provider 引用。
type Embedder struct {
	provider llm.Provider
}

// NewEmbedder 创建嵌入器。
func NewEmbedder(provider llm.Provider) *Embedder {
	return &Embedder{provider: provider}
}

// EmbedChunks 批量嵌入文本块。
//
// 按 EmbedBatchSize 条一批调用 llm.Embed，返回与输入顺序对应的嵌入向量列表。
// model 为嵌入模型 ID（如 "text-embedding-3-small"）。
func (e *Embedder) EmbedChunks(ctx context.Context, chunks []Chunk, model string) ([][]float32, error) {
	if e == nil || e.provider == nil {
		return nil, fmt.Errorf("index: embedder not configured")
	}
	if len(chunks) == 0 {
		return nil, nil
	}

	// 提取文本
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Content
	}

	// 分批嵌入
	var allEmbeddings [][]float32
	for start := 0; start < len(texts); start += EmbedBatchSize {
		end := start + EmbedBatchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[start:end]

		resp, err := e.provider.Embed(ctx, llm.EmbedRequest{
			Model: model,
			Input: batch,
		})
		if err != nil {
			return nil, fmt.Errorf("index: embed batch %d-%d: %w", start, end, err)
		}
		if len(resp.Embeddings) != len(batch) {
			return nil, fmt.Errorf("index: embedding count mismatch: expected %d, got %d", len(batch), len(resp.Embeddings))
		}
		allEmbeddings = append(allEmbeddings, resp.Embeddings...)
	}

	return allEmbeddings, nil
}

// EmbedSingle 嵌入单条文本。
func (e *Embedder) EmbedSingle(ctx context.Context, text, model string) ([]float32, error) {
	if e == nil || e.provider == nil {
		return nil, fmt.Errorf("index: embedder not configured")
	}
	resp, err := e.provider.Embed(ctx, llm.EmbedRequest{
		Model: model,
		Input: []string{text},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("index: empty embedding response")
	}
	return resp.Embeddings[0], nil
}

// SaveEmbeddings 将嵌入向量写入 chunks 表的 embedding BLOB 字段。
//
// 按 chunk ID 更新对应的 embedding。
func SaveEmbeddings(db *sql.DB, chunkIDs []int64, embeddings [][]float32) error {
	if db == nil {
		return fmt.Errorf("index: db is nil")
	}
	if len(chunkIDs) != len(embeddings) {
		return fmt.Errorf("index: id/embedding count mismatch: %d vs %d", len(chunkIDs), len(embeddings))
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`UPDATE chunks SET embedding = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, id := range chunkIDs {
		blob := EncodeFloat32s(embeddings[i])
		if _, err := stmt.Exec(blob, id); err != nil {
			return fmt.Errorf("index: save embedding for chunk %d: %w", id, err)
		}
	}

	return tx.Commit()
}

// LoadEmbeddings 从 chunks 表加载所有带嵌入的 chunk。
//
// 返回 chunk 元数据 + 解码后的嵌入向量。
func LoadEmbeddings(db *sql.DB) ([]Chunk, error) {
	if db == nil {
		return nil, fmt.Errorf("index: db is nil")
	}
	rows, err := db.Query(`SELECT id, uri, pos, content, token_count, embedding FROM chunks WHERE embedding IS NOT NULL ORDER BY uri, pos`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Chunk
	for rows.Next() {
		var c Chunk
		var blob []byte
		if err := rows.Scan(&c.ID, &c.URI, &c.Pos, &c.Content, &c.TokenCount, &blob); err != nil {
			return nil, err
		}
		c.Embedding = DecodeFloat32s(blob)
		result = append(result, c)
	}
	return result, rows.Err()
}

// LoadEmbeddingsByURI 从 chunks 表加载指定 URI 的带嵌入 chunk。
func LoadEmbeddingsByURI(db *sql.DB, uri string) ([]Chunk, error) {
	if db == nil {
		return nil, fmt.Errorf("index: db is nil")
	}
	rows, err := db.Query(`SELECT id, uri, pos, content, token_count, embedding FROM chunks WHERE uri = ? AND embedding IS NOT NULL ORDER BY pos`, uri)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Chunk
	for rows.Next() {
		var c Chunk
		var blob []byte
		if err := rows.Scan(&c.ID, &c.URI, &c.Pos, &c.Content, &c.TokenCount, &blob); err != nil {
			return nil, err
		}
		c.Embedding = DecodeFloat32s(blob)
		result = append(result, c)
	}
	return result, rows.Err()
}

// EncodeFloat32s 将 []float32 编码为 []byte（little-endian，每个 float32 4 字节）。
func EncodeFloat32s(v []float32) []byte {
	buf := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[4*i:], math.Float32bits(f))
	}
	return buf
}

// DecodeFloat32s 将 []byte 解码为 []float32。
func DecodeFloat32s(b []byte) []float32 {
	if len(b) == 0 {
		return nil
	}
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[4*i:]))
	}
	return out
}
