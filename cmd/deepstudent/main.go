// Package deepstudent 是 DeepStudent Go 桌面端入口。
package deepstudent

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/helixnow/deep-student-go/internal/anki"
	"github.com/helixnow/deep-student-go/internal/chat"
	"github.com/helixnow/deep-student-go/internal/essay"
	"github.com/helixnow/deep-student-go/internal/governance"
	"github.com/helixnow/deep-student-go/internal/hub"
	"github.com/helixnow/deep-student-go/internal/llmcfg"
	"github.com/helixnow/deep-student-go/internal/memory"
	"github.com/helixnow/deep-student-go/internal/mindmap"
	"github.com/helixnow/deep-student-go/internal/notes"
	"github.com/helixnow/deep-student-go/internal/paper"
	"github.com/helixnow/deep-student-go/internal/qbank"
	"github.com/helixnow/deep-student-go/internal/reader"
	"github.com/helixnow/deep-student-go/internal/research"
	"github.com/helixnow/deep-student-go/internal/skills"
	"github.com/helixnow/deep-student-go/internal/translate"
	"github.com/helixnow/deep-student-go/pkg/config"
	"github.com/helixnow/deep-student-go/pkg/crypto"
	"github.com/helixnow/deep-student-go/pkg/eventbus"
	"github.com/helixnow/deep-student-go/pkg/index"
	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/logger"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
	"github.com/helixnow/deep-student-go/pkg/webui"
)

var assets = webui.Assets

// App 全局应用对象（暴露给 Wails frontend）。
//
// Wails v2 通过反射导出以大写字母开头的方法。
// 13 个领域服务分别绑定为独立 struct，方法名按 PascalCase 暴露。
type App struct {
	Ctx     context.Context
	cfg     *config.Config
	bus     *eventbus.Bus
	store   *store.Store
	blob    *blob.Store
	vfs     *vfs.FS
	crypto  *crypto.Manager
	llmReg  *llm.Registry
	LLMCfg  *llmcfg.Manager
	Hub     *hub.Service
	Chat    *chat.Service
	Mindmap *mindmap.Service
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
	Notes   *notes.Service
	Index   *index.Service
}

// startup Wails 启动钩子。
func (a *App) startup(ctx context.Context) {
	a.Ctx = ctx
	if err := a.init(); err != nil {
		log.Fatalf("init: %v", err)
	}
}

// shutdown Wails 退出钩子。
func (a *App) shutdown(ctx context.Context) {
	a.bus.Close()
	_ = a.store.Close()
}

// ----- App-level commands (PascalCase → camelCase in JS) -----

// Version returns app version.
func (a *App) Version() string { return a.cfg.Version }

// DataDir returns the user data directory.
func (a *App) DataDir() string { return a.cfg.DataDir }

// LLMProviders lists registered LLM providers.
func (a *App) LLMProviders() []string { return a.llmReg.Names() }

// IsPrivacyMode returns privacy mode flag.
func (a *App) IsPrivacyMode() bool { return a.cfg.PrivacyMode }

// OpenFileDialog opens a file dialog.
func (a *App) OpenFileDialog(title string) (string, error) {
	return wruntime.OpenFileDialog(a.Ctx, wruntime.OpenDialogOptions{Title: title})
}

// SaveFileDialog opens a save file dialog.
func (a *App) SaveFileDialog(title, def string) (string, error) {
	return wruntime.SaveFileDialog(a.Ctx, wruntime.SaveDialogOptions{Title: title, DefaultFilename: def})
}

