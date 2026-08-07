// Command smoke 跑一遍 13 项能力的最小闭环断言，注入 mock LLM/embedder/MCP。
// 退出码：0 全部通过；非 0 失败。
//
// 用法：
//
//	go run ./scripts/smoke
//
// 设计：
//   - 在临时目录建数据，所有写盘动作都隔离
//   - 用脚本式 mock LLM/embedder（基于 system prompt 路由返回 JSON）
//   - 每个能力只走最简 happy-path 一次，断言资源已创建且内容非空
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/helixnow/deep-student-go/internal/anki"
	"github.com/helixnow/deep-student-go/internal/chat"
	"github.com/helixnow/deep-student-go/internal/essay"
	"github.com/helixnow/deep-student-go/internal/governance"
	"github.com/helixnow/deep-student-go/internal/hub"
	"github.com/helixnow/deep-student-go/internal/llmusage"
	"github.com/helixnow/deep-student-go/internal/memory"
	"github.com/helixnow/deep-student-go/internal/multimodal"
	"github.com/helixnow/deep-student-go/internal/ocr"
	"github.com/helixnow/deep-student-go/internal/mindmap"
	"github.com/helixnow/deep-student-go/internal/paper"
	"github.com/helixnow/deep-student-go/internal/pomodoro"
	"github.com/helixnow/deep-student-go/internal/qbank"
	"github.com/helixnow/deep-student-go/internal/reader"
	"github.com/helixnow/deep-student-go/internal/research"
	"github.com/helixnow/deep-student-go/internal/skills"
	"github.com/helixnow/deep-student-go/internal/sync"
	"github.com/helixnow/deep-student-go/internal/templatemgr"
	"github.com/helixnow/deep-student-go/internal/todo"
	"github.com/helixnow/deep-student-go/internal/translate"
	"github.com/helixnow/deep-student-go/pkg/config"
	"github.com/helixnow/deep-student-go/pkg/crypto"
	"github.com/helixnow/deep-student-go/pkg/eventbus"
	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/logger"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// ------------------- mock LLM -------------------

// mockProv 根据 system prompt 路由返回不同的 JSON / 文本，专门为冒烟测试设计。
type mockProv struct {
	calls atomic.Int64
}

// Name 既要充当 "openai"（被各服务查找），也要能被 "mock" 找到（被 Chat 显式选 mock）。
// 用单实例：默认对外 name = "openai"，让各服务按约定查找命中。
func (m *mockProv) Name() string { return "openai" }

func (m *mockProv) Chat(_ context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	m.calls.Add(1)
	return &llm.ChatResponse{
		Content: routeContent(req.Messages),
		Usage:   llm.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
	}, nil
}

func (m *mockProv) Stream(_ context.Context, req llm.ChatRequest) (<-chan llm.Chunk, error) {
	m.calls.Add(1)
	ch := make(chan llm.Chunk, 1)
	ch <- llm.Chunk{Delta: routeContent(req.Messages), Done: true}
	close(ch)
	return ch, nil
}

func (m *mockProv) Embed(_ context.Context, req llm.EmbedRequest) (*llm.EmbedResponse, error) {
	m.calls.Add(1)
	out := make([][]float32, len(req.Input))
	for i := range out {
		// 简单确定性向量：第 i 维 = float32(i+1)/float32(dim+1)
		v := make([]float32, 8)
		for j := range v {
			v[j] = float32(i+1) * 0.1 * float32(j+1) * 0.01
		}
		out[i] = v
	}
	return &llm.EmbedResponse{Embeddings: out}, nil
}

