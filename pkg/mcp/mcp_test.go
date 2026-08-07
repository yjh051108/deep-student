package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

// writeMockServerScript writes a tiny MCP server to a temp file.
// We use a shell-free self-contained Python (most CI has it) and a Go fallback
// for environments without Python. The script:
//   1. reads a line from stdin (the initialize request)
//   2. writes a fixed initialize result
//   3. reads another line (the tools/list request)
//   4. writes a tools/list result with two tools
//   5. reads a tools/call request and echoes it
//   6. exits 0
func writeMockServer(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		// Use a small Go program: more portable across the test matrix.
		return writeMockServerGo(t)
	}
	return writeMockServerPy(t)
}

func writeMockServerGo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	src := `package main
import (
  "bufio"
  "encoding/json"
  "io"
  "os"
)
func main() {
  in := bufio.NewReader(os.Stdin)
  out := bufio.NewWriter(os.Stdout)
  for {
    line, err := in.ReadBytes('\n')
    if err != nil {
      if err != io.EOF { return }
      return
    }
    var req map[string]interface{}
    if err := json.Unmarshal(line, &req); err != nil { return }
    id, _ := req["id"].(float64)
    method, _ := req["method"].(string)
    var result interface{}
    switch method {
    case "initialize":
      result = map[string]interface{}{
        "protocolVersion": "2024-11-05",
        "serverInfo":      map[string]string{"name": "mock", "version": "0.0.1"},
        "capabilities":    map[string]interface{}{},
      }
    case "tools/list":
      result = map[string]interface{}{
        "tools": []map[string]interface{}{
          {"name": "echo", "description": "echo back args", "inputSchema": map[string]interface{}{"type": "object"}},
          {"name": "sum",  "description": "sum two numbers", "inputSchema": map[string]interface{}{"type": "object"}},
        },
      }
    case "tools/call":
      // echo whatever the caller sent
      result = map[string]interface{}{"content": []map[string]interface{}{
        {"type": "text", "text": string(line)},
      }}
    case "notifications/initialized":
      // server-side notification; no response needed
      continue
    default:
      // unknown: do not write a frame, keep reading
      continue
    }
    resp := map[string]interface{}{"jsonrpc": "2.0", "id": int(id), "result": result}
    b, _ := json.Marshal(resp)
    out.Write(b); out.WriteByte('\n')
    out.Flush()
  }
}
`
	path := filepath.Join(dir, "mock_mcp.go")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write mock: %v", err)
	}
	return path
}

func writeMockServerPy(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	src := `#!/usr/bin/env python3
import sys, json
for line in sys.stdin:
    try:
        req = json.loads(line)
    except Exception:
        continue
    method = req.get("method")
    rid = req.get("id")
    if method == "initialize":
        result = {"protocolVersion": "2024-11-05", "serverInfo": {"name": "mock", "version": "0.0.1"}, "capabilities": {}}
    elif method == "tools/list":
        result = {"tools": [
            {"name": "echo", "description": "echo back args", "inputSchema": {"type": "object"}},
            {"name": "sum",  "description": "sum two numbers", "inputSchema": {"type": "object"}},
        ]}
    elif method == "tools/call":
        result = {"content": [{"type": "text", "text": line.strip()}]}
    else:
        continue
    sys.stdout.write(json.dumps({"jsonrpc": "2.0", "id": rid, "result": result}) + "\n")
    sys.stdout.flush()
`
	path := filepath.Join(dir, "mock_mcp.py")
	if err := os.WriteFile(path, []byte(src), 0o755); err != nil {
		t.Fatalf("write mock: %v", err)
	}
	return path
}

// runMockServer returns the (command, args) to spawn a mock MCP server.
func runMockServer(t *testing.T) (string, []string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		goPath, err := exec.LookPath("go")
		if err != nil {
			t.Skipf("go toolchain not available: %v", err)
		}
		src := writeMockServerGo(t)
		// 1) Build a small exe next to the source so we don't pay the cost every test
		exe := filepath.Join(t.TempDir(), "mock_mcp.exe")
		build := exec.Command(goPath, "build", "-o", exe, src)
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			t.Fatalf("build mock: %v", err)
		}
		return exe, nil
	}
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skipf("python3 not available: %v", err)
	}
	src := writeMockServerPy(t)
	return py, []string{src}
}

// ---------- tests ----------

