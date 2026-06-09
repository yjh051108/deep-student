package app

import (
	"deep-student-go/internal/anki"
	"deep-student-go/internal/chat"
	"deep-student-go/internal/dstu"
	"deep-student-go/internal/files"
	"deep-student-go/internal/mcp"
	"deep-student-go/internal/notes"
	"deep-student-go/internal/qbank"
	"deep-student-go/internal/reviewplan"
	"deep-student-go/internal/runtime"
	"deep-student-go/internal/settings"
	"deep-student-go/internal/skills"
	"deep-student-go/internal/system"
	"deep-student-go/internal/templates"
	"deep-student-go/internal/todo"
	"deep-student-go/internal/vfs"
)

type App struct {
	Anki      *anki.Service
	DataDir   string
	Chat      *chat.Service
	Dstu      *dstu.Service
	Events    *runtime.EventBus
	Files     *files.Service
	Mcp       *mcp.Service
	Notes     *notes.Service
	Qbank     *qbank.Service
	Review    *reviewplan.Service
	Settings  *settings.Service
	Skills    *skills.Service
	System    *system.Service
	Templates *templates.Service
	Todo      *todo.Service
	Vfs       *vfs.Service
}

func New() (*App, error) {
	dataDir, err := ResolveDataDir()
	if err != nil {
		return nil, err
	}

	settingsService, err := settings.NewService(dataDir)
	if err != nil {
		return nil, err
	}

	systemService, err := system.NewService(dataDir)
	if err != nil {
		return nil, err
	}

	notesService, err := notes.NewService(dataDir)
	if err != nil {
		return nil, err
	}

	todoService, err := todo.NewService(dataDir)
	if err != nil {
		return nil, err
	}

	vfsService, err := vfs.NewService(dataDir)
	if err != nil {
		return nil, err
	}

	dstuService, err := dstu.NewService(dataDir, vfsService)
	if err != nil {
		return nil, err
	}

	qbankService, err := qbank.NewService(dataDir, vfsService)
	if err != nil {
		return nil, err
	}

	reviewService, err := reviewplan.NewService(dataDir, qbankService)
	if err != nil {
		return nil, err
	}

	ankiService, err := anki.NewService(dataDir)
	if err != nil {
		return nil, err
	}

	templateService, err := templates.NewServiceWithLegacyRoots(dataDir, LegacyDataDirCandidates(dataDir))
	if err != nil {
		return nil, err
	}

	chatService, err := chat.NewService(dataDir)
	if err != nil {
		return nil, err
	}
	mcpService, err := mcp.NewService(dataDir)
	if err != nil {
		return nil, err
	}
	eventBus := runtime.NewEventBus()
	vfsService.SetEventEmitter(eventBus.Emit)
	dstuService.SetEventEmitter(eventBus.Emit)
	mcpService.SetEventEmitter(eventBus.Emit)
	ankiService.SetEventEmitter(eventBus.Emit)
	ankiService.SetAPIConfigLoader(func() (anki.APIConfigState, error) {
		configs, err := settingsService.GetAPIConfigurations()
		if err != nil {
			return anki.APIConfigState{}, err
		}
		assignments, err := settingsService.GetModelAssignments()
		if err != nil {
			return anki.APIConfigState{}, err
		}
		out := make([]anki.ApiConfig, 0, len(configs))
		for _, config := range configs {
			out = append(out, anki.ApiConfig{
				ID:              config.ID,
				Name:            config.Name,
				ApiKey:          config.ApiKey,
				BaseUrl:         config.BaseUrl,
				Model:           config.Model,
				Enabled:         config.Enabled,
				MaxOutputTokens: config.MaxOutputTokens,
				Temperature:     config.Temperature,
				Headers:         config.Headers,
				IsEmbedding:     config.IsEmbedding,
				IsReranker:      config.IsReranker,
			})
		}
		state := anki.APIConfigState{Configs: out}
		if assignments.AnkiCardModelConfigID != nil {
			state.AnkiCardModelConfigID = *assignments.AnkiCardModelConfigID
		}
		return state, nil
	})
	qbankService.SetEventEmitter(eventBus.Emit)
	qbankService.SetAPIConfigLoader(func() (qbank.APIConfigState, error) {
		configs, err := settingsService.GetAPIConfigurations()
		if err != nil {
			return qbank.APIConfigState{}, err
		}
		assignments, err := settingsService.GetModelAssignments()
		if err != nil {
			return qbank.APIConfigState{}, err
		}
		out := make([]qbank.ApiConfig, 0, len(configs))
		for _, config := range configs {
			out = append(out, qbank.ApiConfig{
				ID:              config.ID,
				Name:            config.Name,
				ApiKey:          config.ApiKey,
				BaseUrl:         config.BaseUrl,
				Model:           config.Model,
				Enabled:         config.Enabled,
				MaxOutputTokens: config.MaxOutputTokens,
				Temperature:     config.Temperature,
				Headers:         config.Headers,
				IsEmbedding:     config.IsEmbedding,
				IsReranker:      config.IsReranker,
			})
		}
		state := qbank.APIConfigState{Configs: out}
		if assignments.QbankAIGradingModelConfigID != nil {
			state.QbankAIGradingModelConfigID = *assignments.QbankAIGradingModelConfigID
		}
		return state, nil
	})
	chatService.SetEventEmitter(eventBus.Emit)
	chatService.SetAPIConfigLoader(func() ([]chat.ApiConfig, error) {
		configs, err := settingsService.GetAPIConfigurations()
		if err != nil {
			return nil, err
		}
		out := make([]chat.ApiConfig, 0, len(configs))
		for _, config := range configs {
			out = append(out, chat.ApiConfig{
				ID:                config.ID,
				Name:              config.Name,
				ApiKey:            config.ApiKey,
				BaseUrl:           config.BaseUrl,
				Model:             config.Model,
				Enabled:           config.Enabled,
				ModelAdapter:      config.ModelAdapter,
				MaxOutputTokens:   config.MaxOutputTokens,
				Temperature:       config.Temperature,
				SupportsTools:     config.SupportsTools,
				IsBuiltin:         config.IsBuiltin,
				Headers:           config.Headers,
				ProviderType:      config.ProviderType,
				ProviderScope:     config.ProviderScope,
				ApiProtocol:       config.ApiProtocol,
				VendorID:          config.VendorID,
				IsMultimodal:      config.IsMultimodal,
				IsReasoning:       config.IsReasoning,
				IsEmbedding:       config.IsEmbedding,
				IsReranker:        config.IsReranker,
				ContextWindow:     config.ContextWindow,
				MaxTokensLimit:    config.MaxTokensLimit,
				ReasoningEffort:   config.ReasoningEffort,
				ThinkingEnabled:   config.ThinkingEnabled,
				ThinkingBudget:    config.ThinkingBudget,
				SupportsReasoning: config.SupportsReasoning,
			})
		}
		return out, nil
	})

	return &App{
		Anki:      ankiService,
		DataDir:   dataDir,
		Chat:      chatService,
		Dstu:      dstuService,
		Events:    eventBus,
		Files:     files.NewService(),
		Mcp:       mcpService,
		Notes:     notesService,
		Qbank:     qbankService,
		Review:    reviewService,
		Settings:  settingsService,
		Skills:    skills.NewService(dataDir),
		System:    systemService,
		Templates: templateService,
		Todo:      todoService,
		Vfs:       vfsService,
	}, nil
}
