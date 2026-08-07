package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Anthropic Anthropic Messages API 适配器。
type Anthropic struct {
	APIKey  string
	BaseURL string
	HTTP    HTTPDoer
}

// NewAnthropic 创建适配器。
func NewAnthropic(apiKey string) *Anthropic {
	return &Anthropic{
		APIKey:  apiKey,
		BaseURL: "https://api.anthropic.com",
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (a *Anthropic) Name() string { return "anthropic" }

type anthropicReq struct {
	Model     string             `json:"model"`
	Messages  []anthropicMessage `json:"messages"`
	System    string             `json:"system,omitempty"`
	MaxTokens int                `json:"max_tokens"`
	Stream    bool               `json:"stream"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (a *Anthropic) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	body := a.buildRequest(req)
	var raw struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := a.do(ctx, body, &raw); err != nil {
		return nil, err
	}
	out := &ChatResponse{Usage: Usage{PromptTokens: raw.Usage.InputTokens, CompletionTokens: raw.Usage.OutputTokens, TotalTokens: raw.Usage.InputTokens + raw.Usage.OutputTokens}}
	for _, c := range raw.Content {
		out.Content += c.Text
	}
	return out, nil
}

// Stream Anthropic 流式（SSE：data: {...}\n\n）。
func (a *Anthropic) Stream(ctx context.Context, req ChatRequest) (<-chan Chunk, error) {
	body := a.buildRequest(req)
	buf, _ := json.Marshal(body)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", a.BaseURL+"/v1/messages", bytes.NewReader(buf))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	resp, err := a.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("anthropic: %d: %s", resp.StatusCode, string(raw))
	}
	ch := make(chan Chunk, 32)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			// BUG-003: 监听 ctx 取消，提前退出
			select {
			case <-ctx.Done():
				ch <- Chunk{Err: ctx.Err(), Done: true}
				return
			default:
			}
			line := scanner.Text()
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" || data == "[DONE]" {
				continue
			}
			var ev struct {
				Type  string `json:"type"`
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				continue
			}
			c := Chunk{}
			switch ev.Type {
			case "content_block_delta":
				c.Delta = ev.Delta.Text
			case "message_stop":
				c.Done = true
			}
			select {
			case ch <- c:
				if c.Done {
					return
				}
			case <-ctx.Done():
				ch <- Chunk{Err: ctx.Err(), Done: true}
				return
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- Chunk{Err: err, Done: true}
			return
		}
		ch <- Chunk{Done: true}
	}()
	return ch, nil
}

// Embed Anthropic 不提供直接 Embed（占位）。
func (a *Anthropic) Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error) {
	return nil, fmt.Errorf("anthropic: embed not supported, use Voyage via /llm/voyage or a separate provider")
}

func (a *Anthropic) buildRequest(req ChatRequest) anthropicReq {
	out := anthropicReq{Model: req.Model, MaxTokens: req.MaxTokens, Stream: req.Stream}
	if out.MaxTokens == 0 {
		out.MaxTokens = 2048
	}
	for _, m := range req.Messages {
		switch m.Role {
		case RoleSystem:
			out.System = m.Content
		case RoleUser:
			out.Messages = append(out.Messages, anthropicMessage{Role: "user", Content: m.Content})
		case RoleAssistant:
			out.Messages = append(out.Messages, anthropicMessage{Role: "assistant", Content: m.Content})
		}
	}
	return out
}

func (a *Anthropic) do(ctx context.Context, body any, out any) error {
	buf, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", a.BaseURL+"/v1/messages", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := a.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("anthropic: %d: %s", resp.StatusCode, string(raw))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Google Gemini 适配器（最小实现）。
type Google struct {
	APIKey string
	HTTP   HTTPDoer
}

// NewGoogle 创建 Gemini 适配器。
func NewGoogle(apiKey string) *Google {
	return &Google{APIKey: apiKey, HTTP: &http.Client{Timeout: 60 * time.Second}}
}

func (g *Google) Name() string { return "google" }

// Chat Gemini non-stream。
func (g *Google) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	url := "https://generativelanguage.googleapis.com/v1beta/models/" + req.Model + ":generateContent?key=" + g.APIKey
	type part struct {
		Text string `json:"text"`
	}
	type content struct {
		Parts []part `json:"parts"`
		Role  string `json:"role,omitempty"`
	}
	type body struct {
		Contents []content `json:"contents"`
	}
	var b body
	for _, m := range req.Messages {
		role := "user"
		if m.Role == RoleAssistant {
			role = "model"
		}
		if m.Role == RoleSystem {
			continue
		}
		b.Contents = append(b.Contents, content{Parts: []part{{Text: m.Content}}, Role: role})
	}
	raw, _ := json.Marshal(b)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(raw))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := g.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		r, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google: %d: %s", resp.StatusCode, string(r))
	}
	var r struct {
		Candidates []struct {
			Content struct {
				Parts []part `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	out := &ChatResponse{}
	for _, c := range r.Candidates {
		for _, p := range c.Content.Parts {
			out.Content += p.Text
		}
	}
	return out, nil
}

// Stream Gemini stream（简化）。
func (g *Google) Stream(ctx context.Context, req ChatRequest) (<-chan Chunk, error) {
	url := "https://generativelanguage.googleapis.com/v1beta/models/" + req.Model + ":streamGenerateContent?key=" + g.APIKey
	type part struct {
		Text string `json:"text"`
	}
	type content struct {
		Parts []part `json:"parts"`
		Role  string `json:"role,omitempty"`
	}
	type body struct {
		Contents []content `json:"contents"`
	}
	var b body
	for _, m := range req.Messages {
		role := "user"
		if m.Role == RoleAssistant {
			role = "model"
		}
		if m.Role == RoleSystem {
			continue
		}
		b.Contents = append(b.Contents, content{Parts: []part{{Text: m.Content}}, Role: role})
	}
	raw, _ := json.Marshal(b)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(raw))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := g.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		r, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("google: %d: %s", resp.StatusCode, string(r))
	}
	ch := make(chan Chunk, 32)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		dec := json.NewDecoder(resp.Body)
		for {
			// BUG-003: 监听 ctx 取消，提前退出
			select {
			case <-ctx.Done():
				ch <- Chunk{Err: ctx.Err(), Done: true}
				return
			default:
			}
			var r struct {
				Candidates []struct {
					Content struct {
						Parts []part `json:"parts"`
					} `json:"content"`
				} `json:"candidates"`
			}
			if err := dec.Decode(&r); err != nil {
				if err == io.EOF {
					ch <- Chunk{Done: true}
					return
				}
				ch <- Chunk{Err: err, Done: true}
				return
			}
			for _, c := range r.Candidates {
				for _, p := range c.Content.Parts {
					select {
					case ch <- Chunk{Delta: p.Text}:
					case <-ctx.Done():
						ch <- Chunk{Err: ctx.Err(), Done: true}
						return
					}
				}
			}
		}
	}()
	return ch, nil
}

