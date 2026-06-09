package bindings

import (
	"deep-student-go/internal/app"
	"deep-student-go/internal/settings"
	"deep-student-go/internal/vfs"
)

type SettingsService struct {
	app *app.App
}

func NewSettingsService(app *app.App) *SettingsService {
	return &SettingsService{app: app}
}

func (s *SettingsService) DataDir() string {
	return s.app.DataDir
}

func (s *SettingsService) GetSetting(key string) (string, error) {
	value, _ := s.app.Settings.GetSetting(key)
	return value, nil
}

func (s *SettingsService) GetSettings(keys []string) (map[string]string, error) {
	return s.app.Settings.GetSettings(keys), nil
}

func (s *SettingsService) GetSettingsByPrefix(prefix string) ([][]string, error) {
	return s.app.Settings.GetSettingsByPrefix(prefix), nil
}

func (s *SettingsService) SaveSetting(key string, value string) error {
	return s.app.Settings.SaveSetting(key, value)
}

func (s *SettingsService) SaveSettings(values map[string]string) error {
	return s.app.Settings.SaveSettings(values)
}

func (s *SettingsService) DeleteSetting(key string) error {
	return s.app.Settings.DeleteSetting(key)
}

func (s *SettingsService) GetBackupConfig() (settings.BackupConfig, error) {
	return s.app.Settings.GetBackupConfig()
}

func (s *SettingsService) SetBackupConfig(config settings.BackupConfig) error {
	return s.app.Settings.SetBackupConfig(config)
}

func (s *SettingsService) CheckAPIConfigStatus() (settings.APIConfigStatus, error) {
	return s.app.Settings.CheckAPIConfigStatus()
}

func (s *SettingsService) RestoreDefaultAPIConfigs() (string, error) {
	return s.app.Settings.RestoreDefaultAPIConfigs()
}

func (s *SettingsService) TestAPIConnection(apiKey string, apiBase string, apiProtocol *string, supportsOpenAIResponses *bool, model *string, vendorID *string) (bool, error) {
	return s.app.Settings.TestAPIConnection(apiKey, apiBase, apiProtocol, supportsOpenAIResponses, model, vendorID)
}

func (s *SettingsService) TestSearchEngine(engine string) (settings.SearchEngineTestResult, error) {
	return s.app.Settings.TestSearchEngine(engine)
}

func (s *SettingsService) TestWebSearchConnectivity(engine *string) (settings.WebSearchConnectivityResult, error) {
	return s.app.Settings.TestWebSearchConnectivity(engine)
}

func (s *SettingsService) TestAllSearchEngines() (settings.SearchEngineHealthReport, error) {
	return s.app.Settings.TestAllSearchEngines()
}

func (s *SettingsService) GetStatistics() settings.BasicStatistics {
	return s.app.Settings.GetStatistics()
}

func (s *SettingsService) GetEnhancedStatistics() (settings.EnhancedStatistics, error) {
	imageStats, err := s.getImageStatistics()
	if err != nil {
		return settings.EnhancedStatistics{}, err
	}
	return s.app.Settings.GetEnhancedStatistics(imageStats), nil
}

func (s *SettingsService) GetAttachmentConfig() settings.AttachmentConfig {
	return s.app.Settings.GetAttachmentConfig()
}

func (s *SettingsService) SetAttachmentRootFolder(folderID string) error {
	return s.app.Settings.SetAttachmentRootFolder(folderID)
}

func (s *SettingsService) CreateAttachmentRootFolder(title string) (string, error) {
	return s.app.Settings.CreateAttachmentRootFolder(title)
}

func (s *SettingsService) GetMemoryConfig() settings.MemoryConfig {
	return s.app.Settings.GetMemoryConfig()
}

func (s *SettingsService) GetModelAdapterOptions() []settings.ModelAdapterOption {
	return s.app.Settings.GetModelAdapterOptions()
}

func (s *SettingsService) GetCNWhitelistConfig() settings.CNWhitelistConfigResult {
	return s.app.Settings.GetCNWhitelistConfig()
}

func (s *SettingsService) GetProviderStrategiesConfig() settings.ProviderStrategiesConfigResult {
	return s.app.Settings.GetProviderStrategiesConfig()
}

