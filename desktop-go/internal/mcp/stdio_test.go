package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

const stdioEchoChildEnv = "DEEP_STUDENT_MCP_STDIO_ECHO_CHILD"
const stdioEchoChildFramingEnv = "DEEP_STUDENT_MCP_STDIO_ECHO_FRAMING"
const stdioMcpChildEnv = "DEEP_STUDENT_MCP_STDIO_PROTOCOL_CHILD"

type emittedEvent struct {
	name    string
	payload any
}

func TestMain(m *testing.M) {
	if os.Getenv(stdioMcpChildEnv) == "1" {
		runStdioMcpChild()
		return
	}
	if os.Getenv(stdioEchoChildEnv) == "1" {
		runStdioEchoChild()
		return
	}
	os.Exit(m.Run())
}

func TestStartStdioSessionWithJsonlChildEchoesEventMessage(t *testing.T) {
	sessionID, events := startEchoSession(t, "jsonl")

	const request = `{"jsonrpc":"2.0","id":1,"method":"ping"}`
	if err := events.service.SendStdioMessage(sessionID, request); err != nil {
		t.Fatalf("SendStdioMessage() error = %v", err)
	}

	event := events.waitFor(t, func(event emittedEvent) bool {
		return event.name == "mcp-stdio-"+sessionID+"-message" &&
			payloadString(event.payload, "message") == request
	})
	if got := payloadString(event.payload, "message"); got != request {
		t.Fatalf("message payload = %q, want %q", got, request)
	}
}

func TestStartStdioSessionWithContentLengthChildEchoesEventMessage(t *testing.T) {
	sessionID, events := startEchoSession(t, "content_length")

	const request = `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	if err := events.service.SendStdioMessage(sessionID, request); err != nil {
		t.Fatalf("SendStdioMessage() error = %v", err)
	}

	event := events.waitFor(t, func(event emittedEvent) bool {
		return event.name == "mcp-stdio-"+sessionID+"-message" &&
			payloadString(event.payload, "message") == request
	})
	if got := payloadString(event.payload, "message"); got != request {
		t.Fatalf("message payload = %q, want %q", got, request)
	}
}

func TestStartStdioSessionWithMinimalMcpServer(t *testing.T) {
	for _, framing := range []string{"jsonl", "content_length"} {
		t.Run(framing, func(t *testing.T) {
			sessionID, events := startMcpProtocolSession(t, framing)

			if err := events.service.SendStdioMessage(sessionID, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"dstu-test","version":"1.0.0"}}}`); err != nil {
				t.Fatalf("send initialize: %v", err)
			}
			initialize := waitForRPCResponse(t, events, sessionID, 1)
			if getNestedString(initialize, "result", "serverInfo", "name") != "dstu-mcp-smoke" {
				t.Fatalf("unexpected initialize response: %+v", initialize)
			}

			if err := events.service.SendStdioMessage(sessionID, `{"jsonrpc":"2.0","method":"notifications/initialized"}`); err != nil {
				t.Fatalf("send initialized notification: %v", err)
			}

			if err := events.service.SendStdioMessage(sessionID, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`); err != nil {
				t.Fatalf("send tools/list: %v", err)
			}
			tools := waitForRPCResponse(t, events, sessionID, 2)
			if getFirstToolName(tools) != "smoke_echo" {
				t.Fatalf("unexpected tools/list response: %+v", tools)
			}

			if err := events.service.SendStdioMessage(sessionID, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"smoke_echo","arguments":{"message":"hello"}}}`); err != nil {
				t.Fatalf("send tools/call: %v", err)
			}
			call := waitForRPCResponse(t, events, sessionID, 3)
			if getFirstContentText(call) != "echo: hello" {
				t.Fatalf("unexpected tools/call response: %+v", call)
			}

			if err := events.service.CloseStdioSession(sessionID); err != nil {
				t.Fatalf("CloseStdioSession() error = %v", err)
			}
			if err := events.service.SendStdioMessage(sessionID, `{"jsonrpc":"2.0","id":4,"method":"ping"}`); err == nil {
				t.Fatal("SendStdioMessage() after close error = nil, want rejection")
			}
		})
	}
}

