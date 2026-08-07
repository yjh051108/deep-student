// llmcfg Manager 单元测试 —— 覆盖 CRUD、内置加载、test connection mock、resolve ApiConfig。
package llmcfg

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestManager 在临时目录创建 Manager，返回 Manager 和清理函数。
func newTestManager(t *testing.T) (*Manager, func()) {
	t.Helper()
	dir := t.TempDir()
	m := NewManager(dir)
	return m, func() { _ = os.RemoveAll(dir) }
}

// ---------- 内置加载 ----------

func TestBuiltinVendorsCount(t *testing.T) {
	if got, want := len(builtinVendors), 10; got != want {
		t.Fatalf("builtinVendors count = %d, want %d", got, want)
	}
}

func TestBuiltinVendorsContainAllExpected(t *testing.T) {
	want := map[string]bool{
		"builtin-siliconflow": false,
		"builtin-deepseek":    false,
		"builtin-qwen":        false,
		"builtin-zhipu":       false,
		"builtin-doubao":      false,
		"builtin-minimax":     false,
		"builtin-moonshot":    false,
		"builtin-openai":      false,
		"builtin-nvidia":      false,
		"builtin-mimo":        false,
	}
	for _, v := range builtinVendors {
		if _, ok := want[v.id]; ok {
			want[v.id] = true
		}
	}
	for id, found := range want {
		if !found {
			t.Errorf("missing builtin vendor: %s", id)
		}
	}
}

func TestLoadBuiltinVendorsSkipsExisting(t *testing.T) {
	// 已存在 deepseek，应被跳过
	existing := []string{"builtin-deepseek"}
	got := loadBuiltinVendors(existing)
	for _, v := range got {
		if v.ID == "builtin-deepseek" {
			t.Fatalf("loadBuiltinVendors should skip existing builtin-deepseek")
		}
	}
	if len(got) != 9 {
		t.Fatalf("loadBuiltinVendors with 1 existing = %d, want 9", len(got))
	}
}

func TestLoadBuiltinModelsSkipsExisting(t *testing.T) {
	existing := []string{"builtin-deepseek-v4-flash", "builtin-gpt-5.5"}
	got := loadBuiltinModels(existing)
	for _, p := range got {
		if p.ID == "builtin-deepseek-v4-flash" || p.ID == "builtin-gpt-5.5" {
			t.Fatalf("loadBuiltinModels should skip existing IDs")
		}
	}
}

// ---------- 内置模型转换规则（对齐 Rust 测试）----------