// routeContent 根据 system prompt 决定返回内容。
func routeContent(msgs []llm.Message) string {
	sys := ""
	for _, m := range msgs {
		if m.Role == llm.RoleSystem {
			sys += m.Content + "\n"
		}
	}
	sysLower := strings.ToLower(sys)
	if strings.Contains(sysLower, "translate") || strings.Contains(sysLower, "translator") {
		fmt.Fprintf(os.Stderr, "[debug] translate route hit, sys=%q\n", sys)
	}
	switch {
	case strings.Contains(sysLower, "anki card generator"):
		return `[{"front":"Q1","back":"A1","tags":["t"]},{"front":"Q2","back":"A2","tags":["t"]}]`
	case strings.Contains(sysLower, "translator") || strings.Contains(sysLower, "translate"):
		return "[MOCK-TRANSLATION] " + tailUser(msgs)
	case strings.Contains(sysLower, "extract durable facts"):
		return `["User likes studying","User is preparing for exam"]`
	case strings.Contains(sysLower, "decide whether the new fact"):
		return `{"action":"ADD","target_id":"","content":"new fact"}`
	case strings.Contains(sysLower, "polish") || strings.Contains(sysLower, "essay") || strings.Contains(sysLower, "essay_grader") || strings.Contains(sysLower, "grade"):
		// 作文批改：构造合法 essay.Result JSON
		return essayMockJSON()
	case strings.Contains(sysLower, "knowledge mapper") || strings.Contains(sysLower, "mind map") || strings.Contains(sysLower, "outline") || strings.Contains(sysLower, "node"):
		return mindmapMockJSON()
	case strings.Contains(sysLower, "question") && strings.Contains(sysLower, "json"):
		// QBank.Extract
		return qbankMockJSON()
	case strings.Contains(sysLower, "continue") || strings.Contains(sysLower, "writing assistant"):
		return " [mock-continuation] " + tailUser(msgs)
	default:
		return "[mock-reply] " + tailUser(msgs)
	}
}

func tailUser(msgs []llm.Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == llm.RoleUser {
			c := msgs[i].Content
			if len(c) > 80 {
				c = c[:80]
			}
			return c
		}
	}
	return ""
}

func essayMockJSON() string {
	r := map[string]any{
		"scenario": "gaokao",
		"polished": "Polished essay text",
		"dimensions": []map[string]any{
			{"name": "content", "score": 22, "weight": 0.4, "note": "ok"},
			{"name": "language", "score": 18, "weight": 0.3, "note": "ok"},
			{"name": "structure", "score": 16, "weight": 0.3, "note": "ok"},
		},
		"total":       56,
		"suggestions": []string{"Add more examples", "Tighten conclusion"},
		"highlights":  []string{"Strong opening"},
	}
	b, _ := json.Marshal(r)
	return string(b)
}

