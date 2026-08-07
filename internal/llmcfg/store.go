// llmcfg 持久化层 —— JSON 文件存储。
//
// 路径：<DataDir>/llmcfg.json
// 数据结构：{vendors: [], profiles: [], assignments: {}, version: 1}
//
// 特性：
//   - 文件不存在或解析失败时返回空状态 + 自动加载内置
//   - Save 采用原子写入（先写 .tmp，再 rename）
//   - 线程安全（sync.RWMutex）
//   - 首次加载时自动 seed 内置厂商和模型
package llmcfg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// storeVersion 当前存储格式版本。
const storeVersion = 1

// storeData 持久化数据结构。
type storeData struct {
	Vendors     []VendorConfig   `json:"vendors"`
	Profiles    []ModelProfile   `json:"profiles"`
	Assignments ModelAssignments `json:"assignments"`
	Version     int              `json:"version"`
}

// store JSON 文件持久化（线程安全）。
type store struct {
	mu       sync.RWMutex
	path     string
	data     storeData
	loaded   bool
	seeded   bool
}

// newStore 创建存储实例，path 为 llmcfg.json 完整路径。
func newStore(path string) *store {
	return &store{
		path: path,
		data: storeData{
			Vendors:     []VendorConfig{},
			Profiles:    []ModelProfile{},
			Assignments: ModelAssignments{},
			Version:     storeVersion,
		},
	}
}

// Load 从磁盘加载配置；文件不存在或解析失败时返回空状态并自动 seed 内置。
//
// 首次加载且无内置项时，会写入 builtin 厂商和模型并持久化。
func (s *store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// 文件不存在：返回空状态，由调用方决定是否 seed
			s.data = storeData{
				Vendors:     []VendorConfig{},
				Profiles:    []ModelProfile{},
				Assignments: ModelAssignments{},
				Version:     storeVersion,
			}
			s.loaded = true
			return nil
		}
		return fmt.Errorf("llmcfg: read %s: %w", s.path, err)
	}

	var d storeData
	if err := json.Unmarshal(raw, &d); err != nil {
		// 解析失败：重置为空状态，不破坏应用启动
		s.data = storeData{
			Vendors:     []VendorConfig{},
			Profiles:    []ModelProfile{},
			Assignments: ModelAssignments{},
			Version:     storeVersion,
		}
		s.loaded = true
		return nil
	}
	if d.Vendors == nil {
		d.Vendors = []VendorConfig{}
	}
	if d.Profiles == nil {
		d.Profiles = []ModelProfile{}
	}
	if d.Version == 0 {
		d.Version = storeVersion
	}
	s.data = d
	s.loaded = true
	return nil
}

// Save 原子写入磁盘（先写 .tmp，再 rename）。
func (s *store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("llmcfg: mkdir %s: %w", dir, err)
	}

	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("llmcfg: marshal: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("llmcfg: write tmp %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("llmcfg: rename %s -> %s: %w", tmp, s.path, err)
	}
	return nil
}

// snapshot 返回当前数据的深拷贝快照（调用方持锁）。
func (s *store) snapshot() storeData {
	out := storeData{
		Vendors:     make([]VendorConfig, len(s.data.Vendors)),
		Profiles:    make([]ModelProfile, len(s.data.Profiles)),
		Assignments: s.data.Assignments,
		Version:     s.data.Version,
	}
	copy(out.Vendors, s.data.Vendors)
	copy(out.Profiles, s.data.Profiles)
	return out
}

// getVendors 返回所有供应商的拷贝。
func (s *store) getVendors() []VendorConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]VendorConfig, len(s.data.Vendors))
	copy(out, s.data.Vendors)
	return out
}

// getProfiles 返回所有模型的拷贝。
func (s *store) getProfiles() []ModelProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ModelProfile, len(s.data.Profiles))
	copy(out, s.data.Profiles)
	return out
}

// getAssignments 返回分配配置的拷贝。
func (s *store) getAssignments() ModelAssignments {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Assignments
}

// upsertVendor 插入或更新供应商（按 ID 匹配）。
func (s *store) upsertVendor(v VendorConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Vendors {
		if existing.ID == v.ID {
			s.data.Vendors[i] = v
			return
		}
	}
	s.data.Vendors = append(s.data.Vendors, v)
}

// deleteVendor 删除供应商（按 ID）。
func (s *store) deleteVendor(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Vendors {
		if existing.ID == id {
			s.data.Vendors = append(s.data.Vendors[:i], s.data.Vendors[i+1:]...)
			return true
		}
	}
	return false
}

// findVendor 按 ID 查找供应商。
func (s *store) findVendor(id string) (VendorConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.data.Vendors {
		if v.ID == id {
			return v, true
		}
	}
	return VendorConfig{}, false
}

// upsertProfile 插入或更新模型（按 ID 匹配）。
func (s *store) upsertProfile(p ModelProfile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Profiles {
		if existing.ID == p.ID {
			s.data.Profiles[i] = p
			return
		}
	}
	s.data.Profiles = append(s.data.Profiles, p)
}

// deleteProfile 删除模型（按 ID）。
func (s *store) deleteProfile(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Profiles {
		if existing.ID == id {
			s.data.Profiles = append(s.data.Profiles[:i], s.data.Profiles[i+1:]...)
			return true
		}
	}
	return false
}

// findProfile 按 ID 查找模型。
func (s *store) findProfile(id string) (ModelProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.data.Profiles {
		if p.ID == id {
			return p, true
		}
	}
	return ModelProfile{}, false
}

// setAssignments 替换分配配置。
func (s *store) setAssignments(a ModelAssignments) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Assignments = a
}

// seedBuiltins 若 vendors 或 profiles 为空，自动注入内置项。
//
// 对齐 Rust：仅注入 existingIDs 中不存在的项，避免重复 seed 覆盖用户已填的 Key。
func (s *store) seedBuiltins() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	existingVendorIDs := make([]string, 0, len(s.data.Vendors))
	for _, v := range s.data.Vendors {
		existingVendorIDs = append(existingVendorIDs, v.ID)
	}
	existingProfileIDs := make([]string, 0, len(s.data.Profiles))
	for _, p := range s.data.Profiles {
		existingProfileIDs = append(existingProfileIDs, p.ID)
	}

	newVendors := loadBuiltinVendors(existingVendorIDs)
	newProfiles := loadBuiltinModels(existingProfileIDs)

	if len(newVendors) == 0 && len(newProfiles) == 0 {
		s.seeded = true
		return false
	}

	s.data.Vendors = append(s.data.Vendors, newVendors...)
	s.data.Profiles = append(s.data.Profiles, newProfiles...)
	s.seeded = true
	return true
}
