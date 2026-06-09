package notes

import "testing"

func TestPrefsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := service.SetPref("notes_tabs", `{"openTabs":["n1"],"activeId":"n1"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("SetPref returned false")
	}

	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	value, found := reloaded.GetPref("notes_tabs")
	if !found {
		t.Fatal("expected notes_tabs to be present")
	}
	if value != `{"openTabs":["n1"],"activeId":"n1"}` {
		t.Fatalf("unexpected value: %q", value)
	}
}

func TestEmptyPrefKeyRejected(t *testing.T) {
	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	ok, err := service.SetPref("", "value")
	if err == nil {
		t.Fatal("expected empty key error")
	}
	if ok {
		t.Fatal("SetPref returned true for empty key")
	}
}

func TestAssetRoundTrip(t *testing.T) {
	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	ref, err := service.SaveAsset("_global", "note/1", "aGVsbG8=", "png")
	if err != nil {
		t.Fatal(err)
	}
	if ref.RelativePath == "" || ref.AbsolutePath == "" {
		t.Fatalf("unexpected asset ref: %+v", ref)
	}
	if ref.RelativePath[:len("notes_assets/")] != "notes_assets/" {
		t.Fatalf("unexpected relative path: %s", ref.RelativePath)
	}

	assets, err := service.ListAssets("_global", "note/1")
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 1 || assets[0].RelativePath != ref.RelativePath {
		t.Fatalf("unexpected assets: %+v", assets)
	}

	dataURL, err := service.GetImageAsBase64(ref.RelativePath)
	if err != nil {
		t.Fatal(err)
	}
	if dataURL != "data:image/png;base64,aGVsbG8=" {
		t.Fatalf("unexpected data URL: %s", dataURL)
	}

	deleted, err := service.DeleteAsset(ref.RelativePath)
	if err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected delete to return true")
	}
	deleted, err = service.DeleteAsset(ref.RelativePath)
	if err != nil {
		t.Fatal(err)
	}
	if deleted {
		t.Fatal("expected second delete to return false")
	}
}

func TestAssetMaintenanceAndStats(t *testing.T) {
	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	kept, err := service.SaveAsset("_global", "note1", "a2VwdA==", "png")
	if err != nil {
		t.Fatal(err)
	}
	orphan, err := service.SaveAsset("_global", "note2", "b3JwaGFu", "jpg")
	if err != nil {
		t.Fatal(err)
	}
	if count, err := service.AssetsIndexScan("note1"); err != nil || count != 1 {
		t.Fatalf("AssetsIndexScan(note1) = %d, %v", count, err)
	}

	orphans, err := service.ScanOrphanAssets([]string{kept.RelativePath})
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 1 || orphans[0] != orphan.RelativePath {
		t.Fatalf("unexpected orphans: %+v", orphans)
	}

	stats, err := service.DBStats(3)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalNotes != 3 || stats.TotalAssets != 2 || stats.DBPath == "" {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if ok, err := service.DBVacuum(); err != nil || !ok {
		t.Fatalf("DBVacuum() = %v, %v", ok, err)
	}

	deleted, err := service.BulkDeleteAssets(orphans)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("BulkDeleteAssets() = %d, want 1", deleted)
	}
	remaining, err := service.ScanOrphanAssets([]string{kept.RelativePath})
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected no remaining orphans, got %+v", remaining)
	}
}

func TestAssetPathCannotEscapeDataDir(t *testing.T) {
	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.ResolveAssetPath("../outside.png"); err == nil {
		t.Fatal("expected escaping relative path to be rejected")
	}
	if _, err := service.SaveAsset("_global", "", "aGVsbG8=", "png"); err == nil {
		t.Fatal("expected empty noteId to be rejected")
	}
}
