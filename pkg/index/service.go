// 索引服务，统一入口，管理 FTS5 全文索引 + 向量嵌入 + RAG 检索。
//
// 参考 Rust index_service.rs / indexing.rs：
// - IndexResource：切片 → 写 chunks 表 → FTS 索引 → 可选向量嵌入
// - Search：FTS + 向量混合检索
// - RAGQuery：先取 topK*3 候选，再用查询向量重排序取 topK

package index

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/helixnow/deep-student-go/internal/llmcfg"
	"github.com/helixnow/deep-student-go/pkg/llm"
)

// Service 索引服务。
type Service struct {
	db        *sql.DB
	llmReg    *llm.Registry
	llmcfgMgr *llmcfg.Manager
	opts      IndexOptions
	mu        sync.RWMutex
	tasks     map[string]*IndexTask // URI → 任务状态

	// 可选的直接注入（用于测试或绕过 llmcfg 解析）
	embedProvider llm.Provider
	embedModel    string
}

// New 创建索引服务。
//
// 初始化 FTS5 虚拟表，准备 chunks 表的 created_at 列。
func New(db *sql.DB, llmReg *llm.Registry, llmcfgMgr *llmcfg.Manager) *Service {
	s := &Service{
		db:        db,
		llmReg:    llmReg,
		llmcfgMgr: llmcfgMgr,
		opts:      DefaultOptions(),
		tasks:     map[string]*IndexTask{},
	}
	// 初始化 FTS（忽略错误，后续操作会暴露问题）
	if err := InitFTS(db); err != nil {
		log.Printf("[index] InitFTS warning: %v", err)
	}
	return s
}

// SetEmbedProvider 直接注入嵌入 Provider（用于测试或绕过 llmcfg 解析）。
func (s *Service) SetEmbedProvider(p llm.Provider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.embedProvider = p
}

// SetEmbedModel 直接设置嵌入模型名称。
func (s *Service) SetEmbedModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.embedModel = model
}

// SetOptions 设置默认索引选项。
func (s *Service) SetOptions(opts IndexOptions) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opts = opts
}

// resolveEmbedder 解析嵌入器和模型名称。
//
// 优先级：
// 1. 直接注入的 embedProvider / embedModel
// 2. 从 llmcfg 解析 EmbeddingModelConfigID
func (s *Service) resolveEmbedder() (*Embedder, string, error) {
	s.mu.RLock()
	injected := s.embedProvider
	injectedModel := s.embedModel
	s.mu.RUnlock()

	if injected != nil {
		model := injectedModel
		if model == "" {
			model = "text-embedding-3-small"
		}
		return NewEmbedder(injected), model, nil
	}

	// 从 llmcfg 解析
	if s.llmcfgMgr == nil {
		return nil, "", fmt.Errorf("index: llmcfg manager not configured")
	}

	assignments := s.llmcfgMgr.GetAssignments()
	if assignments.EmbeddingModelConfigID == nil {
		return nil, "", fmt.Errorf("index: no embedding model assigned")
	}

	cfg, err := s.llmcfgMgr.ResolveApiConfig(*assignments.EmbeddingModelConfigID)
	if err != nil {
		return nil, "", fmt.Errorf("index: resolve embedding config: %w", err)
	}

	// 从 registry 查找 provider
	if s.llmReg == nil {
		return nil, "", fmt.Errorf("index: llm registry not configured")
	}

	// 尝试按 ProviderType 和 VendorName 查找
	providerNames := s.llmReg.Names()
	var provider llm.Provider
	for _, name := range providerNames {
		p, ok := s.llmReg.Get(name)
		if !ok {
			continue
		}
		// 按 ProviderType 匹配
		if cfg.ProviderType != nil && name == *cfg.ProviderType {
			provider = p
			break
		}
		// 按 VendorName 匹配
		if cfg.VendorName != nil && name == *cfg.VendorName {
			provider = p
			break
		}
	}
	if provider == nil && len(providerNames) > 0 {
		// 回退到第一个 provider
		provider, _ = s.llmReg.Get(providerNames[0])
	}
	if provider == nil {
		return nil, "", fmt.Errorf("index: no llm provider available")
	}

	return NewEmbedder(provider), cfg.Model, nil
}

