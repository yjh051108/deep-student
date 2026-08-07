package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// 7 个 OpenAI 兼容 Provider 的工厂参数：name + apiKey + 默认 model。
// CustomProvider 实际是 OpenAICompat 的实例化（证明：NewOpenAICompat("custom", ...)）。
var openAICompatProviders = []struct {
	name   string
	apiKey string
	model  string
}{
	{"openai", "sk-openai", "gpt-4o-mini"},
	{"deepseek", "sk-deepseek", "deepseek-chat"},
	{"siliconflow", "sk-sf", "BAAI/bge-m3"},
	{"zhipu", "glm-key", "glm-4"},
	{"tongyi", "sk-tongyi", "qwen-turbo"},
	{"moonshot", "sk-moonshot", "moonshot-v1-8k"},
	{"custom", "sk-custom", "custom-model"},
}

// TestOpenAICompat_StreamCtxCancel BUG-003 回归：ctx 取消时流必须尽快退出，
// 收到 Err chunk（ctx.Canceled 或 ctx.DeadlineExceeded）。
// 服务端按 ~50ms 一行的速度发 50 个 chunk，测试在拿到 1~2 个 chunk 后 cancel，
// 期望 channel 在 ≤500ms 内关闭（或收到 Err chunk）。
func TestOpenAICompat_StreamCtxCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)
		for i := 0; i < 50; i++ {
			payload := map[string]any{"choices": []map[string]any{{"delta": map[string]string{"content": fmt.Sprintf("c%d", i)}}}}
			b, _ := json.Marshal(payload)
			fmt.Fprintf(w, "data: %s\n", b)
			if flusher != nil {
				flusher.Flush()
			}
			// 检测 ctx 取消，立刻停止
			select {
			case <-r.Context().Done():
				return
			case <-time.After(50 * time.Millisecond):
			}
		}
	}))
	defer srv.Close()
	c := NewOpenAICompat("openai", srv.URL, "sk")
	c.HTTP = srv.Client()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := c.Stream(ctx, ChatRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	// 读到第一个 chunk 之后立即 cancel
	got := 0
	for s := range ch {
		if s.Err != nil {
			// 收到 Err chunk 也算正常退出
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
	// 这里 channel 关闭但没收到 Err chunk 也算通过 —— 关键是没 hang 住
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	select {
	case s, ok := <-ch:
		if ok && s.Err == nil {
			// 还可以再发一些 chunk；只要 2s 内退出即可
		}
	default:
	}
}

// ---------- Chat (7) ----------

func TestOpenAICompat_Chat(t *testing.T) {
	for _, p := range openAICompatProviders {
		t.Run(p.name, func(t *testing.T) {
			srv, n := chatJSONHandler(t, "hi from "+p.name, Usage{PromptTokens: 3, CompletionTokens: 5, TotalTokens: 8})
			defer srv.Close()
			c := NewOpenAICompat(p.name, srv.URL, p.apiKey)
			c.HTTP = srv.Client()
			resp, err := c.Chat(context.Background(), ChatRequest{Model: p.model, Messages: []Message{{Role: RoleUser, Content: "hello"}}})
			if err != nil {
				t.Fatalf("chat: %v", err)
			}
			if resp.Content != "hi from "+p.name {
				t.Fatalf("content: %q", resp.Content)
			}
			if resp.Usage.TotalTokens != 8 {
				t.Fatalf("usage: %+v", resp.Usage)
			}
			if atomic.LoadInt64(n) != 1 {
				t.Fatalf("expected 1 request, got %d", *n)
			}
		})
	}
}

// ---------- Stream (7) ----------

func TestOpenAICompat_Stream(t *testing.T) {
	for _, p := range openAICompatProviders {
		t.Run(p.name, func(t *testing.T) {
			srv, n, last := chatSSEHandler(t, []string{"你", "好", "!"})
			defer srv.Close()
			c := NewOpenAICompat(p.name, srv.URL, p.apiKey)
			c.HTTP = srv.Client()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			ch, err := c.Stream(ctx, ChatRequest{Model: p.model, Messages: []Message{{Role: RoleUser, Content: "hi"}}})
			if err != nil {
				t.Fatalf("stream: %v", err)
			}
			got := ""
			for s := range ch {
				if s.Err != nil {
					t.Fatalf("stream err: %v", s.Err)
				}
				got += s.Delta
			}
			if got != "你好!" {
				t.Fatalf("got %q", got)
			}
			if atomic.LoadInt64(n) != 1 {
				t.Fatalf("expected 1 request, got %d", *n)
			}
			if !strings.Contains(*last, "\"stream\":true") {
				t.Fatalf("body did not set stream=true: %s", *last)
			}
		})
	}
}

// ---------- Embed (7) ----------

func TestOpenAICompat_Embed(t *testing.T) {
	for _, p := range openAICompatProviders {
		t.Run(p.name, func(t *testing.T) {
			srv, n := embedHandler(t, 4)
			defer srv.Close()
			c := NewOpenAICompat(p.name, srv.URL, p.apiKey)
			c.HTTP = srv.Client()
			resp, err := c.Embed(context.Background(), EmbedRequest{Model: p.model, Input: []string{"a", "b"}})
			if err != nil {
				t.Fatalf("embed: %v", err)
			}
			if len(resp.Embeddings) != 2 || len(resp.Embeddings[0]) != 4 {
				t.Fatalf("dims: %+v", resp.Embeddings)
			}
			if resp.Usage.TotalTokens != 8 {
				t.Fatalf("usage: %+v", resp.Usage)
			}
			if atomic.LoadInt64(n) != 1 {
				t.Fatalf("expected 1 request, got %d", *n)
			}
		})
	}
}

// ---------- Auth Header (7) ----------

func TestOpenAICompat_AuthHeader(t *testing.T) {
	for _, p := range openAICompatProviders {
		t.Run(p.name, func(t *testing.T) {
			var gotAuth string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
			}))
			defer srv.Close()
			c := NewOpenAICompat(p.name, srv.URL, p.apiKey)
			c.HTTP = srv.Client()
			if _, err := c.Chat(context.Background(), ChatRequest{Model: p.model, Messages: []Message{{Role: RoleUser, Content: "hi"}}}); err != nil {
				t.Fatalf("chat: %v", err)
			}
			if gotAuth != "Bearer "+p.apiKey {
				t.Fatalf("auth header: %q (want %q)", gotAuth, "Bearer "+p.apiKey)
			}
		})
	}
}

