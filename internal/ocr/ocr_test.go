package ocr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestSvc VL mock server。
func newTestSvc(t *testing.T) (*Service, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		// 校验请求里有图片
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		hasImage := false
		for _, m := range req.Messages {
			if strings.Contains(string(m.Content), "image_url") {
				hasImage = true
			}
		}
		if !hasImage {
			http.Error(w, "no image", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "第一行文字\n第二行文字"}},
			},
		})
	}))
	svc := New("test-key")
	svc.SetConfig("test-key", srv.URL, "deepseek-vl2")
	return svc, srv
}

func TestRecognizeVL(t *testing.T) {
	svc, srv := newTestSvc(t)
	defer srv.Close()
	res, err := svc.Recognize(context.Background(), []byte("fake-image"), "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "第一行文字\n第二行文字" {
		t.Fatalf("text=%q", res.Text)
	}
	if res.Engine != EngineDeepSeek {
		t.Fatalf("engine=%s", res.Engine)
	}
}

func TestRecognizeErrors(t *testing.T) {
	svc, srv := newTestSvc(t)
	defer srv.Close()
	// 无 key
	svc.APIKey = ""
	if _, err := svc.Recognize(context.Background(), []byte("x"), "image/png"); err == nil {
		t.Fatal("no key should error")
	}
	// paddle 未安装
	svc.APIKey = "k"
	if err := svc.SetEngineType(EnginePaddle); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Recognize(context.Background(), []byte("x"), "image/png"); err == nil {
		t.Fatal("paddle should error")
	}
}

func TestSetEngineType(t *testing.T) {
	svc := New("")
	if err := svc.SetEngineType("weird"); err == nil {
		t.Fatal("bad engine should error")
	}
	if err := svc.SetEngineType(EngineSystem); err != nil {
		t.Fatal(err)
	}
	if svc.EngineType() != EngineSystem {
		t.Fatal("engine not switched")
	}
}

func TestListEngines(t *testing.T) {
	svc := New("key")
	engines := svc.ListEngines()
	if len(engines) != 3 {
		t.Fatalf("engines=%d", len(engines))
	}
	if !engines[0].Available {
		t.Fatal("deepseek should be available with key")
	}
}

func TestPDFSession(t *testing.T) {
	svc := New("k")
	id, err := svc.StartPDFSession("book.pdf", 100)
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("empty session")
	}
	svc.CancelPDFSession(id)
}

func TestDataURLEncoding(t *testing.T) {
	b64 := base64.StdEncoding.EncodeToString([]byte("img"))
	if b64 != "aW1n" {
		t.Fatalf("b64=%s", b64)
	}
}

func TestPDFTextExtract(t *testing.T) {
	svc := New("")
	// 非 PDF 数据应报错（走 reader.ParsePDF）
	if _, err := svc.ExtractTextFromPDFBytes([]byte("not a pdf")); err == nil {
		t.Fatal("bad pdf should error")
	}
}
