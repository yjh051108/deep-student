package settings

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServiceRoundTrip(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := service.SaveSetting("theme", "dark"); err != nil {
		t.Fatal(err)
	}

	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	value, ok := reloaded.GetSetting("theme")
	if !ok {
		t.Fatal("expected setting to be present")
	}
	if value != "dark" {
		t.Fatalf("expected dark, got %q", value)
	}
}

func TestServiceDeleteSetting(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := service.SaveSetting("theme", "dark"); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteSetting("theme"); err != nil {
		t.Fatal(err)
	}

	_, ok := service.GetSetting("theme")
	if ok {
		t.Fatal("expected setting to be deleted")
	}
}

func TestServiceBatchSettings(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := service.SaveSettings(map[string]string{
		"theme":    "dark",
		"language": "zh-CN",
	}); err != nil {
		t.Fatal(err)
	}

	values := service.GetSettings([]string{"theme", "language", "missing"})
	if values["theme"] != "dark" {
		t.Fatalf("expected dark, got %q", values["theme"])
	}
	if values["language"] != "zh-CN" {
		t.Fatalf("expected zh-CN, got %q", values["language"])
	}
	if _, ok := values["missing"]; ok {
		t.Fatal("expected missing key to be omitted")
	}
}

func TestBackupConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	defaultConfig, err := service.GetBackupConfig()
	if err != nil {
		t.Fatal(err)
	}
	if defaultConfig.BackupDirectory != nil {
		t.Fatalf("expected nil default backup directory, got %+v", defaultConfig.BackupDirectory)
	}
	if defaultConfig.AutoBackupEnabled {
		t.Fatal("expected auto backup to be disabled by default")
	}
	if defaultConfig.AutoBackupIntervalHours != 24 {
		t.Fatalf("expected default interval 24h, got %d", defaultConfig.AutoBackupIntervalHours)
	}
	if defaultConfig.MaxBackupCount == nil || *defaultConfig.MaxBackupCount != 5 {
		t.Fatalf("expected default max backup count 5, got %+v", defaultConfig.MaxBackupCount)
	}
	if defaultConfig.SlimBackup {
		t.Fatal("expected slim backup to be disabled by default")
	}

	backupDir := "/custom/backup"
	maxCount := uint32(10)
	config := BackupConfig{
		BackupDirectory:         &backupDir,
		AutoBackupEnabled:       true,
		AutoBackupIntervalHours: 12,
		MaxBackupCount:          &maxCount,
		SlimBackup:              true,
		BackupTiers:             []string{"core", "important"},
	}
	if err := service.SetBackupConfig(config); err != nil {
		t.Fatal(err)
	}

	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := reloaded.GetBackupConfig()
	if err != nil {
		t.Fatal(err)
	}
	if stored.BackupDirectory == nil || *stored.BackupDirectory != backupDir {
		t.Fatalf("unexpected backup directory: %+v", stored.BackupDirectory)
	}
	if !stored.AutoBackupEnabled || stored.AutoBackupIntervalHours != 12 {
		t.Fatalf("unexpected auto backup fields: %+v", stored)
	}
	if stored.MaxBackupCount == nil || *stored.MaxBackupCount != 10 {
		t.Fatalf("unexpected max count: %+v", stored.MaxBackupCount)
	}
	if !stored.SlimBackup {
		t.Fatal("expected slim backup to round-trip")
	}
	if len(stored.BackupTiers) != 2 || stored.BackupTiers[0] != "core" || stored.BackupTiers[1] != "important" {
		t.Fatalf("unexpected backup tiers: %+v", stored.BackupTiers)
	}
}

func TestBackupConfigPreservesLegacyPartialMaxBackupCount(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SaveSetting(backupConfigKey, `{"autoBackupEnabled":false,"autoBackupIntervalHours":48}`); err != nil {
		t.Fatal(err)
	}

	config, err := service.GetBackupConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.MaxBackupCount != nil {
		t.Fatalf("expected missing legacy maxBackupCount to remain nil, got %+v", config.MaxBackupCount)
	}
	if config.AutoBackupIntervalHours != 48 {
		t.Fatalf("expected stored interval to be preserved, got %d", config.AutoBackupIntervalHours)
	}
}

func TestStatisticsDefaults(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	basic := service.GetStatistics()
	if basic.TotalMistakes != 0 || basic.TotalReviews != 0 {
		t.Fatalf("expected empty basic statistics, got %+v", basic)
	}
	if len(basic.TypeStats) != 0 || len(basic.TagStats) != 0 || len(basic.RecentMistakes) != 0 {
		t.Fatalf("expected empty nested statistics, got %+v", basic)
	}

	enhanced := service.GetEnhancedStatistics(ImageStatistics{TotalFiles: 2, TotalSizeBytes: 128})
	if enhanced.BasicStats.TotalMistakes != 0 {
		t.Fatalf("expected empty enhanced basic stats, got %+v", enhanced.BasicStats)
	}
	if enhanced.ImageStats.TotalFiles != 2 || enhanced.ImageStats.TotalSizeBytes != 128 {
		t.Fatalf("expected supplied image stats, got %+v", enhanced.ImageStats)
	}
	if enhanced.Timestamp == "" {
		t.Fatal("expected enhanced statistics timestamp")
	}
}

func TestAPIConfigRecoveryCompatibility(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	status, err := service.CheckAPIConfigStatus()
	if err != nil {
		t.Fatal(err)
	}
	if status.ConfigCount != 0 || status.EnabledCount != 0 || status.HasAssignments || !status.NeedsRecovery {
		t.Fatalf("unexpected empty status: %+v", status)
	}

	message, err := service.RestoreDefaultAPIConfigs()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(message, "默认API配置已恢复") {
		t.Fatalf("unexpected recovery message: %q", message)
	}

	status, err = service.CheckAPIConfigStatus()
	if err != nil {
		t.Fatal(err)
	}
	if status.ConfigCount != 2 || status.EnabledCount != 0 || status.HasAssignments || status.NeedsRecovery {
		t.Fatalf("unexpected recovered status: %+v", status)
	}

	configs, err := service.GetAPIConfigurations()
	if err != nil {
		t.Fatal(err)
	}
	if configs[0].ID != "openai-gpt4" || configs[0].BaseUrl != "https://api.openai.com/v1" {
		t.Fatalf("unexpected OpenAI recovery config: %+v", configs[0])
	}
	if configs[1].ID != "claude-sonnet" || configs[1].ModelAdapter != "anthropic" {
		t.Fatalf("unexpected Claude recovery config: %+v", configs[1])
	}
}

