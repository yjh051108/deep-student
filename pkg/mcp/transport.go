// Package mcp / transport.go — JSON-RPC 帧读写
//
// 把 mcp.go 里的写请求 / 读响应 / 错误处理抽成可独立测试的 transport 层。
// stdio 和 http+sse 走同一组语义。
package mcp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Frame is a single JSON-RPC message. Use NewFrame/WriteFrame/ReadFrame helpers
// to encode/decode between wire bytes and structured values.
type Frame struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// WriteFrame encodes a Frame as one JSON object terminated by '\n'.
// Stdout is line-buffered in most MCP servers, so we never pretty-print.
func WriteFrame(w io.Writer, f Frame) error {
	b, err := json.Marshal(f)
	if err != nil {
		return fmt.Errorf("mcp: marshal: %w", err)
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

// ReadFrame reads one newline-delimited JSON object. Notifications / unsolicited
// server messages are returned to the caller as the boolean second value, so
// the request/response matcher can skip them.
func ReadFrame(r *bufio.Scanner) (Frame, bool, error) {
	if !r.Scan() {
		if err := r.Err(); err != nil {
			return Frame{}, false, err
		}
		return Frame{}, false, io.EOF
	}
	var f Frame
	if err := json.Unmarshal(r.Bytes(), &f); err != nil {
		return Frame{}, false, fmt.Errorf("mcp: bad frame: %w", err)
	}
	// 通知（无 id）一律不当作 request/response 匹配，跳过
	if f.ID == 0 && f.Method != "" {
		return f, true, nil
	}
	return f, false, nil
}

// Drain reads and discards frames until EOF or a non-notification frame is found.
// Useful at startup to flush "server → client" notifications before sending the
// first real request.
func Drain(r *bufio.Scanner) error {
	for {
		_, notif, err := ReadFrame(r)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if !notif {
			// 出现了一条带 id 的消息，但因为我们已经把它从 buffer 读出来，
			// 把它放回无法做到；这里选择直接报错以暴露协议错误。
			return errors.New("mcp: unexpected id-bearing frame before initialize")
		}
	}
}
