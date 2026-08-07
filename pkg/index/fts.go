// FTS5 全文索引，基于 SQLite FTS5 虚拟表。
//
// 使用 modernc.org/sqlite 内置 FTS5 扩展（默认编译包含）。
// 表名 fts_chunks，避免与 store.Chunks 冲突。

package index

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// InitFTS 初始化 FTS5 虚拟表。
//
// 创建 fts_chunks 虚拟表，使用 trigram 分词器（支持 CJK 子串匹配）。
// 同时为 chunks 表补充 created_at 列（如果缺失）。
func InitFTS(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("index: db is nil")
	}

	// 创建 FTS5 虚拟表
	// 使用 trigram 分词器：支持 CJK 子串匹配（unicode61 会把连续中文当作单个 token，无法子串搜索）
	// trigram 在 SQLite 3.34.0+ 可用，modernc.org/sqlite v1.29.10 已包含
	_, err := db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS fts_chunks USING fts5(uri, content, tokenize='trigram')`)
	if err != nil {
		return fmt.Errorf("index: create fts_chunks: %w", err)
	}

	// 为 chunks 表补充 created_at 列（store.go 未创建此列）
	// ALTER TABLE ADD COLUMN 在列已存在时不会报错（SQLite 不支持 IF NOT EXISTS，需手动检查）
	var colCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('chunks') WHERE name='created_at'`).Scan(&colCount)
	if err != nil {
		return fmt.Errorf("index: check chunks.created_at: %w", err)
	}
	if colCount == 0 {
		_, err = db.Exec(`ALTER TABLE chunks ADD COLUMN created_at INTEGER DEFAULT 0`)
		if err != nil {
			return fmt.Errorf("index: add chunks.created_at: %w", err)
		}
	}

	return nil
}

// IndexFTS 索引单条文本到 FTS5。
func IndexFTS(db *sql.DB, uri, content string) error {
	if db == nil {
		return fmt.Errorf("index: db is nil")
	}
	_, err := db.Exec(`INSERT INTO fts_chunks(uri, content) VALUES (?, ?)`, uri, content)
	return err
}

// IndexFTSBatch 批量索引文本到 FTS5。
func IndexFTSBatch(db *sql.DB, entries []struct{ URI, Content string }) error {
	if db == nil {
		return fmt.Errorf("index: db is nil")
	}
	if len(entries) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT INTO fts_chunks(uri, content) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, e := range entries {
		if _, err := stmt.Exec(e.URI, e.Content); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SearchFTS 使用 FTS5 MATCH + bm25 排序搜索。
//
// 返回的结果按 bm25 相关性排序，Score 归一化到 [0, 1]。
// trigram 分词器支持子串匹配，适合 CJK 文本。
// 查询长度 < 3 字符时（trigram 最小长度），回退到 LIKE 搜索。
func SearchFTS(db *sql.DB, query string, limit int) ([]SearchResult, error) {
	if db == nil {
		return nil, fmt.Errorf("index: db is nil")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	// trigram 最小匹配长度为 3 字符；短查询回退到 LIKE
	if utf8.RuneCountInString(query) < 3 {
		return searchFTSLike(db, query, limit)
	}

	// 构建 FTS5 MATCH 查询（trigram 子串匹配）
	matchExpr := sanitizeFTSQuery(query)
	if matchExpr == "" {
		return nil, nil
	}

	// 使用 bm25 排序 + snippet 高亮
	// bm25 返回负值，越负越相关，ORDER BY bm25 升序（最负在前）
	sqlQuery := `
		SELECT uri, content, bm25(fts_chunks) AS rank, snippet(fts_chunks, 1, '[', ']', '...', 32) AS snip
		FROM fts_chunks
		WHERE fts_chunks MATCH ?
		ORDER BY rank
		LIMIT ?`

	rows, err := db.Query(sqlQuery, matchExpr, limit)
	if err != nil {
		return nil, fmt.Errorf("index: fts search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var uri, content, snip string
		var rank float64
		if err := rows.Scan(&uri, &content, &rank, &snip); err != nil {
			return nil, err
		}

		// bm25 归一化：bm25 为负值，abs(bm25) 越小越相关
		// score = 1 / (1 + abs(bm25))，映射到 (0, 1]
		score := 1.0 / (1.0 + math.Abs(rank))

		results = append(results, SearchResult{
			URI:      uri,
			Snippet:  snip,
			Score:    score,
			FTSScore: score,
		})
	}

	return results, rows.Err()
}

// searchFTSLike 短查询回退：使用 LIKE 搜索。
func searchFTSLike(db *sql.DB, query string, limit int) ([]SearchResult, error) {
	likeQuery := "%" + query + "%"
	rows, err := db.Query(`SELECT uri, content FROM fts_chunks WHERE content LIKE ? LIMIT ?`, likeQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var uri, content string
		if err := rows.Scan(&uri, &content); err != nil {
			return nil, err
		}
		// LIKE 搜索给固定得分 0.5
		results = append(results, SearchResult{
			URI:      uri,
			Snippet:  content,
			Score:    0.5,
			FTSScore: 0.5,
		})
	}
	return results, rows.Err()
}

// DeleteFTS 删除某 URI 的全部 FTS 记录。
func DeleteFTS(db *sql.DB, uri string) error {
	if db == nil {
		return fmt.Errorf("index: db is nil")
	}
	_, err := db.Exec(`DELETE FROM fts_chunks WHERE uri = ?`, uri)
	return err
}

// FTSRowCount 返回 FTS 表的行数。
func FTSRowCount(db *sql.DB) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("index: db is nil")
	}
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM fts_chunks`).Scan(&count)
	return count, err
}

// sanitizeFTSQuery 将用户查询转换为安全的 FTS5 MATCH 表达式。
//
// trigram 分词器策略：转义双引号后，用双引号包裹整个查询，
// 实现子串匹配（trigram 天然支持 CJK 子串搜索）。
func sanitizeFTSQuery(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}
	// 转义双引号（FTS5 用 "" 转义）
	escaped := strings.ReplaceAll(query, `"`, `""`)
	return `"` + escaped + `"`
}