// IndexResource 对单个资源建立索引（FTS + 可选向量）。
//
// 流程：chunker 切片 → 写 chunks 表 → FTS 索引 → 可选向量嵌入。
// 向量嵌入失败不阻塞索引（记日志，返回 partial success）。
func (s *Service) IndexResource(ctx context.Context, uri string, content string, opts IndexOptions) error {
	if s.db == nil {
		return fmt.Errorf("index: db is nil")
	}
	if uri == "" {
		return fmt.Errorf("index: uri is empty")
	}

	// 合并默认选项
	if opts.ChunkSize <= 0 {
		opts.ChunkSize = s.opts.ChunkSize
		if opts.ChunkSize <= 0 {
			opts = DefaultOptions()
		}
	}

	// 记录任务状态
	s.setTask(uri, TaskStatusIndexing, 0.1, "")
	now := time.Now().Unix()

	// 1. 删除旧索引
	if err := s.deleteChunksAndFTS(uri); err != nil {
		s.setTask(uri, TaskStatusError, 0, err.Error())
		return fmt.Errorf("index: delete old index for %s: %w", uri, err)
	}

	// 2. 切片
	chunks := ChunkText(content, opts)
	if len(chunks) == 0 {
		s.setTask(uri, TaskStatusDone, 1.0, "")
		return nil
	}
	s.setTask(uri, TaskStatusIndexing, 0.3, "")

	// 3. 写入 chunks 表
	chunkIDs, err := s.insertChunks(uri, chunks, now)
	if err != nil {
		s.setTask(uri, TaskStatusError, 0, err.Error())
		return fmt.Errorf("index: insert chunks for %s: %w", uri, err)
	}
	s.setTask(uri, TaskStatusIndexing, 0.5, "")

	// 4. FTS 索引
	if err := s.indexFTSForChunks(uri, chunks); err != nil {
		// FTS 失败不阻塞，但记录
		log.Printf("[index] FTS index warning for %s: %v", uri, err)
	}
	s.setTask(uri, TaskStatusIndexing, 0.7, "")

	// 5. 可选向量嵌入
	if opts.Embed {
		if err := s.embedAndSave(ctx, uri, chunks, chunkIDs, opts); err != nil {
			// 嵌入失败不阻塞索引
			log.Printf("[index] embedding warning for %s: %v", uri, err)
			s.setTask(uri, TaskStatusDone, 1.0, "")
			return nil
		}
		s.setTask(uri, TaskStatusEmbedded, 1.0, "")
		s.setTask(uri, TaskStatusDone, 1.0, "")
	} else {
		s.setTask(uri, TaskStatusDone, 1.0, "")
	}

	return nil
}

// IndexBatch 批量索引。
func (s *Service) IndexBatch(ctx context.Context, uris []string, opts IndexOptions) (*BatchIndexResult, error) {
	result := &BatchIndexResult{
		Total: len(uris),
		Tasks: make([]IndexTask, 0, len(uris)),
	}

	for _, uri := range uris {
		// 读取资源内容
		content, err := s.readResourceContent(uri)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, IndexError{URI: uri, Error: err.Error()})
			result.Tasks = append(result.Tasks, IndexTask{
				URI: uri, Status: TaskStatusError, Error: err.Error(), StartedAt: time.Now().Unix(),
			})
			continue
		}

		if content == "" {
			result.Skipped++
			result.Tasks = append(result.Tasks, IndexTask{
				URI: uri, Status: TaskStatusDone, Progress: 1.0, StartedAt: time.Now().Unix(),
			})
			continue
		}

		if err := s.IndexResource(ctx, uri, content, opts); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, IndexError{URI: uri, Error: err.Error()})
			result.Tasks = append(result.Tasks, IndexTask{
				URI: uri, Status: TaskStatusError, Error: err.Error(), StartedAt: time.Now().Unix(),
			})
		} else {
			result.Success++
			result.Tasks = append(result.Tasks, IndexTask{
				URI: uri, Status: TaskStatusDone, Progress: 1.0, StartedAt: time.Now().Unix(),
			})
		}
	}

	return result, nil
}