func TestServiceGetSettingsByPrefix(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := service.SaveSettings(map[string]string{
		"tool_approval.global_bypass":      "true",
		"tool_approval.override.shell.run": "high",
		"theme":                            "dark",
	}); err != nil {
		t.Fatal(err)
	}

	values := service.GetSettingsByPrefix("tool_approval.")
	if len(values) != 2 {
		t.Fatalf("expected two prefixed settings, got %d: %+v", len(values), values)
	}
	if values[0][0] != "tool_approval.global_bypass" || values[0][1] != "true" {
		t.Fatalf("unexpected first row: %+v", values[0])
	}
	if values[1][0] != "tool_approval.override.shell.run" || values[1][1] != "high" {
		t.Fatalf("unexpected second row: %+v", values[1])
	}
	if len(values[0]) != 3 || values[0][2] != "" {
		t.Fatalf("expected legacy three-column row with empty updated_at, got %+v", values[0])
	}
}

func TestAttachmentConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	empty := service.GetAttachmentConfig()
	if empty.AttachmentRootFolderID != nil || empty.AttachmentRootFolderTitle != nil {
		t.Fatalf("expected empty config, got %+v", empty)
	}

	folderID, err := service.CreateAttachmentRootFolder("Uploads")
	if err != nil {
		t.Fatal(err)
	}
	if folderID == "" {
		t.Fatal("expected generated folder id")
	}

	config := service.GetAttachmentConfig()
	if config.AttachmentRootFolderID == nil || *config.AttachmentRootFolderID != folderID {
		t.Fatalf("unexpected folder id: %+v", config)
	}
	if config.AttachmentRootFolderTitle == nil || *config.AttachmentRootFolderTitle != "Uploads" {
		t.Fatalf("unexpected folder title: %+v", config)
	}

	if err := service.SetAttachmentRootFolder("folder_manual"); err != nil {
		t.Fatal(err)
	}

	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	reloadedConfig := reloaded.GetAttachmentConfig()
	if reloadedConfig.AttachmentRootFolderID == nil || *reloadedConfig.AttachmentRootFolderID != "folder_manual" {
		t.Fatalf("unexpected reloaded folder id: %+v", reloadedConfig)
	}
	if reloadedConfig.AttachmentRootFolderTitle != nil {
		t.Fatalf("expected manual folder title to be unknown, got %+v", reloadedConfig)
	}
}

func TestMemoryConfigDefaultsAndStoredValues(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	defaultConfig := service.GetMemoryConfig()
	if defaultConfig.MemoryRootFolderID != nil || defaultConfig.MemoryRootFolderTitle != nil {
		t.Fatalf("expected empty memory root config, got %+v", defaultConfig)
	}
	if !defaultConfig.AutoCreateSubfolders || defaultConfig.DefaultCategory != "通用" || defaultConfig.PrivacyMode || defaultConfig.AutoExtractFrequency != "balanced" {
		t.Fatalf("unexpected default memory config: %+v", defaultConfig)
	}

	if err := service.SaveSettings(map[string]string{
		memoryRootFolderIDKey:         "folder_memory",
		memoryRootFolderTitleKey:      "Memory",
		memoryAutoCreateSubfoldersKey: "false",
		memoryDefaultCategoryKey:      "study",
		memoryPrivacyModeKey:          "true",
		memoryAutoExtractFrequencyKey: "aggressive",
	}); err != nil {
		t.Fatal(err)
	}

	config := service.GetMemoryConfig()
	if config.MemoryRootFolderID == nil || *config.MemoryRootFolderID != "folder_memory" {
		t.Fatalf("unexpected memory root folder id: %+v", config)
	}
	if config.MemoryRootFolderTitle == nil || *config.MemoryRootFolderTitle != "Memory" {
		t.Fatalf("unexpected memory root folder title: %+v", config)
	}
	if config.AutoCreateSubfolders || config.DefaultCategory != "study" || !config.PrivacyMode || config.AutoExtractFrequency != "aggressive" {
		t.Fatalf("unexpected stored memory config: %+v", config)
	}

	if err := service.SaveSettings(map[string]string{
		memoryRootFolderIDKey:         "",
		memoryRootFolderTitleKey:      "Stale",
		memoryAutoExtractFrequencyKey: "unknown",
	}); err != nil {
		t.Fatal(err)
	}
	config = service.GetMemoryConfig()
	if config.MemoryRootFolderID != nil || config.MemoryRootFolderTitle != nil {
		t.Fatalf("expected stale title to be hidden without root id, got %+v", config)
	}
	if config.AutoExtractFrequency != "balanced" {
		t.Fatalf("expected unknown frequency to fall back to balanced, got %+v", config)
	}
}

func TestCNWhitelistConfig(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	empty := service.GetCNWhitelistConfig()
	if empty.UserConfig.Enabled {
		t.Fatal("expected whitelist to be disabled by default")
	}
	if !empty.UserConfig.UseDefaultList {
		t.Fatal("expected default trusted site list to be enabled by default")
	}
	if len(empty.DefaultSites) == 0 {
		t.Fatal("expected default trusted sites")
	}

	if err := service.SaveSettings(map[string]string{
		"web_search.cn_whitelist.enabled":      "true",
		"web_search.cn_whitelist.use_default":  "false",
		"web_search.cn_whitelist.custom_sites": `["example.cn"," example.edu.cn ","example.cn"]`,
	}); err != nil {
		t.Fatal(err)
	}

	config := service.GetCNWhitelistConfig()
	if !config.UserConfig.Enabled {
		t.Fatal("expected whitelist to be enabled")
	}
	if config.UserConfig.UseDefaultList {
		t.Fatal("expected default list to be disabled")
	}
	if len(config.UserConfig.CustomSites) != 2 {
		t.Fatalf("expected two normalized custom sites, got %+v", config.UserConfig.CustomSites)
	}
	if config.UserConfig.CustomSites[0] != "example.cn" || config.UserConfig.CustomSites[1] != "example.edu.cn" {
		t.Fatalf("unexpected custom sites: %+v", config.UserConfig.CustomSites)
	}

	if err := service.SaveSetting("web_search.cn_whitelist.custom_sites", "a.cn, b.cn,,a.cn"); err != nil {
		t.Fatal(err)
	}
	config = service.GetCNWhitelistConfig()
	if len(config.UserConfig.CustomSites) != 2 || config.UserConfig.CustomSites[0] != "a.cn" || config.UserConfig.CustomSites[1] != "b.cn" {
		t.Fatalf("expected comma-separated custom sites to parse, got %+v", config.UserConfig.CustomSites)
	}
}

