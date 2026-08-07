package cloudstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// mockWebDAV 内存 WebDAV mock server（支持 PUT/GET/DELETE/PROPFIND/MKCOL）。
type mockWebDAV struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func newMockWebDAV() *mockWebDAV {
	return &mockWebDAV{objects: map[string][]byte{}}
}

func (m *mockWebDAV) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := strings.TrimPrefix(r.URL.Path, "/")
	switch r.Method {
	case "PUT":
		body, _ := io.ReadAll(r.Body)
		m.objects[key] = body
		w.WriteHeader(http.StatusCreated)
	case "GET":
		if data, ok := m.objects[key]; ok {
			w.Write(data)
		} else {
			http.NotFound(w, r)
		}
	case "DELETE":
		delete(m.objects, key)
		w.WriteHeader(http.StatusNoContent)
	case "MKCOL":
		w.WriteHeader(http.StatusCreated)
	case "PROPFIND":
		// 简单列出所有匹配前缀的对象
		prefix := strings.TrimPrefix(key, "/")
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><D:multistatus xmlns:D="DAV:">`))
		for k := range m.objects {
			if strings.HasPrefix(k, prefix) {
				rel := "/" + k
				fmt.Fprintf(w, `<D:response><D:href>%s</D:href><D:propstat><D:prop><D:getcontentlength>%d</D:getcontentlength></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response>`, rel, len(m.objects[k]))
			}
		}
		w.Write([]byte(`</D:multistatus>`))
	default:
		http.Error(w, "unsupported", http.StatusMethodNotAllowed)
	}
}

func (m *mockWebDAV) count(prefix string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for k := range m.objects {
		if strings.HasPrefix(k, prefix) {
			n++
		}
	}
	return n
}

// newTestManager 构造测试 SyncManager（WebDAV mock）。
func newTestManager(t *testing.T) (*SyncManager, *mockWebDAV) {
	t.Helper()
	store := newMockWebDAV()
	srv := httptest.NewServer(store)
	t.Cleanup(srv.Close)
	cfg := Config{
		Provider:   ProviderWebDAV,
		WebDAVURL:  srv.URL,
		WebDAVUser: "u",
		WebDAVPass: "p",
		RemoteDir:  "backups",
	}
	m, err := NewSyncManager(cfg, "device-test", nil)
	if err != nil {
		t.Fatal(err)
	}
	return m, store
}

func TestUploadDownloadLifecycle(t *testing.T) {
	m, store := newTestManager(t)
	ctx := context.Background()

	// 准备本地备份文件
	dir := t.TempDir()
	backup := filepath.Join(dir, "backup.zip")
	os.WriteFile(backup, []byte("encrypted-backup-content"), 0o600)

	v, err := m.UploadBackup(ctx, backup, "测试备份")
	if err != nil {
		t.Fatal(err)
	}
	if v.Checksum == "" || v.Size == 0 {
		t.Fatalf("version=%+v", v)
	}
	if store.count("backups/device-test/") == 0 {
		t.Fatal("no object uploaded")
	}

	// 上传第二个
	os.WriteFile(backup, []byte("second-backup"), 0o600)
	if _, err := m.UploadBackup(ctx, backup, "第二次"); err != nil {
		t.Fatal(err)
	}

	// 列版本
	versions, err := m.ListVersions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 {
		t.Fatalf("versions=%d", len(versions))
	}
	if versions[0].Note != "第二次" {
		t.Fatalf("newest=%+v", versions[0])
	}

	// 下载最新
	latest, dest, err := m.DownloadLatest(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(dest)
	if string(data) != "second-backup" {
		t.Fatalf("downloaded=%q", string(data))
	}
	if latest.Key != versions[0].Key {
		t.Fatalf("latest mismatch")
	}
}

func TestDownloadSpecificVersion(t *testing.T) {
	m, _ := newTestManager(t)
	ctx := context.Background()
	dir := t.TempDir()
	backup := filepath.Join(dir, "b.zip")
	os.WriteFile(backup, []byte("v1-content"), 0o600)
	if _, err := m.UploadBackup(ctx, backup, ""); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(backup, []byte("v2-content"), 0o600)
	v2, err := m.UploadBackup(ctx, backup, "")
	if err != nil {
		t.Fatal(err)
	}
	// 下载 v2 对应版本（内容应为 v2-content）
	_, dest, err := m.DownloadVersion(ctx, v2.Key, dir)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(dest)
	if string(data) != "v2-content" {
		t.Fatalf("downloaded=%q", string(data))
	}
	// 下载旧版（第一个版本）
	versions, _ := m.ListVersions(ctx)
	v1 := versions[1]
	_, dest1, err := m.DownloadVersion(ctx, v1.Key, dir)
	if err != nil {
		t.Fatal(err)
	}
	data1, _ := os.ReadFile(dest1)
	if string(data1) != "v1-content" {
		t.Fatalf("downloaded old=%q", string(data1))
	}
}

func TestDeleteVersion(t *testing.T) {
	m, store := newTestManager(t)
	ctx := context.Background()
	dir := t.TempDir()
	backup := filepath.Join(dir, "b.zip")
	os.WriteFile(backup, []byte("x"), 0o600)
	v1, _ := m.UploadBackup(ctx, backup, "")
	os.WriteFile(backup, []byte("y"), 0o600)
	_, _ = m.UploadBackup(ctx, backup, "")

	if err := m.DeleteVersion(ctx, v1.Key); err != nil {
		t.Fatal(err)
	}
	versions, _ := m.ListVersions(ctx)
	if len(versions) != 1 {
		t.Fatalf("versions=%d", len(versions))
	}
	_ = store
}

func TestCheckConnectionAndStatus(t *testing.T) {
	m, _ := newTestManager(t)
	ctx := context.Background()
	if err := m.CheckConnection(ctx); err != nil {
		t.Fatal(err)
	}
	status, err := m.GetStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status["connected"] != true {
		t.Fatalf("status=%+v", status)
	}
}

func TestConfigValidation(t *testing.T) {
	if err := (Config{Provider: ProviderWebDAV}).Valid(); err == nil {
		t.Fatal("webdav without url should error")
	}
	if err := (Config{Provider: ProviderS3, S3Endpoint: "http://x", S3Bucket: "b"}).Valid(); err == nil {
		t.Fatal("s3 without keys should error")
	}
	if err := (Config{Provider: ProviderWebDAV, WebDAVURL: "http://x"}).Valid(); err != nil {
		t.Fatal("valid config rejected")
	}
}

func TestManifestJSON(t *testing.T) {
	man := Manifest{DeviceID: "d", Versions: []Version{{Key: "k", Size: 1, Checksum: "c"}}}
	data, _ := json.Marshal(man)
	var back Manifest
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if back.DeviceID != "d" || len(back.Versions) != 1 {
		t.Fatalf("back=%+v", back)
	}
}
