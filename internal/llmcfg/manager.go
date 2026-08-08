// llmcfg Manager —— 对外暴露的 CRUD + 测试连接 + 解析为 ApiConfig 接口。
//
// Manager 封装 store，提供线程安全的配置管理能力。
// App 方法层（cmd/deepstudent/app_methods.go）直接调用 Manager。
package llmcfg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrBuiltinReadOnly 内置项不可删除 / 不可编辑只读字段。
var ErrBuiltinReadOnly = errors.New("llmcfg: builtin item is read-only")

// ErrNotFound 未找到指定项。
var ErrNotFound = errors.New("llmcfg: not found")

// ErrMissingAPIKey 测试连接时缺少 API Key。
var ErrMissingAPIKey = errors.New("llmcfg: missing api key")

// ErrMissingBaseURL 测试连接时缺少 baseURL。
var ErrMissingBaseURL = errors.New("llmcfg: missing base url")

// ErrMissingModel 测试连接时缺少 model。
var ErrMissingModel = errors.New("llmcfg: missing model")

// httpClient 抽象 http.Client 便于测试。
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Manager 配置管理器。
type Manager struct {
	store *store
	http  httpClient
}

// NewManager 创建 Manager，dataDir 为数据目录（llmcfg.json 会写入此目录）。
//
// 创建后立即加载磁盘配置并 seed 内置项。
func NewManager(dataDir string) *Manager {
	path := filepath.Join(dataDir, "llmcfg.json")
	m := &Manager{
		store: newStore(path),
		http:  &http.Client{Timeout: 30 * time.Second},
	}
	_ = m.store.Load()
	// 首次加载自动 seed 内置厂商和模型
	if m.store.seedBuiltins() {
		_ = m.store.Save()
	}
	return m
}

// GetVendors 返回所有供应商。
func (m *Manager) GetVendors() []VendorConfig {
	return m.store.getVendors()
}

// SaveVendor 保存供应商（upsert）。
//
// 内置供应商允许更新 APIKey 等字段，但不可删除。
func (m *Manager) SaveVendor(v VendorConfig) error {
	if v.ID == "" {
		v.ID = "vendor-" + uuid.NewString()
	}
	m.store.upsertVendor(v)
	return m.store.Save()
}

// DeleteVendor 删除供应商（内置不可删）。
func (m *Manager) DeleteVendor(id string) error {
	v, ok := m.store.findVendor(id)
	if !ok {
		return ErrNotFound
	}
	if v.IsBuiltin {
		return ErrBuiltinReadOnly
	}
	m.store.deleteVendor(id)
	return m.store.Save()
}

// GetProfiles 返回所有模型。
func (m *Manager) GetProfiles() []ModelProfile {
	return m.store.getProfiles()
}

// GetProfilesByVendor 按供应商筛选模型。
func (m *Manager) GetProfilesByVendor(vendorID string) []ModelProfile {
	all := m.store.getProfiles()
	out := make([]ModelProfile, 0, len(all))
	for _, p := range all {
		if p.VendorID == vendorID {
			out = append(out, p)
		}
	}
	return out
}

// SaveProfile 保存模型（upsert）。
func (m *Manager) SaveProfile(p ModelProfile) error {
	if p.ID == "" {
		p.ID = "model-" + uuid.NewString()
	}
	if p.ModelAdapter == "" {
		p.ModelAdapter = defaultModelAdapter
	}
	if p.Status == "" {
		p.Status = defaultProfileStatus
	}
	if p.MaxOutputTokens == 0 {
		p.MaxOutputTokens = defaultMaxOutputTokens
	}
	if p.Temperature == 0 {
		p.Temperature = defaultTemperature
	}
	m.store.upsertProfile(p)
	return m.store.Save()
}

// DeleteProfile 删除模型（内置不可删）。
func (m *Manager) DeleteProfile(id string) error {
	p, ok := m.store.findProfile(id)
	if !ok {
		return ErrNotFound
	}
	if p.IsBuiltin {
		return ErrBuiltinReadOnly
	}
	m.store.deleteProfile(id)
	return m.store.Save()
}

// GetAssignments 获取模型分配。
func (m *Manager) GetAssignments() ModelAssignments {
	return m.store.getAssignments()
}

// SaveAssignments 保存模型分配。
func (m *Manager) SaveAssignments(a ModelAssignments) error {
	m.store.setAssignments(a)
	return m.store.Save()
}

