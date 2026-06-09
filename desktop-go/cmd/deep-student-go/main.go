package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	coreapp "deep-student-go/internal/app"
	"deep-student-go/internal/bindings"
	coreruntime "deep-student-go/internal/runtime"
	"deep-student-go/internal/vfs"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

const wailsSmokePortEnv = "DEEP_STUDENT_WAILS_REMOTE_DEBUGGING_PORT"
const wailsSmokeEnabledEnv = "DEEP_STUDENT_WAILS_UI_SMOKE"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--mcp-stdio-smoke-child" {
		runMcpStdioSmokeChild()
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "--smoke-pdfium" {
		result, err := vfs.SmokePDFiumRasterPreview()
		if err != nil {
			log.Fatal(err)
		}
		out, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(out))
		return
	}

	applicationState, err := coreapp.New()
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) > 1 && os.Args[1] == "--smoke" {
		info := map[string]string{
			"dataDir": applicationState.DataDir,
		}
		out, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(out))
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "--cli" {
		fmt.Println("Deep Student Go migration shell")
		fmt.Println("Data directory:", applicationState.DataDir)
		return
	}

	smokeMode := isWailsUISmokeEnabled()
	wailsApp := application.New(application.Options{
		Name:        "Deep Student",
		Description: "Deep Student Go migration shell",
		Services: []application.Service{
			application.NewService(bindings.NewAnkiService(applicationState)),
			application.NewService(bindings.NewChatService(applicationState)),
			application.NewService(bindings.NewDstuService(applicationState)),
			application.NewService(bindings.NewFileService(applicationState)),
			application.NewService(bindings.NewMcpService(applicationState)),
			application.NewService(bindings.NewNotesService(applicationState)),
			application.NewService(bindings.NewQbankService(applicationState)),
			application.NewService(bindings.NewReviewPlanService(applicationState)),
			application.NewService(bindings.NewSettingsService(applicationState)),
			application.NewService(bindings.NewSkillService(applicationState)),
			application.NewService(bindings.NewSystemService(applicationState)),
			application.NewService(bindings.NewTemplateService(applicationState)),
			application.NewService(bindings.NewTodoService(applicationState)),
			application.NewService(bindings.NewVfsService(applicationState)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Flags: map[string]any{
			"deepStudentWailsSmoke": smokeMode,
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Windows: windowsOptionsFromEnv(smokeMode),
	})
	applicationState.Events.Listen("*", func(event coreruntime.Event) {
		wailsApp.Event.Emit(event.Name, event.Payload)
	})

	windowURL := "/"
	if smokeMode {
		windowURL = "/?go-wails-smoke=true"
	}
	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Deep Student",
		BackgroundColour: application.NewRGB(20, 24, 32),
		URL:              windowURL,
	})

	defer applicationState.Mcp.CloseAll()
	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}

func runMcpStdioSmokeChild() {
	reader := bufio.NewReader(os.Stdin)
	for {
		message, err := readMcpContentLengthMessage(reader)
		if err != nil {
			if err != io.EOF {
				fmt.Fprintln(os.Stderr, err.Error())
			}
			return
		}
		response, shouldExit := handleMcpSmokeMessage(message)
		if response != "" {
			fmt.Fprint(os.Stdout, formatMcpContentLengthFrame(response))
		}
		if shouldExit {
			return
		}
	}
}

func handleMcpSmokeMessage(message string) (string, bool) {
	var request map[string]any
	if err := json.Unmarshal([]byte(message), &request); err != nil {
		return encodeMcpSmokeRPC(map[string]any{
			"jsonrpc": "2.0",
			"id":      nil,
			"error": map[string]any{
				"code":    -32700,
				"message": err.Error(),
			},
		}), false
	}

	method, _ := request["method"].(string)
	if strings.HasPrefix(method, "notifications/") {
		return "", false
	}

	id := request["id"]
	response := map[string]any{"jsonrpc": "2.0", "id": id}
	shouldExit := false

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
		echo := "hello"
		if params, ok := request["params"].(map[string]any); ok {
			if args, ok := params["arguments"].(map[string]any); ok {
				if text, ok := args["message"].(string); ok && text != "" {
					echo = text
				}
			}
		}
		response["result"] = map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "echo: " + echo},
			},
		}
	case "shutdown":
		response["result"] = map[string]any{}
		shouldExit = true
	default:
		response["error"] = map[string]any{
			"code":    -32601,
			"message": "method not found",
		}
	}
	return encodeMcpSmokeRPC(response), shouldExit
}

func encodeMcpSmokeRPC(message map[string]any) string {
	encoded, err := json.Marshal(message)
	if err != nil {
		return `{"jsonrpc":"2.0","id":null,"error":{"code":-32603,"message":"encode failed"}}`
	}
	return string(encoded)
}

func formatMcpContentLengthFrame(message string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len([]byte(message)), message)
}

func readMcpContentLengthMessage(reader *bufio.Reader) (string, error) {
	var contentLength int
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		line = strings.TrimRight(line, "\r\n")
		if strings.TrimSpace(line) == "" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "Content-Length") {
			continue
		}
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || parsed <= 0 || parsed > 1024*1024 {
			return "", fmt.Errorf("invalid Content-Length value: %q", value)
		}
		contentLength = parsed
	}
	if contentLength <= 0 {
		return "", fmt.Errorf("missing Content-Length header")
	}
	buffer := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, buffer); err != nil {
		return "", err
	}
	return string(buffer), nil
}

func isWailsUISmokeEnabled() bool {
	return os.Getenv(wailsSmokeEnabledEnv) == "1"
}

func windowsOptionsFromEnv(smokeMode bool) application.WindowsOptions {
	if !smokeMode {
		return application.WindowsOptions{}
	}
	portText := strings.TrimSpace(os.Getenv(wailsSmokePortEnv))
	if portText == "" {
		return application.WindowsOptions{}
	}

	port, err := strconv.Atoi(portText)
	if err != nil || port < 1024 || port > 65535 {
		log.Fatalf("%s must be a local TCP port from 1024 to 65535", wailsSmokePortEnv)
	}

	return application.WindowsOptions{
		AdditionalBrowserArgs: []string{
			fmt.Sprintf("--remote-debugging-port=%d", port),
			"--remote-debugging-address=127.0.0.1",
			"--remote-allow-origins=http://127.0.0.1",
		},
	}
}
