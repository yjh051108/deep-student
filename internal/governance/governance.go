// Package governance 本地数据治理：备份、加密、A/B、审计、导入/导出。
package governance

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/helixnow/deep-student-go/pkg/config"
	"github.com/helixnow/deep-student-go/pkg/crypto"
	"github.com/helixnow/deep-student-go/pkg/eventbus"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Service 治理服务。
type Service struct {
	vfs    *vfs.FS
	store  *store.Store
	crypto *crypto.Manager
	cfg    *config.Config
	bus    *eventbus.Bus
	mu     sync.Mutex
}

// New 创建 Service。
func New(fs *vfs.FS, st *store.Store, cr *crypto.Manager, cfg *config.Config, bus *eventbus.Bus) *Service {
	return &Service{vfs: fs, store: st, crypto: cr, cfg: cfg, bus: bus}
}

// Backup 全量加密备份到目标文件（AES-256-GCM，A 槽位）。
func (s *Service) Backup(target string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	tmp := target + ".tmp"
	zf, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	zw := zip.NewWriter(zf)
	// BUG-004: VFS 写入期间一致性保证 —— 在持锁期间快速 snapshot blob 引用，
	// 然后在锁外安全读取 blob。
	s.vfs.LockForRead()
	vfsSnapshot := s.vfs.Snapshot()
	s.vfs.UnlockRead()
	for _, item := range vfsSnapshot {
		data, _, err := s.vfs.Get(item.URI)
		if err != nil {
			continue
		}
		w, err := zw.Create(item.Type.String() + "/" + item.ID + ".json")
		if err != nil {
			continue
		}
		w.Write(data)
	}
	// 写入 sqlite 一致性快照（VACUUM INTO 内部已加锁）
	dumpPath := tmp + ".db"
	if err := s.store.Backup(dumpPath); err == nil {
		if data, err := os.ReadFile(dumpPath); err == nil {
			w, _ := zw.Create("deepstudent.db")
			w.Write(data)
		}
		os.Remove(dumpPath)
	}
	zw.Close()
	zf.Close()
	// 加密
	plain, err := os.ReadFile(tmp)
	if err != nil {
		return "", err
	}
	slot, _ := s.crypto.ActiveSlot()
	ct, err := s.crypto.Encrypt(slot, plain)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(target, ct, 0o600); err != nil {
		return "", err
	}
	os.Remove(tmp)
	sum := sha256.Sum256(ct)
	ref := hex.EncodeToString(sum[:])
	s.store.LogAudit("governance", "backup", target, time.Now().Unix())
	s.bus.PublishAsync(nil, "governance.backup", target)
	return ref, nil
}

// Restore 从加密备份恢复。
func (s *Service) Restore(source string) error {
	ct, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	slot, _ := s.crypto.ActiveSlot()
	plain, err := s.crypto.Decrypt(slot, ct)
	if err != nil {
		return err
	}
	zr, err := zip.NewReader(bytes.NewReader(plain), int64(len(plain)))
	if err != nil {
		return err
	}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(rc)
		rc.Close()
		if f.Name == "deepstudent.db" {
			// 替换数据库（简化）
			os.WriteFile(filepath.Join(s.cfg.DataDir, "deepstudent.db"), body, 0o600)
			continue
		}
		uri := "vfs://" + f.Name
		s.vfs.Put(uri, body, map[string]string{"restored": "1"})
	}
	s.store.LogAudit("governance", "restore", source, time.Now().Unix())
	return nil
}

// SwitchSlot 切换 A/B。
func (s *Service) SwitchSlot(to string) error {
	if err := s.crypto.SwitchSlot(to); err != nil {
		return err
	}
	s.store.LogAudit("governance", "switch-slot", to, time.Now().Unix())
	return nil
}

// Export 导出资源为 zip（不加密）。
func (s *Service) Export(target string, types []vfs.ResourceType) error {
	zf, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zf.Close()
	zw := zip.NewWriter(zf)
	defer zw.Close()
	for _, t := range types {
		for _, e := range s.vfs.List(t) {
			data, _, err := s.vfs.Get(e.URI)
			if err != nil {
				continue
			}
			w, _ := zw.Create(t.String() + "/" + e.ID + ".json")
			w.Write(data)
		}
	}
	s.store.LogAudit("governance", "export", target, time.Now().Unix())
	return nil
}

// Import 从 zip 导入。
func (s *Service) Import(source string) error {
	r, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		rc, _ := f.Open()
		body, _ := io.ReadAll(rc)
		rc.Close()
		uri := "vfs://" + f.Name
		s.vfs.Put(uri, body, map[string]string{"imported": "1"})
	}
	s.store.LogAudit("governance", "import", source, time.Now().Unix())
	return nil
}

// AuditLogs 读取最近审计日志。
func (s *Service) AuditLogs(limit int) ([]store.AuditEntry, error) { return s.store.AuditLogs(limit) }

func (s *Service) dumpSQLite() ([]byte, error) {
	dbPath := filepath.Join(s.cfg.DataDir, "deepstudent.db")
	return os.ReadFile(dbPath)
}

// Status 治理状态。
func (s *Service) Status() map[string]any {
	slot, _ := s.crypto.ActiveSlot()
	rows, _ := s.store.AuditLogs(5)
	return map[string]any{
		"slot":        slot,
		"data_dir":    s.cfg.DataDir,
		"backup_dir":  s.cfg.BackupDir,
		"audit_count": len(rows),
		"now":         time.Now().Format(time.RFC3339),
	}
}

// MarshalJSON 仅用于 Status 占位。
func (s *Service) MarshalJSON() ([]byte, error) { return json.Marshal(s.Status()) }

// EnsureDir 保证目录存在。
func EnsureDir(p string) error { return os.MkdirAll(p, 0o755) }

// CheckIntegrity 校验关键文件存在性。
func (s *Service) CheckIntegrity() []string {
	var issues []string
	for _, p := range []string{
		filepath.Join(s.cfg.DataDir, "deepstudent.db"),
		filepath.Join(s.cfg.DataDir, "blob"),
		filepath.Join(s.cfg.DataDir, "keys"),
	} {
		if _, err := os.Stat(p); err != nil {
			issues = append(issues, fmt.Sprintf("missing: %s", p))
		}
	}
	return issues
}