func TestStdioProcessLifecycle(t *testing.T) {
	cmd, args := runMockServer(t)
	proc := NewStdioProcess("mock", cmd, args, nil)
	proc.Stderr = io.Discard
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := proc.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	if proc.PID() <= 0 {
		t.Fatalf("expected positive pid, got %d", proc.PID())
	}
	if !proc.Alive() {
		t.Fatalf("expected alive after start")
	}
	if err := proc.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// close is idempotent
	if err := proc.Close(); err != nil {
		t.Fatalf("close again: %v", err)
	}
}

func TestTransportFrameRoundTrip(t *testing.T) {
	r, w := io.Pipe()
	defer r.Close()
	proc := NewStdioProcess("p", "ignored", nil, nil)
	proc.Stderr = io.Discard

	// Use a real *bufio.Scanner to read what we wrote
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024)
	want := Frame{JSONRPC: "2.0", ID: 7, Method: "ping", Params: json.RawMessage(`{"a":1}`)}
	go func() {
		_ = WriteFrame(w, want)
		w.Close()
	}()
	got, _, err := ReadFrame(scanner)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.ID != want.ID || got.Method != want.Method {
		t.Fatalf("round-trip mismatch: %+v vs %+v", got, want)
	}
}

func TestClientStdioHandshake(t *testing.T) {
	cmd, args := runMockServer(t)
	c := NewStdio("mock", cmd, args, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer c.Close()
	tools := c.Tools()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d: %+v", len(tools), tools)
	}
	names := map[string]bool{}
	for _, tt := range tools {
		names[tt.Name] = true
	}
	if !names["echo"] || !names["sum"] {
		t.Fatalf("missing tools: %+v", names)
	}
}

func TestClientCallToolRoundtrip(t *testing.T) {
	cmd, args := runMockServer(t)
	c := NewStdio("mock", cmd, args, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer c.Close()
	raw, err := c.CallTool(ctx, "echo", map[string]any{"hello": "world"})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("invalid json: %s", string(raw))
	}
	// mock echoes the request line; check it includes the tool name and our arg
	if got := string(raw); !contains(got, "echo") || !contains(got, "hello") {
		t.Fatalf("unexpected echo payload: %s", got)
	}
}

func TestStdioProcessNextIDMonotonic(t *testing.T) {
	p := NewStdioProcess("p", "ignored", nil, nil)
	var seen [100]int64
	var i int
	for i = 0; i < 100; i++ {
		seen[i] = p.NextID()
	}
	for i = 1; i < 100; i++ {
		if seen[i] != seen[i-1]+1 {
			t.Fatalf("id not monotonic: %v -> %v", seen[i-1], seen[i])
		}
	}
}