func TestProviderStrategiesConfigDefaultsAndRoundTrip(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	defaults := service.GetProviderStrategiesConfig()
	if defaults.ProviderStrategies.Default.TimeoutMS == nil || *defaults.ProviderStrategies.Default.TimeoutMS != 8000 {
		t.Fatalf("expected default timeout 8000ms, got %+v", defaults.ProviderStrategies.Default.TimeoutMS)
	}
	if defaults.ProviderStrategies.GoogleCSE == nil || defaults.ProviderStrategies.GoogleCSE.TimeoutMS == nil || *defaults.ProviderStrategies.GoogleCSE.TimeoutMS != 6000 {
		t.Fatalf("expected Google CSE provider default, got %+v", defaults.ProviderStrategies.GoogleCSE)
	}
	if defaults.ProviderStrategies.SerpAPI == nil || defaults.ProviderStrategies.SerpAPI.SpecialHandling == nil || !defaults.ProviderStrategies.SerpAPI.SpecialHandling.CircuitBreakerEnabled {
		t.Fatalf("expected SerpAPI circuit breaker default, got %+v", defaults.ProviderStrategies.SerpAPI)
	}
	if defaults.ConfigKeys["provider_strategies"] != webSearchProviderStrategiesKey {
		t.Fatalf("unexpected config key map: %+v", defaults.ConfigKeys)
	}

	strategies := defaults.ProviderStrategies
	strategies.Tavily = &ProviderStrategy{
		TimeoutMS:          uint64Ptr(12345),
		MaxRetries:         uint32Ptr(4),
		RateLimitPerMinute: uint32Ptr(44),
	}
	ok, err := service.SaveProviderStrategiesConfig(strategies)
	if err != nil || !ok {
		t.Fatalf("save provider strategies failed: ok=%v err=%v", ok, err)
	}

	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	loaded := reloaded.GetProviderStrategiesConfig()
	if loaded.ProviderStrategies.Tavily == nil || loaded.ProviderStrategies.Tavily.TimeoutMS == nil || *loaded.ProviderStrategies.Tavily.TimeoutMS != 12345 {
		t.Fatalf("expected saved Tavily strategy, got %+v", loaded.ProviderStrategies.Tavily)
	}
}

func TestProviderStrategiesConfigReadsLegacyJSON(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SaveSetting(webSearchProviderStrategiesKey, `{"brave":{"timeout_ms":4321,"max_retries":1}}`); err != nil {
		t.Fatal(err)
	}

	config := service.GetProviderStrategiesConfig()
	if config.ProviderStrategies.Default.TimeoutMS == nil || *config.ProviderStrategies.Default.TimeoutMS != 8000 {
		t.Fatalf("expected missing legacy default to be filled, got %+v", config.ProviderStrategies.Default)
	}
	if config.ProviderStrategies.Brave == nil || config.ProviderStrategies.Brave.TimeoutMS == nil || *config.ProviderStrategies.Brave.TimeoutMS != 4321 {
		t.Fatalf("expected legacy Brave override, got %+v", config.ProviderStrategies.Brave)
	}
}

func TestPreheatMCPToolsCountsConfiguredEntries(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	empty := service.PreheatMCPTools()
	if !empty.Ok || empty.Count != 0 {
		t.Fatalf("expected empty preheat result, got %+v", empty)
	}

	if err := service.SaveSetting(mcpToolsListKey, `[
		{"id":"docs","transportType":"sse","url":"http://localhost:8080/sse"},
		{"id":"files","transportType":"stdio","command":"npx","args":["@modelcontextprotocol/server-filesystem"]}
	]`); err != nil {
		t.Fatal(err)
	}

	result := service.PreheatMCPTools()
	if !result.Ok || result.Count != 2 {
		t.Fatalf("expected two configured MCP entries, got %+v", result)
	}
}

func TestMCPStatusCompatibility(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	empty := service.GetMCPStatus()
	if empty.Available || empty.Enabled || empty.Connected {
		t.Fatalf("expected disabled backend MCP status, got %+v", empty)
	}
	if empty.LastError != "backend_mcp_disabled" {
		t.Fatalf("unexpected last error: %q", empty.LastError)
	}
	if empty.ConflictResolution != "use_namespace" {
		t.Fatalf("expected default conflict resolution, got %q", empty.ConflictResolution)
	}
	if empty.CacheState.TTLMs != 300_000 || empty.CacheState.LastBuiltAt != nil {
		t.Fatalf("unexpected cache state: %+v", empty.CacheState)
	}

	if err := service.SaveSettings(map[string]string{
		"mcp.tools.namespace_prefix":    "mcp.",
		"mcp.tools.conflict_resolution": "use_mcp",
		"mcp.tools.cache_ttl_ms":        "1200",
		"session.selected_mcp_tools":    "docs,files",
	}); err != nil {
		t.Fatal(err)
	}

	enabled := service.GetMCPStatus()
	if !enabled.Enabled || enabled.EnabledReason != nil {
		t.Fatalf("expected session-selected MCP status to be enabled, got %+v", enabled)
	}
	if enabled.NamespacePrefix != "mcp." || enabled.ConflictResolution != "use_mcp" {
		t.Fatalf("unexpected policy fields: %+v", enabled)
	}
	if enabled.CacheState.TTLMs != 1200 {
		t.Fatalf("unexpected cache ttl: %+v", enabled.CacheState)
	}

	if err := service.SaveSetting("session.selected_mcp_tools", " "); err != nil {
		t.Fatal(err)
	}
	disabled := service.GetMCPStatus()
	if disabled.Enabled || disabled.EnabledReason == nil || *disabled.EnabledReason == "" {
		t.Fatalf("expected empty session selection reason, got %+v", disabled)
	}
}

func TestReloadMCPClientCompatibility(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	result := service.ReloadMCPClient()
	if !result.Success || result.Message == "" || result.Error != nil {
		t.Fatalf("unexpected reload result: %+v", result)
	}
}