func TestDeepSeekBuiltinProfiles(t *testing.T) {
	v4Flash := findBuiltinModel(t, "builtin-deepseek-v4-flash").toModelProfile()
	v4Pro := findBuiltinModel(t, "builtin-deepseek-v4-pro").toModelProfile()
	chatAlias := findBuiltinModel(t, "builtin-deepseek-chat").toModelProfile()
	reasonerAlias := findBuiltinModel(t, "builtin-deepseek-reasoner").toModelProfile()

	if v4Flash.Model != "deepseek-v4-flash" {
		t.Fatalf("v4Flash.Model = %s", v4Flash.Model)
	}
	if v4Pro.Model != "deepseek-v4-pro" {
		t.Fatalf("v4Pro.Model = %s", v4Pro.Model)
	}
	if got, want := derefString(v4Flash.ProviderScope), "deepseek"; got != want {
		t.Fatalf("v4Flash.ProviderScope = %q, want %q", got, want)
	}
	if v4Flash.ModelAdapter != "deepseek" {
		t.Fatalf("v4Flash.ModelAdapter = %s", v4Flash.ModelAdapter)
	}
	if got, want := derefUint32(v4Flash.MaxTokensLimit), uint32(65_536); got != want {
		t.Fatalf("v4Flash.MaxTokensLimit = %d, want %d", got, want)
	}
	if got, want := derefUint32(v4Flash.ContextWindow), uint32(1_000_000); got != want {
		t.Fatalf("v4Flash.ContextWindow = %d, want %d", got, want)
	}
	if got, want := derefUint32(v4Pro.ContextWindow), uint32(1_000_000); got != want {
		t.Fatalf("v4Pro.ContextWindow = %d, want %d", got, want)
	}
	if v4Flash.MaxOutputTokens != 32_768 {
		t.Fatalf("v4Flash.MaxOutputTokens = %d", v4Flash.MaxOutputTokens)
	}
	if got, want := derefString(v4Flash.ReasoningEffort), "high"; got != want {
		t.Fatalf("v4Flash.ReasoningEffort = %q, want %q", got, want)
	}

	if chatAlias.Model != "deepseek-chat" {
		t.Fatalf("chatAlias.Model = %s", chatAlias.Model)
	}
	if chatAlias.ModelAdapter != "deepseek" {
		t.Fatalf("chatAlias.ModelAdapter = %s", chatAlias.ModelAdapter)
	}
	if got, want := derefUint32(chatAlias.ContextWindow), uint32(1_000_000); got != want {
		t.Fatalf("chatAlias.ContextWindow = %d, want %d", got, want)
	}
	if chatAlias.IsReasoning {
		t.Fatalf("chatAlias.IsReasoning should be false")
	}
	if chatAlias.ThinkingEnabled {
		t.Fatalf("chatAlias.ThinkingEnabled should be false")
	}
	if reasonerAlias.Model != "deepseek-reasoner" {
		t.Fatalf("reasonerAlias.Model = %s", reasonerAlias.Model)
	}
	if got, want := derefUint32(reasonerAlias.ContextWindow), uint32(1_000_000); got != want {
		t.Fatalf("reasonerAlias.ContextWindow = %d, want %d", got, want)
	}
	if !reasonerAlias.IsReasoning {
		t.Fatalf("reasonerAlias.IsReasoning should be true")
	}
	if got, want := derefString(reasonerAlias.ReasoningEffort), "high"; got != want {
		t.Fatalf("reasonerAlias.ReasoningEffort = %q, want %q", got, want)
	}
}

func TestNVIDIABuiltinProfiles(t *testing.T) {
	nemotron := findBuiltinModel(t, "builtin-nvidia-nemotron-3-nano").toModelProfile()
	llama := findBuiltinModel(t, "builtin-nvidia-llama-3.1-405b").toModelProfile()

	if nemotron.VendorID != "builtin-nvidia" {
		t.Fatalf("nemotron.VendorID = %s", nemotron.VendorID)
	}
	if got, want := derefString(nemotron.ProviderScope), "nvidia"; got != want {
		t.Fatalf("nemotron.ProviderScope = %q, want %q", got, want)
	}
	if nemotron.ModelAdapter != "general" {
		t.Fatalf("nemotron.ModelAdapter = %s", nemotron.ModelAdapter)
	}
	if nemotron.Model != "nvidia/nemotron-3-nano-30b-a3b" {
		t.Fatalf("nemotron.Model = %s", nemotron.Model)
	}
	if !nemotron.IsReasoning {
		t.Fatalf("nemotron.IsReasoning should be true")
	}
	if nemotron.ThinkingEnabled {
		t.Fatalf("nemotron.ThinkingEnabled should be false (NVIDIA exception)")
	}
	if nemotron.IncludeThoughts {
		t.Fatalf("nemotron.IncludeThoughts should be false (NVIDIA exception)")
	}
	if nemotron.ReasoningEffort != nil {
		t.Fatalf("nemotron.ReasoningEffort should be nil")
	}
	if got, want := derefUint32(nemotron.ContextWindow), uint32(1_000_000); got != want {
		t.Fatalf("nemotron.ContextWindow = %d, want %d", got, want)
	}

	if llama.Model != "meta/llama-3.1-405b-instruct" {
		t.Fatalf("llama.Model = %s", llama.Model)
	}
	if llama.ModelAdapter != "general" {
		t.Fatalf("llama.ModelAdapter = %s", llama.ModelAdapter)
	}
	if llama.ReasoningEffort != nil {
		t.Fatalf("llama.ReasoningEffort should be nil")
	}
}