// DeleteIndex 删除索引。
func (s *Service) DeleteIndex(uri string) error {
	if s.db == nil {
		return fmt.Errorf("index: db is nil")
	}
	if err := s.deleteChunksAndFTS(uri); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.tasks, uri)
	s.mu.Unlock()
	return nil
}

// RebuildAll 重建全部索引。
func (s *Service) RebuildAll(ctx context.Context, opts IndexOptions) (*BatchIndexResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("index: db is nil")
	}

	// 清空所有索引
	_, err := s.db.Exec(`DELETE FROM chunks`)
	if err != nil {
		return nil, fmt.Errorf("index: clear chunks: %w", err)
	}
	_, err = s.db.Exec(`DELETE FROM fts_chunks`)
	if err != nil {
		return nil, fmt.Errorf("index: clear fts_chunks: %w", err)
	}

	// 获取所有资源 URI
	rows, err := s.db.Query(`SELECT uri FROM resources ORDER BY uri`)
	if err != nil {
		return nil, fmt.Errorf("index: list resources: %w", err)
	}
	var uris []string
	for rows.Next() {
		var uri string
		if err := rows.Scan(&uri); err != nil {
			rows.Close()
			return nil, err
		}
		uris = append(uris, uri)
	}
	rows.Close()

	return s.IndexBatch(ctx, uris, opts)
}