func mindmapMockJSON() string {
	// 严格按 mindmap.parseRoot 的期望：包一层 {"root": {...}}
	m := map[string]any{
		"root": map[string]any{
			"id":    "root",
			"topic": "Photosynthesis",
			"children": []map[string]any{
				{"id": "1", "topic": "Light reactions", "children": []map[string]any{
					{"id": "1a", "topic": "Thylakoid"},
					{"id": "1b", "topic": "ATP synthase"},
				}},
				{"id": "2", "topic": "Calvin cycle", "children": []map[string]any{
					{"id": "2a", "topic": "RuBisCO"},
				}},
			},
		},
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func qbankMockJSON() string {
	// 严格按 qbank.Extract 的期望：包一层 {"questions": [...]}
	q := map[string]any{
		"questions": []map[string]any{
			{
				"id": "q1", "stem": "1+1=?", "options": []string{"1", "2", "3"}, "answer": "2",
				"type": "single", "knowledge": []string{"math.basic"},
			},
			{
				"id": "q2", "stem": "Capital of France?", "options": []string{"Paris", "London", "Berlin"}, "answer": "Paris",
				"type": "single", "knowledge": []string{"geography.europe"},
			},
		},
	}
	b, _ := json.Marshal(q)
	return string(b)
}

// ------------------- main -------------------

type runner struct {
	ctx   context.Context
	app   *app
	calls atomic.Int64
}

type app struct {
	Hub     *hub.Service
	Chat    *chat.Service
	Mind    *mindmap.Service
	QBank   *qbank.Service
	Anki    *anki.Service
	Reader  *reader.Service
	Trans   *translate.Service
	Essay   *essay.Service
	Res     *research.Service
	Paper   *paper.Service
	Mem     *memory.Service
	Skills  *skills.Service
	Gov     *governance.Service
	Todo    *todo.Service
	Pomodoro *pomodoro.Service
	LLMUsage *llmusage.Service
	Templates *templatemgr.Service
	Sync     *sync.Service
	OCR      *ocr.Service
	Multi    *multimodal.Service
	vfs     *vfs.FS
}

// VaultDir 返回 vault 根目录。
func (a *app) VaultDir() string {
	if a.vfs == nil {
		return ""
	}
	return a.vfs.VaultDir()
}

// vfsPut 通过 vfs 写入资源。
func (a *app) vfsPut(uri string, data []byte, meta map[string]string) (vfs.Entry, error) {
	return a.vfs.Put(uri, data, meta)
}

// StatVFS 查询资源元数据。
func (a *app) StatVFS(uri string) (vfs.Entry, bool) {
	return a.vfs.Stat(uri)
}

// VFSLinks 返回出链。
func (a *app) VFSLinks(uri string) []vfs.LinkEntry { return a.vfs.Links(uri) }

// RunSmoke 执行一次完整的冒烟：boot + walk。返回 error 表示失败，
// 退出码 0 表示全部通过。main() 与 smoke_test.go 都通过它跑断言，
// 避免在两处重复 happy-path 逻辑。
func RunSmoke(ctx context.Context) error {
	r := &runner{ctx: ctx}
	if err := r.boot(); err != nil {
		return fmt.Errorf("boot: %w", err)
	}
	r.walk()
	return nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := RunSmoke(ctx); err != nil {
		fail("smoke", err)
	}
	fmt.Println("SMOKE OK")
}

func fail(stage string, err error) {
	fmt.Fprintf(os.Stderr, "SMOKE FAIL [%s]: %v\n", stage, err)
	os.Exit(1)
}

func (r *runner) boot() error {
	tmp, err := os.MkdirTemp("", "deepstudent-smoke-*")
	if err != nil {
		return err
	}
	// 把数据目录指向临时目录
	os.Setenv("DEEPSTUDENT_DATA", tmp)
	os.Setenv("DEEPSTUDENT_CACHE", filepath.Join(tmp, "cache"))
	os.Setenv("DEEPSTUDENT_LOG", filepath.Join(tmp, "logs"))
	os.Setenv("DEEPSTUDENT_BACKUP", filepath.Join(tmp, "backups"))
	// config 是单例；测试/冒烟走自己的 init 流程。
	config.Reset()
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load cfg: %w", err)
	}
	if err := logger.Init(cfg.LogDir, "error"); err != nil {
		return fmt.Errorf("logger: %w", err)
	}
	st, err := store.Open(filepath.Join(cfg.DataDir, "deepstudent.db"))
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	bs, err := blob.New(filepath.Join(cfg.DataDir, "blob"))
	if err != nil {
		return fmt.Errorf("blob: %w", err)
	}
	// Obsidian 式 vault 文件系统：内容落盘真实文件，可验证落盘与双链
	fs, err := vfs.NewVaultFS(cfg.VaultDir, bs)
	if err != nil {
		return fmt.Errorf("vault vfs: %w", err)
	}
	cry, err := crypto.NewManager(filepath.Join(cfg.DataDir, "keys"))
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}
	bus := eventbus.New()
	// 各服务都按 "openai" 取 LLM；mock 复用该 name 即可走通所有能力。
	reg := llm.NewRegistry()
	prov := &mockProv{}
	reg.Register(prov)
	r.app = &app{
		Hub:     hub.New(fs, st, reg),
		Chat:    chat.New(fs, st, reg, bus),
		Mind:    mindmap.New(fs, reg),
		QBank:   qbank.New(fs, st, reg),
		Anki:    anki.New(fs, reg, bus),
		Reader:  reader.New(fs, reg),
		Trans:   translate.New(fs, reg),
		Essay:   essay.New(fs, reg),
		Res:     research.New(fs, reg, bus),
		Mem:     memory.New(fs, st, reg),
		Skills:  skills.New(fs, st, reg, bus),
		Gov:     governance.New(fs, st, cry, cfg, bus),
		Todo:    todo.New(fs, st, reg),
		Pomodoro: pomodoro.New(fs, st, reg),
		LLMUsage: llmusage.New(fs, st, reg),
		Templates: templatemgr.New(fs, st, reg),
		Sync:     sync.New(st),
		OCR:      ocr.New("smoke-key"),
		Multi:    multimodal.New(st, reg, fs),
		vfs:     fs,
	}
	if err := r.app.Sync.EnsureTriggers(); err != nil {
		return fmt.Errorf("sync triggers: %w", err)
	}
	// 注册 3 个 demo tool，验证 Tools() / 工具注册路径
	demoEcho := []byte(`{"type":"object","properties":{"msg":{"type":"string"}}}`)
	for _, name := range []string{"echo", "now", "ping"} {
		nm := name
		r.app.Skills.RegisterTool(skills.ToolBinding{
			Name:   nm,
			Desc:   "demo tool: " + nm,
			Schema: demoEcho,
			Handler: func(_ context.Context, args json.RawMessage) (any, error) {
				return map[string]any{"tool": nm, "args": string(args)}, nil
			},
		})
	}
	return nil
}

