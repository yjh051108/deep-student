// Package crypto 提供 AES-256-GCM 加密、双槽 A/B 切换、密钥派生。
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/pbkdf2"
)

const (
	saltSize   = 16
	keySize    = 32 // AES-256
	nonceSize  = 12
	slotA      = "A"
	slotB      = "B"
	magic      = "DSG1"
	pbkdf2Iter = 200_000
)

// EncryptedBlob 加密文件格式：[magic(4)][salt(16)][nonce(12)][ciphertext...]
type EncryptedBlob struct {
	Salt       []byte
	Nonce      []byte
	CipherText []byte
}

// Slot 表示一个加密槽位。
type Slot struct {
	Name        string
	KeyFile     string
	Encrypted   bool
	LastUpdated int64
}

// Manager 管理 A/B 加密槽。
type Manager struct {
	mu      sync.RWMutex
	keysDir string
	slots   map[string][]byte
}

// NewManager 创建加密管理器，从密钥文件加载两个槽位的密钥。
func NewManager(keysDir string) (*Manager, error) {
	if err := os.MkdirAll(keysDir, 0o700); err != nil {
		return nil, err
	}
	m := &Manager{keysDir: keysDir, slots: map[string][]byte{}}
	for _, name := range []string{slotA, slotB} {
		key, err := loadOrCreateKey(filepath.Join(keysDir, "slot-"+name+".key"))
		if err != nil {
			return nil, err
		}
		m.slots[name] = key
	}
	return m, nil
}

func loadOrCreateKey(path string) ([]byte, error) {
	if data, err := os.ReadFile(path); err == nil && len(data) == keySize {
		return data, nil
	}
	key := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, key, 0o600); err != nil {
		return nil, err
	}
	return key, nil
}

// DeriveKey 从密码派生密钥（PBKDF2-SHA256）。
func DeriveKey(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, pbkdf2Iter, keySize, sha256.New)
}

// DeriveKeyArgon2id 使用 Argon2id 派生（更安全，慢）。
func DeriveKeyArgon2id(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, keySize)
}

// Encrypt 使用指定槽位加密数据。
func (m *Manager) Encrypt(slotName string, plaintext []byte) ([]byte, error) {
	m.mu.RLock()
	key, ok := m.slots[slotName]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("slot %s not found", slotName)
	}
	return encryptWithKey(key, plaintext)
}

// Decrypt 使用指定槽位解密。
func (m *Manager) Decrypt(slotName string, blob []byte) ([]byte, error) {
	m.mu.RLock()
	key, ok := m.slots[slotName]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("slot %s not found", slotName)
	}
	return decryptWithKey(key, blob)
}

// SwitchSlot 切换主槽（仅写元信息，不动数据）。
func (m *Manager) SwitchSlot(to string) error {
	if to != slotA && to != slotB {
		return errors.New("invalid slot")
	}
	meta := filepath.Join(m.keysDir, "active.txt")
	return os.WriteFile(meta, []byte(to), 0o600)
}

// ActiveSlot 读取当前激活的槽位。
func (m *Manager) ActiveSlot() (string, error) {
	meta := filepath.Join(m.keysDir, "active.txt")
	data, err := os.ReadFile(meta)
	if err != nil {
		return slotA, nil
	}
	s := string(data)
	if s != slotA && s != slotB {
		return slotA, nil
	}
	return s, nil
}

func encryptWithKey(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ct := gcm.Seal(nil, nonce, plaintext, nil)
	out := make([]byte, 0, len(magic)+saltSize+nonceSize+len(ct))
	out = append(out, []byte(magic)...)
	salt := make([]byte, saltSize)
	io.ReadFull(rand.Reader, salt)
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

func decryptWithKey(key, blob []byte) ([]byte, error) {
	if len(blob) < len(magic)+saltSize+nonceSize {
		return nil, errors.New("blob too short")
	}
	if string(blob[:len(magic)]) != magic {
		return nil, errors.New("invalid magic")
	}
	nonceStart := len(magic) + saltSize
	nonceEnd := nonceStart + nonceSize
	nonce := blob[nonceStart:nonceEnd]
	ct := blob[nonceEnd:]
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, err
	}
	return pt, nil
}

// EncryptToFile 加密并写入文件。
func (m *Manager) EncryptToFile(slotName, path string, plaintext []byte) error {
	blob, err := m.Encrypt(slotName, plaintext)
	if err != nil {
		return err
	}
	return os.WriteFile(path, blob, 0o600)
}

// DecryptFromFile 从文件读取并解密。
func (m *Manager) DecryptFromFile(slotName, path string) ([]byte, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return m.Decrypt(slotName, blob)
}

// EncToBase64 / Base64ToEnc 仅供调试。
func EncToBase64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
