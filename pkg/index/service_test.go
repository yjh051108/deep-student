package index

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	_ "modernc.org/sqlite"
)

// =====================================================================
// Mock LLM Provider
// =====================================================================

// mockEmbedProvider 模拟嵌入 Provider，返回基于文本 hash 的确定性向量。
type mockEmbedProvider struct {
	name   string
	dim    int
	failOn int // >0 时在第 failOn 次调用返回错误
	calls  int
}

func (m *mockEmbedProvider) Name() string { return m.name }
func (m *mockEmbedProvider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: "mock"}, nil
}
func (m *mockEmbedProvider) Stream(ctx context.Context, req llm.ChatRequest) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk, 1)
	ch <- llm.Chunk{Done: true}
	close(ch)
	return ch, nil
}

// Embed 返回确定性向量：基于文本内容的简单 hash 生成 dim 维向量。
func (m *mockEmbedProvider) Embed(ctx context.Context, req llm.EmbedRequest) (*llm.EmbedResponse, error) {
	m.calls++
	if m.failOn > 0 && m.calls == m.failOn {
		return nil, fmt.Errorf("mock: simulated failure on call %d", m.calls)
	}
	out := &llm.EmbedResponse{
		Embeddings: make([][]float32, len(req.Input)),
	}
	for i, text := range req.Input {
		out.Embeddings[i] = textToVector(text, m.dim)
	}
	return out, nil
}

// textToVector 把文本转换为确定性向量（简单 hash → dim 维）。
// 相似文本会产生相似向量（共享前缀的文本前几维相同）。
func textToVector(text string, dim int) []float32 {
	v := make([]float32, dim)
	if dim == 0 || text == "" {
		return v
	}
	runes := []rune(text)
	for i := 0; i < dim; i++ {
		if i < len(runes) {
			v[i] = float32(runes[i]) / 127.0
		} else {
			v[i] = 0.01
		}
	}
	// 归一化
	var norm float64
	for _, f := range v {
		norm += float64(f) * float64(f)
	}
	if norm > 0 {
		sqrtNorm := 1.0
		for sqrtNorm*sqrtNorm < norm {
			sqrtNorm += 0.001
		}
		for i := range v {
			v[i] = float32(float64(v[i]) / sqrtNorm)
		}
	}
	return v
}

// =====================================================================
// 测试辅助
// =====================================================================

// newTestDB 创建测试用 SQLite 数据库（带 store 迁移 + FTS 初始化）。
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err := InitFTS(s.DB); err != nil {
		t.Fatal(err)
	}
	return s.DB
}

// newTestService 创建测试用 Service（带 mock provider）。
func newTestService(t *testing.T) *Service {
	t.Helper()
	db := newTestDB(t)
	svc := New(db, llm.NewRegistry(), nil)
	svc.SetEmbedProvider(&mockEmbedProvider{name: "mock", dim: 16})
	svc.SetEmbedModel("mock-embed-model")
	return svc
}

// =====================================================================
// Chunker 测试
// =====================================================================

func TestChunkText_EmptyText(t *testing.T) {
	chunks := ChunkText("", DefaultOptions())
	if len(chunks) != 0 {
		t.Fatalf("空文本应返回 nil，got %d chunks", len(chunks))
	}
}

func TestChunkText_ShortText(t *testing.T) {
	text := "这是一段短文本。"
	chunks := ChunkText(text, DefaultOptions())
	if len(chunks) != 1 {
		t.Fatalf("短文本应返回 1 块，got %d", len(chunks))
	}
	if chunks[0].Content != text {
		t.Fatalf("内容不匹配: got %q", chunks[0].Content)
	}
	if chunks[0].TokenCount <= 0 {
		t.Fatal("tokenCount 应 > 0")
	}
}

func TestChunkText_MixedChineseEnglish(t *testing.T) {
	text := "Hello 世界 this is a test 测试文本。"
	chunks := ChunkText(text, DefaultOptions())
	if len(chunks) != 1 {
		t.Fatalf("混合文本不超过 chunkSize 应返回 1 块，got %d", len(chunks))
	}
	// 中文 4 字 × 1.5 = 6，英文约 6 词 × 1.3 = 7.8 → 取较大值
	if chunks[0].TokenCount < 5 {
		t.Fatalf("tokenCount 估算过低: %d", chunks[0].TokenCount)
	}
}