func startEchoSession(t *testing.T, framing string) (string, *eventRecorder) {
	t.Helper()

	service := newTestService(t)
	events := newEventRecorder()
	events.service = service
	service.SetEventEmitter(events.emit)

	sessionID, err := service.StartStdioSession(os.Args[0], []string{"-test.run=TestMain"}, map[string]string{
		stdioEchoChildEnv:        "1",
		stdioEchoChildFramingEnv: framing,
	}, &framing, nil)
	if err != nil {
		t.Fatalf("StartStdioSession() error = %v", err)
	}
	t.Cleanup(func() {
		if err := service.CloseStdioSession(sessionID); err != nil {
			t.Fatalf("CloseStdioSession() cleanup error = %v", err)
		}
	})
	if sessionID == "" {
		t.Fatal("StartStdioSession() returned empty session ID")
	}
	return sessionID, events
}

func startMcpProtocolSession(t *testing.T, framing string) (string, *eventRecorder) {
	t.Helper()

	service := newTestService(t)
	events := newEventRecorder()
	events.service = service
	service.SetEventEmitter(events.emit)

	sessionID, err := service.StartStdioSession(os.Args[0], []string{"-test.run=TestMain"}, map[string]string{
		stdioMcpChildEnv:         "1",
		stdioEchoChildFramingEnv: framing,
	}, &framing, nil)
	if err != nil {
		t.Fatalf("StartStdioSession() error = %v", err)
	}
	t.Cleanup(func() {
		if err := service.CloseStdioSession(sessionID); err != nil {
			t.Fatalf("CloseStdioSession() cleanup error = %v", err)
		}
	})
	if sessionID == "" {
		t.Fatal("StartStdioSession() returned empty session ID")
	}
	return sessionID, events
}

func TestStartStdioSessionReturnsErrorWhenCommandIsMissing(t *testing.T) {
	service := newTestService(t)

	framing := "jsonl"
	if _, err := service.StartStdioSession("  ", nil, nil, &framing, nil); err == nil {
		t.Fatal("StartStdioSession() error = nil, want missing command error")
	}
}

