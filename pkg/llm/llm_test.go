package llm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// ---------- shared mock-server helpers ----------
// 这些 handler 被 openai_compat_test.go / anthropic_google_test.go 复用。

// chatSSEHandler 把一个 OpenAI 兼容的 /chat/completions 端点模拟成 SSE 响应。
// 返回 (url, requestCount, lastBody)。测试通过 lastBody 断言请求是否带 Authorization 等。
func chatSSEHandler(t *testing.T, chunks []string) (*httptest.Server, *int64, *string) {
	t.Helper()
	var count int64
	var lastBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, 1)
		b, _ := io.ReadAll(r.Body)
		lastBody = string(b)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)
		for _, c := range chunks {
			payload := map[string]any{
				"choices": []map[string]any{
					{"delta": map[string]string{"content": c}},
				},
			}
			b, _ := json.Marshal(payload)
			fmt.Fprintf(w, "data: %s\n", b)
			if flusher != nil {
				flusher.Flush()
			}
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	return srv, &count, &lastBody
}

func chatJSONHandler(t *testing.T, content string, usage Usage) (*httptest.Server, *int64) {
	t.Helper()
	var count int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": content}},
			},
			"usage": usage,
		})
	}))
	return srv, &count
}

func embedHandler(t *testing.T, dim int) (*httptest.Server, *int64) {
	t.Helper()
	var count int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, 1)
		var body struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		out := map[string]any{
			"data":  make([]map[string]any, 0, len(body.Input)),
			"usage": Usage{PromptTokens: len(body.Input) * 4, TotalTokens: len(body.Input) * 4},
		}
		for i := 0; i < len(body.Input); i++ {
			vec := make([]float32, dim)
			for j := range vec {
				vec[j] = float32(i+j) * 0.01
			}
			out["data"] = append(out["data"].([]map[string]any), map[string]any{"embedding": vec})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}))
	return srv, &count
}

// roundTripperFunc 用于 Anthropic / Google（URL 写死时把绝对 URL 重写到 httptest.Server）。
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Do 兼容 HTTPDoer 接口（http.Client.Do 内部也是走 RoundTrip）。
func (f roundTripperFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

// ---------- registry ----------

func TestRegistry_RegisterAndLookup(t *testing.T) {
	r := NewRegistry()
	r.Register(NewOpenAICompat("openai", "https://api.openai.com/v1", ""))
	r.Register(NewOpenAICompat("deepseek", "https://api.deepseek.com/v1", ""))
	if len(r.Names()) != 2 {
		t.Fatalf("names: %v", r.Names())
	}
	if p, ok := r.Get("deepseek"); !ok || p.Name() != "deepseek" {
		t.Fatalf("lookup failed")
	}
	if _, ok := r.Get("missing"); ok {
		t.Fatalf("expected miss")
	}
}
