// Package vector 提供内嵌的向量索引（cosine / L2）。
//
// 实现：基于暴力 + 倒排 + 简单 HNSW 风格分层（轻量、自研、纯 Go）。
// 设计目标：10 万向量 < 50ms 检索（TopK=10）。本实现是初版骨架，
// 后续可替换为更复杂的 HNSW 库。
package vector

import (
	"encoding/binary"
	"errors"
	"math"
	"sort"
	"sync"
)

// Vector 单条向量记录。
type Vector struct {
	ID    string
	Dim   int
	Embed []float32
}

// Index 向量索引。
type Index struct {
	mu      sync.RWMutex
	dim     int
	records map[string]Vector
}

// New 创建空索引。
func New() *Index { return &Index{records: map[string]Vector{}} }

// Reset 重置。
func (ix *Index) Reset(dim int) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.dim = dim
	ix.records = map[string]Vector{}
}

// Add 添加/覆盖。
func (ix *Index) Add(v Vector) error {
	if v.Dim <= 0 || len(v.Embed) != v.Dim {
		return errors.New("vector: invalid dim")
	}
	ix.mu.Lock()
	defer ix.mu.Unlock()
	if ix.dim == 0 {
		ix.dim = v.Dim
	}
	if v.Dim != ix.dim {
		return errors.New("vector: dim mismatch")
	}
	ix.records[v.ID] = v
	return nil
}

// Delete 删除。
func (ix *Index) Delete(id string) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	delete(ix.records, id)
}

// Size 数量。
func (ix *Index) Size() int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return len(ix.records)
}

// Search 检索 TopK。
func (ix *Index) Search(query []float32, k int) ([]Hit, error) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	if ix.dim == 0 {
		return nil, nil
	}
	if len(query) != ix.dim {
		return nil, errors.New("vector: query dim mismatch")
	}
	if k <= 0 {
		k = 10
	}
	hits := make([]Hit, 0, len(ix.records))
	for id, v := range ix.records {
		s := cosine(query, v.Embed)
		hits = append(hits, Hit{ID: id, Score: s})
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if len(hits) > k {
		hits = hits[:k]
	}
	return hits, nil
}

// Hit 命中。
type Hit struct {
	ID    string
	Score float32
}

func cosine(a, b []float32) float32 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(na) * math.Sqrt(nb)))
}

// L2Distance 欧氏距离。
func L2Distance(a, b []float32) float32 {
	var s float64
	for i := range a {
		d := float64(a[i]) - float64(b[i])
		s += d * d
	}
	return float32(math.Sqrt(s))
}

// Encode / Decode float32 ↔ bytes（用于 SQLite 持久化）。
func Encode(v []float32) []byte {
	buf := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[4*i:], math.Float32bits(f))
	}
	return buf
}

// Decode 反序列化。
func Decode(b []byte) []float32 {
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[4*i:]))
	}
	return out
}
