// Package config 提供统一的配置加载（环境变量 + 用户配置文件）。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Config 全局配置，运行时只读。
type Config struct {
	AppName     string
	Version     string
	DataDir     string
	CacheDir    string
	LogDir      string
	BackupDir   string
	LogLevel    string
	PrivacyMode bool
	UseSlot     string // "A" or "B"
	HTTPListen  string
	MCPServers  map[string]MCPServerConfig
	LLM         LLMConfig
}

// LLMConfig 默认 LLM 设置。
type LLMConfig struct {
	DefaultProvider string
	DefaultModel    string
	OpenAIKey       string
	AnthropicKey    string
	GoogleKey       string
	DeepSeekKey     string
	SiliconKey      string
	ZhipuKey        string
	TongyiKey       string
	MoonshotKey     string
}

// MCPServerConfig 单个 MCP 服务配置。
type MCPServerConfig struct {
	Command string            `json:"command" yaml:"command"`
	Args    []string          `json:"args" yaml:"args"`
	Env     map[string]string `json:"env" yaml:"env"`
	URL     string            `json:"url" yaml:"url"`
	Enabled bool              `json:"enabled" yaml:"enabled"`
}

var (
	once   sync.Once
	cfg    *Config
	loadEr error
)

// Load 加载配置（进程内单次）。
func Load() (*Config, error) {
	once.Do(func() {
		dataDir := envOr("DEEPSTUDENT_DATA", defaultDataDir())
		cfg = &Config{
			AppName:     "DeepStudent",
			Version:     "0.1.0-go",
			DataDir:     dataDir,
			CacheDir:    envOr("DEEPSTUDENT_CACHE", filepath.Join(dataDir, "cache")),
			LogDir:      envOr("DEEPSTUDENT_LOG", filepath.Join(dataDir, "logs")),
			BackupDir:   envOr("DEEPSTUDENT_BACKUP", filepath.Join(dataDir, "backups")),
			LogLevel:    envOr("DEEPSTUDENT_LOG_LEVEL", "info"),
			PrivacyMode: envBool("DEEPSTUDENT_PRIVACY", false),
			UseSlot:     envOr("DEEPSTUDENT_SLOT", "A"),
			HTTPListen:  envOr("DEEPSTUDENT_HTTP", "127.0.0.1:0"),
			MCPServers:  map[string]MCPServerConfig{},
			LLM: LLMConfig{
				DefaultProvider: envOr("DEEPSTUDENT_PROVIDER", "openai"),
				DefaultModel:    envOr("DEEPSTUDENT_MODEL", "gpt-4o-mini"),
				OpenAIKey:       os.Getenv("OPENAI_API_KEY"),
				AnthropicKey:    os.Getenv("ANTHROPIC_API_KEY"),
				GoogleKey:       os.Getenv("GOOGLE_API_KEY"),
				DeepSeekKey:     os.Getenv("DEEPSEEK_API_KEY"),
				SiliconKey:      os.Getenv("SILICONFLOW_API_KEY"),
				ZhipuKey:        os.Getenv("ZHIPU_API_KEY"),
				TongyiKey:       os.Getenv("TONGYI_API_KEY"),
				MoonshotKey:     os.Getenv("MOONSHOT_API_KEY"),
			},
		}
		// 确保关键目录存在
		for _, d := range []string{cfg.DataDir, cfg.CacheDir, cfg.LogDir, cfg.BackupDir, filepath.Join(cfg.DataDir, "blob"), filepath.Join(cfg.DataDir, "vector")} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				loadEr = fmt.Errorf("create dir %s: %w", d, err)
				return
			}
		}
	})
	return cfg, loadEr
}

// Reset 重置单例（仅供测试）。
func Reset() {
	once = sync.Once{}
	cfg = nil
	loadEr = nil
}

func envOr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".deepstudent"
	}
	return filepath.Join(home, ".deepstudent-go")
}