// Stats 统计索引状态。
func (s *Service) Stats() (*IndexStats, error) {
	if s.db == nil {
		return nil, fmt.Errorf("index: db is nil")
	}

	stats := &IndexStats{}

	// 资源总数
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM resources`).Scan(&stats.TotalResources); err != nil {
		return nil, err
	}

	// 已索引资源数（有 chunk 记录的 URI 数）
	if err := s.db.QueryRow(`SELECT COUNT(DISTINCT uri) FROM chunks`).Scan(&stats.IndexedResources); err != nil {
		return nil, err
	}

	// 总 chunk 数
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM chunks`).Scan(&stats.TotalChunks); err != nil {
		return nil, err
	}

	// 已嵌入 chunk 数（embedding 非空）
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM chunks WHERE embedding IS NOT NULL`).Scan(&stats.EmbeddedChunks); err != nil {
		return nil, err
	}

	// FTS 行数
	ftsRows, err := FTSRowCount(s.db)
	if err != nil {
		return nil, err
	}
	stats.FTSRows = ftsRows

	// 平均 token 数
	if stats.TotalChunks > 0 {
		var avgFloat float64
		if err := s.db.QueryRow(`SELECT COALESCE(AVG(token_count), 0) FROM chunks`).Scan(&avgFloat); err != nil {
			return nil, err
		}
		stats.AvgChunkTokens = int(avgFloat)
	}

	return stats, nil
}

// Search 综合搜索（FTS + 向量混合）。
func (s *Service) Search(ctx context.Context, q SearchQuery) ([]SearchResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("index: db is nil")
	}
	if strings.TrimSpace(q.Query) == "" {
		return nil, nil
	}
	if q.Limit <= 0 {
		q.Limit = 10
	}

	var ftsResults, vecResults []SearchResult

	// FTS 搜索
	if q.UseFTS {
		ftsResults, _ = SearchFTS(s.db, q.Query, q.Limit*2)
		s.enrichResults(ftsResults)
	}

	// 向量搜索
	if q.UseVector {
		vecResults = s.vectorSearch(ctx, q.Query, q.Limit*2)
	}

	// 合并
	var results []SearchResult
	if q.UseFTS && q.UseVector {
		results = MergeResults(ftsResults, vecResults, q.Limit)
	} else if q.UseFTS {
		results = ftsResults
		if len(results) > q.Limit {
			results = results[:q.Limit]
		}
	} else if q.UseVector {
		results = vecResults
		if len(results) > q.Limit {
			results = results[:q.Limit]
		}
	}

	// 过滤最低分
	if q.MinScore > 0 {
		filtered := results[:0]
		for _, r := range results {
			if r.Score >= q.MinScore {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	// 类型过滤
	if len(q.Types) > 0 {
		typeSet := make(map[string]bool)
		for _, t := range q.Types {
			typeSet[t] = true
		}
		filtered := results[:0]
		for _, r := range results {
			if typeSet[r.Type] {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	return results, nil
}

// RAGQuery RAG 检索（带重排序）。
//
// 先取 topK*3 候选，再用查询向量重排序取 topK。
func (s *Service) RAGQuery(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("index: db is nil")
	}
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	if topK <= 0 {
		topK = 5
	}

	// 1. 取 topK*3 候选
	candidateLimit := topK * 3
	q := SearchQuery{
		Query:     query,
		Limit:     candidateLimit,
		UseFTS:    true,
		UseVector: true,
	}
	candidates, err := s.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	// 2. 用查询向量重排序
	embedder, model, err := s.resolveEmbedder()
	if err != nil {
		// 无法嵌入查询，直接返回候选
		if len(candidates) > topK {
			return candidates[:topK], nil
		}
		return candidates, nil
	}

	queryVec, err := embedder.EmbedSingle(ctx, query, model)
	if err != nil {
		if len(candidates) > topK {
			return candidates[:topK], nil
		}
		return candidates, nil
	}

	// 加载候选对应的 chunk 向量
	var chunkCandidates []Chunk
	for _, c := range candidates {
		chunks, _ := LoadEmbeddingsByURI(s.db, c.URI)
		chunkCandidates = append(chunkCandidates, chunks...)
	}

	if len(chunkCandidates) == 0 {
		if len(candidates) > topK {
			return candidates[:topK], nil
		}
		return candidates, nil
	}

	// 3. 重排序
	return Rerank(ctx, queryVec, chunkCandidates, topK)
}

// GetTask 获取任务状态。
func (s *Service) GetTask(uri string) *IndexTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if t, ok := s.tasks[uri]; ok {
		cp := *t
		return &cp
	}
	return nil
}

// ---------- 内部方法 ----------

// setTask 设置任务状态。
func (s *Service) setTask(uri, status string, progress float64, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[uri] = &IndexTask{
		URI:       uri,
		Status:    status,
		Progress:  progress,
		Error:     errMsg,
		StartedAt: time.Now().Unix(),
	}
}

// deleteChunksAndFTS 删除指定 URI 的 chunks 和 FTS 记录。
func (s *Service) deleteChunksAndFTS(uri string) error {
	_, err := s.db.Exec(`DELETE FROM chunks WHERE uri = ?`, uri)
	if err != nil {
		return err
	}
	return DeleteFTS(s.db, uri)
}

// insertChunks 批量插入 chunks，返回自增 ID 列表。
func (s *Service) insertChunks(uri string, chunks []Chunk, now int64) ([]int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO chunks(uri, pos, content, token_count, created_at) VALUES(?, ?, ?, ?, ?)`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	ids := make([]int64, 0, len(chunks))
	for _, c := range chunks {
		res, err := stmt.Exec(uri, c.Pos, c.Content, c.TokenCount, now)
		if err != nil {
			return nil, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, tx.Commit()
}

// indexFTSForChunks 为每个 chunk 建立 FTS 索引。
func (s *Service) indexFTSForChunks(uri string, chunks []Chunk) error {
	// 先删除旧 FTS 记录
	if err := DeleteFTS(s.db, uri); err != nil {
		return err
	}
	// 逐条插入 FTS
	for _, c := range chunks {
		if err := IndexFTS(s.db, uri, c.Content); err != nil {
			return err
		}
	}
	return nil
}

// embedAndSave 嵌入并保存向量。
func (s *Service) embedAndSave(ctx context.Context, uri string, chunks []Chunk, chunkIDs []int64, opts IndexOptions) error {
	embedder, model, err := s.resolveEmbedder()
	if err != nil {
		return err
	}

	// 确定模型名称
	if opts.EmbedModel != "" {
		model = opts.EmbedModel
	}

	embeddings, err := embedder.EmbedChunks(ctx, chunks, model)
	if err != nil {
		return err
	}

	return SaveEmbeddings(s.db, chunkIDs, embeddings)
}

// vectorSearch 向量搜索。
func (s *Service) vectorSearch(ctx context.Context, query string, limit int) []SearchResult {
	embedder, model, err := s.resolveEmbedder()
	if err != nil {
		return nil
	}

	queryVec, err := embedder.EmbedSingle(ctx, query, model)
	if err != nil {
		return nil
	}

	// 加载所有带嵌入的 chunk
	allChunks, err := LoadEmbeddings(s.db)
	if err != nil || len(allChunks) == 0 {
		return nil
	}

	return VectorSearch(queryVec, allChunks, limit)
}

// enrichResults 从 resources 表补充 Type/Title 信息。
func (s *Service) enrichResults(results []SearchResult) {
	if len(results) == 0 {
		return
	}
	uriSet := make(map[string]bool)
	for _, r := range results {
		uriSet[r.URI] = true
	}
	if len(uriSet) == 0 {
		return
	}

	// 批量查询 resources 表
	placeholders := make([]string, 0, len(uriSet))
	args := make([]any, 0, len(uriSet))
	for uri := range uriSet {
		placeholders = append(placeholders, "?")
		args = append(args, uri)
	}
	query := fmt.Sprintf(`SELECT uri, type, title FROM resources WHERE uri IN (%s)`, strings.Join(placeholders, ","))
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return
	}
	defer rows.Close()

	meta := make(map[string]struct{ Type, Title string })
	for rows.Next() {
		var uri, typ, title string
		if err := rows.Scan(&uri, &typ, &title); err != nil {
			continue
		}
		meta[uri] = struct{ Type, Title string }{typ, title}
	}

	for i := range results {
		if m, ok := meta[results[i].URI]; ok {
			results[i].Type = m.Type
			results[i].Title = m.Title
		}
	}
}