// Embed Gemini embed（v1beta embedContent）。
func (g *Google) Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error) {
	url := "https://generativelanguage.googleapis.com/v1beta/models/" + req.Model + ":batchEmbedContents?key=" + g.APIKey
	type req2 struct {
		Requests []struct {
			Model   string `json:"model"`
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"requests"`
	}
	var r req2
	for _, in := range req.Input {
		var item struct {
			Model   string `json:"model"`
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		}
		item.Model = "models/" + req.Model
		item.Content.Parts = append(item.Content.Parts, struct {
			Text string `json:"text"`
		}{Text: in})
		r.Requests = append(r.Requests, item)
	}
	raw, _ := json.Marshal(r)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(raw))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := g.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google embed: %d: %s", resp.StatusCode, string(b))
	}
	var raw2 struct {
		Embeddings []struct {
			Values []float32 `json:"values"`
		} `json:"embeddings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw2); err != nil {
		return nil, err
	}
	out := &EmbedResponse{Embeddings: make([][]float32, 0, len(raw2.Embeddings))}
	for _, e := range raw2.Embeddings {
		out.Embeddings = append(out.Embeddings, e.Values)
	}
	return out, nil
}

// TrimAll 返回去掉前后空白的副本（占位，避免 import errors）。
func TrimAll(s string) string { return strings.TrimSpace(s) }
