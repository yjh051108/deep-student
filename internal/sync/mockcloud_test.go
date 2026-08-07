package sync

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// mockCloud 内存对象存储（模拟 WebDAV/S3 云端）。
type mockCloud struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func newMockCloud() *mockCloud { return &mockCloud{objects: map[string][]byte{}} }

func (m *mockCloud) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
			_, _ = w.Write(data)
		} else {
			http.NotFound(w, r)
		}
	case "DELETE":
		delete(m.objects, key)
		w.WriteHeader(http.StatusNoContent)
	case "PROPFIND":
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?><D:multistatus xmlns:D="DAV:">`))
		for k := range m.objects {
			if strings.HasPrefix(k, strings.TrimPrefix(key, "/")) {
				_, _ = w.Write([]byte(`<D:response><D:href>/` + k + `</D:href><D:propstat><D:prop><D:getcontentlength>` + itoa(len(m.objects[k])) + `</D:getcontentlength></D:prop><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response>`))
			}
		}
		_, _ = w.Write([]byte(`</D:multistatus>`))
	default:
		http.Error(w, "unsupported", http.StatusMethodNotAllowed)
	}
}

func (m *mockCloud) count(prefix string) int {
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

// newCloudServer 启动 mock 云 HTTP 服务。
func newCloudServer(t *testing.T, m *mockCloud) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	return srv
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