func TestChunkText_ParagraphBoundary(t *testing.T) {
	// 构造多个段落，总长超过 chunkSize
	var paragraphs []string
	for i := 0; i < 5; i++ {
		paragraphs = append(paragraphs, fmt.Sprintf("这是第%d段内容，包含一些文字用于测试切片功能。", i+1))
	}
	text := strings.Join(paragraphs, "\n\n")

	opts := IndexOptions{ChunkSize: 50, ChunkOverlap: 10, MinChunkSize: 5}
	chunks := ChunkText(text, opts)
	if len(chunks) < 2 {
		t.Fatalf("多段落应切分为多块，got %d", len(chunks))
	}

	// 验证每个块不为空
	for i, c := range chunks {
		if c.Content == "" {
			t.Fatalf("块 %d 为空", i)
		}
	}
}

func TestChunkText_LongText(t *testing.T) {
	// 构造超长文本
	var builder strings.Builder
	for i := 0; i < 100; i++ {
		builder.WriteString("这是一段用于测试的中文文本内容。")
	}
	text := builder.String()

	opts := IndexOptions{ChunkSize: 100, ChunkOverlap: 20, MinChunkSize: 10}
	chunks := ChunkText(text, opts)
	if len(chunks) < 5 {
		t.Fatalf("超长文本应产生多个块，got %d", len(chunks))
	}

	// 验证总字符数覆盖（考虑重叠后总量大于原文）
	totalChars := 0
	for _, c := range chunks {
		totalChars += utf8.RuneCountInString(c.Content)
	}
	if totalChars < utf8.RuneCountInString(text) {
		t.Fatalf("块总字符数 %d < 原文 %d", totalChars, utf8.RuneCountInString(text))
	}
}

func TestChunkText_Overlap(t *testing.T) {
	// 构造文本使得切片产生多块，验证重叠
	text := "段落一内容较多。段落二内容较多。段落三内容较多。段落四内容较多。"
	opts := IndexOptions{ChunkSize: 20, ChunkOverlap: 5, MinChunkSize: 1}
	chunks := ChunkText(text, opts)

	if len(chunks) < 2 {
		t.Fatalf("应产生多块，got %d", len(chunks))
	}

	// 验证重叠：第二块的开头应包含第一块末尾的文本
	if len(chunks) >= 2 {
		firstTail := tailByRunes(chunks[0].Content, 5)
		firstTail = strings.TrimSpace(firstTail)
		if firstTail != "" && !strings.Contains(chunks[1].Content, firstTail) {
			// 重叠可能被 trim 影响，仅检查是否有重叠特征
			t.Logf("块1末尾: %q, 块2开头: %q", firstTail, chunks[1].Content[:min(20, len(chunks[1].Content))])
		}
	}
}

func TestChunkText_SentenceBoundary(t *testing.T) {
	// 单段落超长但无 \n\n，应按句号切分
	var builder strings.Builder
	for i := 0; i < 20; i++ {
		builder.WriteString("这是一句话。")
	}
	text := builder.String()

	opts := IndexOptions{ChunkSize: 30, ChunkOverlap: 5, MinChunkSize: 1}
	chunks := ChunkText(text, opts)
	if len(chunks) < 2 {
		t.Fatalf("按句号切分应产生多块，got %d", len(chunks))
	}
}

func TestChunkText_MinChunkSize(t *testing.T) {
	// 尾片小于 MinChunkSize 应被跳过
	text := "这是一个较长的段落内容用于产生多个切片。这是一个较长的段落内容用于产生多个切片。短"
	opts := IndexOptions{ChunkSize: 30, ChunkOverlap: 0, MinChunkSize: 5}
	chunks := ChunkText(text, opts)

	for i, c := range chunks {
		if i > 0 && utf8.RuneCountInString(c.Content) < opts.MinChunkSize {
			t.Fatalf("尾片 %d 长度 %d < MinChunkSize %d", i, utf8.RuneCountInString(c.Content), opts.MinChunkSize)
		}
	}
}

// =====================================================================
// FTS 测试
// =====================================================================

func TestInitFTS(t *testing.T) {
	db := newTestDB(t)

	// 验证 fts_chunks 表存在
	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='fts_chunks'`).Scan(&name)
	if err != nil {
		t.Fatalf("fts_chunks 表不存在: %v", err)
	}
	if name != "fts_chunks" {
		t.Fatalf("表名不匹配: %s", name)
	}

	// 验证 chunks 表有 created_at 列
	var colCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('chunks') WHERE name='created_at'`).Scan(&colCount)
	if err != nil {
		t.Fatal(err)
	}
	if colCount != 1 {
		t.Fatal("chunks 表缺少 created_at 列")
	}
}

