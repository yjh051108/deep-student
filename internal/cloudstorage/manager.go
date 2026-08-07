// cloudstorage 的管理层：配置持久化（加密凭据）+ SyncManager 工厂。
//
// 配置存于 SQLite app_settings（cloudstorage.config JSON）；
// 敏感字段（密码/密钥）用 crypto.Manager AES-256-GCM 加密后入库。

package cloudstorage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/helixnow/deep-student-go/pkg/crypto"
	"github.com/helixnow/deep-student-go/pkg/store"
)

const configKey = "cloudstorage.config"
const deviceIDKey = "cloudstorage.device_id"

// Manager 云存储管理入口。
type Manager struct {
	st    *store.Store
	crypto *crypto.Manager
	mu    sync.Mutex
}

// NewManager 构造管理器。
func NewManager(st *store.Store, cr *crypto.Manager) *Manager {
	return &Manager{st: st, crypto: cr}
}

// SaveConfig 保存配置（凭据加密存储）。
func (m *Manager) SaveConfig(cfg Config) error {
	if err := cfg.Valid(); err != nil {
		return err
	}
	// 加密敏感字段
	enc := encryptCreds(m.crypto, cfg)
	data, err := json.Marshal(enc)
	if err != nil {
		return err
	}
	return m.setSetting(configKey, string(data))
}

// LoadConfig 读取配置（解密凭据）。
func (m *Manager) LoadConfig() (Config, bool, error) {
	raw, ok := m.getSetting(configKey)
	if !ok {
		return Config{}, false, nil
	}
	var enc Config
	if err := json.Unmarshal([]byte(raw), &enc); err != nil {
		return Config{}, false, fmt.Errorf("cloudstorage: bad saved config: %w", err)
	}
	cfg := decryptCreds(m.crypto, enc)
	return cfg, true, nil
}

// ClearConfig 清除配置。
func (m *Manager) ClearConfig() error { return m.setSetting(configKey, "") }

// DeviceID 获取/创建设备 ID。
func (m *Manager) DeviceID() (string, error) {
	if id, ok := m.getSetting(deviceIDKey); ok && id != "" {
		return id, nil
	}
	id := fmt.Sprintf("device-%d", os.Getpid()) + "-" + randHex(6)
	if err := m.setSetting(deviceIDKey, id); err != nil {
		return "", err
	}
	return id, nil
}

// SyncManager 返回配置对应的同步管理器（每次重新构造，读取最新配置）。
func (m *Manager) SyncManager(onProgress func(Progress)) (*SyncManager, error) {
	cfg, ok, err := m.LoadConfig()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("cloudstorage: not configured")
	}
	deviceID, err := m.DeviceID()
	if err != nil {
		return nil, err
	}
	return NewSyncManager(cfg, deviceID, onProgress)
}

// CheckConnection 测试连接。
func (m *Manager) CheckConnection(ctx context.Context) error {
	sm, err := m.SyncManager(nil)
	if err != nil {
		return err
	}
	return sm.CheckConnection(ctx)
}

// UploadBackup 上传备份。
func (m *Manager) UploadBackup(ctx context.Context, localPath, note string) (*Version, error) {
	sm, err := m.SyncManager(nil)
	if err != nil {
		return nil, err
	}
	return sm.UploadBackup(ctx, localPath, note)
}

// DownloadLatest 下载最新备份到 dir，返回 (版本, 本地路径)。
func (m *Manager) DownloadLatest(ctx context.Context, dir string) (*Version, string, error) {
	sm, err := m.SyncManager(nil)
	if err != nil {
		return nil, "", err
	}
	return sm.DownloadLatest(ctx, dir)
}

// DownloadVersion 下载指定版本。
func (m *Manager) DownloadVersion(ctx context.Context, key, dir string) (*Version, string, error) {
	sm, err := m.SyncManager(nil)
	if err != nil {
		return nil, "", err
	}
	return sm.DownloadVersion(ctx, key, dir)
}

// ListVersions 列出云端版本。
func (m *Manager) ListVersions(ctx context.Context) ([]Version, error) {
	sm, err := m.SyncManager(nil)
	if err != nil {
		return nil, err
	}
	return sm.ListVersions(ctx)
}

// GetStatus 同步状态。
func (m *Manager) GetStatus(ctx context.Context) (map[string]any, error) {
	sm, err := m.SyncManager(nil)
	if err != nil {
		return map[string]any{"configured": false, "error": err.Error()}, nil
	}
	return sm.GetStatus(ctx)
}

// DeleteVersion 删除远端版本。
func (m *Manager) DeleteVersion(ctx context.Context, key string) error {
	sm, err := m.SyncManager(nil)
	if err != nil {
		return err
	}
	return sm.DeleteVersion(ctx, key)
}

// RestoreDir 返回下载目录（数据目录下）。
func RestoreDir(dataDir string) string {
	return filepath.Join(dataDir, "cloud-downloads")
}

// ===================== 加密凭据 =====================

// encryptCreds 对敏感字段加密（前缀 enc: 标记）。
func encryptCreds(cr *crypto.Manager, cfg Config) Config {
	out := cfg
	if cr == nil {
		return out
	}
	slot, _ := cr.ActiveSlot()
	enc := func(s string) string {
		if s == "" {
			return s
		}
		ct, err := cr.Encrypt(slot, []byte(s))
		if err != nil {
			return s
		}
		return "enc:" + string(ct)
	}
	out.WebDAVPass = enc(cfg.WebDAVPass)
	out.S3SecretKey = enc(cfg.S3SecretKey)
	return out
}

// decryptCreds 解密敏感字段。
func decryptCreds(cr *crypto.Manager, cfg Config) Config {
	out := cfg
	if cr == nil {
		return out
	}
	slot, _ := cr.ActiveSlot()
	dec := func(s string) string {
		if len(s) < 5 || s[:4] != "enc:" {
			return s
		}
		plain, err := cr.Decrypt(slot, []byte(s[4:]))
		if err != nil {
			return s
		}
		return string(plain)
	}
	out.WebDAVPass = dec(cfg.WebDAVPass)
	out.S3SecretKey = dec(cfg.S3SecretKey)
	return out
}

// ===================== settings kv =====================

func (m *Manager) setSetting(key, value string) error {
	if m.st == nil || m.st.DB == nil {
		return errors.New("cloudstorage: store not ready")
	}
	if value == "" {
		_, err := m.st.DB.Exec(`DELETE FROM app_settings WHERE key=?`, key)
		return err
	}
	_, err := m.st.DB.Exec(`INSERT INTO app_settings(key, value) VALUES (?,?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

func (m *Manager) getSetting(key string) (string, bool) {
	if m.st == nil || m.st.DB == nil {
		return "", false
	}
	var v string
	err := m.st.DB.QueryRow(`SELECT value FROM app_settings WHERE key=?`, key).Scan(&v)
	if err != nil || v == "" {
		return "", false
	}
	return v, true
}

// randHex 生成随机 hex（无外部依赖）。
func randHex(n int) string {
	const hex = "0123456789abcdef"
	b := make([]byte, n)
	seed := uint64(os.Getpid())*2654435761 + uint64(time.Now().UnixNano())
	for i := range b {
		seed = seed*6364136223846793005 + 1442695040888963407
		b[i] = hex[(seed>>33)&0xf]
	}
	return string(b)
}