func TestMiMoBuiltinProfiles(t *testing.T) {
	pro := findBuiltinModel(t, "builtin-mimo-v2.5-pro").toModelProfile()
	omni := findBuiltinModel(t, "builtin-mimo-v2.5").toModelProfile()
	flash := findBuiltinModel(t, "builtin-mimo-v2-flash").toModelProfile()

	if pro.VendorID != "builtin-mimo" {
		t.Fatalf("pro.VendorID = %s", pro.VendorID)
	}
	if got, want := derefString(pro.ProviderScope), "mimo"; got != want {
		t.Fatalf("pro.ProviderScope = %q, want %q", got, want)
	}
	if pro.ModelAdapter != "mimo" {
		t.Fatalf("pro.ModelAdapter = %s", pro.ModelAdapter)
	}
	if pro.Model != "mimo-v2.5-pro" {
		t.Fatalf("pro.Model = %s", pro.Model)
	}
	if !pro.IsReasoning {
		t.Fatalf("pro.IsReasoning should be true")
	}
	if !pro.ThinkingEnabled {
		t.Fatalf("pro.ThinkingEnabled should be true")
	}
	if !pro.IncludeThoughts {
		t.Fatalf("pro.IncludeThoughts should be true")
	}
	if pro.MaxOutputTokens != 131_072 {
		t.Fatalf("pro.MaxOutputTokens = %d", pro.MaxOutputTokens)
	}
	if got, want := derefUint32(pro.ContextWindow), uint32(1_000_000); got != want {
		t.Fatalf("pro.ContextWindow = %d, want %d", got, want)
	}

	if omni.Model != "mimo-v2.5" {
		t.Fatalf("omni.Model = %s", omni.Model)
	}
	if !omni.IsMultimodal {
		t.Fatalf("omni.IsMultimodal should be true")
	}
	if got, want := derefUint32(omni.ContextWindow), uint32(1_000_000); got != want {
		t.Fatalf("omni.ContextWindow = %d, want %d", got, want)
	}

	if flash.Model != "mimo-v2-flash" {
		t.Fatalf("flash.Model = %s", flash.Model)
	}
	if flash.MaxOutputTokens != 65_536 {
		t.Fatalf("flash.MaxOutputTokens = %d", flash.MaxOutputTokens)
	}
	if got, want := derefUint32(flash.ContextWindow), uint32(256_000); got != want {
		t.Fatalf("flash.ContextWindow = %d, want %d", got, want)
	}
	if flash.ThinkingEnabled {
		t.Fatalf("flash.ThinkingEnabled should be false (MiMo flash exception)")
	}
}

func TestOpenAIBuiltinProfilesReasoningEffort(t *testing.T) {
	flagship := findBuiltinModel(t, "builtin-gpt-5.5").toModelProfile()
	pro := findBuiltinModel(t, "builtin-gpt-5.5-pro").toModelProfile()
	nano := findBuiltinModel(t, "builtin-gpt-5.4-nano").toModelProfile()

	if flagship.ModelAdapter != "general" {
		t.Fatalf("flagship.ModelAdapter = %s", flagship.ModelAdapter)
	}
	if got, want := derefString(flagship.ReasoningEffort), "medium"; got != want {
		t.Fatalf("flagship.ReasoningEffort = %q, want %q", got, want)
	}
	if got, want := derefString(flagship.Verbosity), "medium"; got != want {
		t.Fatalf("flagship.Verbosity = %q, want %q", got, want)
	}
	if !flagship.ThinkingEnabled {
		t.Fatalf("flagship.ThinkingEnabled should be true")
	}
	if !flagship.IncludeThoughts {
		t.Fatalf("flagship.IncludeThoughts should be true")
	}

	if got, want := derefString(pro.ReasoningEffort), "high"; got != want {
		t.Fatalf("pro.ReasoningEffort = %q, want %q", got, want)
	}
	if got, want := derefString(pro.Verbosity), "medium"; got != want {
		t.Fatalf("pro.Verbosity = %q, want %q", got, want)
	}

	if got, want := derefString(nano.ReasoningEffort), "low"; got != want {
		t.Fatalf("nano.ReasoningEffort = %q, want %q", got, want)
	}
	if got, want := derefString(nano.Verbosity), "low"; got != want {
		t.Fatalf("nano.Verbosity = %q, want %q", got, want)
	}
}