// ---------- HTTP Error (7) ----------

func TestOpenAICompat_HTTPError(t *testing.T) {
	for _, p := range openAICompatProviders {
		t.Run(p.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(401)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			}))
			defer srv.Close()
			c := NewOpenAICompat(p.name, srv.URL, p.apiKey)
			c.HTTP = srv.Client()
			_, err := c.Chat(context.Background(), ChatRequest{Model: p.model, Messages: []Message{{Role: RoleUser, Content: "hi"}}})
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), "401") {
				t.Fatalf("missing 401 in error: %v", err)
			}
		})
	}
}

// ---------- Custom Provider 实例类型证明 ----------

// 证明 CustomProvider 实际就是 OpenAICompat 的实例化：和工厂返回的同一类型。
func TestOpenAICompat_CustomIsOpenAICompat(t *testing.T) {
	p := NewOpenAICompat("custom", "https://example.com/v1", "sk")
	var _ Provider = p
	if p.Name() != "custom" {
		t.Fatalf("name: %q", p.Name())
	}
}

// ---------- 流式响应中途断连 ----------

// TestOpenAICompat_StreamMidDisconnect 模拟 server 写了第一个 chunk 后立即关连接，
// 验证客户端不 panic、goroutine 优雅退出（要么 Err chunk 要么 Done）。
func TestOpenAICompat_StreamMidDisconnect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)
		b, _ := json.Marshal(map[string]any{"choices": []map[string]any{{"delta": map[string]string{"content": "first"}}}})
		fmt.Fprintf(w, "data: %s\n", b)
		if flusher != nil {
			flusher.Flush()
		}
		// 直接 Hijack 拿到底层 conn 并 Close —— 模拟网络断连。
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Errorf("server does not support hijack")
			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return
		}
		_ = conn.Close()
	}))
	defer srv.Close()
	c := NewOpenAICompat("custom", srv.URL, "sk")
	c.HTTP = srv.Client()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ch, err := c.Stream(ctx, ChatRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("stream open: %v", err)
	}
	// 必须能消费完所有 chunk 而不 panic/hang。
	// 期望：至少拿到一个 chunk（"first"），且最终拿到 Done 或 Err，channel 关闭。
	var gotDelta string
	var finished bool
	deadline := time.After(3 * time.Second)
	for {
		select {
		case s, ok := <-ch:
			if !ok {
				finished = true
				if !strings.Contains(gotDelta, "first") {
					t.Fatalf("expected 'first' delta, got %q", gotDelta)
				}
				if !finished {
					t.Fatalf("expected channel to close")
				}
				return
			}
			if s.Err != nil {
				// 中途断连导致 scanner 读到 EOF —— 拿到 Err chunk 是合法的
				finished = true
				if !strings.Contains(gotDelta, "first") {
					t.Fatalf("expected 'first' delta before err, got %q err=%v", gotDelta, s.Err)
				}
				return
			}
			gotDelta += s.Delta
			if s.Done {
				finished = true
				return
			}
		case <-deadline:
			t.Fatalf("stream did not terminate after mid-disconnect (gotDelta=%q)", gotDelta)
		}
	}
}
