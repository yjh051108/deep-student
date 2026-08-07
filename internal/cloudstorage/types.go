// Package cloudstorage 提供统一云存储（WebDAV / S3）+ 加密 ZIP 备份版本同步。
//
// 对齐 Rust 原版 src-tauri/src/cloud_storage/（traits/webdav/s3/sync_manager）：
//   - Backend 接口统一 WebDAV 与 S3 对象存储的 PUT/GET/LIST/DELETE/STAT；
//   - SyncManager 负责把本地治理备份（AES 加密 ZIP）上传/下载，
//     用 CloudManifest 记录版本历史与 device_id。
package cloudstorage

import "time"

// Provider 存储供应商类型。
type Provider string

const (
	ProviderWebDAV Provider = "webdav"
	ProviderS3     Provider = "s3"
)

// Config 云存储配置。
type Config struct {
	Provider Provider `json:"provider"`
	// WebDAV
	WebDAVURL  string `json:"webdavUrl,omitempty"`
	WebDAVUser string `json:"webdavUser,omitempty"`
	WebDAVPass string `json:"webdavPass,omitempty"`
	// S3
	S3Endpoint  string `json:"s3Endpoint,omitempty"`
	S3Region    string `json:"s3Region,omitempty"`
	S3Bucket    string `json:"s3Bucket,omitempty"`
	S3AccessKey string `json:"s3AccessKey,omitempty"`
	S3SecretKey string `json:"s3SecretKey,omitempty"`
	// 公共
	RemoteDir string `json:"remoteDir,omitempty"` // 远端前缀目录（默认 deepstudent-backups）
}

// Valid 校验配置完整性。
func (c Config) Valid() error {
	switch c.Provider {
	case ProviderWebDAV:
		if c.WebDAVURL == "" {
			return errField("webdavUrl")
		}
	case ProviderS3:
		if c.S3Endpoint == "" || c.S3Bucket == "" {
			return errField("s3Endpoint/s3Bucket")
		}
		if c.S3AccessKey == "" || c.S3SecretKey == "" {
			return errField("s3AccessKey/s3SecretKey")
		}
	default:
		return errField("provider")
	}
	return nil
}

func errField(f string) error { return &ConfigError{Field: f} }

// ConfigError 配置缺失错误。
type ConfigError struct{ Field string }

func (e *ConfigError) Error() string { return "cloudstorage: missing " + e.Field }

// Object 远端对象元数据。
type Object struct {
	Key  string    `json:"key"`
	Size int64     `json:"size"`
	ETag string    `json:"etag,omitempty"`
	Mod  time.Time `json:"modified"`
}

// Manifest 云端备份清单（记录版本历史）。
type Manifest struct {
	DeviceID  string    `json:"deviceId"`
	UpdatedAt time.Time `json:"updatedAt"`
	Versions  []Version `json:"versions"`
}

// Version 单个备份版本。
type Version struct {
	Key      string    `json:"key"`
	Size     int64     `json:"size"`
	Created  time.Time `json:"created"`
	Checksum string    `json:"checksum"` // sha256 hex
	Note     string    `json:"note,omitempty"`
}

// Progress 上传/下载进度。
type Progress struct {
	Phase   string  `json:"phase"` // preparing | uploading | downloading | completed
	Current int64   `json:"current"`
	Total   int64   `json:"total"`
	Ratio   float64 `json:"ratio"`
}

// SyncResult 同步结果。
type SyncResult struct {
	Uploaded   int    `json:"uploaded"`
	Downloaded int    `json:"downloaded"`
	Versions   []Version `json:"versions"`
}