// ---------- Manager CRUD ----------

func TestManagerSeedsBuiltinsOnFirstLoad(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	vendors := m.GetVendors()
	if len(vendors) < 10 {
		t.Fatalf("expected >=10 builtin vendors after seed, got %d", len(vendors))
	}
	profiles := m.GetProfiles()
	if len(profiles) < 40 {
		t.Fatalf("expected >=40 builtin profiles after seed, got %d", len(profiles))
	}

	// 验证文件已落盘
	if _, err := os.Stat(filepath.Join(t.TempDir(), "..", "llmcfg.json")); err != nil {
		// 上面 TempDir 是新的，文件在 m 的目录里；这里仅检查 m.store.path
		if _, err2 := os.Stat(m.store.path); err2 != nil {
			t.Fatalf("llmcfg.json not persisted: %v", err2)
		}
	}
}

func TestManagerSaveAndDeleteVendor(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	custom := VendorConfig{
		ID:           "custom-vendor-1",
		Name:         "My Custom",
		ProviderType: "openai",
		BaseURL:      "https://example.com/v1",
		APIKey:       "sk-test",
		Headers:      map[string]string{},
	}
	if err := m.SaveVendor(custom); err != nil {
		t.Fatalf("SaveVendor: %v", err)
	}

	found := false
	for _, v := range m.GetVendors() {
		if v.ID == "custom-vendor-1" {
			found = true
			if v.APIKey != "sk-test" {
				t.Fatalf("APIKey = %s", v.APIKey)
			}
		}
	}
	if !found {
		t.Fatalf("custom vendor not saved")
	}

	// 更新 APIKey
	custom.APIKey = "sk-updated"
	if err := m.SaveVendor(custom); err != nil {
		t.Fatalf("SaveVendor update: %v", err)
	}
	v, ok := m.store.findVendor("custom-vendor-1")
	if !ok || v.APIKey != "sk-updated" {
		t.Fatalf("vendor APIKey not updated")
	}

	// 删除自定义
	if err := m.DeleteVendor("custom-vendor-1"); err != nil {
		t.Fatalf("DeleteVendor: %v", err)
	}
	if _, ok := m.store.findVendor("custom-vendor-1"); ok {
		t.Fatalf("custom vendor should be deleted")
	}
}

func TestManagerDeleteBuiltinVendorFails(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	if err := m.DeleteVendor("builtin-deepseek"); err != ErrBuiltinReadOnly {
		t.Fatalf("DeleteVendor builtin should return ErrBuiltinReadOnly, got %v", err)
	}
}

func TestManagerSaveAndDeleteProfile(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	custom := ModelProfile{
		ID:              "custom-model-1",
		VendorID:        "builtin-deepseek",
		Label:           "My Model",
		Model:           "my-model",
		ModelAdapter:    "general",
		MaxOutputTokens: 4096,
		Temperature:     0.7,
		Status:          "enabled",
		Enabled:         true,
	}
	if err := m.SaveProfile(custom); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	// 内置模型 is_builtin=false，可以删除
	if err := m.DeleteProfile("custom-model-1"); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}
}

func TestManagerGetProfilesByVendor(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	deepseekProfiles := m.GetProfilesByVendor("builtin-deepseek")
	if len(deepseekProfiles) != 4 {
		t.Fatalf("deepseek profiles = %d, want 4", len(deepseekProfiles))
	}
	for _, p := range deepseekProfiles {
		if p.VendorID != "builtin-deepseek" {
			t.Fatalf("profile vendor mismatch")
		}
	}
}

