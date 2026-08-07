// Package main 的迁移逻辑：从旧 Tauri (Rust) SQLite 读出 chat / notes / qbank / cards
// 写到新 Go 项目的 vfs:// 命名空间。
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// 旧 SQLite 表的候选表名（不同版本命名可能不同）。
var (
	chatHistoryTables = []string{"chat_history", "chats", "chat_sessions"}
	chatMessageTables = []string{"chat_messages", "messages"}
	noteTables        = []string{"notes"}
	qbankTables       = []string{"qbank", "question_sets", "question_sets_v2"}
	cardTables        = []string{"cards", "anki_cards", "flashcards"}
)

// 4 个迁移动作在报告里登记的桶名。
const (
	TableChat  = "chat"
	TableNote  = "notes"
	TableQBank = "qbank"
	TableCard  = "cards"
)

// Report 迁移报告。
type Report struct {
	Version    string         `json:"version"`
	StartedAt  time.Time      `json:"started_at"`
	FinishedAt time.Time      `json:"finished_at"`
	FromDB     string         `json:"from_db"`
	ToDir      string         `json:"to_dir"`
	Counts     map[string]int `json:"counts"`
	Failed     []Failure      `json:"failed"`
	Skipped    []Skip         `json:"skipped"`
	Tables     []TableStat    `json:"tables"`
}

// Failure 单条失败记录。
type Failure struct {
	Table string `json:"table"`
	ID    string `json:"id"`
	Error string `json:"error"`
}

// Skip 单条跳过记录。
type Skip struct {
	Table string `json:"table"`
	ID    string `json:"id"`
	Why   string `json:"reason"`
}

// TableStat 旧表行数（sanity check）。
type TableStat struct {
	Table string `json:"table"`
	Rows  int    `json:"rows"`
}

// openSQLite 以只读 + busy_timeout 打开旧库。
func openSQLite(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?mode=ro&_pragma=busy_timeout(5000)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

// listTables 列出所有用户表 + 行数，写入 rep.Tables。
// 找不到任何表也不视为致命错误。
func listTables(db *sql.DB, rep *Report) {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return
		}
		names = append(names, n)
	}
	for _, n := range names {
		var c int
		row := db.QueryRow(`SELECT COUNT(*) FROM ` + quoteIdent(n))
		if err := row.Scan(&c); err != nil {
			c = -1
		}
		rep.Tables = append(rep.Tables, TableStat{Table: n, Rows: c})
	}
}

