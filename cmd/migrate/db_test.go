// Package main 的迁移工具测试：构造一个旧版 SQLite 库（4 张表），跑迁移，
// 校验落盘 blob 数量 / 报告字段 / 抽样 payload 内容。
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// buildOldSQLiteFixture 在 dir 下生成 old.db，写入 4 张表 + chat_messages 关联子表。
//
// 关键场景：
//   - chat_history   3 行：2 个有效 session，1 个空 id（→ skipped）
//   - chat_messages  4 行：3 条属于 s1，1 条属于 s2（验按 session_id 关联 + seq 排序）
//   - notes          3 行：2 个有效，1 个空 body（→ skipped）
//   - qbank          2 行：1 个含 JSON 数组，1 个空 questions
//   - cards          3 行：1 正常 + 1 "broken" id（→ failed）+ 1 空 front/back（→ skipped）
func buildOldSQLiteFixture(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "old.db")
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer db.Close()

	stmts := []string{
		`CREATE TABLE chat_history (
			id TEXT PRIMARY KEY,
			title TEXT,
			model TEXT,
			provider TEXT,
			created_at INTEGER,
			updated_at INTEGER,
			system_hint TEXT,
			tags TEXT,
			group_id TEXT,
			branch_of TEXT
		)`,
		`CREATE TABLE chat_messages (
			session_id TEXT,
			seq INTEGER,
			role TEXT,
			content TEXT,
			reasoning TEXT,
			refs TEXT,
			created_at INTEGER
		)`,
		`CREATE TABLE notes (
			id TEXT PRIMARY KEY,
			title TEXT,
			content TEXT,
			tags TEXT,
			created_at INTEGER,
			updated_at INTEGER
		)`,
		`CREATE TABLE qbank (
			id TEXT PRIMARY KEY,
			title TEXT,
			questions TEXT,
			created_at INTEGER
		)`,
		`CREATE TABLE cards (
			id TEXT PRIMARY KEY,
			deck TEXT,
			front TEXT,
			back TEXT,
			tags TEXT,
			source TEXT,
			template TEXT
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("create table: %v\nSQL: %s", err, s)
		}
	}

	// chat_history（最后一行 id 为空 → skipped）
	if _, err := db.Exec(`INSERT INTO chat_history VALUES
		('s1', 'first chat',  'gpt-4o',   'openai',    1700000000, 1700000100, 'be helpful', 'demo,test', '', ''),
		('s2', 'second chat', 'claude-3', 'anthropic', 1700000200, 1700000300, '',           'demo',      '', ''),
		('',  'no id row',   'gpt-4o',   'openai',    1700000400, 1700000400, '',           '',          '', '')
	`); err != nil {
		t.Fatalf("insert chat_history: %v", err)
	}
	// chat_messages（注意 s1 的 seq 是乱序写入，期望按 seq ASC 重新排）
	if _, err := db.Exec(`INSERT INTO chat_messages VALUES
		('s1', 3, 'user',      'explain notes', '',          'vfs://note/x', 1700000003),
		('s1', 1, 'user',      'hi',            '',          '',             1700000001),
		('s1', 2, 'assistant', 'hello there',   'thinking',  '',             1700000002),
		('s2', 1, 'user',      'second session','',          '',             1700000201)
	`); err != nil {
		t.Fatalf("insert chat_messages: %v", err)
	}

	// notes（n3 body 为空 → skipped）
	if _, err := db.Exec(`INSERT INTO notes VALUES
		('n1', 'note one',   'body one', 'tagA,tagB', 1700000000, 1700000000),
		('n2', 'note two',   'body two', 'tagA',      1700000100, 1700000100),
		('n3', 'empty body', '',         '',          1700000200, 1700000200)
	`); err != nil {
		t.Fatalf("insert notes: %v", err)
	}

	// qbank
	if _, err := db.Exec(`INSERT INTO qbank VALUES
		('q1', 'qbank one', '[{"id":"q1","stem":"1+1?","answer":"2","type":"single","knowledge":["math"]}]', 1700000000),
		('q2', 'qbank two', '', 1700000100)
	`); err != nil {
		t.Fatalf("insert qbank: %v", err)
	}

	// cards
	if _, err := db.Exec(`INSERT INTO cards VALUES
		('c1',       'deckA', 'front1',     'back1',       'tagA,tagB', 'vfs://note/n1', 'default'),
		('broken-1', 'deckA', 'front bad',  'back bad',    'tagBroken', '',              'default'),
		('c3',       'deckB', '',           '',            'tagC',      '',              'default')
	`); err != nil {
		t.Fatalf("insert cards: %v", err)
	}
	return path
}

// countBlobs 递归统计 blob 目录下所有文件数（blob 是分片存储的）。
func countBlobs(t *testing.T, dir string) int {
	t.Helper()
	n := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			n++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk blob: %v", err)
	}
	return n
}

// findBlobWithPrefix 找包含子串的 blob 内容，返回 (sha256 引用, 内容)。
func findBlobWithPrefix(t *testing.T, dir, needle string) (string, []byte) {
	t.Helper()
	var found string
	var data []byte
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if bytes.Contains(b, []byte(needle)) {
			found = filepath.Base(path)
			data = b
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if found == "" {
		t.Fatalf("no blob contains %q", needle)
	}
	return found, data
}

// TestRunWithRealFixture 端到端：用真实 fixture 跑一次完整迁移，校验报告 + 落盘 blob。
//
// 期望结果：
//   - counts: {chat:2, notes:2, qbank:2, cards:1}
//   - failed: 1 (cards/broken-1)
//   - skipped: 3 (chat 空 id, notes 空 body, cards 空 front/back)
//   - exit 1 (partial failure)
//   - blob 目录里写入了 7 个 blob（4+2+1+0 跳过 + 1 失败 = 没写的跳过不计入；broken-1 没写）
//   - tables 列出所有 5 张表
//   - 抽样：chat s1 的 blob 包含 messages 数组且按 seq 排序
//   - 抽样：note n1 的 blob 是 markdown，含 frontmatter
func TestRunWithRealFixture(t *testing.T) {
	root := t.TempDir()
	srcDB := buildOldSQLiteFixture(t, root)
	toDir := filepath.Join(root, "new")
	if err := os.MkdirAll(toDir, 0o755); err != nil {
		t.Fatal(err)
	}

	reportPath := filepath.Join(toDir, "migrate-report.json")
	exitCode, err := run(srcDB, toDir, reportPath)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	// cards 有 1 条 failed，totalOK>0 且 failed>0 → exit 1
	if exitCode != 1 {
		t.Fatalf("exitCode=%d, want 1 (partial failure)", exitCode)
	}

	// 1) 报告必须存在并可解析
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var rep Report
	if err := json.Unmarshal(data, &rep); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	if rep.FromDB != srcDB {
		t.Fatalf("FromDB=%q, want %q", rep.FromDB, srcDB)
	}
	if rep.ToDir != toDir {
		t.Fatalf("ToDir=%q, want %q", rep.ToDir, toDir)
	}
	if rep.Version == "" {
		t.Fatalf("version empty")
	}
	if rep.StartedAt.IsZero() || rep.FinishedAt.IsZero() {
		t.Fatalf("started/finished unset")
	}

	// 2) counts 校验
	wantCounts := map[string]int{
		TableChat:  2,
		TableNote:  2,
		TableQBank: 2,
		TableCard:  1,
	}
	for k, v := range wantCounts {
		if rep.Counts[k] != v {
			t.Errorf("counts[%s]=%d, want %d (full report=%+v)", k, rep.Counts[k], v, rep.Counts)
		}
	}

	// 3) failed / skipped 校验
	if len(rep.Failed) != 1 {
		t.Fatalf("failed len=%d, want 1; got=%+v", len(rep.Failed), rep.Failed)
	}
	if rep.Failed[0].Table != TableCard || rep.Failed[0].ID != "broken-1" {
		t.Errorf("unexpected failed[0]: %+v", rep.Failed[0])
	}
	if len(rep.Skipped) != 3 {
		t.Fatalf("skipped len=%d, want 3; got=%+v", len(rep.Skipped), rep.Skipped)
	}

	// 4) Tables sanity check 应包含所有 5 张表，且行数正确
	rowExpect := map[string]int{
		"chat_history":  3,
		"chat_messages": 4,
		"notes":         3,
		"qbank":         2,
		"cards":         3,
	}
	tableSeen := map[string]int{}
	for _, ts := range rep.Tables {
		tableSeen[ts.Table] = ts.Rows
	}
	for tbl, want := range rowExpect {
		if tableSeen[tbl] != want {
			t.Errorf("table %s: rows=%d, want %d", tbl, tableSeen[tbl], want)
		}
	}

	// 5) 落盘 blob 数 = counts[chat]+counts[notes]+counts[qbank]+counts[cards] = 2+2+2+1 = 7
	blobDir := filepath.Join(toDir, "blob")
	blobs := countBlobs(t, blobDir)
	wantBlobs := 7
	if blobs != wantBlobs {
		t.Errorf("blob files=%d, want %d", blobs, wantBlobs)
	}

	// 6) 抽样：chat s1 的 blob 含 messages 数组，且按 seq 排序
	_, chatBody := findBlobWithPrefix(t, blobDir, `"messages"`)
	if !bytes.Contains(chatBody, []byte(`"hello there"`)) {
		t.Errorf("chat blob missing assistant content: %s", chatBody)
	}
	// 检查 seq 顺序：hi 必须在 hello there 之前
	idxHi := bytes.Index(chatBody, []byte(`"hi"`))
	idxHello := bytes.Index(chatBody, []byte(`"hello there"`))
	if idxHi < 0 || idxHello < 0 || idxHi >= idxHello {
		t.Errorf("chat messages not sorted by seq: hi@%d hello@%d", idxHi, idxHello)
	}

	// 7) 抽样：note n1 是 markdown 含 frontmatter
	_, noteBody := findBlobWithPrefix(t, blobDir, "body one")
	if !bytes.HasPrefix(noteBody, []byte("---\ntitle: note one\n---\n")) {
		t.Errorf("note n1 missing frontmatter: %s", noteBody)
	}

	// 8) 抽样：qbank q1 含 questions 数组
	_, qBody := findBlobWithPrefix(t, blobDir, `"1+1?"`)
	if !bytes.Contains(qBody, []byte(`"answer":"2"`)) {
		t.Errorf("qbank q1 missing answer: %s", qBody)
	}

	// 9) 抽样：card c1 的 blob
	_, cardBody := findBlobWithPrefix(t, blobDir, `"front1"`)
	if !bytes.Contains(cardBody, []byte(`"back1"`)) {
		t.Errorf("card c1 missing back1: %s", cardBody)
	}

	// 10) 报告序列化稳定性：counts 必须是 map，failed/skipped 是 slice
	raw, _ := json.Marshal(rep)
	for _, k := range []string{`"counts":`, `"failed":`, `"skipped":`, `"tables":`} {
		if !bytes.Contains(raw, []byte(k)) {
			t.Errorf("report json missing key %s: %s", k, raw)
		}
	}
}

// TestMigrateMissingDB 旧 db 不存在：必须返回 exit 2 + 报告里有 open 失败条目。
// 注意：run() 不把"打不开旧库"当 Go 错误抛出，而是把它记在报告里。
func TestMigrateMissingDB(t *testing.T) {
	root := t.TempDir()
	toDir := filepath.Join(root, "new")
	_ = os.MkdirAll(toDir, 0o755)
	reportPath := filepath.Join(toDir, "r.json")
	exitCode, err := run(filepath.Join(root, "nope.db"), toDir, reportPath)
	if err != nil {
		t.Fatalf("run err=%v, want nil (错误应该写进报告)", err)
	}
	if exitCode != 2 {
		t.Fatalf("exitCode=%d, want 2", exitCode)
	}
	// 报告里要有 open 失败
	data, _ := os.ReadFile(reportPath)
	var rep Report
	_ = json.Unmarshal(data, &rep)
	if len(rep.Failed) != 1 || rep.Failed[0].Table != "*" {
		t.Fatalf("expected 1 failed with table=*, got=%+v", rep.Failed)
	}
	if !strings.Contains(rep.Failed[0].Error, "open old db") {
		t.Errorf("failed[0].Error missing 'open old db': %q", rep.Failed[0].Error)
	}
}

// TestMigrateEmptyDB 旧 db 存在但没有任何目标表：counts 全 0，跳过 4 个表，exit 2。
func TestMigrateEmptyDB(t *testing.T) {
	root := t.TempDir()
	srcPath := filepath.Join(root, "empty.db")
	db, err := sql.Open("sqlite", "file:"+srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE other (id INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO other VALUES (1)`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	toDir := filepath.Join(root, "new")
	_ = os.MkdirAll(toDir, 0o755)
	reportPath := filepath.Join(toDir, "r.json")
	exitCode, err := run(srcPath, toDir, reportPath)
	if err != nil {
		t.Fatal(err)
	}
	if exitCode != 2 {
		t.Fatalf("exitCode=%d, want 2", exitCode)
	}
	data, _ := os.ReadFile(reportPath)
	var rep Report
	_ = json.Unmarshal(data, &rep)
	if len(rep.Skipped) != 4 {
		t.Errorf("skipped len=%d, want 4 (chat/notes/qbank/cards); got=%+v", len(rep.Skipped), rep.Skipped)
	}
	for _, s := range rep.Skipped {
		if !strings.HasPrefix(s.Table, "chat") && s.Table != TableNote && s.Table != TableQBank && s.Table != TableCard {
			t.Errorf("unexpected skipped table: %+v", s)
		}
	}
}

// TestParseReportVersionField 报告里的 version 字段是稳定的字符串。
func TestParseReportVersionField(t *testing.T) {
	root := t.TempDir()
	srcDB := buildOldSQLiteFixture(t, root)
	toDir := filepath.Join(root, "new")
	_ = os.MkdirAll(toDir, 0o755)
	if _, err := run(srcDB, toDir, filepath.Join(toDir, "r.json")); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(toDir, "r.json"))
	var rep Report
	if err := json.Unmarshal(data, &rep); err != nil {
		t.Fatal(err)
	}
	if rep.Version != "1.0" {
		t.Errorf("version=%q, want 1.0", rep.Version)
	}
}