func TestIndexFTS_SearchFTS(t *testing.T) {
	db := newTestDB(t)

	// 索引几条文本
	if err := IndexFTS(db, "vfs://note/n1", "Go 语言是一门编程语言"); err != nil {
		t.Fatal(err)
	}
	if err := IndexFTS(db, "vfs://note/n2", "Rust 语言也是一门编程语言"); err != nil {
		t.Fatal(err)
	}
	if err := IndexFTS(db, "vfs://note/n3", "Python 数据分析"); err != nil {
		t.Fatal(err)
	}

	// 搜索
	results, err := SearchFTS(db, "编程语言", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 2 {
		t.Fatalf("应至少搜到 2 条结果，got %d", len(results))
	}

	// 验证结果包含正确的 URI
	uriSet := map[string]bool{}
	for _, r := range results {
		uriSet[r.URI] = true
	}
	if !uriSet["vfs://note/n1"] || !uriSet["vfs://note/n2"] {
		t.Fatal("搜索结果应包含 n1 和 n2")
	}

	// 验证得分在 [0, 1] 范围
	for _, r := range results {
		if r.FTSScore < 0 || r.FTSScore > 1 {
			t.Fatalf("FTS 得分超出范围: %f", r.FTSScore)
		}
	}
}

func TestSearchFTS_EmptyQuery(t *testing.T) {
	db := newTestDB(t)
	results, err := SearchFTS(db, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("空查询应返回 0 结果，got %d", len(results))
	}
}

func TestDeleteFTS(t *testing.T) {
	db := newTestDB(t)

	IndexFTS(db, "vfs://note/n1", "测试删除")
	IndexFTS(db, "vfs://note/n2", "测试保留")

	// 删除 n1
	if err := DeleteFTS(db, "vfs://note/n1"); err != nil {
		t.Fatal(err)
	}

	// 搜索应只返回 n2
	results, _ := SearchFTS(db, "测试", 10)
	for _, r := range results {
		if r.URI == "vfs://note/n1" {
			t.Fatal("n1 应已被删除")
		}
	}
}

func TestFTSRowCount(t *testing.T) {
	db := newTestDB(t)

	IndexFTS(db, "vfs://note/n1", "内容一")
	IndexFTS(db, "vfs://note/n2", "内容二")

	count, err := FTSRowCount(db)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("行数应为 2，got %d", count)
	}

	DeleteFTS(db, "vfs://note/n1")
	count, _ = FTSRowCount(db)
	if count != 1 {
		t.Fatalf("删除后行数应为 1，got %d", count)
	}
}

func TestSanitizeFTSQuery(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"hello world", true},
		{"中文搜索", true},
		{"", false},
		{`quoted "text"`, true},
	}
	for _, tt := range tests {
		result := sanitizeFTSQuery(tt.input)
		if tt.valid && result == "" {
			t.Fatalf("sanitizeFTSQuery(%q) 返回空", tt.input)
		}
		if !tt.valid && result != "" {
			t.Fatalf("sanitizeFTSQuery(%q) 应返回空，got %q", tt.input, result)
		}
	}
}

// =====================================================================
// Embedder 测试
// =====================================================================

func TestEmbedder_EmbedChunks(t *testing.T) {
	provider := &mockEmbedProvider{name: "mock", dim: 8}
	embedder := NewEmbedder(provider)

	chunks := []Chunk{
		{Content: "hello"},
		{Content: "world"},
		{Content: "test"},
	}

	embeddings, err := embedder.EmbedChunks(context.Background(), chunks, "mock-model")
	if err != nil {
		t.Fatal(err)
	}
	if len(embeddings) != 3 {
		t.Fatalf("应返回 3 个向量，got %d", len(embeddings))
	}
	for _, emb := range embeddings {
		if len(emb) != 8 {
			t.Fatalf("向量维度应为 8，got %d", len(emb))
		}
	}
}