// readResourceContent 读取资源内容。
//
// 从 resources 表读取 metadata 中的 content 字段，
// 或从 blob_ref 读取（当前简化为从 metadata JSON 提取）。
func (s *Service) readResourceContent(uri string) (string, error) {
	var metadata sql.NullString
	var blobRef sql.NullString
	err := s.db.QueryRow(`SELECT metadata, blob_ref FROM resources WHERE uri = ?`, uri).Scan(&metadata, &blobRef)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("index: resource not found: %s", uri)
	}
	if err != nil {
		return "", err
	}

	metaStr := ""
	if metadata.Valid {
		metaStr = metadata.String
	}

	// 尝试从 metadata JSON 中提取 content 字段
	// metadata 格式为 {"content":"...", ...}
	content := extractJSONField(metaStr, "content")
	if content != "" {
		return content, nil
	}

	// 如果 metadata 本身就是纯文本内容
	if metaStr != "" && !strings.HasPrefix(metaStr, "{") {
		return metaStr, nil
	}

	return "", nil
}

// extractJSONField 从简单 JSON 字符串中提取字段值（避免引入 encoding/json 的开销）。
func extractJSONField(jsonStr, field string) string {
	if jsonStr == "" {
		return ""
	}
	key := `"` + field + `":`
	idx := strings.Index(jsonStr, key)
	if idx < 0 {
		return ""
	}
	rest := jsonStr[idx+len(key):]
	rest = strings.TrimLeft(rest, " ")
	if len(rest) == 0 {
		return ""
	}
	if rest[0] == '"' {
		// 字符串值
		end := strings.Index(rest[1:], `"`)
		if end < 0 {
			return ""
		}
		return rest[1 : 1+end]
	}
	// 非字符串值，取到逗号或花括号
	end := strings.IndexAny(rest, ",}")
	if end < 0 {
		return rest
	}
	return rest[:end]
}
