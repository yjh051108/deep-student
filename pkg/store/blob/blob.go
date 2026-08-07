// Package blob 提供按 SHA256 内容寻址的本地 Blob 存储。
package blob

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Store 本地 Blob 存储。
type Store struct {
	mu   sync.RWMutex
	root string
}

// New 创建 Blob 存储。
func New(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &Store{root: root}, nil
}

// Put 写入数据并返回 sha256 引用与字节数。
func (s *Store) Put(data []byte) (string, int64, error) {
	if s == nil {
		return "", 0, errors.New("blob: nil store")
	}
	sum := sha256.Sum256(data)
	ref := hex.EncodeToString(sum[:])
	path := s.path(ref)
	if _, err := os.Stat(path); err == nil {
		return ref, int64(len(data)), nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", 0, err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", 0, err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", 0, err
	}
	return ref, int64(len(data)), nil
}

// Get 读取数据。
func (s *Store) Get(ref string) ([]byte, error) {
	if s == nil {
		return nil, errors.New("blob: nil store")
	}
	return os.ReadFile(s.path(ref))
}

// Open 返回 Reader。
func (s *Store) Open(ref string) (io.ReadCloser, error) {
	f, err := os.Open(s.path(ref))
	if err != nil {
		return nil, err
	}
	return f, nil
}

// Has 是否存在。
func (s *Store) Has(ref string) bool {
	_, err := os.Stat(s.path(ref))
	return err == nil
}

// Delete 删除（引用计数由 vfs 维护）。
func (s *Store) Delete(ref string) error {
	if !s.Has(ref) {
		return nil
	}
	return os.Remove(s.path(ref))
}

// Path 返回磁盘路径（内部）。
func (s *Store) path(ref string) string {
	if len(ref) < 4 {
		return filepath.Join(s.root, ref)
	}
	return filepath.Join(s.root, ref[:2], ref[2:4], ref)
}

// Size 返回文件大小。
func (s *Store) Size(ref string) (int64, error) {
	fi, err := os.Stat(s.path(ref))
	if err != nil {
		return 0, fmt.Errorf("blob: %w", err)
	}
	return fi.Size(), nil
}