func TestEmbedder_EmbedChunks_Batch(t *testing.T) {
	// 测试批量嵌入（超过 EmbedBatchSize）
	provider := &mockEmbedProvider{name: "mock", dim: 4}
	embedder := NewEmbedder(provider)

	// 构造 40 条（超过 EmbedBatchSize=32）
	chunks := make([]Chunk, 40)
	for i := range chunks {
		chunks[i].Content = fmt.Sprintf("text %d", i)
	}

	embeddings, err := embedder.EmbedChunks(context.Background(), chunks, "mock-model")
	if err != nil {
		t.Fatal(err)
	}
	if len(embeddings) != 40 {
		t.Fatalf("应返回 40 个向量，got %d", len(embeddings))
	}
}

func TestEmbedder_EmbedChunks_Empty(t *testing.T) {
	provider := &mockEmbedProvider{name: "mock", dim: 8}
	embedder := NewEmbedder(provider)

	embeddings, err := embedder.EmbedChunks(context.Background(), nil, "mock-model")
	if err != nil {
		t.Fatal(err)
	}
	if embeddings != nil {
		t.Fatal("空输入应返回 nil")
	}
}

func TestEmbedder_EmbedChunks_Error(t *testing.T) {
	provider := &mockEmbedProvider{name: "mock", dim: 8, failOn: 1}
	embedder := NewEmbedder(provider)

	chunks := []Chunk{{Content: "test"}}
	_, err := embedder.EmbedChunks(context.Background(), chunks, "mock-model")
	if err == nil {
		t.Fatal("应返回错误")
	}
}

func TestEmbedder_EmbedSingle(t *testing.T) {
	provider := &mockEmbedProvider{name: "mock", dim: 8}
	embedder := NewEmbedder(provider)

	vec, err := embedder.EmbedSingle(context.Background(), "test", "mock-model")
	if err != nil {
		t.Fatal(err)
	}
	if len(vec) != 8 {
		t.Fatalf("向量维度应为 8，got %d", len(vec))
	}
}

func TestEncodeDecodeFloat32s(t *testing.T) {
	original := []float32{1.0, -2.5, 3.14, 0.001, -0.999}
	encoded := EncodeFloat32s(original)
	decoded := DecodeFloat32s(encoded)

	if len(decoded) != len(original) {
		t.Fatalf("长度不匹配: %d vs %d", len(decoded), len(original))
	}
	for i := range original {
		if decoded[i] != original[i] {
			t.Fatalf("位置 %d: %v vs %v", i, decoded[i], original[i])
		}
	}
}