func TestGetMCPToolsCompatibility(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	tools := service.GetMCPTools()
	if tools == nil {
		t.Fatal("expected empty MCP tool slice, got nil")
	}
	if len(tools) != 0 {
		t.Fatalf("expected backend MCP tools to be empty, got %+v", tools)
	}
}

func TestOCRSettingsCompatibility(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	if service.GetOCREngineType() != "paddle_ocr_vl" {
		t.Fatalf("expected default OCR engine, got %q", service.GetOCREngineType())
	}
	if service.GetOCRThinkingEnabled() {
		t.Fatal("expected OCR thinking to be disabled by default")
	}
	if ok, err := service.SetOCRThinkingEnabled(true); err != nil || !ok {
		t.Fatalf("set OCR thinking failed: ok=%v err=%v", ok, err)
	}
	if !service.GetOCRThinkingEnabled() {
		t.Fatal("expected OCR thinking setting to persist")
	}

	engines := service.GetOCREngines()
	if len(engines) < 5 {
		t.Fatalf("expected OCR engine metadata, got %+v", engines)
	}
	if engines[0].EngineType != "deepseek_ocr" || engines[0].RecommendedModel == "" {
		t.Fatalf("unexpected first OCR engine metadata: %+v", engines[0])
	}

	if err := service.SaveAPIConfigurations([]ApiConfig{
		{ID: "api_paddle", Name: "Paddle", Model: "PaddlePaddle/PaddleOCR-VL", Enabled: true, IsMultimodal: true},
		{ID: "api_glm", Name: "GLM", Model: "zai-org/GLM-4.6V", Enabled: true, IsMultimodal: true},
	}); err != nil {
		t.Fatal(err)
	}

	if ok, err := service.SaveAvailableOCRModels([]SaveOCRModelRequest{
		{
			ConfigID:   "api_paddle",
			Model:      "PaddlePaddle/PaddleOCR-VL",
			EngineType: "paddle_ocr_vl",
			Name:       "PaddleOCR-VL",
			IsFree:     true,
			Enabled:    boolPtr(true),
		},
	}); err != nil || !ok {
		t.Fatalf("save OCR models failed: ok=%v err=%v", ok, err)
	}

	models, err := service.GetAvailableOCRModels()
	if err != nil {
		t.Fatal(err)
	}
	paddle := findOCRModel(models, "api_paddle")
	if paddle == nil {
		t.Fatalf("expected paddle OCR model, got %+v", models)
	}
	if paddle.Model != "PaddlePaddle/PaddleOCR-VL-1.5" {
		t.Fatalf("expected PaddleOCR migration to 1.5, got %+v", paddle)
	}
	if paddle.Description == nil || *paddle.Description == "" || !paddle.SupportsGrounding {
		t.Fatalf("expected merged OCR engine metadata, got %+v", paddle)
	}

	if ok, err := service.AddOCREngine("api_glm", "zai-org/GLM-4.6V", "GLM", nil); err != nil || !ok {
		t.Fatalf("add OCR engine failed: ok=%v err=%v", ok, err)
	}
	if ok, err := service.UpdateOCREnginePriority([]UpdateOCRPriorityItem{
		{ConfigID: "api_glm", Enabled: false},
		{ConfigID: "api_paddle", Enabled: true},
	}); err != nil || !ok {
		t.Fatalf("update OCR priority failed: ok=%v err=%v", ok, err)
	}
	models, err = service.GetAvailableOCRModels()
	if err != nil {
		t.Fatal(err)
	}
	glm := findOCRModel(models, "api_glm")
	if glm == nil || glm.Priority != 0 || glm.Enabled || glm.EngineType != "glm4v_ocr" {
		t.Fatalf("expected disabled GLM OCR engine first, got %+v in %+v", glm, models)
	}

	if ok, err := service.RemoveOCREngine("api_glm"); err != nil || !ok {
		t.Fatalf("remove OCR engine failed: ok=%v err=%v", ok, err)
	}
	models, err = service.GetAvailableOCRModels()
	if err != nil {
		t.Fatal(err)
	}
	if findOCRModel(models, "api_glm") != nil {
		t.Fatalf("expected GLM OCR engine to be removed, got %+v", models)
	}
}