func (s *SettingsService) SaveProviderStrategiesConfig(strategies settings.ProviderStrategies) (bool, error) {
	return s.app.Settings.SaveProviderStrategiesConfig(strategies)
}

func (s *SettingsService) PreheatMCPTools() settings.MCPPreheatResult {
	return s.app.Settings.PreheatMCPTools()
}

func (s *SettingsService) GetMCPStatus() settings.MCPStatus {
	return s.app.Settings.GetMCPStatus()
}

func (s *SettingsService) ReloadMCPClient() settings.MCPReloadResult {
	return s.app.Settings.ReloadMCPClient()
}

func (s *SettingsService) GetMCPTools() []settings.MCPToolInfo {
	return s.app.Settings.GetMCPTools()
}

func (s *SettingsService) GetOCREngines() []settings.OCREngineInfo {
	return s.app.Settings.GetOCREngines()
}

func (s *SettingsService) GetOCREngineType() string {
	return s.app.Settings.GetOCREngineType()
}

func (s *SettingsService) GetOCRThinkingEnabled() bool {
	return s.app.Settings.GetOCRThinkingEnabled()
}

func (s *SettingsService) SetOCRThinkingEnabled(enabled bool) (bool, error) {
	return s.app.Settings.SetOCRThinkingEnabled(enabled)
}

func (s *SettingsService) GetAvailableOCRModels() ([]settings.AvailableOCRModel, error) {
	return s.app.Settings.GetAvailableOCRModels()
}

func (s *SettingsService) TestOCREngine(request settings.OCRTestRequest) (settings.OCRTestResponse, error) {
	return s.app.Settings.TestOCREngine(request)
}

func (s *SettingsService) SaveAvailableOCRModels(models []settings.SaveOCRModelRequest) (bool, error) {
	return s.app.Settings.SaveAvailableOCRModels(models)
}

func (s *SettingsService) UpdateOCREnginePriority(engineList []settings.UpdateOCRPriorityItem) (bool, error) {
	return s.app.Settings.UpdateOCREnginePriority(engineList)
}

func (s *SettingsService) AddOCREngine(configID string, model string, name string, engineType *string) (bool, error) {
	return s.app.Settings.AddOCREngine(configID, model, name, engineType)
}

func (s *SettingsService) RemoveOCREngine(configID string) (bool, error) {
	return s.app.Settings.RemoveOCREngine(configID)
}

func (s *SettingsService) GetAPIConfigurations() ([]settings.ApiConfig, error) {
	return s.app.Settings.GetAPIConfigurations()
}

func (s *SettingsService) SaveAPIConfigurations(configs []settings.ApiConfig) error {
	return s.app.Settings.SaveAPIConfigurations(configs)
}

func (s *SettingsService) GetVendorConfigs() ([]settings.VendorConfig, error) {
	return s.app.Settings.GetVendorConfigs()
}

func (s *SettingsService) SaveVendorConfigs(configs []settings.VendorConfig) error {
	return s.app.Settings.SaveVendorConfigs(configs)
}

func (s *SettingsService) GetModelProfiles() ([]settings.ModelProfile, error) {
	return s.app.Settings.GetModelProfiles()
}

func (s *SettingsService) SaveModelProfiles(profiles []settings.ModelProfile) error {
	return s.app.Settings.SaveModelProfiles(profiles)
}

func (s *SettingsService) GetModelAssignments() (settings.ModelAssignments, error) {
	return s.app.Settings.GetModelAssignments()
}

func (s *SettingsService) SaveModelAssignments(assignments settings.ModelAssignments) error {
	return s.app.Settings.SaveModelAssignments(assignments)
}

func (s *SettingsService) getImageStatistics() (settings.ImageStatistics, error) {
	const pageSize = 500
	stats := settings.ImageStatistics{}
	for offset := 0; ; {
		files, err := s.app.Vfs.ListFiles(vfs.ListFilesInput{
			FileType: "image",
			Limit:    pageSize,
			Offset:   offset,
		})
		if err != nil {
			return settings.ImageStatistics{}, err
		}
		if len(files) == 0 {
			return stats, nil
		}
		for _, file := range files {
			stats.TotalFiles++
			if file.Size > 0 {
				stats.TotalSizeBytes += file.Size
			}
		}
		if len(files) < pageSize {
			return stats, nil
		}
		offset += len(files)
	}
}