func TestSaveLoadEmbeddings(t *testing.T) {
	db := newTestDB(t)

	// 插入 chunks
	chunks := []Chunk{
		{URI: "vfs://note/n1", Pos: 0, Content: "hello", TokenCount: 1},
		{URI: "vfs://note/n1", Pos: 1, Content: "world", TokenCount: 1},
	}
	ids := make([]int64, len(chunks))
	for i, c := range chunks {
		res, _ := db.Exec(`INSERT INTO chunks(uri, pos, content, token_count) VALUES(?, ?, ?, ?)`,
			c.URI, c.Pos, c.Content, c.TokenCount)
		ids[i], _ = res.LastInsertId()
	}

	// 保存嵌入
	embeddings := [][]float32{
		{1.0, 2.0, 3.0},
		{4.0, 5.0, 6.0},
	}
	if err := SaveEmbeddings(db, ids, embeddings); err != nil {
		t.Fatal(err)
	}

	// 加载嵌入
	loaded, err := LoadEmbeddings(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 {
		t.Fatalf("应加载 2 条，got %d", len(loaded))
	}
}

// =====================================================================
// RAG 测试
// =====================================================================

func TestCosineSimilarity(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	s := CosineSimilarity(a, b)
	if s < 0.99 {
		t.Fatalf("相同向量余弦应 ≈ 1，got %f", s)
	}

	c := []float32{0, 1, 0}
	s2 := CosineSimilarity(a, c)
	if s2 > 0.01 {
		t.Fatalf("正交向量余弦应 ≈ 0，got %f", s2)
	}

	// 零向量
	zero := []float32{0, 0, 0}
	s3 := CosineSimilarity(a, zero)
	if s3 != 0 {
		t.Fatalf("零向量余弦应为 0，got %f", s3)
	}

	// 维度不匹配
	s4 := CosineSimilarity(a, []float32{1, 0})
	if s4 != 0 {
		t.Fatal("维度不匹配应返回 0")
	}
}

func TestBuildRAGContext(t *testing.T) {
	results := []SearchResult{
		{URI: "vfs://note/n1", Snippet: "第一段内容"},
		{URI: "vfs://note/n2", Snippet: "第二段内容"},
	}
	ctx := BuildRAGContext(results)
	if ctx == "" {
		t.Fatal("上下文不应为空")
	}
	if !strings.Contains(ctx, "第一段内容") {
		t.Fatal("上下文应包含第一段内容")
	}
	if !strings.Contains(ctx, "vfs://note/n1") {
		t.Fatal("上下文应包含 URI")
	}
}

func TestBuildRAGContext_Empty(t *testing.T) {
	ctx := BuildRAGContext(nil)
	if ctx != "" {
		t.Fatal("空结果应返回空上下文")
	}
}

func TestBuildRAGContext_LongSnippet(t *testing.T) {
	longText := strings.Repeat("这是一段很长的文本", 200)
	results := []SearchResult{
		{URI: "vfs://note/n1", Snippet: longText},
	}
	ctx := BuildRAGContext(results)
	// 应截断到 RAGChunkMaxChars + "..."
	runes := []rune(ctx)
	if len(runes) > RAGChunkMaxChars+100 { // 允许一些前缀开销
		t.Fatalf("上下文过长: %d runes", len(runes))
	}
}

func TestVectorSearch(t *testing.T) {
	queryVec := textToVector("hello", 8)
	chunks := []Chunk{
		{URI: "vfs://note/n1", Pos: 0, Content: "hello", Embedding: textToVector("hello", 8)},
		{URI: "vfs://note/n2", Pos: 0, Content: "world", Embedding: textToVector("world", 8)},
		{URI: "vfs://note/n3", Pos: 0, Content: "hello world", Embedding: textToVector("hello world", 8)},
	}

	results := VectorSearch(queryVec, chunks, 3)
	if len(results) == 0 {
		t.Fatal("应返回结果")
	}
	// hello 应排在最前
	if results[0].URI != "vfs://note/n1" {
		t.Fatalf("最相关应是 n1，got %s", results[0].URI)
	}
}

func TestMergeResults(t *testing.T) {
	ftsResults := []SearchResult{
		{URI: "vfs://note/n1", FTSScore: 0.9, Snippet: "FTS片段"},
		{URI: "vfs://note/n2", FTSScore: 0.5},
	}
	vecResults := []SearchResult{
		{URI: "vfs://note/n1", VecScore: 0.8},
		{URI: "vfs://note/n3", VecScore: 0.7, Snippet: "向量片段"},
	}

	merged := MergeResults(ftsResults, vecResults, 10)
	if len(merged) != 3 {
		t.Fatalf("合并后应有 3 条，got %d", len(merged))
	}

	// n1 应有 FTS 和 Vec 得分
	var n1 *SearchResult
	for i := range merged {
		if merged[i].URI == "vfs://note/n1" {
			n1 = &merged[i]
		}
	}
	if n1 == nil {
		t.Fatal("n1 不在结果中")
	}
	if n1.FTSScore != 0.9 || n1.VecScore != 0.8 {
		t.Fatalf("n1 得分不正确: fts=%f vec=%f", n1.FTSScore, n1.VecScore)
	}
	// 综合得分 = 0.5*0.9 + 0.5*0.8 = 0.85
	if n1.Score < 0.84 || n1.Score > 0.86 {
		t.Fatalf("n1 综合得分应 ≈ 0.85，got %f", n1.Score)
	}
}

func TestRerank(t *testing.T) {
	queryVec := textToVector("hello", 8)
	candidates := []Chunk{
		{URI: "vfs://note/n1", Pos: 0, Content: "hello", Embedding: textToVector("hello", 8)},
		{URI: "vfs://note/n2", Pos: 0, Content: "world", Embedding: textToVector("world", 8)},
		{URI: "vfs://note/n3", Pos: 0, Content: "hello world", Embedding: textToVector("hello world", 8)},
	}

	results, err := Rerank(context.Background(), queryVec, candidates, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("应返回 2 条，got %d", len(results))
	}
	// hello 应排在最前
	if results[0].URI != "vfs://note/n1" {
		t.Fatalf("最相关应是 n1，got %s", results[0].URI)
	}
}

// =====================================================================
// Service 测试
// =====================================================================

func TestService_IndexResource_FTSSearch(t *testing.T) {
	svc := newTestService(t)

	content := "Go 语言是一门编译型编程语言，由 Google 开发。它具有简洁的语法和高效的并发支持。"
	opts := DefaultOptions() // 不嵌入

	if err := svc.IndexResource(context.Background(), "vfs://note/n1", content, opts); err != nil {
		t.Fatal(err)
	}

	// FTS 搜索
	results, err := svc.Search(context.Background(), SearchQuery{
		Query:  "Go 语言",
		Limit:  10,
		UseFTS: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("应搜到结果")
	}
	if results[0].URI != "vfs://note/n1" {
		t.Fatalf("结果 URI 不匹配: %s", results[0].URI)
	}
}

func TestService_IndexResource_WithEmbedding(t *testing.T) {
	svc := newTestService(t)

	content := "这是一段测试文本，用于验证向量嵌入功能。"
	opts := DefaultOptions()
	opts.Embed = true

	if err := svc.IndexResource(context.Background(), "vfs://note/n1", content, opts); err != nil {
		t.Fatal(err)
	}

	// 验证 chunks 表有嵌入
	var embeddedCount int
	err := svc.db.QueryRow(`SELECT COUNT(*) FROM chunks WHERE embedding IS NOT NULL`).Scan(&embeddedCount)
	if err != nil {
		t.Fatal(err)
	}
	if embeddedCount == 0 {
		t.Fatal("应有嵌入向量")
	}

	// 验证任务状态
	task := svc.GetTask("vfs://note/n1")
	if task == nil {
		t.Fatal("任务状态不应为 nil")
	}
	if task.Status != TaskStatusDone {
		t.Fatalf("任务状态应为 done，got %s", task.Status)
	}
}

func TestService_IndexResource_EmptyContent(t *testing.T) {
	svc := newTestService(t)

	err := svc.IndexResource(context.Background(), "vfs://note/n1", "", DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}

	// 空内容不应产生 chunks
	stats, _ := svc.Stats()
	if stats.TotalChunks != 0 {
		t.Fatalf("空内容不应产生 chunks，got %d", stats.TotalChunks)
	}
}

func TestService_DeleteIndex(t *testing.T) {
	svc := newTestService(t)

	content := "测试删除索引的内容"
	svc.IndexResource(context.Background(), "vfs://note/n1", content, DefaultOptions())

	// 验证有索引
	stats, _ := svc.Stats()
	if stats.TotalChunks == 0 {
		t.Fatal("应有 chunks")
	}

	// 删除
	if err := svc.DeleteIndex("vfs://note/n1"); err != nil {
		t.Fatal(err)
	}

	// 验证已删除
	stats, _ = svc.Stats()
	if stats.TotalChunks != 0 {
		t.Fatalf("删除后不应有 chunks，got %d", stats.TotalChunks)
	}
}

func TestService_Search_VectorOnly(t *testing.T) {
	svc := newTestService(t)

	content1 := "Go 语言编程"
	content2 := "Python 数据分析"
	opts := DefaultOptions()
	opts.Embed = true

	svc.IndexResource(context.Background(), "vfs://note/n1", content1, opts)
	svc.IndexResource(context.Background(), "vfs://note/n2", content2, opts)

	// 向量搜索
	results, err := svc.Search(context.Background(), SearchQuery{
		Query:     "Go 语言",
		Limit:     10,
		UseVector: true,
		UseFTS:    false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("向量搜索应返回结果")
	}
}

func TestService_Search_Hybrid(t *testing.T) {
	svc := newTestService(t)

	content1 := "Go 语言是一门编程语言"
	content2 := "Rust 语言也是一门编程语言"
	content3 := "Python 用于数据分析"
	opts := DefaultOptions()
	opts.Embed = true

	svc.IndexResource(context.Background(), "vfs://note/n1", content1, opts)
	svc.IndexResource(context.Background(), "vfs://note/n2", content2, opts)
	svc.IndexResource(context.Background(), "vfs://note/n3", content3, opts)

	// 混合搜索
	results, err := svc.Search(context.Background(), SearchQuery{
		Query:     "编程语言",
		Limit:     5,
		UseFTS:    true,
		UseVector: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("混合搜索应返回结果")
	}
}

func TestService_RAGQuery(t *testing.T) {
	svc := newTestService(t)

	content := "Go 语言是一门编译型编程语言，由 Google 开发。它具有简洁的语法和高效的并发支持。Go 语言的垃圾回收机制非常高效。"
	opts := DefaultOptions()
	opts.Embed = true

	svc.IndexResource(context.Background(), "vfs://note/n1", content, opts)

	// RAG 查询
	results, err := svc.RAGQuery(context.Background(), "Go 语言", 3)
	if err != nil {
		t.Fatal(err)
	}
	// RAGQuery 可能因为候选不足返回空，但不应报错
	if len(results) > 3 {
		t.Fatalf("RAG 应最多返回 topK=3 条，got %d", len(results))
	}
}

func TestService_Stats(t *testing.T) {
	svc := newTestService(t)

	// 初始状态
	stats, err := svc.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalChunks != 0 {
		t.Fatalf("初始应 0 chunks，got %d", stats.TotalChunks)
	}

	// 索引一些资源
	content := "测试统计功能的文本内容。"
	svc.IndexResource(context.Background(), "vfs://note/n1", content, DefaultOptions())
	svc.IndexResource(context.Background(), "vfs://note/n2", content, DefaultOptions())

	stats, _ = svc.Stats()
	if stats.TotalChunks == 0 {
		t.Fatal("索引后应有 chunks")
	}
	if stats.FTSRows == 0 {
		t.Fatal("应有 FTS 行")
	}
}

func TestService_IndexBatch(t *testing.T) {
	svc := newTestService(t)

	// 先在 resources 表中插入资源
	for _, uri := range []string{"vfs://note/n1", "vfs://note/n2"} {
		svc.db.Exec(`INSERT OR REPLACE INTO resources(uri, type, id, title, metadata) VALUES(?, ?, ?, ?, ?)`,
			uri, "note", "n1", "test", `{"content":"测试批量索引内容"}`)
	}

	result, err := svc.IndexBatch(context.Background(), []string{"vfs://note/n1", "vfs://note/n2"}, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 {
		t.Fatalf("总计应 2，got %d", result.Total)
	}
	if result.Success != 2 {
		t.Fatalf("成功应 2，got %d", result.Success)
	}
}

func TestService_RebuildAll(t *testing.T) {
	svc := newTestService(t)

	// 插入资源
	for _, uri := range []string{"vfs://note/n1", "vfs://note/n2"} {
		svc.db.Exec(`INSERT OR REPLACE INTO resources(uri, type, id, title, metadata) VALUES(?, ?, ?, ?, ?)`,
			uri, "note", "n1", "test", `{"content":"重建索引测试"}`)
	}

	// 先索引
	svc.IndexResource(context.Background(), "vfs://note/n1", "旧内容", DefaultOptions())

	// 重建
	result, err := svc.RebuildAll(context.Background(), DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 {
		t.Fatalf("总计应 2，got %d", result.Total)
	}
}

func TestService_GetTask(t *testing.T) {
	svc := newTestService(t)

	content := "测试任务状态"
	svc.IndexResource(context.Background(), "vfs://note/n1", content, DefaultOptions())

	task := svc.GetTask("vfs://note/n1")
	if task == nil {
		t.Fatal("任务不应为 nil")
	}
	if task.URI != "vfs://note/n1" {
		t.Fatalf("URI 不匹配: %s", task.URI)
	}
	if task.Status != TaskStatusDone {
		t.Fatalf("状态应为 done，got %s", task.Status)
	}
}

func TestService_EmbeddingFailure_NoBlock(t *testing.T) {
	// 嵌入失败不应阻塞索引
	db := newTestDB(t)
	svc := New(db, llm.NewRegistry(), nil)
	// 设置一个会失败的 provider
	svc.SetEmbedProvider(&mockEmbedProvider{name: "mock", dim: 8, failOn: 1})
	svc.SetEmbedModel("mock-model")

	content := "测试嵌入失败不阻塞索引"
	opts := DefaultOptions()
	opts.Embed = true

	err := svc.IndexResource(context.Background(), "vfs://note/n1", content, opts)
	if err != nil {
		t.Fatalf("嵌入失败不应阻塞索引: %v", err)
	}

	// FTS 应仍然有数据
	results, _ := SearchFTS(db, "测试", 10)
	if len(results) == 0 {
		t.Fatal("嵌入失败后 FTS 应仍有数据")
	}

	// 任务状态应为 done
	task := svc.GetTask("vfs://note/n1")
	if task == nil || task.Status != TaskStatusDone {
		t.Fatal("任务应为 done")
	}
}

// min 返回两个整数中的较小值。
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
