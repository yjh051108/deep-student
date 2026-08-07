package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ---------- Anthropic ----------

func TestAnthropic_Chat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "ak-test" {
			t.Errorf("missing api key: %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Errorf("missing version header")
		}
		var body struct {
			Model     string `json:"model"`
			System    string `json:"system"`
			MaxTokens int    `json:"max_tokens"`
			Messages  []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.System != "you are claude" {
			t.Errorf("system not extracted: %q", body.System)
		}
		if body.MaxTokens == 0 {
			t.Errorf("default max tokens not applied")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "content": [{"type": "text", "text": "hello from claude"}],
            "usage": {"input_tokens": 11, "output_tokens": 5}
        }`))
	}))
	defer srv.Close()
	a := NewAnthropic("ak-test")
	a.HTTP = srv.Client()
	a.BaseURL = srv.URL
	resp, err := a.Chat(context.Background(), ChatRequest{
		Model:     "claude-3-5-sonnet",
		MaxTokens: 0,
		Messages:  []Message{{Role: RoleSystem, Content: "you are claude"}, {Role: RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if resp.Content != "hello from claude" {
		t.Fatalf("content: %q", resp.Content)
	}
	if resp.Usage.PromptTokens != 11 || resp.Usage.CompletionTokens != 5 {
		t.Fatalf("usage: %+v", resp.Usage)
	}
}

func TestAnthropic_Stream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)
		events := []string{
			`{"type":"content_block_start"}`,
			`{"type":"content_block_delta","delta":{"type":"text_delta","text":"foo"}}`,
			`{"type":"content_block_delta","delta":{"type":"text_delta","text":"bar"}}`,
			`{"type":"message_stop"}`,
		}
		for _, e := range events {
			fmt.Fprintf(w, "data: %s\n\n", e)
			flusher.Flush()
		}
	}))
	defer srv.Close()
	a := NewAnthropic("ak")
	a.HTTP = srv.Client()
	a.BaseURL = srv.URL
	ch, err := a.Stream(context.Background(), ChatRequest{Model: "claude", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	got := ""
	for c := range ch {
		if c.Err != nil {
			t.Fatalf("err: %v", c.Err)
		}
		got += c.Delta
		if c.Done {
			break
		}
	}
	if got != "foobar" {
		t.Fatalf("got %q", got)
	}
}

func TestAnthropic_Embed_Unsupported(t *testing.T) {
	a := NewAnthropic("ak")
	_, err := a.Embed(context.Background(), EmbedRequest{Model: "x", Input: []string{"a"}})
	if err == nil {
		t.Fatalf("expected embed to be unsupported")
	}
}

// TestAnthropic_StreamCtxCancel BUG-003 回归：ctx 取消时 Anthropic stream
// goroutine 必须在合理时间内退出，channel 关闭（可能伴随 Err chunk）。
func TestAnthropic_StreamCtxCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)
		for i := 0; i < 50; i++ {
			ev := fmt.Sprintf(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"d%d"}}`, i)
			fmt.Fprintf(w, "data: %s\n\n", ev)
			flusher.Flush()
			select {
			case <-r.Context().Done():
				return
			case <-time.After(50 * time.Millisecond):
			}
		}
	}))
	defer srv.Close()
	a := NewAnthropic("ak")
	a.HTTP = srv.Client()
	a.BaseURL = srv.URL
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := a.Stream(ctx, ChatRequest{Model: "claude", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	got := 0
	for s := range ch {
		if s.Err != nil {
			if !strings.Contains(s.Err.Error(), "context") {
				t.Fatalf("unexpected err: %v", s.Err)
			}
			return
		}
		got++
		if got >= 2 {
			cancel()
		}
	}
	if got < 1 {
		t.Fatalf("expected at least 1 chunk before cancel")
	}
}

// ---------- Google ----------

func TestGoogle_Chat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "key=gk") {
			t.Errorf("missing api key: %q", r.URL.RawQuery)
		}
		var body struct {
			Contents []struct {
				Role  string `json:"role"`
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"contents"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if len(body.Contents) < 2 {
			t.Errorf("expected 2+ contents, got %d", len(body.Contents))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "candidates": [
                {"content": {"parts": [{"text": "Hello "}, {"text": "from Gemini"}]}}
            ]
        }`))
	}))
	defer srv.Close()
	g := NewGoogle("gk")
	// Gemini URL 在代码中硬编码为 https://generativelanguage.googleapis.com —— 用 roundTripperFunc 把请求重写到 httptest.Server。
	g.HTTP = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req2 := req.Clone(req.Context())
		req2.URL.Scheme = "http"
		req2.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return srv.Client().Do(req2)
	})
	resp, err := g.Chat(context.Background(), ChatRequest{
		Model:    "gemini-1.5-pro",
		Messages: []Message{{Role: RoleSystem, Content: "be nice"}, {Role: RoleUser, Content: "hi"}, {Role: RoleAssistant, Content: "hello"}, {Role: RoleUser, Content: "how are you?"}},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if resp.Content != "Hello from Gemini" {
		t.Fatalf("content: %q", resp.Content)
	}
}

func TestGoogle_Stream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 一次写完整响应（两个 JSON 对象），避免 Content-Length 截断。
		w.Header().Set("Content-Type", "application/json")
		body := `{"candidates": [{"content": {"parts": [{"text": "A"}]}}]}
{"candidates": [{"content": {"parts": [{"text": "B"}]}}]}
`
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	g := NewGoogle("gk")
	g.HTTP = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req2 := req.Clone(req.Context())
		req2.URL.Scheme = "http"
		req2.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return srv.Client().Do(req2)
	})
	ch, err := g.Stream(context.Background(), ChatRequest{Model: "gemini-1.5-pro", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	got := ""
	for c := range ch {
		if c.Err != nil {
			t.Fatalf("err: %v", c.Err)
		}
		got += c.Delta
		if c.Done {
			break
		}
	}
	if got != "AB" {
		t.Fatalf("got %q", got)
	}
}

// TestGoogle_StreamCtxCancel BUG-003 回归：ctx 取消时 Gemini stream
// goroutine 必须在合理时间内退出，channel 关闭。
func TestGoogle_StreamCtxCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Gemini 流式响应：一次写多个 JSON 对象，但服务端模拟"每隔 50ms 写一个对象"
		// 通过 flush 控制节奏。
		w.Header().Set("Content-Type", "application/json")
		flusher, _ := w.(http.Flusher)
		for i := 0; i < 20; i++ {
			body := fmt.Sprintf(`{"candidates": [{"content": {"parts": [{"text": "g%d"}]}}]}`+"\n", i)
			_, _ = w.Write([]byte(body))
			flusher.Flush()
			select {
			case <-r.Context().Done():
				return
			case <-time.After(50 * time.Millisecond):
			}
		}
	}))
	defer srv.Close()
	g := NewGoogle("gk")
	g.HTTP = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req2 := req.Clone(req.Context())
		req2.URL.Scheme = "http"
		req2.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return srv.Client().Do(req2)
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := g.Stream(ctx, ChatRequest{Model: "gemini-1.5-pro", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	got := 0
	for c := range ch {
		if c.Err != nil {
			if !strings.Contains(c.Err.Error(), "context") {
				t.Fatalf("unexpected err: %v", c.Err)
			}
			return
		}
		got++
		if got >= 2 {
			cancel()
		}
	}
	if got < 1 {
		t.Fatalf("expected at least 1 chunk before cancel")
	}
}

func TestGoogle_Embed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Requests []struct {
				Model string `json:"model"`
			} `json:"requests"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if len(body.Requests) != 2 {
			t.Errorf("expected 2 requests, got %d", len(body.Requests))
		}
		if body.Requests[0].Model != "models/gemini-1" {
			t.Errorf("model name not prefixed: %q", body.Requests[0].Model)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "embeddings": [
                {"values": [0.1, 0.2, 0.3]},
                {"values": [0.4, 0.5, 0.6]}
            ]
        }`))
	}))
	defer srv.Close()
	g := NewGoogle("gk")
	g.HTTP = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req2 := req.Clone(req.Context())
		req2.URL.Scheme = "http"
		req2.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return srv.Client().Do(req2)
	})
	resp, err := g.Embed(context.Background(), EmbedRequest{Model: "gemini-1", Input: []string{"a", "b"}})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(resp.Embeddings) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(resp.Embeddings))
	}
	if resp.Embeddings[0][0] != 0.1 || resp.Embeddings[1][2] != 0.6 {
		t.Fatalf("vectors wrong: %+v", resp.Embeddings)
	}
}