// ResolveApiConfig 合并 vendor + profile 为运行时 ApiConfig。
//
// profile 字段优先于 vendor 字段；缺失 vendor 时仅用 profile 信息。
func (m *Manager) ResolveApiConfig(profileID string) (*ApiConfig, error) {
	p, ok := m.store.findProfile(profileID)
	if !ok {
		return nil, ErrNotFound
	}

	cfg := &ApiConfig{
		ID:                p.ID,
		Name:              p.Label,
		Model:             p.Model,
		IsMultimodal:      p.IsMultimodal,
		IsReasoning:       p.IsReasoning,
		IsEmbedding:       p.IsEmbedding,
		IsReranker:        p.IsReranker,
		IsImageGeneration: p.IsImageGeneration,
		Enabled:           p.Enabled,
		ModelAdapter:      p.ModelAdapter,
		MaxOutputTokens:   p.MaxOutputTokens,
		Temperature:       p.Temperature,
		SupportsTools:     p.SupportsTools,
		GeminiAPIVersion:  defaultGeminiAPIVer,
		IsBuiltin:         p.IsBuiltin,
		ReasoningEffort:   p.ReasoningEffort,
		ThinkingEnabled:   p.ThinkingEnabled,
		ThinkingBudget:    p.ThinkingBudget,
		IncludeThoughts:   p.IncludeThoughts,
		MinP:              p.MinP,
		TopK:              p.TopK,
		EnableThinking:    p.EnableThinking,
		SupportsReasoning: p.SupportsReasoning,
		RepetitionPenalty: p.RepetitionPenalty,
		ReasoningSplit:    p.ReasoningSplit,
		Effort:            p.Effort,
		Verbosity:         p.Verbosity,
		IsFavorite:        p.IsFavorite,
		MaxTokensLimit:    p.MaxTokensLimit,
		ContextWindow:     p.ContextWindow,
	}
	if p.APIProtocol != nil {
		cfg.APIProtocol = p.APIProtocol
	}
	if p.GeminiAPIVersion != nil && *p.GeminiAPIVersion != "" {
		cfg.GeminiAPIVersion = *p.GeminiAPIVersion
	}
	if p.ProviderScope != nil {
		cfg.ProviderScope = p.ProviderScope
	}

	// 合并 vendor 信息
	if v, ok := m.store.findVendor(p.VendorID); ok {
		vendorID := v.ID
		vendorName := v.Name
		providerType := v.ProviderType
		cfg.VendorID = &vendorID
		cfg.VendorName = &vendorName
		cfg.ProviderType = &providerType
		cfg.APIKey = v.APIKey
		cfg.BaseURL = v.BaseURL
		cfg.IsReadOnly = v.IsReadOnly
		if cfg.APIProtocol == nil && v.APIProtocol != nil {
			cfg.APIProtocol = v.APIProtocol
		}
		if v.SupportsOpenAIResponses != nil {
			cfg.SupportsOpenAIResponses = v.SupportsOpenAIResponses
		}
		if len(v.Headers) > 0 {
			cfg.Headers = v.Headers
		}
		if v.MaxTokensLimit != nil && cfg.MaxTokensLimit == nil {
			cfg.MaxTokensLimit = v.MaxTokensLimit
		}
	}
	return cfg, nil
}

// TestConnection 测试连接：发起一个最简 chat 请求（messages:[{role:user,content:"ping"}], max_tokens:5）。
//
// HTTP 2xx 视为成功，记录 latency。
func (m *Manager) TestConnection(ctx context.Context, profileID string) (*TestConnectionResult, error) {
	cfg, err := m.ResolveApiConfig(profileID)
	if err != nil {
		return nil, err
	}
	if cfg.APIKey == "" {
		return nil, ErrMissingAPIKey
	}
	if cfg.BaseURL == "" {
		return nil, ErrMissingBaseURL
	}
	if cfg.Model == "" {
		return nil, ErrMissingModel
	}

	// 构造最简 OpenAI 兼容 chat 请求
	body := map[string]any{
		"model":       cfg.Model,
		"max_tokens":  5,
		"temperature": cfg.Temperature,
		"messages": []map[string]string{
			{"role": "user", "content": "ping"},
		},
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("llmcfg: marshal test body: %w", err)
	}

	url := strings.TrimRight(cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("llmcfg: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := m.http.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return &TestConnectionResult{
			OK:         false,
			Message:    err.Error(),
			LatencyMs:  latency,
			Model:      cfg.Model,
			VendorName: derefString(cfg.VendorName),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &TestConnectionResult{
			OK:         true,
			Message:    "OK",
			LatencyMs:  latency,
			Model:      cfg.Model,
			VendorName: derefString(cfg.VendorName),
		}, nil
	}

	raw, _ := io.ReadAll(resp.Body)
	return &TestConnectionResult{
		OK:         false,
		Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncate(string(raw), 200)),
		LatencyMs:  latency,
		Model:      cfg.Model,
		VendorName: derefString(cfg.VendorName),
	}, nil
}

// ReloadBuiltins 重新加载内置（用于重置）。
//
// 仅补充缺失的内置项，不会删除用户已添加的自定义项。
func (m *Manager) ReloadBuiltins() error {
	if m.store.seedBuiltins() {
		return m.store.Save()
	}
	return nil
}

// derefString 解引用 *string，nil 返回空串。
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// truncate 截断字符串到指定长度并加省略号。
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// ListApiConfigurations 列出全部运行时可调用的配置（对齐原版 get_api_configurations）。
func (m *Manager) ListApiConfigurations() []ApiConfig {
	profiles := m.store.getProfiles()
	out := make([]ApiConfig, 0, len(profiles))
	for _, p := range profiles {
		if cfg, err := m.ResolveApiConfig(p.ID); err == nil && cfg != nil {
			out = append(out, *cfg)
		}
	}
	return out
}
