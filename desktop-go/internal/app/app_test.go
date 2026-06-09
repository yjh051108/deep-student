package app

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	"deep-student-go/internal/dstu"
	"deep-student-go/internal/runtime"
	"deep-student-go/internal/vfs"
)

func TestNewWiresVfsEventsToEventBus(t *testing.T) {
	t.Setenv("DEEP_STUDENT_DATA_DIR", t.TempDir())

	application, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	seen := []runtime.Event{}
	stop := application.Events.Listen("*", func(event runtime.Event) {
		seen = append(seen, event)
	})
	defer stop()

	uploaded, err := application.Vfs.UploadFile(vfs.UploadFileInput{
		Name:          "eventbus.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n")),
	})
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}
	if err := application.Vfs.StartPdfProcessing(uploaded.SourceID, nil); err != nil {
		t.Fatalf("StartPdfProcessing() error = %v", err)
	}
	if _, err := application.Vfs.ReindexResource(uploaded.SourceID); err != nil {
		t.Fatalf("ReindexResource() error = %v", err)
	}

	sawMedia := false
	sawLegacyPdf := false
	sawIndex := false
	for _, event := range seen {
		switch event.Name {
		case "media-processing-progress":
			sawMedia = true
		case "pdf-processing-progress":
			sawLegacyPdf = true
		case "vfs-index-progress":
			sawIndex = true
		}
	}
	if !sawMedia || !sawLegacyPdf || !sawIndex {
		t.Fatalf("expected VFS events to reach app event bus, saw %+v", seen)
	}
}

func TestNewWiresDstuTextbookImportProgressToEventBus(t *testing.T) {
	t.Setenv("DEEP_STUDENT_DATA_DIR", t.TempDir())

	application, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	seen := []runtime.Event{}
	stop := application.Events.Listen("*", func(event runtime.Event) {
		seen = append(seen, event)
	})
	defer stop()

	sourcePath := filepath.Join(t.TempDir(), "eventbus-textbook.pdf")
	if err := os.WriteFile(sourcePath, []byte("%PDF-1.7\napp eventbus textbook"), 0o600); err != nil {
		t.Fatalf("WriteFile(textbook) error = %v", err)
	}
	records, err := application.Dstu.AddTextbooks(dstu.AddTextbooksRequest{Sources: []string{sourcePath}})
	if err != nil {
		t.Fatalf("AddTextbooks() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("AddTextbooks() records = %+v", records)
	}

	sawDone := false
	for _, event := range seen {
		if event.Name != "textbook-import-progress" {
			continue
		}
		payload := eventPayloadMap(t, event.Payload)
		if payload["file_name"] != "eventbus-textbook.pdf" {
			t.Fatalf("unexpected textbook import payload: %+v", payload)
		}
		if payload["stage"] == "done" {
			sawDone = true
			if payload["textbook_id"] != records[0].ID || payload["resource_id"] != records[0].ResourceID {
				t.Fatalf("done payload missing record identity: payload=%+v record=%+v", payload, records[0])
			}
		}
	}
	if !sawDone {
		t.Fatalf("expected textbook-import-progress done event on app event bus, saw %+v", seen)
	}
}

func TestLegacyDataDirCandidatesRespectExplicitDataDir(t *testing.T) {
	explicit := t.TempDir()
	t.Setenv("DEEP_STUDENT_DATA_DIR", explicit)
	if got := LegacyDataDirCandidates(explicit); len(got) != 0 {
		t.Fatalf("explicit data dir should isolate legacy search, got %#v", got)
	}
}

func TestLegacyDataDirCandidatesIncludeOldTauriRoots(t *testing.T) {
	t.Setenv("DEEP_STUDENT_DATA_DIR", "")
	current := filepath.Join(t.TempDir(), "Deep Student")

	switch goruntime.GOOS {
	case "windows":
		t.Setenv("APPDATA", filepath.Join(t.TempDir(), "roaming"))
		t.Setenv("LOCALAPPDATA", filepath.Join(t.TempDir(), "local"))
	case "darwin":
		t.Setenv("HOME", t.TempDir())
	default:
		t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "xdg"))
		t.Setenv("HOME", t.TempDir())
	}

	roots := LegacyDataDirCandidates(current)
	joined := strings.Join(roots, "\n")
	if !strings.Contains(joined, "com.deepstudent.app") {
		t.Fatalf("expected old Tauri identifier root in %#v", roots)
	}
	if !strings.Contains(joined, "DeepStudent") {
		t.Fatalf("expected old writable fallback root in %#v", roots)
	}
}

func eventPayloadMap(t *testing.T, payload any) map[string]any {
	t.Helper()
	bytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal(payload) error = %v", err)
	}
	out := map[string]any{}
	if err := json.Unmarshal(bytes, &out); err != nil {
		t.Fatalf("Unmarshal(payload) error = %v", err)
	}
	return out
}
