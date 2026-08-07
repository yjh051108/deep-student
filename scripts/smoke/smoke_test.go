package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/helixnow/deep-student-go/internal/essay"
	"github.com/helixnow/deep-student-go/internal/paper"
	"github.com/helixnow/deep-student-go/internal/translate"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// TestSmoke 跑一次端到端冒烟（13 项能力 + 工具注册），与 `go run ./scripts/smoke`
// 行为等价：全部通过 → 测试通过；任一断言失败 → 测试 fail。
//
// 不依赖网络 / 不依赖真实 LLM：mockProv 已就位。
//
// Task 27.4 要求："go run ./scripts/smoke 在 60 秒内退出 0"。
// 这里用 60s ctx 跑 RunSmoke()，确保 walk 不会卡死；CI 在 -timeout=120s 下面
// 跑本测试是稳的。
func TestSmoke(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := RunSmoke(ctx); err != nil {
		t.Fatalf("RunSmoke: %v", err)
	}
}

// TestMain 重定向临时数据目录（DEEPSTUDENT_DATA），避免污染真实 ~/.deepstudent-go；
// 同时让 go test 进程退码与 go run 退码语义对齐。
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "deepstudent-smoke-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mktemp: %v\n", err)
		os.Exit(1)
	}
	os.Setenv("DEEPSTUDENT_DATA", tmp)
	os.Setenv("DEEPSTUDENT_CACHE", filepath.Join(tmp, "cache"))
	os.Setenv("DEEPSTUDENT_LOG", filepath.Join(tmp, "logs"))
	os.Setenv("DEEPSTUDENT_BACKUP", filepath.Join(tmp, "backups"))
	code := m.Run()
	_ = os.RemoveAll(tmp)
	os.Exit(code)
}

// TestSmokeSubtests 独立验证 13 个能力各自有 happy-path。
// 当 TestSmoke 整体失败时，可以靠 subtest 名称快速定位是哪个能力挂了。
//
// 每个 subtest 自己 boot 一份 app（用独立子目录），互不污染。
func TestSmokeSubtests(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T, r *runner)
	}{
		{"1_hub", subtestHub},
		{"2_chat", subtestChat},
		{"3_mindmap", subtestMindmap},
		{"4_qbank", subtestQBank},
		{"5_anki", subtestAnki},
		{"6_reader", subtestReader},
		{"7_translate", subtestTranslate},
		{"8_essay", subtestEssay},
		{"9_research", subtestResearch},
		{"10_paper", subtestPaper},
		{"11_memory", subtestMemory},
		{"12_skills", subtestSkills},
		{"13_governance", subtestGovernance},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			r := &runner{ctx: ctx}
			if err := r.boot(); err != nil {
				t.Fatalf("boot: %v", err)
			}
			tc.fn(t, r)
		})
	}
}

// ---- 13 个能力 subtest ----

func subtestHub(t *testing.T, r *runner) {
	uri, err := r.app.Hub.ImportResource(r.ctx, vfs.TypeNote, "h1", []byte("hi"), []string{"smoke"})
	if err != nil || uri == "" {
		t.Fatalf("import: uri=%q err=%v", uri, err)
	}
	if got := len(r.app.Hub.List(vfs.TypeNote)); got < 1 {
		t.Fatalf("list: got %d", got)
	}
}

func subtestChat(t *testing.T, r *runner) {
	g := r.app.Chat.CreateGroup("g", "you are a tutor", "", []string{"smoke"})
	se := r.app.Chat.CreateSession(g.ID, "s", "gpt-4o-mini", "openai")
	if se == nil || se.ID == "" {
		t.Fatalf("session: %+v", se)
	}
	ch, err := r.app.Chat.Send(r.ctx, se.ID, "ping", nil, false)
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if drainStrings(ch) == "" {
		t.Fatal("empty reply")
	}
}

func subtestMindmap(t *testing.T, r *runner) {
	m, err := r.app.Mind.Generate(r.ctx, "Photosynthesis")
	if err != nil {
		t.Fatalf("gen: %v", err)
	}
	if m.Root == nil || m.Root.Topic != "Photosynthesis" {
		t.Fatalf("bad root: %+v", m.Root)
	}
}

func subtestQBank(t *testing.T, r *runner) {
	srcURI, err := r.app.Hub.ImportResource(r.ctx, vfs.TypeTextbook, "math-1", []byte("..."), nil)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	set, err := r.app.QBank.Extract(r.ctx, srcURI, "M")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(set.Questions) < 1 {
		t.Fatalf("no questions")
	}
}

func subtestAnki(t *testing.T, r *runner) {
	job, err := r.app.Anki.GenerateFromText(r.ctx, "smoke", "Some text...", "", 1, nil)
	if err != nil {
		t.Fatalf("gen: %v", err)
	}
	if len(job.Cards) < 1 {
		t.Fatalf("no cards")
	}
	apkg, err := r.app.Anki.ExportAPKG(job)
	if err != nil || len(apkg) < 10 {
		t.Fatalf("export: err=%v len=%d", err, len(apkg))
	}
}

func subtestReader(t *testing.T, r *runner) {
	uri, _ := r.app.Hub.ImportResource(r.ctx, vfs.TypeNote, "r1", []byte("hello"), nil)
	doc, err := r.app.Reader.Open(uri)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if len(doc.Pages) < 1 {
		t.Fatalf("no pages")
	}
}

func subtestTranslate(t *testing.T, r *runner) {
	res, err := r.app.Trans.Translate(r.ctx, translate.Request{
		Text: "Hello world", Source: "en", Target: "zh", Domain: translate.DomainGeneral,
	})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	if !strings.Contains(res.Text, "MOCK-TRANSLATION") {
		t.Fatalf("bad text: %q", res.Text)
	}
}

func subtestEssay(t *testing.T, r *runner) {
	res, err := r.app.Essay.Grade(r.ctx, "My essay", essay.ScenarioGaokao, []string{"content"})
	if err != nil {
		t.Fatalf("grade: %v", err)
	}
	if res.Polished == "" {
		t.Fatal("empty polished")
	}
}

func subtestResearch(t *testing.T, r *runner) {
	plan, err := r.app.Res.ConfirmAndPlan(r.ctx, "topic", "deep", "outline")
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan == nil || len(plan.Steps) < 1 {
		t.Fatalf("bad plan: %+v", plan)
	}
}

func subtestPaper(t *testing.T, r *runner) {
	src := paper.Source{ID: "1", Title: "T", Authors: []string{"Smith, J."}, Year: 2023}
	cited := paper.Cite(src, "bibtex")
	if !strings.Contains(cited, "@") {
		t.Fatalf("bad cite: %q", cited)
	}
}

func subtestMemory(t *testing.T, r *runner) {
	items, err := r.app.Mem.Ingest(r.ctx, "I am a student")
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if len(items) < 1 {
		t.Fatalf("no items")
	}
}

func subtestSkills(t *testing.T, r *runner) {
	if got := len(r.app.Skills.Skills()); got < 1 {
		t.Fatalf("no skills: %d", got)
	}
	if got := len(r.app.Skills.Tools()); got < 1 {
		t.Fatalf("no tools: %d", got)
	}
}

func subtestGovernance(t *testing.T, r *runner) {
	tmp, _ := os.MkdirTemp("", "smoke-gov-")
	defer os.RemoveAll(tmp)
	target := filepath.Join(tmp, "backup.zip")
	out, err := r.app.Gov.Backup(target)
	if err != nil || out == "" {
		t.Fatalf("backup: out=%q err=%v", out, err)
	}
}
