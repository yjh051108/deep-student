//go:build windows

package vfs

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestPdfiumIntegrationRendersMinimalPdfWhenConfigured(t *testing.T) {
	if os.Getenv("DEEP_STUDENT_PDFIUM_PATH") == "" && os.Getenv("DEEP_STUDENT_ENABLE_DEV_PDFIUM_PATHS") != "1" {
		t.Skip("set DEEP_STUDENT_PDFIUM_PATH or DEEP_STUDENT_ENABLE_DEV_PDFIUM_PATHS=1 to run PDFium integration smoke")
	}
	rendered, err := SmokePDFiumRasterPreview()
	if err != nil {
		t.Fatalf("SmokePDFiumRasterPreview() error = %v", err)
	}
	if rendered.TotalPages != 1 || rendered.RenderedPages != 1 {
		t.Fatalf("expected one rendered page, got %+v", rendered)
	}
	if rendered.FirstPageWidth <= 0 || rendered.FirstPageHeight <= 0 || rendered.FirstPageBytes <= 0 {
		t.Fatalf("expected non-empty PNG page, got %+v", rendered)
	}
}

func TestPdfiumCandidatePathsIncludePackagedWindowsLocations(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "app", "Deep Student.exe")
	candidates, failures := pdfiumCandidatePathsFor("", executable, "", false)
	if len(failures) != 0 {
		t.Fatalf("unexpected failures: %v", failures)
	}

	wantAdjacent := filepath.Clean(filepath.Join(filepath.Dir(executable), "pdfium.dll"))
	wantResource := filepath.Clean(filepath.Join(filepath.Dir(executable), "..", "Resources", "pdfium.dll"))
	for _, want := range []string{wantAdjacent, wantResource} {
		if !slices.Contains(candidates, want) {
			t.Fatalf("expected candidate %q in %v", want, candidates)
		}
	}
}

func TestPdfiumCandidatePathsRejectRelativeConfiguredPath(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "app", "Deep Student.exe")
	candidates, failures := pdfiumCandidatePathsFor("pdfium.dll", executable, "", false)
	if len(candidates) != 2 {
		t.Fatalf("expected packaged candidates to remain after invalid configured path, got %v", candidates)
	}
	if len(failures) != 1 || !strings.Contains(failures[0], "path must be absolute") {
		t.Fatalf("expected relative configured path failure, got %v", failures)
	}
}

func TestPdfiumCandidatePathsDevRepoIsExplicitlyGated(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "desktop-go"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "desktop-go", "go.mod"), []byte("module smoke\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(t.TempDir(), "app", "Deep Student.exe")

	disabled, failures := pdfiumCandidatePathsFor("", executable, filepath.Join(root, "desktop-go"), false)
	if len(failures) != 0 {
		t.Fatalf("unexpected disabled failures: %v", failures)
	}
	devRootCandidate := filepath.Clean(filepath.Join(root, "pdfium.dll"))
	if slices.Contains(disabled, devRootCandidate) {
		t.Fatalf("dev repo candidate should be gated off, got %v", disabled)
	}

	enabled, failures := pdfiumCandidatePathsFor("", executable, filepath.Join(root, "desktop-go"), true)
	if len(failures) != 0 {
		t.Fatalf("unexpected enabled failures: %v", failures)
	}
	if !slices.Contains(enabled, devRootCandidate) {
		t.Fatalf("expected gated dev repo candidate %q in %v", devRootCandidate, enabled)
	}
}
