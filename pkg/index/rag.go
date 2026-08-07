// RAG 检索模块，提供上下文构造、余弦相似度和重排序能力。
//
// 参考 Rust lance_vector_store.rs 的向量检索和 RAG 上下文构造逻辑。

package index

import (
	"context"
	"fmt"
	"math"
	"strings"
)

// CosineSimilarity 计算两个向量的余弦相似度。
//
// 处理零向量：任一向量为零时返回 0。
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		af := float64(a[i])
		bf := float64(b[i])
		dot += af * bf
		na += af * af
		nb += bf * bf
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(na) * math.Sqrt(nb)))
}

// BuildRAGContext 从搜索结果构造给 LLM 的上下文字符串。
//
// 格式：
//
//	[1] (uri) content...
//	[2] (uri) content...
//
// 截取每个片段的最大长度为 RAGChunkMaxChars。
const RAGChunkMaxChars = 800

func BuildRAGContext(results []SearchResult) string {
	if len(results) == 0 {
		return ""
	}

	var b strings.Builder
	for i, r := range results {
		snippet := r.Snippet
		if snippet == "" {
			snippet = r.URI
		}
		// 截取前 RAGChunkMaxChars 字符
		runes := []rune(snippet)
		if len(runes) > RAGChunkMaxChars {
			snippet = string(runes[:RAGChunkMaxChars]) + "..."
		}
		fmt.Fprintf(&b, "[%d] (%s) %s\n\n", i+1, r.URI, snippet)
	}
	return strings.TrimSpace(b.String())
}

// Rerank 对候选结果进行重排序。
//
// 简单重排序：用查询向量与候选向量计算余弦相似度，重新排序取 topK。
// 可后续扩展为 LLM rerank（调用 LLM 对候选对查询的相关性打分）。
func Rerank(ctx context.Context, queryVec []float32, candidates []Chunk, topK int) ([]SearchResult, error) {
	if len(candidates) == 0 {
		return nil, nil
	}
	if topK <= 0 {
		topK = 10
	}

	type scored struct {
		chunk Chunk
		score float32
	}

	scoredList := make([]scored, 0, len(candidates))
	for _, c := range candidates {
		s := CosineSimilarity(queryVec, c.Embedding)
		scoredList = append(scoredList, scored{chunk: c, score: s})
	}

	// 按得分降序排序
	for i := 0; i < len(scoredList); i++ {
		for j := i + 1; j < len(scoredList); j++ {
			if scoredList[j].score > scoredList[i].score {
				scoredList[i], scoredList[j] = scoredList[j], scoredList[i]
			}
		}
	}

	// 取 topK
	if topK > len(scoredList) {
		topK = len(scoredList)
	}

	results := make([]SearchResult, 0, topK)
	for i := 0; i < topK; i++ {
		s := scoredList[i]
		// 余弦相似度映射到 [0,1]：(cos + 1) / 2
		normalizedScore := float64(s.score+1.0) / 2.0
		results = append(results, SearchResult{
			URI:      s.chunk.URI,
			Snippet:  s.chunk.Content,
			Score:    normalizedScore,
			VecScore: normalizedScore,
			ChunkPos: s.chunk.Pos,
		})
	}

	return results, nil
}

// VectorSearch 向量检索：在候选 chunk 中按查询向量搜索 topK。
func VectorSearch(queryVec []float32, candidates []Chunk, topK int) []SearchResult {
	if len(candidates) == 0 || len(queryVec) == 0 {
		return nil
	}
	if topK <= 0 {
		topK = 10
	}

	type scored struct {
		chunk Chunk
		score float32
	}

	scoredList := make([]scored, 0, len(candidates))
	for _, c := range candidates {
		s := CosineSimilarity(queryVec, c.Embedding)
		scoredList = append(scoredList, scored{chunk: c, score: s})
	}

	// 按得分降序排序
	for i := 0; i < len(scoredList); i++ {
		for j := i + 1; j < len(scoredList); j++ {
			if scoredList[j].score > scoredList[i].score {
				scoredList[i], scoredList[j] = scoredList[j], scoredList[i]
			}
		}
	}

	if topK > len(scoredList) {
		topK = len(scoredList)
	}

	results := make([]SearchResult, 0, topK)
	for i := 0; i < topK; i++ {
		s := scoredList[i]
		normalizedScore := float64(s.score+1.0) / 2.0
		results = append(results, SearchResult{
			URI:      s.chunk.URI,
			Snippet:  s.chunk.Content,
			Score:    normalizedScore,
			VecScore: normalizedScore,
			ChunkPos: s.chunk.Pos,
		})
	}

	return results
}

// MergeResults 融合 FTS 和向量搜索结果。
//
// 按 0.5*fts_norm + 0.5*vec_norm 融合得分。
func MergeResults(ftsResults, vecResults []SearchResult, limit int) []SearchResult {
	if limit <= 0 {
		limit = 10
	}

	// 按 URI 去重合并
	merged := map[string]*SearchResult{}
	for _, r := range ftsResults {
		key := r.URI
		if existing, ok := merged[key]; ok {
			existing.FTSScore = r.FTSScore
			if r.Snippet != "" {
				existing.Snippet = r.Snippet
			}
		} else {
			merged[key] = &SearchResult{
				URI:      r.URI,
				Type:     r.Type,
				Title:    r.Title,
				Snippet:  r.Snippet,
				FTSScore: r.FTSScore,
			}
		}
	}

	for _, r := range vecResults {
		key := r.URI
		if existing, ok := merged[key]; ok {
			existing.VecScore = r.VecScore
			if existing.Snippet == "" {
				existing.Snippet = r.Snippet
			}
			existing.ChunkPos = r.ChunkPos
		} else {
			merged[key] = &SearchResult{
				URI:      r.URI,
				Type:     r.Type,
				Title:    r.Title,
				Snippet:  r.Snippet,
				VecScore: r.VecScore,
				ChunkPos: r.ChunkPos,
			}
		}
	}

	// 计算综合得分
	list := make([]SearchResult, 0, len(merged))
	for _, r := range merged {
		r.Score = 0.5*r.FTSScore + 0.5*r.VecScore
		list = append(list, *r)
	}

	// 按综合得分降序排序
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].Score > list[i].Score {
				list[i], list[j] = list[j], list[i]
			}
		}
	}

	if len(list) > limit {
		list = list[:limit]
	}

	return list
}
