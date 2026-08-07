// Command migrate 把旧版 Tauri (Rust) DeepStudent 的 SQLite 数据迁移到新 Go 项目的
// vfs:// 命名空间（chat/notes/qbank/cards）。
//
// 用法：
//
//	deepstudent migrate --from <old.db> --to <new-data-dir> [--report <path>]
//
// 报告写入到 <to>/migrate-report.json（可通过 --report 覆盖）。
// 退出码：0=全成功；1=部分失败（仍有成功项）；2=参数错或全部失败。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

func main() {
	from := flag.String("from", "", "source old sqlite file (e.g. ~/.deepstudent/deepstudent.db)")
	to := flag.String("to", "", "target go data dir (e.g. ~/Documents/deepstudent-go)")
	report := flag.String("report", "", "report json path (default: <to>/migrate-report.json)")
	flag.Parse()

	if *from == "" || *to == "" {
		fmt.Fprintln(os.Stderr, "usage: migrate --from <old.db> --to <new-data-dir> [--report <path>]")
		os.Exit(2)
	}

	exitCode, err := run(*from, *to, *report)
	if err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
	}
	os.Exit(exitCode)
}

// run 负责拉起 store/blob/vfs，跑 4 个迁移动作，写报告。
// 返回 (exitCode, err)：err 表示顶层 fatal（如旧 DB 没法打开）；
// exitCode 是进程退出码。
func run(from, to, report string) (int, error) {
	if err := os.MkdirAll(to, 0o755); err != nil {
		return 2, fmt.Errorf("mkdir to-dir: %w", err)
	}
	if report == "" {
		report = filepath.Join(to, "migrate-report.json")
	}
	tmpReport := report + ".tmp"

	rep := &Report{
		Version:   "1.0",
		StartedAt: time.Now(),
		FromDB:    from,
		ToDir:     to,
		Counts:    map[string]int{},
		Failed:    []Failure{},
		Skipped:   []Skip{},
		Tables:    []TableStat{},
	}

	bs, err := blob.New(filepath.Join(to, "blob"))
	if err != nil {
		return 2, fmt.Errorf("blob store: %w", err)
	}
	fs := vfs.NewFS(bs)

	db, err := openSQLite(from)
	if err != nil {
		rep.Failed = append(rep.Failed, Failure{Table: "*", ID: "", Error: "open old db: " + err.Error()})
		return finalize(rep, tmpReport, report)
	}
	defer db.Close()

	// 1) 列出所有表 + 行数（sanity check）。
	listTables(db, rep)

	// 2) 4 个迁移。每个函数内部已经处理了表不存在/字段缺失/单行失败。
	migrators := []struct {
		name string
		fn   func() error
	}{
		{"chat", func() error { return migrateChatHistory(db, fs, rep) }},
		{"notes", func() error { return migrateNotes(db, fs, rep) }},
		{"qbank", func() error { return migrateQBank(db, fs, rep) }},
		{"cards", func() error { return migrateCards(db, fs, rep) }},
	}
	for _, m := range migrators {
		if err := m.fn(); err != nil {
			// 单个迁移动作整体失败（例如表查询失败），记为顶层 failed。
			rep.Failed = append(rep.Failed, Failure{Table: m.name, ID: "", Error: err.Error()})
		}
	}

	return finalize(rep, tmpReport, report)
}

// finalize 写报告并根据 counts/failed 决定退出码。
func finalize(rep *Report, tmp, final string) (int, error) {
	rep.FinishedAt = time.Now()
	out, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return 2, fmt.Errorf("marshal report: %w", err)
	}
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return 2, fmt.Errorf("write tmp report: %w", err)
	}
	if err := os.Rename(tmp, final); err != nil {
		return 2, fmt.Errorf("rename report: %w", err)
	}
	fmt.Println(string(out))

	// 退出码：
	//  - 全部成功（counts>0 且 failed=0）：0
	//  - 部分失败（counts>0 且 failed>0）：1
	//  - 全失败（counts=0）：2
	totalOK := rep.Counts[TableChat] + rep.Counts[TableNote] + rep.Counts[TableQBank] + rep.Counts[TableCard]
	switch {
	case totalOK > 0 && len(rep.Failed) == 0:
		return 0, nil
	case totalOK > 0 && len(rep.Failed) > 0:
		return 1, nil
	default:
		return 2, nil
	}
}