func TestOCREngineSendsVisionRequestToConfiguredProvider(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	vendorID := "vendor_ocr"
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("expected /v1/chat/completions, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer config-key" {
			t.Fatalf("expected config bearer key, got %q", got)
		}
		if got := r.Header.Get("X-Vendor"); got != "vendor-header" {
			t.Fatalf("expected vendor header, got %q", got)
		}
		if got := r.Header.Get("X-Override"); got != "config-header" {
			t.Fatalf("expected config header override, got %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"recognized text"}}]}`))
	}))
	defer server.Close()

	if err := service.SaveVendorConfigs([]VendorConfig{{
		ID:      vendorID,
		Name:    "OCR Vendor",
		ApiKey:  "vendor-key",
		Headers: map[string]string{"X-Vendor": "vendor-header", "X-Override": "vendor-header"},
	}}); err != nil {
		t.Fatal(err)
	}
	if err := service.SaveAPIConfigurations([]ApiConfig{{
		ID:              "api_paddle",
		Name:            "Paddle Config",
		VendorID:        &vendorID,
		ApiKey:          "config-key",
		BaseUrl:         server.URL + "/v1",
		Model:           "PaddlePaddle/PaddleOCR-VL-1.5",
		IsMultimodal:    true,
		Enabled:         false,
		MaxOutputTokens: 8192,
		Headers:         map[string]string{"X-Override": "config-header"},
	}}); err != nil {
		t.Fatal(err)
	}
	if ok, err := service.SaveAvailableOCRModels([]SaveOCRModelRequest{{
		ConfigID:   "api_paddle",
		Model:      "PaddlePaddle/PaddleOCR-VL-1.5",
		EngineType: "paddle_ocr_vl",
		Name:       "Paddle UI",
		IsFree:     true,
		Enabled:    boolPtr(true),
	}}); err != nil || !ok {
		t.Fatalf("save OCR model failed: ok=%v err=%v", ok, err)
	}

	response, err := service.TestOCREngine(OCRTestRequest{
		ImageBase64: "data:image/png;base64,AQID",
		EngineType:  "generic_vlm",
		ConfigID:    strPtr("api_paddle"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !response.Success || response.Error != nil {
		t.Fatalf("expected successful OCR test, got %+v", response)
	}
	if response.EngineType != "paddle_ocr_vl" || response.EngineName != "Paddle UI" {
		t.Fatalf("expected OCR model metadata to win, got %+v", response)
	}
	if response.Text != "recognized text" || len(response.Regions) != 1 || response.Regions[0].Text != "recognized text" {
		t.Fatalf("expected fallback text region, got %+v", response)
	}
	if requestBody["model"] != "PaddlePaddle/PaddleOCR-VL-1.5" {
		t.Fatalf("unexpected OCR model in body: %+v", requestBody)
	}
	if requestBody["stream"] != false || int(requestBody["max_tokens"].(float64)) != 8000 {
		t.Fatalf("expected non-streaming clamped body, got %+v", requestBody)
	}
	if requestBody["repetition_penalty"] != 1.1 {
		t.Fatalf("expected Paddle repetition penalty, got %+v", requestBody)
	}
	messages := requestBody["messages"].([]any)
	content := messages[0].(map[string]any)["content"].([]any)
	if content[0].(map[string]any)["type"] != "image_url" || content[1].(map[string]any)["type"] != "text" {
		t.Fatalf("expected image-first multimodal prompt, got %+v", content)
	}
	imageURL := content[0].(map[string]any)["image_url"].(map[string]any)["url"]
	if imageURL != "data:image/png;base64,AQID" {
		t.Fatalf("expected uploaded data URL, got %+v", imageURL)
	}
}

func TestOCREngineReturnsStructuredFailureWhenConfigIsMissing(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	service.SetHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("OCR diagnostic should not call provider when config is missing")
		return nil, nil
	})})

	response, err := service.TestOCREngine(OCRTestRequest{
		ImageBase64: "AQID",
		EngineType:  "deepseek_ocr",
		ConfigID:    strPtr("missing_config"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Success || response.Error == nil || !strings.Contains(*response.Error, "未找到 OCR API 配置") {
		t.Fatalf("expected structured missing-config failure, got %+v", response)
	}
	if response.EngineType != "deepseek_ocr" || response.EngineName == "" {
		t.Fatalf("expected canonical metadata on failure, got %+v", response)
	}
}

func TestOCREngineRejectsInvalidImageBase64(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.TestOCREngine(OCRTestRequest{
		ImageBase64: "data:image/png;base64",
		EngineType:  "paddle_ocr_vl",
	})
	if err == nil || !strings.Contains(err.Error(), "图片解析失败") {
		t.Fatalf("expected image parse error, got %v", err)
	}

	_, err = service.TestOCREngine(OCRTestRequest{
		ImageBase64: "not base64",
		EngineType:  "paddle_ocr_vl",
	})
	if err == nil || !strings.Contains(err.Error(), "Base64 decode error") {
		t.Fatalf("expected base64 decode error, got %v", err)
	}
}

func TestOCREngineParsesPaddleRegions(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"blocks\":[{\"type\":\"text\",\"content\":\"Block A\",\"bbox\":[0.1,0.2,0.3,0.4]}]}"}}]}`))
	}))
	defer server.Close()

	if err := service.SaveAPIConfigurations([]ApiConfig{{
		ID:           "api_paddle",
		Name:         "Paddle Config",
		ApiKey:       "config-key",
		BaseUrl:      server.URL + "/v1",
		Model:        "PaddlePaddle/PaddleOCR-VL-1.5",
		IsMultimodal: true,
		Enabled:      true,
	}}); err != nil {
		t.Fatal(err)
	}
	if ok, err := service.SaveAvailableOCRModels([]SaveOCRModelRequest{{
		ConfigID:   "api_paddle",
		Model:      "PaddlePaddle/PaddleOCR-VL-1.5",
		EngineType: "paddle_ocr_vl",
		Name:       "Paddle UI",
		IsFree:     true,
		Enabled:    boolPtr(true),
	}}); err != nil || !ok {
		t.Fatalf("save OCR model failed: ok=%v err=%v", ok, err)
	}

	response, err := service.TestOCREngine(OCRTestRequest{
		ImageBase64: "AQID",
		EngineType:  "paddle_ocr_vl",
		ConfigID:    strPtr("api_paddle"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !response.Success || response.Text != "Block A" || len(response.Regions) != 1 {
		t.Fatalf("expected parsed Paddle region, got %+v", response)
	}
	bbox := response.Regions[0].BBox
	if bbox == nil || *bbox != [4]float64{0.1, 0.2, 0.3, 0.4} {
		t.Fatalf("expected normalized bbox to survive, got %+v", bbox)
	}
}

func TestAPIConfigurationsRoundTrip(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	providerScope := "deepseek"
	reasoningEffort := "medium"
	if err := service.SaveAPIConfigurations([]ApiConfig{
		{
			ID:              " api_main ",
			Name:            "",
			VendorID:        strPtr(" vendor_openai "),
			ProviderType:    strPtr("openai"),
			ProviderScope:   &providerScope,
			ApiKey:          "secret",
			BaseUrl:         " https://api.example.com/v1 ",
			Model:           " gpt-test ",
			IsMultimodal:    true,
			Enabled:         true,
			ReasoningEffort: &reasoningEffort,
		},
	}); err != nil {
		t.Fatal(err)
	}

	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	configs, err := reloaded.GetAPIConfigurations()
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected one config, got %d", len(configs))
	}
	config := configs[0]
	if config.ID != "api_main" {
		t.Fatalf("expected trimmed id, got %q", config.ID)
	}
	if config.Name != "gpt-test" {
		t.Fatalf("expected model fallback name, got %q", config.Name)
	}
	if config.BaseUrl != "https://api.example.com/v1" {
		t.Fatalf("expected trimmed baseUrl, got %q", config.BaseUrl)
	}
	if config.Model != "gpt-test" {
		t.Fatalf("expected trimmed model, got %q", config.Model)
	}
	if config.ModelAdapter != "general" {
		t.Fatalf("expected default adapter, got %q", config.ModelAdapter)
	}
	if config.MaxOutputTokens != 4096 {
		t.Fatalf("expected default max output tokens, got %d", config.MaxOutputTokens)
	}
	if config.Temperature != 0.7 {
		t.Fatalf("expected default temperature, got %f", config.Temperature)
	}
	if config.GeminiApiVersion != "v1" {
		t.Fatalf("expected default Gemini API version, got %q", config.GeminiApiVersion)
	}
}

func TestAPIConnectionUsesChatCompletionsWhenProtocolIsChat(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("expected /v1/chat/completions, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer real-key" {
			t.Fatalf("expected bearer auth, got %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "model-a" {
			t.Fatalf("expected selected model, got %#v", body["model"])
		}
		if body["stream"] != false {
			t.Fatalf("expected non-streaming request, got %#v", body["stream"])
		}
		if body["max_tokens"] != float64(1) {
			t.Fatalf("expected max_tokens=1, got %#v", body["max_tokens"])
		}
		if _, ok := body["input"]; ok {
			t.Fatalf("chat-completions body should not include input: %+v", body)
		}
		if _, ok := body["max_output_tokens"]; ok {
			t.Fatalf("chat-completions body should not include max_output_tokens: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	ok, err := service.TestAPIConnection("real-key", server.URL+"/v1", strPtr("openai_chat_completions"), nil, strPtr("model-a"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected successful connection test")
	}
}

func TestAPIConnectionUsesOpenAIResponsesWhenProtocolIsExplicit(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("expected /v1/responses, got %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "gpt-5" || body["input"] != "Hi" {
			t.Fatalf("unexpected responses body: %+v", body)
		}
		if body["max_output_tokens"] != float64(1) || body["stream"] != false {
			t.Fatalf("expected responses max_output_tokens/stream, got %+v", body)
		}
		if _, ok := body["messages"]; ok {
			t.Fatalf("responses body should not include messages: %+v", body)
		}
		if _, ok := body["max_tokens"]; ok {
			t.Fatalf("responses body should not include max_tokens: %+v", body)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	ok, err := service.TestAPIConnection("real-key", server.URL+"/v1", strPtr("openai_responses"), boolPtr(true), strPtr("gpt-5"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected successful responses connection test")
	}
}

func TestAPIConnectionDefaultsOfficialOpenAIToResponses(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	service.SetHTTPClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://api.openai.com/v1/responses" {
			t.Fatalf("expected official OpenAI responses endpoint, got %s", r.URL.String())
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "gpt-4o-mini" || body["input"] != "Hi" {
			t.Fatalf("unexpected default responses body: %+v", body)
		}
		return jsonResponse(http.StatusOK, `{}`), nil
	})})

	ok, err := service.TestAPIConnection("real-key", "https://api.openai.com/v1", nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected successful official OpenAI responses connection test")
	}
}

func TestAPIConnectionUsesResponsesForThirdPartyWhenDeclaredSupported(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("expected /v1/responses, got %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	ok, err := service.TestAPIConnection("real-key", server.URL+"/v1", nil, boolPtr(true), strPtr("gpt-4o-mini"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected successful supported third-party responses test")
	}
}

func TestAPIConnectionDowngradesUnsupportedExplicitResponsesToChat(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("expected downgrade to /v1/chat/completions, got %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, ok := body["messages"]; !ok {
			t.Fatalf("expected chat body after downgrade, got %+v", body)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	ok, err := service.TestAPIConnection("real-key", server.URL+"/v1", strPtr("openai_responses"), boolPtr(false), strPtr("deepseek-chat"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected successful downgraded chat connection test")
	}
}

func TestAPIConnectionReportsProviderErrors(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad key", http.StatusUnauthorized)
	}))
	defer server.Close()

	ok, err := service.TestAPIConnection("real-key", server.URL+"/v1", nil, nil, strPtr("model-a"), nil)
	if ok {
		t.Fatal("expected failed connection test")
	}
	if err == nil || !strings.Contains(err.Error(), "401 Unauthorized") || !strings.Contains(err.Error(), "bad key") {
		t.Fatalf("expected status/body error, got %v", err)
	}
}

func TestAPIConnectionReportsResponsesProviderErrors(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("expected /v1/responses, got %s", r.URL.Path)
		}
		http.Error(w, `{"error":{"message":"bad responses key"}}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	ok, err := service.TestAPIConnection("real-key", server.URL+"/v1", strPtr("openai_responses"), boolPtr(true), strPtr("gpt-5"), nil)
	if ok {
		t.Fatal("expected failed responses connection test")
	}
	if err == nil || !strings.Contains(err.Error(), "401 Unauthorized") || !strings.Contains(err.Error(), "bad responses key") {
		t.Fatalf("expected responses status/body error, got %v", err)
	}
}

