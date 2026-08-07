// cloudstorage 的同步管理层：加密 ZIP 备份上传/下载 + CloudManifest 版本管理。
//
// 流程（对齐上游 sync_manager）：
//   1. UploadBackup：本地治理备份（AES 加密 ZIP）→ 远端
//      （object key: <remoteDir>/<deviceID>/backup-<timestamp>.zip）
//   2. 每次上传后更新 CloudManifest（版本列表，device_id）；
//   3. DownloadLatest / ListVersions / DeleteVersion 管理远端版本。

package cloudstorage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ManifestKey 清单对象 key（各设备一份）。
func ManifestKey(remoteDir, deviceID string) string {
	return keyJoin(remoteDir, deviceID, "manifest.json")
}

// backupKey 生成备份对象 key（纳秒精度避免同秒冲突）。
func backupKey(remoteDir, deviceID string, t time.Time) string {
	return keyJoin(remoteDir, deviceID, "backup-"+t.UTC().Format("20060102-150405.000000000")+".zip")
}

// SyncManager 云同步管理器。
type SyncManager struct {
	cfg      Config
	backend  Backend
	deviceID string
	mu       sync.Mutex
	onProgress func(Progress)
}

// NewSyncManager 构造同步管理器。deviceID 用于区分多设备（不同设备上传到各自子目录）。
func NewSyncManager(cfg Config, deviceID string, onProgress func(Progress)) (*SyncManager, error) {
	be, err := NewBackend(cfg)
	if err != nil {
		return nil, err
	}
	if deviceID == "" {
		deviceID = "device-default"
	}
	return &SyncManager{cfg: cfg, backend: be, deviceID: deviceID, onProgress: onProgress}, nil
}

// remoteDir 远端前缀目录。
func (m *SyncManager) remoteDir() string {
	d := strings.Trim(m.cfg.RemoteDir, "/")
	if d == "" {
		d = "deepstudent-backups"
	}
	return d
}

// CheckConnection 测试连接（List 远端目录）。
func (m *SyncManager) CheckConnection(ctx context.Context) error {
	_, err := m.backend.List(ctx, m.remoteDir())
	return err
}

// UploadBackup 上传本地加密备份文件到远端，更新清单。
// localPath 为治理备份生成的加密 ZIP 文件路径。
func (m *SyncManager) UploadBackup(ctx context.Context, localPath, note string) (*Version, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(localPath)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	key := backupKey(m.remoteDir(), m.deviceID, now)
	if m.onProgress != nil {
		m.onProgress(Progress{Phase: "uploading", Current: 0, Total: info.Size()})
	}
	if err := m.backend.Put(ctx, key, strings.NewReader(string(data)), info.Size()); err != nil {
		return nil, err
	}

	sum := sha256Hex(data)
	v := Version{Key: key, Size: info.Size(), Created: now, Checksum: sum, Note: note}
	man, _ := m.loadManifest(ctx)
	man.Versions = append([]Version{v}, man.Versions...)
	// 保留最近 20 个版本
	if len(man.Versions) > 20 {
		// 删除最旧版本对象
		for _, old := range man.Versions[20:] {
			_ = m.backend.Delete(ctx, old.Key)
		}
		man.Versions = man.Versions[:20]
	}
	man.DeviceID = m.deviceID
	man.UpdatedAt = now
	if err := m.saveManifest(ctx, man); err != nil {
		return nil, err
	}
	if m.onProgress != nil {
		m.onProgress(Progress{Phase: "completed", Current: info.Size(), Total: info.Size(), Ratio: 1})
	}
	return &v, nil
}

// DownloadLatest 下载最新版本到本地目标路径，返回 Version 与写入的本地路径。
func (m *SyncManager) DownloadLatest(ctx context.Context, destDir string) (*Version, string, error) {
	man, err := m.loadManifest(ctx)
	if err != nil {
		return nil, "", err
	}
	if len(man.Versions) == 0 {
		return nil, "", fmt.Errorf("cloudstorage: no versions available")
	}
	v := man.Versions[0]
	dest := filepath.Join(destDir, filepath.Base(v.Key))
	if err := m.download(ctx, v, dest); err != nil {
		return nil, "", err
	}
	return &v, dest, nil
}