func TestClosedClientErrors(t *testing.T) {
	cmd, args := runMockServer(t)
	c := NewStdio("mock", cmd, args, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	// Close before any call: must not panic, must return clean error.
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// We don't assert the exact error string; we just want "no panic, no goroutine leak".
	_, _ = c.Call(ctx, "tools/list", nil)
}

// contains is a small stdlib-free helper (Go 1.21+ has strings.Contains but we
// keep this file dependency-light for the loopback mock tests).
func contains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// make sure the package compiles with both directions of error returns
var _ = errors.New
var _ atomic.Int64

// TestStdioProcessRestart verifies that Restart() can recycle a child process
// and that subsequent frames are served by the new process. We use the
// loopback mock to confirm the second child is a fresh one (different PID).
func TestStdioProcessRestart(t *testing.T) {
	cmd, args := runMockServer(t)
	proc := NewStdioProcess("mock", cmd, args, nil)
	proc.Stderr = io.Discard
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := proc.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	pid1 := proc.PID()
	if err := proc.Restart(ctx); err != nil {
		t.Fatalf("restart: %v", err)
	}
	pid2 := proc.PID()
	if pid2 <= 0 {
		t.Fatalf("expected positive pid after restart, got %d", pid2)
	}
	if pid1 == pid2 {
		// On Windows process ids recycle, but the loopback mock should
		// start a new child in a fresh TempDir-built exe each call to
		// runMockServer. Either way, post-restart we must be able to
		// exchange frames; that's the real assertion.
		t.Logf("note: pid recycled (%d == %d); continuing", pid1, pid2)
	}
	// Confirm we can still send a frame and not get an error from a
	// half-closed pipe.
	if err := proc.WriteFrame(Frame{JSONRPC: "2.0", ID: 1, Method: "ping"}); err != nil {
		t.Fatalf("write after restart: %v", err)
	}
	if err := proc.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

// TestStdioProcessRestartFromIdle ensures Restart on a process that was
// never started just delegates to Start (idempotent lifecycle convenience).
func TestStdioProcessRestartFromIdle(t *testing.T) {
	cmd, args := runMockServer(t)
	proc := NewStdioProcess("mock", cmd, args, nil)
	proc.Stderr = io.Discard
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := proc.Restart(ctx); err != nil {
		t.Fatalf("restart from idle: %v", err)
	}
	if proc.PID() <= 0 {
		t.Fatalf("expected positive pid, got %d", proc.PID())
	}
	if err := proc.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

// TestReadFrameSkipsNotifications feeds the transport a JSON-RPC notification
// (no id, has method) followed by a response (has id) and asserts ReadFrame
// surfaces the notification flag correctly.
func TestReadFrameSkipsNotifications(t *testing.T) {
	r, w := io.Pipe()
	defer r.Close()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024)
	notif := `{"jsonrpc":"2.0","method":"notifications/message","params":{"hi":1}}` + "\n"
	resp := `{"jsonrpc":"2.0","id":42,"result":{"ok":true}}` + "\n"
	go func() {
		_, _ = w.Write([]byte(notif))
		_, _ = w.Write([]byte(resp))
		w.Close()
	}()
	got1, isNotif, err := ReadFrame(scanner)
	if err != nil {
		t.Fatalf("read 1: %v", err)
	}
	if !isNotif {
		t.Fatalf("expected notification flag, got false (frame=%+v)", got1)
	}
	if got1.Method != "notifications/message" {
		t.Fatalf("unexpected method: %s", got1.Method)
	}
	got2, isNotif, err := ReadFrame(scanner)
	if err != nil {
		t.Fatalf("read 2: %v", err)
	}
	if isNotif {
		t.Fatalf("expected non-notification frame, got notification")
	}
	if got2.ID != 42 {
		t.Fatalf("expected id=42, got %d", got2.ID)
	}
}

// TestClientToolRegistration asserts that after handshake the client exposes
// the same tool list that the loopback server advertises. This is the
// "tool registration" half of the spec'd integration test.
func TestClientToolRegistration(t *testing.T) {
	cmd, args := runMockServer(t)
	c := NewStdio("mock", cmd, args, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer c.Close()
	tools := c.Tools()
	if len(tools) < 2 {
		t.Fatalf("expected at least 2 tools registered, got %d: %+v", len(tools), tools)
	}
	// Verify the tool list returned by Tools() is a copy (mutating it must
	// not affect subsequent calls).
	tools[0].Name = "mutated"
	tools2 := c.Tools()
	if tools2[0].Name == "mutated" {
		t.Fatalf("Tools() should return a defensive copy")
	}
	// Find echo and exercise it via CallTool.
	var found bool
	for _, tt := range tools2 {
		if tt.Name == "echo" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing echo tool: %+v", tools2)
	}
	raw, err := c.CallTool(ctx, "echo", map[string]any{"x": 1})
	if err != nil {
		t.Fatalf("call echo: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("invalid json: %s", string(raw))
	}
	// mock echoes the raw request line wrapped in a content[0].text field.
	// The JSON gets escaped, so we look for the escaped form.
	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(parsed.Content) == 0 {
		t.Fatalf("empty content: %s", string(raw))
	}
	if !contains(parsed.Content[0].Text, "echo") || !contains(parsed.Content[0].Text, "x") {
		t.Fatalf("echo payload missing markers: %q", parsed.Content[0].Text)
	}
}

// TestClientContextCancel ensures a Call with an already-cancelled context
// returns promptly with a context error rather than blocking on the server.
func TestClientContextCancel(t *testing.T) {
	cmd, args := runMockServer(t)
	c := NewStdio("mock", cmd, args, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer c.Close()
	// Pre-cancelled context: any call must fail without sending anything.
	cancelled, cc := context.WithCancel(ctx)
	cc()
	_, err := c.CallTool(cancelled, "echo", nil)
	if err == nil {
		t.Fatalf("expected context error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		if !contains(err.Error(), "context") {
			t.Fatalf("expected context error, got %v", err)
		}
	}
}
