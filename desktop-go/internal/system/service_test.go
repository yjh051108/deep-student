package system

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServicePathsStayInsideDataDir(t *testing.T) {
	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	logDir, err := service.LogDir("frontend")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(logDir, service.AppDataDir()) {
		t.Fatalf("expected %q to stay inside %q", logDir, service.AppDataDir())
	}

	debugDir, err := service.EnsureDebugLogDir()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(debugDir) != "debug-logs" {
		t.Fatalf("expected debug-logs dir, got %q", debugDir)
	}
}

func TestServiceRejectsUnsafeLogType(t *testing.T) {
	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.LogDir("../outside"); err == nil {
		t.Fatal("expected unsafe log type to be rejected")
	}
}

func TestServiceWritesFrontendLog(t *testing.T) {
	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if err := service.ReportFrontendLog(FrontendLogPayload{
		Level:   "warning",
		Message: "render failed",
		Kind:    "UI_ERROR",
	}); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(service.AppDataDir(), "logs", "frontend", "frontend-go.jsonl")
	bytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(bytes)
	if !strings.Contains(text, `"level":"WARN"`) || !strings.Contains(text, "render failed") {
		t.Fatalf("unexpected log content: %s", text)
	}
}

func TestServiceSavesWebviewSettings(t *testing.T) {
	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.SaveWebviewSettings(map[string]any{
		"deep-student-theme": "dark",
		"sidebar_collapsed":  "true",
		"font_scale":         1.2,
		"feature_flags": map[string]any{
			"denseMode": true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatal("expected success result")
	}
	if result.Size <= 0 {
		t.Fatalf("expected positive size, got %d", result.Size)
	}
	if !strings.HasPrefix(result.Path, service.AppDataDir()) {
		t.Fatalf("expected %q to stay inside %q", result.Path, service.AppDataDir())
	}

	bytes, err := os.ReadFile(filepath.Join(service.AppDataDir(), "webview_settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var stored map[string]any
	if err := json.Unmarshal(bytes, &stored); err != nil {
		t.Fatal(err)
	}
	if stored["deep-student-theme"] != "dark" || stored["sidebar_collapsed"] != "true" {
		t.Fatalf("unexpected stored settings: %#v", stored)
	}
	if stored["font_scale"] != 1.2 {
		t.Fatalf("expected numeric setting to be preserved, got %#v", stored["font_scale"])
	}
	flags, ok := stored["feature_flags"].(map[string]any)
	if !ok || flags["denseMode"] != true {
		t.Fatalf("expected nested setting to be preserved, got %#v", stored["feature_flags"])
	}
}