// DownloadVersion 下载指定版本。
func (m *SyncManager) DownloadVersion(ctx context.Context, key, destDir string) (*Version, string, error) {
	man, err := m.loadManifest(ctx)
	if err != nil {
		return nil, "", err
	}
	for i := range man.Versions {
		if man.Versions[i].Key == key {
			dest := filepath.Join(destDir, filepath.Base(key))
			if err := m.download(ctx, man.Versions[i], dest); err != nil {
				return nil, "", err
			}
			return &man.Versions[i], dest, nil
		}
	}
	return nil, "", fmt.Errorf("cloudstorage: version not found: %s", key)
}

// download 下载对象到本地并校验 checksum。
func (m *SyncManager) download(ctx context.Context, v Version, dest string) error {
	rc, size, err := m.backend.Get(ctx, v.Key)
	if err != nil {
		return err
	}
	defer rc.Close()
	if m.onProgress != nil {
		m.onProgress(Progress{Phase: "downloading", Current: 0, Total: size})
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	buf := make([]byte, 256*1024)
	var total int64
	for {
		n, rerr := rc.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
			total += int64(n)
			if m.onProgress != nil {
				m.onProgress(Progress{Phase: "downloading", Current: total, Total: size, Ratio: float64(total) / float64(size)})
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	if m.onProgress != nil {
		m.onProgress(Progress{Phase: "completed", Current: total, Total: size, Ratio: 1})
	}
	return nil
}

// ListVersions 列出云端版本。
func (m *SyncManager) ListVersions(ctx context.Context) ([]Version, error) {
	man, err := m.loadManifest(ctx)
	if err != nil {
		return nil, err
	}
	return man.Versions, nil
}

// GetStatus 返回同步状态（连接可用性 + 版本数）。
func (m *SyncManager) GetStatus(ctx context.Context) (map[string]any, error) {
	connErr := m.CheckConnection(ctx)
	versions, verr := m.ListVersions(ctx)
	status := map[string]any{
		"connected":  connErr == nil,
		"deviceId":   m.deviceID,
		"remoteDir":  m.remoteDir(),
		"versionCount": 0,
	}
	if connErr != nil {
		status["error"] = connErr.Error()
	}
	if verr == nil {
		status["versionCount"] = len(versions)
	}
	return status, nil
}

// DeleteVersion 删除远端版本并更新清单。
func (m *SyncManager) DeleteVersion(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	man, err := m.loadManifest(ctx)
	if err != nil {
		return err
	}
	var kept []Version
	found := false
	for _, v := range man.Versions {
		if v.Key == key {
			found = true
			_ = m.backend.Delete(ctx, v.Key)
			continue
		}
		kept = append(kept, v)
	}
	if !found {
		return fmt.Errorf("cloudstorage: version not found: %s", key)
	}
	man.Versions = kept
	man.UpdatedAt = time.Now().UTC()
	return m.saveManifest(ctx, man)
}

// loadManifest 读取远端清单（不存在返回空清单）。
func (m *SyncManager) loadManifest(ctx context.Context) (*Manifest, error) {
	key := ManifestKey(m.remoteDir(), m.deviceID)
	rc, _, err := m.backend.Get(ctx, key)
	if err != nil {
		if IsNotFound(err) {
			return &Manifest{DeviceID: m.deviceID, UpdatedAt: time.Now().UTC()}, nil
		}
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	var man Manifest
	if err := json.Unmarshal(data, &man); err != nil {
		return nil, fmt.Errorf("cloudstorage: bad manifest: %w", err)
	}
	return &man, nil
}

// saveManifest 写远端清单。
func (m *SyncManager) saveManifest(ctx context.Context, man *Manifest) error {
	data, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		return err
	}
	key := ManifestKey(m.remoteDir(), m.deviceID)
	return m.backend.Put(ctx, key, strings.NewReader(string(data)), int64(len(data)))
}

// sha256Hex 计算数据 SHA-256 hex。
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
