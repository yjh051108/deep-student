// Package mcp / stdio.go — stdio 子进程生命周期
//
// 把 "启动子进程 / 关闭子进程 / 重启 / 信号转发" 这一组语义从 mcp.go 中抽出，
// 便于集成测试里用 loopback mock 进程验证协议握手、tools/list、tools/call。
package mcp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

// StdioProcess wraps a long-running MCP server child process connected via
// stdin/stdout. The struct is goroutine-safe: writes are serialized through
// wmu, reads go through the per-call scanner.
type StdioProcess struct {
	Name   string
	Cmd    string
	Args   []string
	Env    []string
	Stderr io.Writer // default os.Stderr; tests can override to /dev/null

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner

	wmu sync.Mutex     // 序列化写
	ctr atomic.Int64   // id 自增
	closed atomic.Bool // 防止重复 close
}

// NewStdioProcess builds a process wrapper. The actual subprocess is started
// by Start().
func NewStdioProcess(name, command string, args, env []string) *StdioProcess {
	return &StdioProcess{Name: name, Cmd: command, Args: args, Env: env, Stderr: os.Stderr}
}

// Start spawns the child process and waits for the first frame so we can fail
// fast on a misconfigured command. It does NOT perform the MCP handshake;
// use Client.Start for that.
func (p *StdioProcess) Start(ctx context.Context) error {
	if p.cmd != nil {
		return fmt.Errorf("mcp: process %q already started", p.Name)
	}
	cmd := exec.CommandContext(ctx, p.Cmd, p.Args...)
	cmd.Env = append(os.Environ(), p.Env...)
	if p.Stderr != nil {
		cmd.Stderr = p.Stderr
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("mcp: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("mcp: stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("mcp: start %s: %w", p.Cmd, err)
	}
	p.cmd = cmd
	p.stdin = stdin
	p.stdout = bufio.NewScanner(stdout)
	p.stdout.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return nil
}

// WriteFrame serializes f and writes a newline-delimited frame to stdin.
func (p *StdioProcess) WriteFrame(f Frame) error {
	p.wmu.Lock()
	defer p.wmu.Unlock()
	if p.closed.Load() {
		return fmt.Errorf("mcp: process %q is closed", p.Name)
	}
	return WriteFrame(p.stdin, f)
}

// ReadFrame pulls one JSON-RPC frame from stdout. The bool is true when the
// frame is a notification (no id, has method).
func (p *StdioProcess) ReadFrame() (Frame, bool, error) {
	return ReadFrame(p.stdout)
}

// NextID returns a monotonic request id. Safe for concurrent callers.
func (p *StdioProcess) NextID() int64 { return p.ctr.Add(1) }

// PID returns the child OS pid, or 0 if not yet started.
func (p *StdioProcess) PID() int {
	if p.cmd == nil || p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

// Alive reports whether the child is still running. We treat "ProcessState !=
// nil && Success()==false" as exited; this is a best-effort check and is racy
// by design — callers wanting strong semantics should Wait().
func (p *StdioProcess) Alive() bool {
	if p.cmd == nil || p.cmd.Process == nil {
		return false
	}
	if p.closed.Load() {
		return false
	}
	// ProcessState is set only after Wait; if non-nil the process exited.
	return p.cmd.ProcessState == nil
}

// Close stops the child. It is safe to call multiple times; only the first
// call does real work.
func (p *StdioProcess) Close() error {
	if p.closed.Swap(true) {
		return nil
	}
	var firstErr error
	if p.stdin != nil {
		if err := p.stdin.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
		// Best-effort reap; cap the wait to 2s to avoid hanging the caller.
		done := make(chan struct{})
		go func() { _, _ = p.cmd.Process.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
	// Clear the bookkeeping so a subsequent Start / Restart succeeds.
	p.cmd = nil
	p.stdin = nil
	p.stdout = nil
	return firstErr
}

// Restart closes the existing child (if any) and spawns a fresh one. It
// preserves the Name / Cmd / Args / Env / Stderr configuration. After a
// successful restart the caller still needs to perform the MCP handshake —
// use Client.Start for the full re-initialization.
func (p *StdioProcess) Restart(ctx context.Context) error {
	if p.cmd == nil {
		return p.Start(ctx)
	}
	_ = p.Close() // Close clears cmd/stdin/stdout bookkeeping.
	// Reset the closed flag that Close set; Start requires a clean state.
	p.closed.Store(false)
	return p.Start(ctx)
}