func (a *App) init() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	a.cfg = cfg

	// BUG-005: 启动时若数据目录不存在主动创建；容忍中文 / 空格 / unicode 路径
	for _, d := range []string{cfg.DataDir, cfg.CacheDir, cfg.LogDir, cfg.BackupDir,
		filepath.Join(cfg.DataDir, "blob"),
		filepath.Join(cfg.DataDir, "keys"),
		filepath.Join(cfg.DataDir, "vector")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}

	if err := logger.Init(cfg.LogDir, cfg.LogLevel); err != nil {
		return err
	}
	logger.Info("starting", "version", cfg.Version, "data", cfg.DataDir)

	a.bus = eventbus.New()

	st, err := store.Open(filepath.Join(cfg.DataDir, "deepstudent.db"))
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	a.store = st

	bs, err := blob.New(filepath.Join(cfg.DataDir, "blob"))
	if err != nil {
		return err
	}
	a.blob = bs

	a.vfs = vfs.NewFS(bs)

	cry, err := crypto.NewManager(filepath.Join(cfg.DataDir, "keys"))
	if err != nil {
		return err
	}
	a.crypto = cry

	a.llmReg = llm.NewRegistry()
	a.llmReg.Register(llm.NewOpenAICompat("openai", "https://api.openai.com/v1", cfg.LLM.OpenAIKey))
	a.llmReg.Register(llm.NewOpenAICompat("deepseek", "https://api.deepseek.com/v1", cfg.LLM.DeepSeekKey))
	a.llmReg.Register(llm.NewOpenAICompat("siliconflow", "https://api.siliconflow.cn/v1", cfg.LLM.SiliconKey))
	a.llmReg.Register(llm.NewOpenAICompat("zhipu", "https://open.bigmodel.cn/api/paas/v4", cfg.LLM.ZhipuKey))
	a.llmReg.Register(llm.NewOpenAICompat("tongyi", "https://dashscope.aliyuncs.com/compatible-mode/v1", cfg.LLM.TongyiKey))
	a.llmReg.Register(llm.NewOpenAICompat("moonshot", "https://api.moonshot.cn/v1", cfg.LLM.MoonshotKey))
	a.llmReg.Register(llm.NewAnthropic(cfg.LLM.AnthropicKey))
	a.llmReg.Register(llm.NewGoogle(cfg.LLM.GoogleKey))

	a.Hub = hub.New(a.vfs, a.store, a.llmReg)
	a.Chat = chat.New(a.vfs, a.store, a.llmReg, a.bus)
	a.Mindmap = mindmap.New(a.vfs, a.llmReg)
	a.QBank = qbank.New(a.vfs, a.store, a.llmReg)
	a.Anki = anki.New(a.vfs, a.llmReg, a.bus)
	a.Reader = reader.New(a.vfs, a.llmReg)
	a.Trans = translate.New(a.vfs, a.llmReg)
	a.Essay = essay.New(a.vfs, a.llmReg)
	a.Res = research.New(a.vfs, a.llmReg, a.bus)
	a.Paper = paper.New(a.vfs, a.store)
	a.Mem = memory.New(a.vfs, a.store, a.llmReg)
	a.Skills = skills.New(a.vfs, a.store, a.llmReg, a.bus)
	a.Gov = governance.New(a.vfs, a.store, a.crypto, cfg, a.bus)
	a.Notes = notes.New(a.store, a.blob, a.vfs)

	// P0-A，模型厂商配置系统 —— 加载磁盘配置并 seed 内置厂商/模型
	a.LLMCfg = llmcfg.NewManager(cfg.DataDir)

	// P0-B，索引服务 —— FTS5 全文索引 + 向量嵌入 + RAG 检索
	a.Index = index.New(a.store.DB, a.llmReg, a.LLMCfg)
	return nil
}

// RunApp 是 Wails 应用入口，被根目录 main.go 调用。
func RunApp() {
	app := &App{}
	if err := wails.Run(&options.App{
		Title:     "DeepStudent",
		Width:     1400,
		Height:    900,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 22, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHiddenInset(),
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			About: &mac.AboutInfo{
				Title:   "DeepStudent",
				Message: "An open-source, local-first AI learning workbench.",
			},
		},
		Linux: &linux.Options{
			ProgramName:         "deepstudent",
			Icon:                nil,
			WindowIsTranslucent: false,
		},
	}); err != nil {
		fmt.Fprintf(os.Stderr, "wails: %v\n", err)
		os.Exit(1)
	}
}
