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

// HTTPDoer 抽象 http.Client 便于测试。
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// OpenAICompat 通用 OpenAI 兼容适配器。
type OpenAICompat struct {
	name    string
	BaseURL string
	APIKey  string
	HTTP    HTTPDoer
}

// NewOpenAICompat 创建 OpenAI 兼容适配器。
func NewOpenAICompat(name, baseURL, apiKey string) *OpenAICompat {
	return &OpenAICompat{
		name:    name,
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (o *OpenAICompat) Name() string { return o.name }

func (o *OpenAICompat) chatURL() string  { return o.BaseURL + "/chat/completions" }
func (o *OpenAICompat) embedURL() string { return o.BaseURL + "/embeddings" }

func (o *OpenAICompat) doRequest(ctx context.Context, url string, body any, out any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if o.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.APIKey)
	}
	resp, err := o.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("llm: %s: %d: %s", o.Name(), resp.StatusCode, string(raw))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Chat 非流式。
func (o *OpenAICompat) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	req.Stream = false
	var raw struct {
		Choices []struct {
			Message struct {
				Content   string     `json:"content"`
				Reasoning string     `json:"reasoning_content,omitempty"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage Usage `json:"usage"`
	}
	if err := o.doRequest(ctx, o.chatURL(), req, &raw); err != nil {
		return nil, err
	}
	out := &ChatResponse{Usage: raw.Usage}
	if len(raw.Choices) > 0 {
		out.Content = raw.Choices[0].Message.Content
		out.Reasoning = raw.Choices[0].Message.Reasoning
		out.ToolCalls = raw.Choices[0].Message.ToolCalls
	}
	return out, nil
}

// Stream 流式（SSE）。
func (o *OpenAICompat) Stream(ctx context.Context, req ChatRequest) (<-chan Chunk, error) {
	req.Stream = true
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.chatURL(), bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if o.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.APIKey)
	}
	httpReq.Header.Set("Accept", "text/event-stream")
	resp, err := o.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("llm: %s: %d: %s", o.Name(), resp.StatusCode, string(raw))
	}
	// BUG-003: 使用 1 缓冲，保证 ctx 取消时生产者最多领先 1 个 chunk。
	// 配合循环顶部的 ctx 检查，调用方一旦取消就能在 ≤2 步内收到错误 chunk。
	ch := make(chan Chunk, 1)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			// 1) 循环顶部检查：ctx 取消立即发错误 chunk 并退出
			if err := ctx.Err(); err != nil {
				ch <- Chunk{Err: err, Done: true}
				return
			}
			line := scanner.Text()
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				ch <- Chunk{Done: true}
				return
			}
			var s struct {
				Choices []struct {
					Delta struct {
						Content   string `json:"content"`
						Reasoning string `json:"reasoning_content,omitempty"`
					} `json:"delta"`
				} `json:"choices"`
				Usage Usage `json:"usage"`
			}
			if err := json.Unmarshal([]byte(data), &s); err != nil {
				continue
			}
			c := Chunk{Usage: s.Usage}
			if len(s.Choices) > 0 {
				c.Delta = s.Choices[0].Delta.Content
				c.Reason = s.Choices[0].Delta.Reasoning
			}
			// 2) 发送时也检查 ctx：ctx 已取消时优先走取消分支
			select {
			case ch <- c:
				// 3) 发送成功后再次检查 ctx，确保下一次循环顶部能尽快感知取消
				if err := ctx.Err(); err != nil {
					ch <- Chunk{Err: err, Done: true}
					return
				}
			case <-ctx.Done():
				ch <- Chunk{Err: ctx.Err(), Done: true}
				return
			}
		}
		// BUG-003: scanner 退出后再检查 ctx
		if err := scanner.Err(); err != nil {
			ch <- Chunk{Err: err, Done: true}
			return
		}
		ch <- Chunk{Done: true}
	}()
	return ch, nil
}

// Embed 嵌入。
func (o *OpenAICompat) Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error) {
	var raw struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
		Usage Usage `json:"usage"`
	}
	if err := o.doRequest(ctx, o.embedURL(), req, &raw); err != nil {
		return nil, err
	}
	out := &EmbedResponse{Usage: raw.Usage, Embeddings: make([][]float32, 0, len(raw.Data))}
	for _, d := range raw.Data {
		out.Embeddings = append(out.Embeddings, d.Embedding)
	}
	return out, nil
}