func (r *runner) walk() {
	ctx := r.ctx
	// 1. Hub: 导入一个 txt + 一个 md，然后列表
	txtURI, err := r.app.Hub.ImportResource(ctx, vfs.TypeNote, "note-1", []byte("hello world from smoke test"), []string{"demo"})
	must("hub.import.txt", err)
	mdURI, err := r.app.Hub.ImportResource(ctx, vfs.TypeNote, "note-2", []byte("# title\nbody text\n"), []string{"demo", "math"})
	must("hub.import.md", err)
	notes := r.app.Hub.List(vfs.TypeNote)
	mustOK("hub.list", len(notes) >= 2, fmt.Sprintf("got %d notes", len(notes)))
	// 引用其中一个 URI 到续写
	ch, err := r.app.Hub.ContinueNote(ctx, txtURI, "继续写下一句")
	must("hub.continue", err)
	got := drainStrings(ch)
	mustOK("hub.continue.content", strings.Contains(got, "mock-continuation"), "got="+got)
	_ = mdURI

	// 2. Chat: 创 group/session/send/branch
	g := r.app.Chat.CreateGroup("g1", "you are a tutor", "", []string{"demo"})
	se := r.app.Chat.CreateSession(g.ID, "first", "gpt-4o-mini", "openai")
	cch, err := r.app.Chat.Send(ctx, se.ID, "1+1=?", nil, false)
	must("chat.send", err)
	smokeCollected := drainStrings(cch)
	mustOK("chat.send.content", smokeCollected != "", "empty reply")
	// branch
	last := se.Messages[len(se.Messages)-1]
	br, err := r.app.Chat.Branch(se.ID, last.ID)
	must("chat.branch", err)
	mustOK("chat.branch.id", br.ID != "", "empty branch id")
	// compare
	cmp := r.app.Chat.Compare(ctx, se.ID, "test", []string{"openai"})
	fmt.Printf("    compare debug: cmp=%v\n", cmp)
	mustOK("chat.compare", len(cmp) >= 1, fmt.Sprintf("cmp=%+v", cmp))

	// 3. Mindmap: generate + save + load
	mm, err := r.app.Mind.Generate(ctx, "Photosynthesis")
	must("mindmap.generate", err)
	mustOK("mindmap.root", mm.Root != nil && mm.Root.Topic == "Photosynthesis", "topic wrong")
	mmURI, err := r.app.Mind.Save(mm)
	must("mindmap.save", err)
	mm2, err := r.app.Mind.Load(mmURI)
	must("mindmap.load", err)
	mustOK("mindmap.load.match", mm2.Title == mm.Title, "titles differ")
	outline := mm2.ToOutline()
	mustOK("mindmap.outline", strings.Contains(outline, "Photosynthesis"), "outline missing root")
	mm2.Mask(0.5)
	mustOK("mindmap.mask", mm2.Root.Masked, "mask not applied")

	// 4. QBank: extract + start + answer + submit + mastery
	// 先准备一个 source doc
	srcURI, err := r.app.Hub.ImportResource(ctx, vfs.TypeTextbook, "math-101", []byte("Some math content."), []string{"math"})
	must("qbank.src", err)
	set, err := r.app.QBank.Extract(ctx, srcURI, "Math Quiz")
	must("qbank.extract", err)
	mustOK("qbank.extract.questions", len(set.Questions) >= 2, "questions not extracted")
	att, err := r.app.QBank.StartAttempt(set.ID)
	must("qbank.attempt", err)
	for _, q := range set.Questions {
		must("qbank.answer", r.app.QBank.Answer(att.ID, q.ID, q.Answer))
	}
	fin, err := r.app.QBank.Submit(att.ID)
	must("qbank.submit", err)
	mustOK("qbank.submit.score", fin.Score == len(set.Questions), fmt.Sprintf("score=%d", fin.Score))
	m := r.app.QBank.Mastery()
	mustOK("qbank.mastery", len(m) >= 1, "no mastery")

	// 5. Anki: generate + save + export
	job, err := r.app.Anki.GenerateFromText(ctx, "smoke-deck", "Photosynthesis is the process...", "", 2, nil)
	must("anki.generate", err)
	mustOK("anki.cards", len(job.Cards) >= 1, fmt.Sprintf("got %d cards", len(job.Cards)))
	ankiURI, err := r.app.Anki.SaveToVFS(job)
	must("anki.save", err)
	apkg, err := r.app.Anki.ExportAPKG(job)
	must("anki.export", err)
	mustOK("anki.apkg", len(apkg) > 10, "apkg too small")
	_ = ankiURI

	// 6. Reader: open text + inject
	doc, err := r.app.Reader.Open(txtURI)
	must("reader.open", err)
	mustOK("reader.pages", len(doc.Pages) >= 1, "no pages")
	prompt := r.app.Reader.InjectToChat(doc, 1, 1, "selected text")
	mustOK("reader.inject", strings.Contains(prompt, "Selected:") || strings.Contains(prompt, "Page 1"), "prompt wrong")

	// 7. Translate: text + domain + glossary
	res, err := r.app.Trans.Translate(ctx, translate.Request{
		Text: "Hello world", Source: "en", Target: "zh",
		Domain:   translate.DomainTech,
		Glossary: []translate.GlossaryEntry{{Source: "world", Target: "世界"}},
	})
	must("translate", err)
	mustOK("translate.content", strings.Contains(res.Text, "MOCK-TRANSLATION"), "got="+res.Text)
	// document translation
	transURI, err := r.app.Trans.TranslateDocument(ctx, txtURI, "en", "zh", translate.DomainGeneral)
	must("translate.doc", err)
	mustOK("translate.doc.uri", strings.HasPrefix(transURI, "vfs://translation/"), "uri="+transURI)

	// 8. Essay: grade + save
	gr, err := r.app.Essay.Grade(ctx, "My essay content", essay.ScenarioGaokao, []string{"content", "language"})
	must("essay.grade", err)
	mustOK("essay.polished", gr.Polished == "Polished essay text", "polished="+gr.Polished)
	mustOK("essay.total", gr.Total > 0, "total not computed")
	essayURI, err := r.app.Essay.Save(gr)
	must("essay.save", err)
	mustOK("essay.uri", strings.HasPrefix(essayURI, "vfs://translation/"), "uri="+essayURI)

	// 9. Research: 调 ConfirmAndPlan 走 LLM 路径，断言返回的 plan 不为空
	//    （不调 Run —— Run 会进一步 Search/analyze；本冒烟只验 plan 链路能跑通）
	plan, err := r.app.Res.ConfirmAndPlan(ctx, "Transformer 架构演进", "deep", "outline")
	must("research.plan", err)
	mustOK("research.plan.steps", plan != nil && len(plan.Steps) >= 1, fmt.Sprintf("plan=%+v", plan))

	// 10. Paper: 不发网络；只验 cite 函数
	src := paper.Source{ID: "2301.00001", Title: "Mock Paper", Authors: []string{"Smith, J."}, Year: 2023, Venue: "arXiv", DOI: "10.0000/mock"}
	cited := paper.Cite(src, "bibtex")
	mustOK("paper.cite", strings.Contains(cited, "@") || strings.Contains(cited, "Smith"), "cite="+cited)

	// 11. Memory: ingest + search + profile + privacy
	items, err := r.app.Mem.Ingest(ctx, "I am a student preparing for the gaokao exam")
	must("memory.ingest", err)
	mustOK("memory.items", len(items) >= 1, fmt.Sprintf("items=%d", len(items)))
	_ = r.app.Mem.Search("exam")
	prof := r.app.Mem.Profile()
	mustOK("memory.profile", prof != "", "empty profile")
	r.app.Mem.PrivacyMode(true)
	r.app.Mem.Decay()

	// 12. Skills: list + tools + builtin call
	allSkills := r.app.Skills.Skills()
	mustOK("skills.list", len(allSkills) >= 3, fmt.Sprintf("skills=%d", len(allSkills)))
	tools := r.app.Skills.Tools()
	mustOK("skills.tools", len(tools) >= 3, fmt.Sprintf("tools=%d", len(tools)))

	// 13. Governance: backup + export/import + audit + integrity
	backupPath := filepath.Join(os.TempDir(), fmt.Sprintf("smoke-backup-%d.zip", time.Now().UnixNano()))
	out, err := r.app.Gov.Backup(backupPath)
	must("gov.backup", err)
	mustOK("gov.backup.path", out != "", "empty backup path")
	st2 := r.app.Gov.Status()
	mustOK("gov.status", st2 != nil, "status nil")
	auditRows, _ := r.app.Gov.AuditLogs(10)
	mustOK("gov.audit", len(auditRows) >= 1, "no audit rows")
	_ = r.app.Gov.CheckIntegrity()

	// 14. Todo: ensure inbox + list CRUD + item CRUD + toggle + view
	inbox, err := r.app.Todo.EnsureInbox()
	must("todo.inbox", err)
	mustOK("todo.inbox.flag", inbox.IsInbox, "inbox flag")
	lst, err := r.app.Todo.CreateList(todo.CreateListParams{Name: "学习"})
	must("todo.list.create", err)
	it, err := r.app.Todo.CreateItem(todo.CreateItemParams{ListID: lst.ID, Title: "读论文"})
	must("todo.item.create", err)
	_, err = r.app.Todo.CreateItem(todo.CreateItemParams{ListID: lst.ID, Title: "写总结"})
	must("todo.item.create2", err)
	toggled, err := r.app.Todo.ToggleItem(it.ID)
	must("todo.toggle", err)
	mustOK("todo.toggle.done", toggled.CompletedAt != nil, "not completed")
	sum, err := r.app.Todo.Summary()
	must("todo.summary", err)
	mustOK("todo.summary.pending", sum != nil && sum.TotalPending >= 1, fmt.Sprintf("summary=%+v", sum))
	_ = r.app.Todo.PurgeList(lst.ID)

	// 15. Pomodoro: create + stats
	prec, err := r.app.Pomodoro.Create(pomodoro.CreateParams{ActualDuration: 1500})
	must("pomodoro.create", err)
	mustOK("pomodoro.duration", prec.Duration == pomodoro.DefaultWorkSeconds, fmt.Sprintf("duration=%d", prec.Duration))
	pstats, err := r.app.Pomodoro.TodayStats()
	must("pomodoro.stats", err)
	mustOK("pomodoro.stats.count", pstats.CompletedCount >= 1, fmt.Sprintf("stats=%+v", pstats))
	pdaily, err := r.app.Pomodoro.DailyStats(7)
	must("pomodoro.daily", err)
	mustOK("pomodoro.daily.count", len(pdaily) >= 1, fmt.Sprintf("daily=%d", len(pdaily)))

	// 16. LLM usage: record + query + summary
	ulog, err := r.app.LLMUsage.Record(llmusage.LogEntry{
		Provider: "openai", Model: "gpt-4o-mini",
		PromptTokens: 100, CompletionTokens: 20,
		CallerType: "smoke",
	})
	must("llmusage.record", err)
	mustOK("llmusage.total", ulog.TotalTokens == 120, fmt.Sprintf("total=%d", ulog.TotalTokens))
	ulogs, err := r.app.LLMUsage.Query(llmusage.LogFilter{CallerType: "smoke"})
	must("llmusage.query", err)
	mustOK("llmusage.query.count", len(ulogs) >= 1, fmt.Sprintf("logs=%d", len(ulogs)))
	usum, err := r.app.LLMUsage.Summary()
	must("llmusage.summary", err)
	mustOK("llmusage.summary.req", usum.TotalRequests >= 1, fmt.Sprintf("summary=%+v", usum))

	// 17. Vault: 资源真实落盘 + 双链解析
	vaultDir := r.app.VaultDir()
	_, err = r.app.vfsPut("vfs://note/vault-source", []byte("参见 [[目标笔记]]"), map[string]string{"title": "源笔记"})
	must("vault.put", err)
	_, err = r.app.vfsPut("vfs://note/vault-target", []byte("内容"), map[string]string{"title": "目标笔记"})
	must("vault.put2", err)
	fp := ""
	if e, ok := r.app.StatVFS("vfs://note/vault-source"); ok {
		fp = e.FilePath
	}
	mustOK("vault.file.exists", fp != "" && fileExists(fp), fmt.Sprintf("file=%s", fp))
	links := r.app.VFSLinks("vfs://note/vault-source")
	mustOK("vault.links", len(links) == 1 && links[0].TargetURI == "vfs://note/vault-target", fmt.Sprintf("links=%+v", links))
	mustOK("vault.dir", vaultDir != "", "empty vault dir")

	// 18. Templates: 内置 seed + CRUD + 默认模板
	tpls, err := r.app.Templates.List()
	must("templates.list", err)
	mustOK("templates.builtins", len(tpls) >= 4, fmt.Sprintf("templates=%d", len(tpls)))
	nt, err := r.app.Templates.Create(templatemgr.CreateParams{Name: "冒烟模板", FrontTmpl: "F", BackTmpl: "B"})
	must("templates.create", err)
	mustOK("templates.created", nt.ID != "", "empty id")
	if err := r.app.Templates.SetDefault(nt.ID); err != nil {
		must("templates.setdefault", err)
	}
	defID, err := r.app.Templates.DefaultID()
	must("templates.defaultid", err)
	mustOK("templates.default", defID == nt.ID, fmt.Sprintf("default=%s", defID))

	// 19. Sync: 变更日志触发器 + 导出/应用 + 隔离区
	cur, cerr := r.app.Sync.Cursor()
	must("sync.cursor", cerr)
	max, merr := r.app.Sync.MaxSeq()
	must("sync.maxseq", merr)
	mustOK("sync.status.hasCursor", max >= cur, fmt.Sprintf("cur=%d max=%d", cur, max))
	pending := max - cur
	mustOK("sync.pending.nonneg", pending >= 0, fmt.Sprintf("pending=%d", pending))
	// 触发一个变更（todo），应产生日志
	smLst, err := r.app.Todo.CreateList(todo.CreateListParams{Name: "同步冒烟"})
	must("sync.todo", err)
	_ = smLst
	qc, err := r.app.Sync.QuarantineList(10)
	must("sync.quarantine", err)
	mustOK("sync.quarantine.empty", len(qc) == 0, fmt.Sprintf("quarantine=%d", len(qc)))

	// 20. OCR: 引擎列表 + 无 key 时 VL 识别报错（不走网络）
	engines := r.app.OCR.ListEngines()
	mustOK("ocr.engines", len(engines) == 3, fmt.Sprintf("engines=%d", len(engines)))
	r.app.OCR.APIKey = ""
	_, oerr := r.app.OCR.Recognize(ctx, []byte("img"), "image/png")
	mustOK("ocr.nokey", oerr != nil, "expected error without key")

	// 21. Multimodal: 索引 + 关键词检索 + 统计
	mmN, err := r.app.Multi.IndexResource(ctx, "vfs://note/mm-1", "多模态索引测试内容：机器学习与深度学习。")
	must("multimodal.index", err)
	mustOK("multimodal.chunks", mmN >= 1, fmt.Sprintf("chunks=%d", mmN))
	mmResults, err := r.app.Multi.Search(ctx, "机器学习", 5)
	must("multimodal.search", err)
	mustOK("multimodal.results", len(mmResults) >= 1, fmt.Sprintf("results=%d", len(mmResults)))
	mmStats, err := r.app.Multi.Stats()
	must("multimodal.stats", err)
	mustOK("multimodal.units", mmStats.TotalUnits >= 1, fmt.Sprintf("units=%d", mmStats.TotalUnits))

	// 22. chat_v2: 会话持久化 + 标签 + 工具循环
	v2g := r.app.Chat.CreateGroup("v2g", "assistant", "", nil)
	v2se := r.app.Chat.CreateSession(v2g.ID, "v2会话", "gpt-4o-mini", "openai")
	mustOK("chatv2.session", v2se.ID != "", "empty session")
	if err := r.app.Chat.UpdateSessionTags(v2se.ID, []string{"标签A"}); err != nil {
		must("chatv2.tags", err)
	}
	r.app.Chat.RegisterTool("smoke_tool", func(_ context.Context, args string) (any, error) {
		return map[string]any{"echo": args}, nil
	})
	mustOK("chatv2.tools", len(r.app.Chat.Tools()) >= 1, "no tools")
	v2reply, _, err := r.app.Chat.SendWithTools(ctx, v2se.ID, "你好", nil, nil)
	must("chatv2.send", err)
	mustOK("chatv2.reply", v2reply != "", "empty reply")
	sess, err := r.app.Chat.GetSession(v2se.ID)
	must("chatv2.get", err)
	mustOK("chatv2.messages", sess != nil && len(sess.Messages) >= 2, fmt.Sprintf("messages=%d", len(sess.Messages)))
	// 持久化：同一 db 重启后仍可读（通过内存加载验证）
	v2count, err := r.app.Chat.CountSessions()
	must("chatv2.count", err)
	mustOK("chatv2.count.n", v2count >= 1, fmt.Sprintf("count=%d", v2count))
}

// fileExists 判断文件是否存在。
func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// ------------------- helpers -------------------

func mustOK(stage string, ok bool, info string) {
	if !ok {
		fail(stage, fmt.Errorf("assertion failed: %s", info))
	}
	fmt.Printf("  [ok] %s\n", stage)
}

func must(stage string, err error) {
	if err != nil {
		fail(stage, err)
	}
}

func drainStrings(ch <-chan string) string {
	var sb strings.Builder
	for s := range ch {
		sb.WriteString(s)
	}
	return sb.String()
}
