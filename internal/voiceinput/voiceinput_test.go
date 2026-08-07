package voiceinput

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestSvc 用 httptest mock server 构造服务。
func newTestSvc(t *testing.T) (*Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		_ = r.ParseMultipartForm(10 << 20)
		// 校验 multipart 里有 file 与 model
		f, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "no file field", http.StatusBadRequest)
			return
		}
		_ = f.Close()
		_ = r.FormValue("model")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "你好，世界"})
	}))
	svc := NewWithProvider(Provider{
		Name:    "test",
		BaseURL: srv.URL + "/v1",
		APIKey:  "test-key",
		Model:   "mock-model",
	})
	return svc, srv
}

func TestTranscribe(t *testing.T) {
	svc, srv := newTestSvc(t)
	defer srv.Close()

	res, err := svc.Transcribe(context.Background(), []byte("fake-audio-bytes"), "audio/wav")
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "你好，世界" {
		t.Fatalf("text=%q", res.Text)
	}
	if res.Provider != "test" {
		t.Fatalf("provider=%s", res.Provider)
	}
}

func TestTranscribeErrors(t *testing.T) {
	svc, srv := newTestSvc(t)
	defer srv.Close()

	// 空音频
	if _, err := svc.Transcribe(context.Background(), nil, "audio/wav"); err == nil {
		t.Fatal("empty audio should error")
	}
	// 无 key
	svc.SetProvider(Provider{Name: "x", BaseURL: srv.URL, Model: "m"})
	if _, err := svc.Transcribe(context.Background(), []byte("x"), "audio/wav"); err == nil {
		t.Fatal("no key should error")
	}
}

func TestServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	svc := NewWithProvider(Provider{Name: "e", BaseURL: srv.URL + "/v1", APIKey: "k", Model: "m"})
	_, err := svc.Transcribe(context.Background(), []byte("x"), "audio/wav")
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected 500 error, got %v", err)
	}
}

func TestMimeExt(t *testing.T) {
	cases := map[string]string{
		"audio/wav":    ".wav",
		"audio/mpeg":   ".mp3",
		"audio/webm":   ".webm",
		"audio/x-m4a":  ".m4a",
		"audio/unknown": ".wav",
	}
	for in, want := range cases {
		if got := mimeExt(in); got != want {
			t.Fatalf("mimeExt(%q)=%q want %q", in, got, want)
		}
	}
}