func TestAPIConnectionResolvesMaskedVendorKey(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer stored-key" {
			t.Fatalf("expected stored key, got %q", got)
		}
		if got := r.Header.Get("X-Vendor-Test"); got != "yes" {
			t.Fatalf("expected custom vendor header, got %q", got)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	if err := service.SaveVendorConfigs([]VendorConfig{
		{
			ID:           "vendor-main",
			Name:         "Vendor",
			ProviderType: "openai",
			BaseUrl:      server.URL + "/v1/chat/completions",
			ApiKey:       "stored-key",
			Headers:      map[string]string{"X-Vendor-Test": "yes"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	ok, err := service.TestAPIConnection("***", "", nil, nil, strPtr("model-a"), strPtr("vendor-main"))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected successful masked-key connection test")
	}
}

func TestAPIConnectionResolvesMaskedVendorKeyForResponses(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("expected /v1/responses, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer stored-key" {
			t.Fatalf("expected stored key, got %q", got)
		}
		if got := r.Header.Get("X-Vendor-Test"); got != "yes" {
			t.Fatalf("expected custom vendor header, got %q", got)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	if err := service.SaveVendorConfigs([]VendorConfig{
		{
			ID:                      "vendor-main",
			Name:                    "Vendor",
			ProviderType:            "openai",
			BaseUrl:                 server.URL + "/v1",
			ApiKey:                  "stored-key",
			Headers:                 map[string]string{"X-Vendor-Test": "yes"},
			SupportsOpenAIResponses: boolPtr(true),
		},
	}); err != nil {
		t.Fatal(err)
	}

	ok, err := service.TestAPIConnection("***", "", strPtr("openai_responses"), boolPtr(true), strPtr("gpt-5"), strPtr("vendor-main"))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected successful masked-key responses connection test")
	}
}

func TestAPIConnectionDoesNotDuplicateChatCompletionsSuffix(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("expected no duplicated suffix, got %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	ok, err := service.TestAPIConnection("real-key", server.URL+"/v1/chat/completions", nil, nil, strPtr("model-a"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected successful connection test")
	}
}

func TestAPIConnectionDoesNotDuplicateResponsesSuffix(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("expected no duplicated responses suffix, got %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	ok, err := service.TestAPIConnection("real-key", server.URL+"/v1/responses", strPtr("openai_responses"), boolPtr(true), strPtr("gpt-5"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected successful responses connection test")
	}
}

func TestAPIConnectionSwitchesChatSuffixToResponses(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("expected switched responses suffix, got %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	ok, err := service.TestAPIConnection("real-key", server.URL+"/v1/chat/completions", strPtr("openai_responses"), boolPtr(true), strPtr("gpt-5"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected successful switched responses connection test")
	}
}

func TestSearchEngineReportsMissingConfiguration(t *testing.T) {
	clearSearchEnv(t)
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.TestSearchEngine("google_cse")
	if err != nil {
		t.Fatal(err)
	}
	if result.Ok {
		t.Fatalf("expected missing config to fail, got %+v", result)
	}
	if !strings.Contains(result.Message, "缺少API密钥或端点配置") {
		t.Fatalf("expected missing configuration message, got %q", result.Message)
	}
}

func TestSearchEngineUsesSearxngEndpoint(t *testing.T) {
	clearSearchEnv(t)
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Fatalf("expected /search, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("q"); got != "AI artificial intelligence" {
			t.Fatalf("expected test query, got %q", got)
		}
		if got := r.URL.Query().Get("format"); got != "json" {
			t.Fatalf("expected json format, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"title":"ok","url":"https://example.test"}]}`))
	}))
	defer server.Close()

	if err := service.SaveSetting("web_search.searxng.endpoint", server.URL); err != nil {
		t.Fatal(err)
	}

	result, err := service.TestSearchEngine("searxng")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ok {
		t.Fatalf("expected successful searxng probe, got %+v", result)
	}
	if result.ResultsCount == nil || *result.ResultsCount != 1 {
		t.Fatalf("expected one search result, got %+v", result.ResultsCount)
	}
}

