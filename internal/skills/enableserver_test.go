package skills

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/helixnow/deep-student-go/pkg/config"
)

// mockServerSrc is the loopback MCP server source. It is intentionally a
// hand-written single file so we don't depend on any third-party test
// harness; it implements just enough of the JSON-RPC protocol to drive the
// real Client.Start handshake + tools/list + tools/call paths.
const mockServerSrc = `package main
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
      continue
    }
    resp := map[string]interface{}{"jsonrpc": "2.0", "id": int(id), "result": result}
    b, _ := json.Marshal(resp)
    out.Write(b); out.WriteByte('\n')
    out.Flush()
  }
}
`

// runMockServerCmd returns a (command, args) tuple that spawns the loopback
// mock. On every platform we build a tiny Go exe in t.TempDir() from
// mockServerSrc above. The build is the slow part of the test; using the
// same source keeps behaviour identical across OSes.
func runMockServerCmd(t *testing.T) (string, []string) {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "mock_mcp.go")
	if err := os.WriteFile(src, []byte(mockServerSrc), 0o644); err != nil {
		t.Fatalf("write mock: %v", err)
	}
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("go toolchain not available: %v", err)
	}
	exe := filepath.Join(t.TempDir(), "mock_mcp")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	build := exec.Command(goPath, "build", "-o", exe, src)
	build.Stderr = io.Discard
	if err := build.Run(); err != nil {
		t.Fatalf("build mock: %v", err)
	}
	return exe, nil
}

// TestEnableServerStdIO drives the spec'd SubTask 23.3 entry point end to
// end: spawn a loopback MCP server via EnableServer, confirm the tool
// registration is namespaced correctly, exercise one tool call through the
// Service.Tool dispatcher, then DisableServer and confirm cleanup.
func TestEnableServerStdIO(t *testing.T) {
	cmd, args := runMockServerCmd(t)
	svc := newSvc(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := svc.EnableServer(ctx, "mock", config.MCPServerConfig{
		Command: cmd, Args: args, Env: map[string]string{"FOO": "BAR"},
	}); err != nil {
		t.Fatalf("EnableServer: %v", err)
	}

	// Server must be listed.
	var found bool
	for _, n := range svc.ListMCPServers() {
		if n == "mock" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("mock not in ListMCPServers: %+v", svc.ListMCPServers())
	}

	// The two tools from the loopback mock must be registered, namespaced
	// with "mock.".
	names := map[string]bool{}
	for _, t2 := range svc.Tools() {
		names[t2.Name] = true
	}
	if !names["mock.echo"] || !names["mock.sum"] {
		t.Fatalf("expected namespaced tools, got %+v", names)
	}

	// Dispatch a tool call through Service.Tool and confirm the loopback
	// echo comes back. The mock wraps the raw request in content[0].text.
	out, err := svc.Tool(ctx, "mock.echo", json.RawMessage(`{"hello":"world"}`))
	if err != nil {
		t.Fatalf("Tool(mock.echo): %v", err)
	}
	raw, ok := out.(json.RawMessage)
	if !ok {
		t.Fatalf("expected json.RawMessage, got %T (%+v)", out, out)
	}
	if !json.Valid(raw) {
		t.Fatalf("invalid json: %s", string(raw))
	}
	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("parse echo: %v", err)
	}
	if len(parsed.Content) == 0 {
		t.Fatalf("empty content: %s", string(raw))
	}
	if parsed.Content[0].Text == "" {
		t.Fatalf("echo text empty: %+v", parsed)
	}

	// DisableServer should drop the tools and the registered client.
	svc.DisableServer("mock")
	if found := false; found {
		_ = found
	}
	for _, n := range svc.ListMCPServers() {
		if n == "mock" {
			t.Fatalf("mock still listed after DisableServer: %+v", svc.ListMCPServers())
		}
	}
	for _, t2 := range svc.Tools() {
		if t2.Name == "mock.echo" || t2.Name == "mock.sum" {
			t.Fatalf("tools not unregistered: %+v", t2)
		}
	}
}

// TestEnableServerReEnable exercises the "if it already exists, stop the
// old one first" branch of EnableServer. The second EnableServer should
// succeed (and quietly close the first mock).
func TestEnableServerReEnable(t *testing.T) {
	cmd, args := runMockServerCmd(t)
	svc := newSvc(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for i := 0; i < 2; i++ {
		if err := svc.EnableServer(ctx, "mock", config.MCPServerConfig{
			Command: cmd, Args: args,
		}); err != nil {
			t.Fatalf("EnableServer #%d: %v", i, err)
		}
	}
	if got := len(svc.ListMCPServers()); got != 1 {
		t.Fatalf("expected 1 server after re-enable, got %d", got)
	}
	svc.DisableServer("mock")
}