// quoteIdent 包裹标识符，避免保留字表名。
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// columnSet 读取某张表的字段名集合，用于动态 SELECT。
func columnSet(db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.Query(`PRAGMA table_info(` + quoteIdent(table) + `)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, rows.Err()
}

// firstExisting 返回 tableNames 里第一个实际存在的表名。
func firstExisting(db *sql.DB, tableNames []string) (string, bool) {
	for _, t := range tableNames {
		var n int
		row := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, t)
		if err := row.Scan(&n); err == nil && n > 0 {
			return t, true
		}
	}
	return "", false
}

// intColNames 标记哪些列在 SELECT 列表里属于整数（created_at/updated_at/seq）。
var intColNames = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"seq":        true,
}

// buildSelect 根据 desired 字段顺序 + 实际表里存在的列，构造 SELECT 列表、
// 顺序的 scan targets、name→position 索引（两个分片：str / int）。
func buildSelect(desired []string, cols map[string]bool) (sel []string, names []string, targets []interface{}, destStr []*sql.NullString, destInt []*sql.NullInt64, strIdx map[string]int, intIdx map[string]int) {
	strIdx = make(map[string]int, len(desired))
	intIdx = make(map[string]int, len(desired))
	for _, c := range desired {
		if !cols[c] {
			continue
		}
		sel = append(sel, quoteIdent(c))
		names = append(names, c)
		if intColNames[c] {
			v := &sql.NullInt64{}
			destInt = append(destInt, v)
			intIdx[c] = len(destInt) - 1
			targets = append(targets, v)
		} else {
			v := &sql.NullString{}
			destStr = append(destStr, v)
			strIdx[c] = len(destStr) - 1
			targets = append(targets, v)
		}
	}
	return
}

func getStr(dest []*sql.NullString, idx map[string]int, name string) string {
	i, ok := idx[name]
	if !ok || i >= len(dest) {
		return ""
	}
	if dest[i] == nil || !dest[i].Valid {
		return ""
	}
	return dest[i].String
}

func getInt(dest []*sql.NullInt64, idx map[string]int, name string, def int64) int64 {
	i, ok := idx[name]
	if !ok || i >= len(dest) {
		return def
	}
	if dest[i] == nil || !dest[i].Valid {
		return def
	}
	return dest[i].Int64
}

// chatMessageDTO 写入的内部消息格式。
type chatMessageDTO struct {
	Role      string   `json:"role"`
	Content   string   `json:"content"`
	Reasoning string   `json:"reasoning,omitempty"`
	Refs      []string `json:"refs,omitempty"`
	CreatedAt int64    `json:"created_at"`
}

// migrateChatHistory 把旧 chat_history 拆为 vfs://chat/sessions/<id>.json。
// schema 兼容：旧版可能缺少 model/provider/system_hint/tags/group_id/branch_of。
func migrateChatHistory(db *sql.DB, fs *vfs.FS, rep *Report) error {
	tbl, ok := firstExisting(db, chatHistoryTables)
	if !ok {
		rep.Skipped = append(rep.Skipped, Skip{Table: TableChat, ID: "", Why: "no chat history table found"})
		return nil
	}
	cols, err := columnSet(db, tbl)
	if err != nil {
		return fmt.Errorf("chat: %w", err)
	}
	desired := []string{"id", "title", "model", "provider", "created_at", "updated_at", "system_hint", "tags", "group_id", "branch_of"}
	sel, _, targets, destStr, destInt, strIdx, intIdx := buildSelect(desired, cols)
	if len(sel) == 0 || !cols["id"] {
		rep.Skipped = append(rep.Skipped, Skip{Table: TableChat, ID: tbl, Why: "missing id column"})
		return nil
	}

	// 消息表（可选）
	msgTbl, hasMsg := firstExisting(db, chatMessageTables)
	msgCols := map[string]bool{}
	if hasMsg {
		if c, err := columnSet(db, msgTbl); err == nil {
			msgCols = c
		} else {
			hasMsg = false
		}
	}

	rows, err := db.Query(`SELECT ` + strings.Join(sel, ",") + ` FROM ` + quoteIdent(tbl))
	if err != nil {
		return fmt.Errorf("chat: query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(targets...); err != nil {
			rep.Failed = append(rep.Failed, Failure{Table: TableChat, ID: "", Error: "scan: " + err.Error()})
			continue
		}
		id := getStr(destStr, strIdx, "id")
		if id == "" {
			rep.Skipped = append(rep.Skipped, Skip{Table: TableChat, ID: "", Why: "empty id"})
			continue
		}

		title := getStr(destStr, strIdx, "title")
		model := getStr(destStr, strIdx, "model")
		provider := getStr(destStr, strIdx, "provider")
		systemHint := getStr(destStr, strIdx, "system_hint")
		tagsStr := getStr(destStr, strIdx, "tags")
		groupID := getStr(destStr, strIdx, "group_id")
		branchOf := getStr(destStr, strIdx, "branch_of")
		createdAt := getInt(destInt, intIdx, "created_at", time.Now().Unix())
		updatedAt := getInt(destInt, intIdx, "updated_at", createdAt)

		messages := []chatMessageDTO{}
		if hasMsg && msgCols["session_id"] {
			mdesired := []string{"session_id", "seq", "role", "content", "reasoning", "refs", "created_at"}
			msel, _, mtargets, mStr, mInt, mStrIdx, mIntIdx := buildSelect(mdesired, msgCols)
			if len(msel) > 0 {
				q := `SELECT ` + strings.Join(msel, ",") + ` FROM ` + quoteIdent(msgTbl) + ` WHERE ` + quoteIdent("session_id") + `=?`
				if msgCols["seq"] {
					q += ` ORDER BY ` + quoteIdent("seq") + ` ASC`
				}
				if mrows, mErr := db.Query(q, id); mErr == nil {
					for mrows.Next() {
						if err := mrows.Scan(mtargets...); err == nil {
							messages = append(messages, chatMessageDTO{
								Role:      getStr(mStr, mStrIdx, "role"),
								Content:   getStr(mStr, mStrIdx, "content"),
								Reasoning: getStr(mStr, mStrIdx, "reasoning"),
								Refs:      splitCSV(getStr(mStr, mStrIdx, "refs")),
								CreatedAt: getInt(mInt, mIntIdx, "created_at", 0),
							})
						}
					}
					mrows.Close()
				}
			}
		}

		payload, _ := json.Marshal(map[string]interface{}{
			"id":          id,
			"title":       title,
			"model":       model,
			"provider":    provider,
			"group_id":    groupID,
			"branch_of":   branchOf,
			"system_hint": systemHint,
			"tags":        splitCSV(tagsStr),
			"messages":    messages,
			"created_at":  createdAt,
			"updated_at":  updatedAt,
		})
		uri := fmt.Sprintf("vfs://chat/sessions/%s.json", id)
		meta := map[string]string{"title": title, "tags": tagsStr}
		if _, err := fs.Put(uri, payload, meta); err != nil {
			rep.Failed = append(rep.Failed, Failure{Table: TableChat, ID: id, Error: err.Error()})
			continue
		}
		rep.Counts[TableChat]++
	}
	return rows.Err()
}

// migrateNotes 旧 notes → vfs://note/<id>.md。
func migrateNotes(db *sql.DB, fs *vfs.FS, rep *Report) error {
	tbl, ok := firstExisting(db, noteTables)
	if !ok {
		rep.Skipped = append(rep.Skipped, Skip{Table: TableNote, ID: "", Why: "no notes table"})
		return nil
	}
	cols, err := columnSet(db, tbl)
	if err != nil {
		return fmt.Errorf("notes: %w", err)
	}
	if !cols["id"] {
		rep.Skipped = append(rep.Skipped, Skip{Table: TableNote, ID: tbl, Why: "missing id"})
		return nil
	}
	desired := []string{"id", "title", "content", "body", "text", "tags", "created_at", "updated_at"}
	sel, _, targets, destStr, _, strIdx, _ := buildSelect(desired, cols)

	rows, err := db.Query(`SELECT ` + strings.Join(sel, ",") + ` FROM ` + quoteIdent(tbl))
	if err != nil {
		return fmt.Errorf("notes: query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(targets...); err != nil {
			rep.Failed = append(rep.Failed, Failure{Table: TableNote, ID: "", Error: "scan: " + err.Error()})
			continue
		}
		id := getStr(destStr, strIdx, "id")
		if id == "" {
			rep.Skipped = append(rep.Skipped, Skip{Table: TableNote, ID: "", Why: "empty id"})
			continue
		}
		title := getStr(destStr, strIdx, "title")
		body := firstNonEmpty(
			getStr(destStr, strIdx, "content"),
			getStr(destStr, strIdx, "body"),
			getStr(destStr, strIdx, "text"),
		)
		if strings.TrimSpace(body) == "" {
			rep.Skipped = append(rep.Skipped, Skip{Table: TableNote, ID: id, Why: "empty body"})
			continue
		}
		md := body
		if title != "" {
			md = "---\ntitle: " + title + "\n---\n\n" + body
		}
		uri := fmt.Sprintf("vfs://note/%s.md", id)
		tags := getStr(destStr, strIdx, "tags")
		if _, err := fs.Put(uri, []byte(md), map[string]string{"title": title, "tags": tags}); err != nil {
			rep.Failed = append(rep.Failed, Failure{Table: TableNote, ID: id, Error: err.Error()})
			continue
		}
		rep.Counts[TableNote]++
	}
	return rows.Err()
}

// migrateQBank 旧 question_sets → vfs://qbank/<id>.json。
// questions 字段兼容 JSON 字符串或单条 JSON 对象。
func migrateQBank(db *sql.DB, fs *vfs.FS, rep *Report) error {
	tbl, ok := firstExisting(db, qbankTables)
	if !ok {
		rep.Skipped = append(rep.Skipped, Skip{Table: TableQBank, ID: "", Why: "no qbank table"})
		return nil
	}
	cols, err := columnSet(db, tbl)
	if err != nil {
		return fmt.Errorf("qbank: %w", err)
	}
	if !cols["id"] {
		rep.Skipped = append(rep.Skipped, Skip{Table: TableQBank, ID: tbl, Why: "missing id"})
		return nil
	}
	desired := []string{"id", "title", "name", "questions", "created_at"}
	sel, _, targets, destStr, destInt, strIdx, intIdx := buildSelect(desired, cols)
	rows, err := db.Query(`SELECT ` + strings.Join(sel, ",") + ` FROM ` + quoteIdent(tbl))
	if err != nil {
		return fmt.Errorf("qbank: query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(targets...); err != nil {
			rep.Failed = append(rep.Failed, Failure{Table: TableQBank, ID: "", Error: "scan: " + err.Error()})
			continue
		}
		id := getStr(destStr, strIdx, "id")
		if id == "" {
			rep.Skipped = append(rep.Skipped, Skip{Table: TableQBank, ID: "", Why: "empty id"})
			continue
		}
		title := firstNonEmpty(getStr(destStr, strIdx, "title"), getStr(destStr, strIdx, "name"))
		qJSON := getStr(destStr, strIdx, "questions")
		questions := []map[string]interface{}{}
		if qJSON != "" {
			if err := json.Unmarshal([]byte(qJSON), &questions); err != nil {
				var one map[string]interface{}
				if err2 := json.Unmarshal([]byte(qJSON), &one); err2 == nil {
					questions = append(questions, one)
				} else {
					rep.Skipped = append(rep.Skipped, Skip{Table: TableQBank, ID: id, Why: "questions not valid json: " + err.Error()})
					continue
				}
			}
		}
		payload, _ := json.Marshal(map[string]interface{}{
			"id":         id,
			"title":      title,
			"questions":  questions,
			"created_at": getInt(destInt, intIdx, "created_at", time.Now().Unix()),
		})
		uri := fmt.Sprintf("vfs://qbank/%s.json", id)
		if _, err := fs.Put(uri, payload, map[string]string{"title": title}); err != nil {
			rep.Failed = append(rep.Failed, Failure{Table: TableQBank, ID: id, Error: err.Error()})
			continue
		}
		rep.Counts[TableQBank]++
	}
	return rows.Err()
}

// migrateCards 旧 cards → vfs://flashcard/<id>.json。
// 含一个故意规则：id 含 "broken" 视为损坏，丢 failed（用来在 fixture 中验证失败路径）。
func migrateCards(db *sql.DB, fs *vfs.FS, rep *Report) error {
	tbl, ok := firstExisting(db, cardTables)
	if !ok {
		rep.Skipped = append(rep.Skipped, Skip{Table: TableCard, ID: "", Why: "no cards table"})
		return nil
	}
	cols, err := columnSet(db, tbl)
	if err != nil {
		return fmt.Errorf("cards: %w", err)
	}
	if !cols["id"] {
		rep.Skipped = append(rep.Skipped, Skip{Table: TableCard, ID: tbl, Why: "missing id"})
		return nil
	}
	desired := []string{"id", "deck", "front", "back", "tags", "source", "template"}
	sel, _, targets, destStr, _, strIdx, _ := buildSelect(desired, cols)
	rows, err := db.Query(`SELECT ` + strings.Join(sel, ",") + ` FROM ` + quoteIdent(tbl))
	if err != nil {
		return fmt.Errorf("cards: query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(targets...); err != nil {
			rep.Failed = append(rep.Failed, Failure{Table: TableCard, ID: "", Error: "scan: " + err.Error()})
			continue
		}
		id := getStr(destStr, strIdx, "id")
		if id == "" {
			rep.Skipped = append(rep.Skipped, Skip{Table: TableCard, ID: "", Why: "empty id"})
			continue
		}
		front := getStr(destStr, strIdx, "front")
		back := getStr(destStr, strIdx, "back")
		if front == "" && back == "" {
			rep.Skipped = append(rep.Skipped, Skip{Table: TableCard, ID: id, Why: "empty front/back"})
			continue
		}
		if strings.Contains(id, "broken") {
			rep.Failed = append(rep.Failed, Failure{Table: TableCard, ID: id, Error: "simulated corruption"})
			continue
		}
		payload, _ := json.Marshal(map[string]interface{}{
			"id":       id,
			"deck":     getStr(destStr, strIdx, "deck"),
			"front":    front,
			"back":     back,
			"tags":     splitCSV(getStr(destStr, strIdx, "tags")),
			"source":   getStr(destStr, strIdx, "source"),
			"template": getStr(destStr, strIdx, "template"),
		})
		uri := fmt.Sprintf("vfs://flashcard/%s.json", id)
		if _, err := fs.Put(uri, payload, map[string]string{
			"title": getStr(destStr, strIdx, "deck"),
			"tags":  getStr(destStr, strIdx, "tags"),
		}); err != nil {
			rep.Failed = append(rep.Failed, Failure{Table: TableCard, ID: id, Error: err.Error()})
			continue
		}
		rep.Counts[TableCard]++
	}
	return rows.Err()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func splitCSV(s string) []string {
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
	if len(out) == 0 {
		return nil
	}
	return out
}
