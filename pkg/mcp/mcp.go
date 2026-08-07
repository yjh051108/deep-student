// Package mcp 实现 Model Context Protocol 客户端（stdio + http+sse）。
//
// 本文件保留对外的 Client API；内部把 stdio 子进程管理交给 stdio.go，
// 把 JSON-RPC 帧编解码交给 transport.go。
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Request JSON-RPC 请求。
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response JSON-RPC 响应。
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error JSON-RPC 错误。
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *Error) Error() string { return fmt.Sprintf("mcp: %d: %s", e.Code, e.Message) }

// Tool MCP 工具声明。
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// Client MCP 客户端。
type Client struct {
	Name    string
	Cmd     string
	Args    []string
	Env     []string
	URL     string // http+sse transport
	Headers map[string]string
	tools   []Tool
	mu      sync.Mutex
	proc    *StdioProcess
	cmd     string // legacy alias
	stdin   io.WriteCloser
	stdout  *bufio.Scanner
	httpCli *http.Client
	id      int64
	closed  atomic.Bool
}

// NewStdio 创建 stdio 客户端。
func NewStdio(name, cmd string, args, env []string) *Client {
	return &Client{Name: name, Cmd: cmd, Args: args, Env: env, cmd: cmd}
}

// NewSSE 创建 http+sse 客户端。
func NewSSE(name, url string, headers map[string]string) *Client {
	return &Client{Name: name, URL: url, Headers: headers, httpCli: &http.Client{Timeout: 60 * time.Second}}
}

// Start 初始化连接与握手。
func (c *Client) Start(ctx context.Context) error {
	if c.URL != "" {
		return c.startSSE(ctx)
	}
	return c.startStdio(ctx)
}

func (c *Client) startStdio(ctx context.Context) error {
	proc := NewStdioProcess(c.Name, c.Cmd, c.Args, c.Env)
	if err := proc.Start(ctx); err != nil {
		return err
	}
	c.proc = proc
	c.stdin = proc.stdin
	c.stdout = proc.stdout
	// initialize
	if _, err := c.Call(ctx, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "deepstudent-go", "version": "0.1.0"},
	}); err != nil {
		_ = proc.Close()
		return fmt.Errorf("mcp initialize: %w", err)
	}
	// initialized notification
	_ = c.notify(ctx, "notifications/initialized", nil)
	// list tools
	raw, err := c.Call(ctx, "tools/list", nil)
	if err != nil {
		_ = proc.Close()
		return fmt.Errorf("mcp list tools: %w", err)
	}
	var r struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		_ = proc.Close()
		return err
	}
	c.mu.Lock()
	c.tools = r.Tools
	c.mu.Unlock()
	return nil
}

func (c *Client) startSSE(ctx context.Context) error {
	// initialize over HTTP POST to URL.
	raw, err := c.httpCall(ctx, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "deepstudent-go", "version": "0.1.0"},
	})
	if err != nil {
		return err
	}
	_ = raw
	raw2, err := c.httpCall(ctx, "tools/list", nil)
	if err != nil {
		return err
	}
	var r struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(raw2, &r); err != nil {
		return err
	}
	c.mu.Lock()
	c.tools = r.Tools
	c.mu.Unlock()
	return nil
}

// Tools 工具列表。
func (c *Client) Tools() []Tool {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Tool, len(c.tools))
	copy(out, c.tools)
	return out
}

// Call 同步调用。
func (c *Client) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if c.closed.Load() {
		return nil, errors.New("mcp: closed")
	}
	id := atomic.AddInt64(&c.id, 1)
	req := Request{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	if c.URL != "" {
		return c.httpCall(ctx, method, params)
	}
	return c.stdioCall(ctx, req)
}

func (c *Client) stdioCall(ctx context.Context, req Request) (json.RawMessage, error) {
	// write request
	c.mu.Lock()
	enc := json.NewEncoder(c.stdin)
	if err := enc.Encode(req); err != nil {
		c.mu.Unlock()
		return nil, err
	}
	c.mu.Unlock()

	type result struct {
		raw json.RawMessage
		err error
	}
	done := make(chan result, 1)
	go func() {
		for c.stdout.Scan() {
			line := c.stdout.Bytes()
			var r Response
			if err := json.Unmarshal(line, &r); err != nil {
				continue
			}
			if r.ID == req.ID {
				if r.Error != nil {
					done <- result{err: r.Error}
				} else {
					done <- result{raw: r.Result}
				}
				return
			}
		}
		done <- result{err: errors.New("mcp: eof")}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-done:
		return r.raw, r.err
	}
}

func (c *Client) notify(ctx context.Context, method string, params any) error {
	req := Request{JSONRPC: "2.0", Method: method, Params: params}
	c.mu.Lock()
	defer c.mu.Unlock()
	return json.NewEncoder(c.stdin).Encode(req)
}

func (c *Client) httpCall(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := atomic.AddInt64(&c.id, 1)
	body, _ := json.Marshal(Request{JSONRPC: "2.0", ID: id, Method: method, Params: params})
	req, _ := http.NewRequestWithContext(ctx, "POST", c.URL, nil)
	req.Body = io.NopCloser(newReader(body))
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}
	resp, err := c.httpCli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mcp http: %d: %s", resp.StatusCode, string(b))
	}
	var r Response
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	if r.Error != nil {
		return nil, r.Error
	}
	return r.Result, nil
}

// Close 关闭连接。
func (c *Client) Close() error {
	if c.closed.Swap(true) {
		return nil
	}
	if c.proc != nil {
		return c.proc.Close()
	}
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	return nil
}

// CallTool 便捷方法。
func (c *Client) CallTool(ctx context.Context, name string, args any) (json.RawMessage, error) {
	return c.Call(ctx, "tools/call", map[string]any{"name": name, "arguments": args})
}

// small in-memory reader helper
type memReader struct {
	b []byte
	i int
}

func newReader(b []byte) *memReader { return &memReader{b: b} }
func (m *memReader) Read(p []byte) (int, error) {
	if m.i >= len(m.b) {
		return 0, io.EOF
	}
	n := copy(p, m.b[m.i:])
	m.i += n
	return n, nil
}

// ensure context import doesn't get optimized away on platforms without os
var _ = os.Stderr