func TestManagerAssignments(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	assignments := m.GetAssignments()
	if assignments.Model2ConfigID != nil {
		t.Fatalf("initial assignments should be empty")
	}

	id := "builtin-deepseek-v4-flash"
	assignments.Model2ConfigID = &id
	assignments.ChatTitleModelConfigID = &id
	if err := m.SaveAssignments(assignments); err != nil {
		t.Fatalf("SaveAssignments: %v", err)
	}

	got := m.GetAssignments()
	if got.Model2ConfigID == nil || *got.Model2ConfigID != id {
		t.Fatalf("Model2ConfigID not saved")
	}
	if got.ChatTitleModelConfigID == nil || *got.ChatTitleModelConfigID != id {
		t.Fatalf("ChatTitleModelConfigID not saved")
	}
}

// ---------- ResolveApiConfig ----------

func TestResolveApiConfig(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	// 给 deepseek 填 key
	v, ok := m.store.findVendor("builtin-deepseek")
	if !ok {
		t.Fatalf("builtin-deepseek vendor missing")
	}
	v.APIKey = "sk-test-key"
	m.store.upsertVendor(v)

	cfg, err := m.ResolveApiConfig("builtin-deepseek-v4-flash")
	if err != nil {
		t.Fatalf("ResolveApiConfig: %v", err)
	}
	if cfg.Model != "deepseek-v4-flash" {
		t.Fatalf("cfg.Model = %s", cfg.Model)
	}
	if cfg.APIKey != "sk-test-key" {
		t.Fatalf("cfg.APIKey = %s", cfg.APIKey)
	}
	if cfg.BaseURL != "https://api.deepseek.com/v1" {
		t.Fatalf("cfg.BaseURL = %s", cfg.BaseURL)
	}
	if cfg.ModelAdapter != "deepseek" {
		t.Fatalf("cfg.ModelAdapter = %s", cfg.ModelAdapter)
	}
	if cfg.VendorName == nil || *cfg.VendorName != "DeepSeek" {
		t.Fatalf("cfg.VendorName = %v", cfg.VendorName)
	}
	if cfg.ProviderType == nil || *cfg.ProviderType != "deepseek" {
		t.Fatalf("cfg.ProviderType = %v", cfg.ProviderType)
	}
	if cfg.MaxOutputTokens != 32_768 {
		t.Fatalf("cfg.MaxOutputTokens = %d", cfg.MaxOutputTokens)
	}
	if !cfg.IsReasoning {
		t.Fatalf("cfg.IsReasoning should be true")
	}
	if !cfg.ThinkingEnabled {
		t.Fatalf("cfg.ThinkingEnabled should be true")
	}
}

func TestResolveApiConfigNotFound(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	if _, err := m.ResolveApiConfig("nonexistent"); err != ErrNotFound {
		t.Fatalf("ResolveApiConfig nonexistent should return ErrNotFound, got %v", err)
	}
}

// ---------- TestConnection（mock HTTP）----------