func TestSearchEngineUsesGoogleCSERequestShape(t *testing.T) {
	clearSearchEnv(t)
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	service.SetHTTPClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Host != "www.googleapis.com" || r.URL.Path != "/customsearch/v1" {
			t.Fatalf("unexpected Google CSE URL: %s", r.URL.String())
		}
		if got := r.URL.Query().Get("key"); got != "google-key" {
			t.Fatalf("expected key query, got %q", got)
		}
		if got := r.URL.Query().Get("cx"); got != "cx-id" {
			t.Fatalf("expected cx query, got %q", got)
		}
		if got := r.URL.Query().Get("num"); got != "1" {
			t.Fatalf("expected num=1, got %q", got)
		}
		return jsonResponse(http.StatusOK, `{"items":[{},{}]}`), nil
	})})
	if err := service.SaveSettings(map[string]string{
		"web_search.api_key.google_cse": "google-key",
		"web_search.google_cse.cx":      "cx-id",
	}); err != nil {
		t.Fatal(err)
	}

	result, err := service.TestSearchEngine("google_cse")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ok || result.ResultsCount == nil || *result.ResultsCount != 2 {
		t.Fatalf("expected successful Google CSE probe with two results, got %+v", result)
	}
}

func TestWebSearchConnectivityUsesConfiguredDefault(t *testing.T) {
	clearSearchEnv(t)
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("q"); got != "connectivity test" {
			t.Fatalf("expected connectivity query, got %q", got)
		}
		_, _ = w.Write([]byte(`{"results":[{}]}`))
	}))
	defer server.Close()

	if err := service.SaveSettings(map[string]string{
		"web_search.engine":           "searxng",
		"web_search.searxng.endpoint": server.URL,
	}); err != nil {
		t.Fatal(err)
	}

	result, err := service.TestWebSearchConnectivity(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected successful connectivity probe, got %+v", result)
	}
	if result.Usage["provider"] != "searxng" {
		t.Fatalf("expected searxng provider usage, got %+v", result.Usage)
	}
}

