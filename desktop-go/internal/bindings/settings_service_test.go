package bindings

import (
	"deep-student-go/internal/app"
	"deep-student-go/internal/vfs"
	"testing"
)

func TestSettingsServiceEnhancedStatisticsIncludesVfsImages(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DEEP_STUDENT_DATA_DIR", dir)

	application, err := app.New()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.Vfs.UploadFile(vfs.UploadFileInput{
		Name:          "image.png",
		MimeType:      "image/png",
		FileType:      strPtr("image"),
		Base64Content: "aW1hZ2U=",
	}); err != nil {
		t.Fatal(err)
	}

	service := NewSettingsService(application)
	stats, err := service.GetEnhancedStatistics()
	if err != nil {
		t.Fatal(err)
	}
	if stats.ImageStats.TotalFiles != 1 {
		t.Fatalf("expected one VFS image, got %+v", stats.ImageStats)
	}
	if stats.ImageStats.TotalSizeBytes != 5 {
		t.Fatalf("expected image size 5, got %+v", stats.ImageStats)
	}
	if stats.BasicStats.TotalMistakes != 0 {
		t.Fatalf("expected empty basic stats, got %+v", stats.BasicStats)
	}
}

func strPtr(value string) *string {
	return &value
}