func TestSendStdioMessageRejectsClosedSession(t *testing.T) {
	service := newTestService(t)

	framing := "jsonl"
	sessionID, err := service.StartStdioSession(os.Args[0], []string{"-test.run=TestMain"}, map[string]string{
		stdioEchoChildEnv: "1",
	}, &framing, nil)
	if err != nil {
		t.Fatalf("StartStdioSession() error = %v", err)
	}
	if err := service.CloseStdioSession(sessionID); err != nil {
		t.Fatalf("CloseStdioSession() error = %v", err)
	}

	if err := service.SendStdioMessage(sessionID, `{"jsonrpc":"2.0","id":1}`); err == nil {
		t.Fatal("SendStdioMessage() error = nil, want closed session rejection")
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

type eventRecorder struct {
	service *Service
	events  chan emittedEvent
	mu      sync.Mutex
	seen    []emittedEvent
}

func newEventRecorder() *eventRecorder {
	return &eventRecorder{
		events: make(chan emittedEvent, 16),
	}
}

func (r *eventRecorder) emit(name string, payload any) {
	event := emittedEvent{name: name, payload: payload}
	r.mu.Lock()
	r.seen = append(r.seen, event)
	r.mu.Unlock()
	r.events <- event
}

func (r *eventRecorder) waitFor(t *testing.T, match func(emittedEvent) bool) emittedEvent {
	t.Helper()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()

	for {
		select {
		case event := <-r.events:
			if match(event) {
				return event
			}
		case <-timer.C:
			r.mu.Lock()
			seen := append([]emittedEvent(nil), r.seen...)
			r.mu.Unlock()
			t.Fatalf("timed out waiting for event; seen=%s", formatEvents(seen))
		}
	}
}

func payloadString(payload any, key string) string {
	if values, ok := payload.(map[string]any); ok {
		if value, ok := values[key].(string); ok {
			return value
		}
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	var values map[string]string
	if err := json.Unmarshal(encoded, &values); err != nil {
		return ""
	}
	return values[key]
}

func waitForRPCResponse(t *testing.T, events *eventRecorder, sessionID string, id int) map[string]any {
	t.Helper()
	event := events.waitFor(t, func(event emittedEvent) bool {
		if event.name != "mcp-stdio-"+sessionID+"-message" {
			return false
		}
		var message map[string]any
		if err := json.Unmarshal([]byte(payloadString(event.payload, "message")), &message); err != nil {
			return false
		}
		return int(message["id"].(float64)) == id
	})
	var message map[string]any
	if err := json.Unmarshal([]byte(payloadString(event.payload, "message")), &message); err != nil {
		t.Fatalf("parse JSON-RPC response: %v", err)
	}
	return message
}

func getNestedString(value map[string]any, keys ...string) string {
	var current any = value
	for _, key := range keys {
		values, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = values[key]
	}
	if text, ok := current.(string); ok {
		return text
	}
	return ""
}

func getFirstToolName(value map[string]any) string {
	result, ok := value["result"].(map[string]any)
	if !ok {
		return ""
	}
	tools, ok := result["tools"].([]any)
	if !ok || len(tools) == 0 {
		return ""
	}
	tool, ok := tools[0].(map[string]any)
	if !ok {
		return ""
	}
	name, _ := tool["name"].(string)
	return name
}

func getFirstContentText(value map[string]any) string {
	result, ok := value["result"].(map[string]any)
	if !ok {
		return ""
	}
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		return ""
	}
	item, ok := content[0].(map[string]any)
	if !ok {
		return ""
	}
	text, _ := item["text"].(string)
	return text
}

func formatEvents(events []emittedEvent) string {
	if len(events) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(events))
	for _, event := range events {
		parts = append(parts, fmt.Sprintf("%s:%v", event.name, event.payload))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func runStdioEchoChild() {
	if os.Getenv(stdioEchoChildFramingEnv) == "content_length" {
		runContentLengthEchoChild()
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Fprintln(os.Stdout, scanner.Text())
	}
}

func runStdioMcpChild() {
	if os.Getenv(stdioEchoChildFramingEnv) == "content_length" {
		runContentLengthMcpChild()
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if response, ok := handleMcpProtocolMessage(scanner.Text()); ok {
			fmt.Fprintln(os.Stdout, response)
		}
	}
}

func runContentLengthMcpChild() {
	reader := bufio.NewReader(os.Stdin)
	for {
		message, err := readContentLengthMessage(reader)
		if err != nil {
			if !errorsIsEOF(err) {
				fmt.Fprintln(os.Stderr, err.Error())
			}
			return
		}
		if response, ok := handleMcpProtocolMessage(message); ok {
			if _, err := fmt.Fprint(os.Stdout, formatContentLengthFrame(response)); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				return
			}
		}
	}
}

func handleMcpProtocolMessage(message string) (string, bool) {
	var request map[string]any
	if err := json.Unmarshal([]byte(message), &request); err != nil {
		return encodeRPC(map[string]any{
			"jsonrpc": "2.0",
			"id":      nil,
			"error": map[string]any{
				"code":    -32700,
				"message": err.Error(),
			},
		}), true
	}

	method, _ := request["method"].(string)
	if strings.HasPrefix(method, "notifications/") {
		return "", false
	}
	id := request["id"]
	response := map[string]any{"jsonrpc": "2.0", "id": id}

	switch method {
	case "initialize":
		response["result"] = map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "dstu-mcp-smoke",
				"version": "1.0.0",
			},
		}
	case "tools/list":
		response["result"] = map[string]any{
			"tools": []map[string]any{
				{
					"name":        "smoke_echo",
					"description": "Echoes a smoke-test message.",
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"message": map[string]any{"type": "string"},
						},
					},
				},
			},
		}
	case "tools/call":
		message := "hello"
		if params, ok := request["params"].(map[string]any); ok {
			if args, ok := params["arguments"].(map[string]any); ok {
				if text, ok := args["message"].(string); ok && text != "" {
					message = text
				}
			}
		}
		response["result"] = map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "echo: " + message},
			},
		}
	default:
		response["error"] = map[string]any{
			"code":    -32601,
			"message": "method not found",
		}
	}
	return encodeRPC(response), true
}

func encodeRPC(message map[string]any) string {
	encoded, err := json.Marshal(message)
	if err != nil {
		return `{"jsonrpc":"2.0","id":null,"error":{"code":-32603,"message":"encode failed"}}`
	}
	return string(encoded)
}

func runContentLengthEchoChild() {
	reader := bufio.NewReader(os.Stdin)
	for {
		message, err := readContentLengthMessage(reader)
		if err != nil {
			if !errorsIsEOF(err) {
				fmt.Fprintln(os.Stderr, err.Error())
			}
			return
		}
		if _, err := fmt.Fprint(os.Stdout, formatContentLengthFrame(message)); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return
		}
	}
}

func errorsIsEOF(err error) bool {
	return err == io.EOF || strings.Contains(err.Error(), "EOF")
}