func TestAllSearchEnginesSummarizesConfiguredEngines(t *testing.T) {
	clearSearchEnv(t)
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{}]}`))
	}))
	defer server.Close()

	if err := service.SaveSetting("web_search.searxng.endpoint", server.URL); err != nil {
		t.Fatal(err)
	}

	report, err := service.TestAllSearchEngines()
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Total != 7 || report.Summary.Configured != 1 || report.Summary.Success != 1 || report.Summary.Failed != 0 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	if report.Results["searxng"].Status != "success" {
		t.Fatalf("expected searxng success, got %+v", report.Results["searxng"])
	}
	if report.Results["google_cse"].Status != "not_configured" {
		t.Fatalf("expected google_cse not configured, got %+v", report.Results["google_cse"])
	}
}

func TestSearchEngineReportsProviderErrors(t *testing.T) {
	clearSearchEnv(t)
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":{"message":"bad key"}}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	if err := service.SaveSetting("web_search.searxng.endpoint", server.URL); err != nil {
		t.Fatal(err)
	}

	result, err := service.TestSearchEngine("searxng")
	if err != nil {
		t.Fatal(err)
	}
	if result.Ok {
		t.Fatalf("expected failed provider probe, got %+v", result)
	}
	if result.ErrorDetails == nil || !strings.Contains(*result.ErrorDetails, "401 Unauthorized") || !strings.Contains(*result.ErrorDetails, "bad key") {
		t.Fatalf("expected status/body error details, got %+v", result.ErrorDetails)
	}
}

func TestVendorConfigsRoundTrip(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	website := "https://example.com"
	if err := service.SaveVendorConfigs([]VendorConfig{
		{
			ID:           " vendor_main ",
			Name:         "",
			ProviderType: "",
			BaseUrl:      " https://api.vendor.test/v1 ",
			ApiKey:       "secret",
			WebsiteUrl:   &website,
		},
	}); err != nil {
		t.Fatal(err)
	}

	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	configs, err := reloaded.GetVendorConfigs()
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected one vendor, got %d", len(configs))
	}
	config := configs[0]
	if config.ID != "vendor_main" {
		t.Fatalf("expected trimmed id, got %q", config.ID)
	}
	if config.ProviderType != "openai" {
		t.Fatalf("expected default provider, got %q", config.ProviderType)
	}
	if config.Name != "vendor_main" {
		t.Fatalf("expected id fallback name, got %q", config.Name)
	}
	if config.BaseUrl != "https://api.vendor.test/v1" {
		t.Fatalf("expected trimmed baseUrl, got %q", config.BaseUrl)
	}
	if config.Headers == nil {
		t.Fatal("expected headers to default to an empty map")
	}
	if config.WebsiteUrl == nil || *config.WebsiteUrl != website {
		t.Fatalf("expected website url to persist, got %+v", config.WebsiteUrl)
	}
}

func TestModelProfilesRoundTrip(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := service.SaveModelProfiles([]ModelProfile{
		{
			ID:         " profile_main ",
			VendorID:   " vendor_main ",
			Label:      "",
			Model:      " model-main ",
			Enabled:    true,
			IsReranker: true,
		},
	}); err != nil {
		t.Fatal(err)
	}

	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := reloaded.GetModelProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected one profile, got %d", len(profiles))
	}
	profile := profiles[0]
	if profile.ID != "profile_main" {
		t.Fatalf("expected trimmed id, got %q", profile.ID)
	}
	if profile.VendorID != "vendor_main" {
		t.Fatalf("expected trimmed vendor id, got %q", profile.VendorID)
	}
	if profile.Label != "model-main" {
		t.Fatalf("expected model fallback label, got %q", profile.Label)
	}
	if profile.ModelAdapter != "general" {
		t.Fatalf("expected default adapter, got %q", profile.ModelAdapter)
	}
	if profile.Status != "active" {
		t.Fatalf("expected active status, got %q", profile.Status)
	}
	if profile.MaxOutputTokens != 4096 {
		t.Fatalf("expected default max output tokens, got %d", profile.MaxOutputTokens)
	}
	if profile.Temperature != 0.7 {
		t.Fatalf("expected default temperature, got %f", profile.Temperature)
	}
}

func TestModelAssignmentsRoundTrip(t *testing.T) {
	dir := t.TempDir()

	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := service.SaveModelAssignments(ModelAssignments{
		Model2ConfigID:              strPtr(" api_main "),
		AnkiCardModelConfigID:       strPtr(" "),
		TranslationModelConfigID:    strPtr(" translator "),
		VoiceInputASRModelConfigID:  strPtr(" asr "),
		TranslationDisplayMode:      strPtr("side-by-side"),
		ReviewAnalysisModelConfigID: strPtr(" reviewer "),
	}); err != nil {
		t.Fatal(err)
	}

	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	assignments, err := reloaded.GetModelAssignments()
	if err != nil {
		t.Fatal(err)
	}
	if assignments.Model2ConfigID == nil || *assignments.Model2ConfigID != "api_main" {
		t.Fatalf("expected trimmed model2 assignment, got %+v", assignments.Model2ConfigID)
	}
	if assignments.AnkiCardModelConfigID != nil {
		t.Fatalf("expected empty assignment to be cleared, got %+v", assignments.AnkiCardModelConfigID)
	}
	if assignments.TranslationModelConfigID == nil || *assignments.TranslationModelConfigID != "translator" {
		t.Fatalf("expected translation assignment to persist, got %+v", assignments.TranslationModelConfigID)
	}
	if assignments.VoiceInputASRModelConfigID == nil || *assignments.VoiceInputASRModelConfigID != "asr" {
		t.Fatalf("expected ASR assignment to persist, got %+v", assignments.VoiceInputASRModelConfigID)
	}
	if assignments.ReviewAnalysisModelConfigID == nil || *assignments.ReviewAnalysisModelConfigID != "reviewer" {
		t.Fatalf("expected review assignment to persist, got %+v", assignments.ReviewAnalysisModelConfigID)
	}
	if assignments.TranslationDisplayMode == nil || *assignments.TranslationDisplayMode != "aligned" {
		t.Fatalf("expected invalid translation display mode to normalize to aligned, got %+v", assignments.TranslationDisplayMode)
	}

	if err := service.SaveModelAssignments(ModelAssignments{
		TranslationDisplayMode: strPtr("streaming"),
	}); err != nil {
		t.Fatal(err)
	}
	assignments, err = service.GetModelAssignments()
	if err != nil {
		t.Fatal(err)
	}
	if assignments.TranslationDisplayMode == nil || *assignments.TranslationDisplayMode != "streaming" {
		t.Fatalf("expected streaming display mode, got %+v", assignments.TranslationDisplayMode)
	}
}

func TestModelAdapterOptions(t *testing.T) {
	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	options := service.GetModelAdapterOptions()
	if len(options) != 11 {
		t.Fatalf("expected 11 model adapter options, got %d", len(options))
	}
	if options[0].Value != "general" || options[0].Label == "" || !options[0].IsDefault {
		t.Fatalf("expected general default adapter first, got %+v", options[0])
	}

	expectedValues := []string{
		"general",
		"google",
		"anthropic",
		"deepseek",
		"qwen",
		"zhipu",
		"doubao",
		"moonshot",
		"grok",
		"minimax",
		"mimo",
	}
	for index, expected := range expectedValues {
		if options[index].Value != expected {
			t.Fatalf("expected adapter %d to be %q, got %q", index, expected, options[index].Value)
		}
		if options[index].Label == "" || options[index].Description == "" {
			t.Fatalf("expected adapter %q to have display metadata, got %+v", expected, options[index])
		}
	}

	options[0].Value = "mutated"
	again := service.GetModelAdapterOptions()
	if again[0].Value != "general" {
		t.Fatalf("expected adapter options to be copied, got %q", again[0].Value)
	}
}

func strPtr(value string) *string {
	return &value
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func clearSearchEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"GOOGLE_API_KEY",
		"GOOGLE_CSE_CX",
		"SERPAPI_KEY",
		"TAVILY_API_KEY",
		"BRAVE_API_KEY",
		"SEARXNG_ENDPOINT",
		"SEARXNG_API_KEY",
		"ZHIPU_API_KEY",
		"BOCHA_API_KEY",
	} {
		t.Setenv(key, "")
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func findOCRModel(models []AvailableOCRModel, configID string) *AvailableOCRModel {
	for index := range models {
		if models[index].ConfigID == configID {
			return &models[index]
		}
	}
	return nil
}