func TestTestConnectionSuccess(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	// 启动一个 mock OpenAI 兼容服务
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.String(), "/chat/completions") {
			t.Errorf("unexpected url: %s", r.URL.String())
		}
		// 校验 Authorization
		if got := r.Header.Get("Authorization"); got != "Bearer sk-success" {
			t.Errorf("Authorization = %q", got)
		}
		// 校验 body 包含 model
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		if body["model"] != "deepseek-v4-flash" {
			t.Errorf("body.model = %v", body["model"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"pong"}}]}`))
	}))
	defer server.Close()

	// 注入一个指向 mock server 的 vendor + profile
	vendor := VendorConfig{
		ID:           "test-vendor",
		Name:         "TestVendor",
		ProviderType: "openai",
		BaseURL:      server.URL,
		APIKey:       "sk-success",
		Headers:      map[string]string{},
	}
	if err := m.SaveVendor(vendor); err != nil {
		t.Fatalf("SaveVendor: %v", err)
	}
	profile := ModelProfile{
		ID:              "test-model",
		VendorID:        "test-vendor",
		Label:           "TestModel",
		Model:           "deepseek-v4-flash",
		ModelAdapter:    "general",
		MaxOutputTokens: 4096,
		Temperature:     0.7,
		Status:          "enabled",
		Enabled:         true,
	}
	if err := m.SaveProfile(profile); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	// 替换 http client 为默认（mock server 已是真实 HTTP）
	m.http = server.Client()

	res, err := m.TestConnection(context.Background(), "test-model")
	if err != nil {
		t.Fatalf("TestConnection: %v", err)
	}
	if !res.OK {
		t.Fatalf("TestConnection should succeed, message: %s", res.Message)
	}
	if res.Model != "deepseek-v4-flash" {
		t.Fatalf("res.Model = %s", res.Model)
	}
	if res.VendorName != "TestVendor" {
		t.Fatalf("res.VendorName = %s", res.VendorName)
	}
	if res.LatencyMs < 0 {
		t.Fatalf("res.LatencyMs = %d", res.LatencyMs)
	}
}

func TestTestConnectionFailure(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer server.Close()

	vendor := VendorConfig{
		ID:           "test-vendor-fail",
		Name:         "FailVendor",
		ProviderType: "openai",
		BaseURL:      server.URL,
		APIKey:       "sk-bad",
		Headers:      map[string]string{},
	}
	_ = m.SaveVendor(vendor)
	profile := ModelProfile{
		ID:           "test-model-fail",
		VendorID:     "test-vendor-fail",
		Label:        "Fail",
		Model:        "fail-model",
		ModelAdapter: "general",
		Status:       "enabled",
		Enabled:      true,
	}
	_ = m.SaveProfile(profile)
	m.http = server.Client()

	res, err := m.TestConnection(context.Background(), "test-model-fail")
	if err != nil {
		t.Fatalf("TestConnection returned err: %v", err)
	}
	if res.OK {
		t.Fatalf("TestConnection should fail")
	}
	if !strings.Contains(res.Message, "401") {
		t.Fatalf("res.Message should contain 401, got: %s", res.Message)
	}
}

func TestTestConnectionMissingAPIKey(t *testing.T) {
	m, cleanup := newTestManager(t)
	defer cleanup()

	// deepseek 未填 key
	if _, err := m.TestConnection(context.Background(), "builtin-deepseek-v4-flash"); err != ErrMissingAPIKey {
		t.Fatalf("TestConnection without key should return ErrMissingAPIKey, got %v", err)
	}
}

// ---------- 持久化 ----------

func TestStorePersistAndReload(t *testing.T) {
	dir := t.TempDir()
	m1 := NewManager(dir)
	v, _ := m1.store.findVendor("builtin-deepseek")
	v.APIKey = "sk-persisted"
	if err := m1.SaveVendor(v); err != nil {
		t.Fatalf("SaveVendor: %v", err)
	}

	// 新建 Manager 读取同一目录
	m2 := NewManager(dir)
	got, ok := m2.store.findVendor("builtin-deepseek")
	if !ok {
		t.Fatalf("builtin-deepseek not found after reload")
	}
	if got.APIKey != "sk-persisted" {
		t.Fatalf("APIKey not persisted, got %s", got.APIKey)
	}
	// seedBuiltins 不应重复注入
	if len(m2.GetVendors()) != len(m1.GetVendors()) {
		t.Fatalf("vendor count changed after reload: %d vs %d", len(m2.GetVendors()), len(m1.GetVendors()))
	}
}

// ---------- ReloadBuiltins ----------

func TestReloadBuiltins(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir)

	// 删除一个内置模型（注意：内置模型 is_builtin=false，可以删）
	_ = m.DeleteProfile("builtin-deepseek-v4-flash")

	// ReloadBuiltins 应补回
	if err := m.ReloadBuiltins(); err != nil {
		t.Fatalf("ReloadBuiltins: %v", err)
	}
	found := false
	for _, p := range m.GetProfiles() {
		if p.ID == "builtin-deepseek-v4-flash" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("ReloadBuiltins should restore deleted builtin model")
	}
}

// ---------- helpers ----------

func findBuiltinModel(t *testing.T, id string) builtinModel {
	t.Helper()
	for _, m := range builtinModels {
		if m.id == id {
			return m
		}
	}
	t.Fatalf("builtin model %s not found", id)
	return builtinModel{}
}

func derefUint32(v *uint32) uint32 {
	if v == nil {
		return 0
	}
	return *v
}
