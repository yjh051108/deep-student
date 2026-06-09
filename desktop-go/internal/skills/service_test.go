package skills

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSkillServiceCRUDInsideAllowedRoot(t *testing.T) {
	tmp := t.TempDir()
	service := NewService(tmp)
	base := filepath.Join(tmp, "skills")

	created, err := service.Create(base, "biology-review", "# Biology\n")
	if err != nil {
		t.Fatalf("create skill: %v", err)
	}
	if filepath.Base(created.Path) != skillFileName {
		t.Fatalf("expected SKILL.md path, got %q", created.Path)
	}

	entries, err := service.ListDirectories(base)
	if err != nil {
		t.Fatalf("list directories: %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "biology-review" {
		t.Fatalf("unexpected entries: %+v", entries)
	}

	read, err := service.ReadFile(created.Path)
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	if read.Content != "# Biology\n" {
		t.Fatalf("unexpected content: %q", read.Content)
	}

	updated, err := service.Update(created.Path, "# Biology updated\n")
	if err != nil {
		t.Fatalf("update skill: %v", err)
	}
	if updated.Content != "# Biology updated\n" {
		t.Fatalf("unexpected updated content: %q", updated.Content)
	}

	if err := service.Delete(filepath.Dir(created.Path)); err != nil {
		t.Fatalf("delete skill: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(created.Path)); !os.IsNotExist(err) {
		t.Fatalf("expected deleted skill directory, stat err=%v", err)
	}
}

func TestSkillServiceRejectsTraversalAndUnsafeIDs(t *testing.T) {
	tmp := t.TempDir()
	service := NewService(tmp)

	if _, err := service.Create(filepath.Join(tmp, "skills"), "../bad", "# Bad\n"); err == nil {
		t.Fatal("expected unsafe skill ID to fail")
	}

	outside := filepath.Join(tmp, "..", "outside", "SKILL.md")
	if _, err := service.ReadFile(outside); err == nil {
		t.Fatal("expected outside path to fail")
	}

	for _, id := range []string{"中文", " NUL", "NUL", "-bad", "bad/id"} {
		if _, err := service.Create(filepath.Join(tmp, "skills"), id, "# Bad\n"); err == nil {
			t.Fatalf("expected unsafe skill ID %q to fail", id)
		}
	}
}

func TestSkillServiceAllowsProjectSkillsUnderCurrentDirectory(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(old); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	service := NewService(filepath.Join(tmp, "data"))
	created, err := service.Create(".skills", "project-skill", "# Project\n")
	if err != nil {
		t.Fatalf("create project skill on %s: %v", runtime.GOOS, err)
	}
	if !filepath.IsAbs(created.Path) {
		t.Fatalf("expected absolute path, got %q", created.Path)
	}
}

func TestSkillServiceRestrictsWriteTargetsToSkillFiles(t *testing.T) {
	tmp := t.TempDir()
	service := NewService(tmp)
	base := filepath.Join(tmp, "skills")

	created, err := service.Create(base, "biology-review", "# Biology\n")
	if err != nil {
		t.Fatalf("create skill: %v", err)
	}

	sidecar := filepath.Join(filepath.Dir(created.Path), "notes.md")
	if err := os.WriteFile(sidecar, []byte("notes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ReadFile(sidecar); err == nil {
		t.Fatal("expected reading non-SKILL.md file to fail")
	}
	if _, err := service.Update(sidecar, "updated"); err == nil {
		t.Fatal("expected updating non-SKILL.md file to fail")
	}
}

func TestSkillServiceRejectsRootDelete(t *testing.T) {
	tmp := t.TempDir()
	service := NewService(tmp)
	base := filepath.Join(tmp, "skills")
	if err := os.MkdirAll(base, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, skillFileName), []byte("# Root\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := service.Delete(base); err == nil {
		t.Fatal("expected deleting the skill root to fail")
	}
	if _, err := os.Stat(base); err != nil {
		t.Fatalf("skill root should remain after failed delete: %v", err)
	}
}

func TestSkillServiceRejectsSymlinkedSkillFile(t *testing.T) {
	tmp := t.TempDir()
	service := NewService(tmp)
	base := filepath.Join(tmp, "skills")
	skillDir := filepath.Join(base, "biology-review")
	if err := os.MkdirAll(skillDir, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(tmp, "outside.md")
	if err := os.WriteFile(outside, []byte("# Outside\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(skillDir, skillFileName)
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable on %s: %v", runtime.GOOS, err)
	}

	if _, err := service.ReadFile(link); err == nil {
		t.Fatal("expected symlinked skill file read to fail")
	}
	if _, err := service.Update(link, "# Updated\n"); err == nil {
		t.Fatal("expected symlinked skill file update to fail")
	}
}

func TestSkillServiceRejectsOversizedContent(t *testing.T) {
	tmp := t.TempDir()
	service := NewService(tmp)
	base := filepath.Join(tmp, "skills")
	oversized := strings.Repeat("x", maxSkillFileBytes+1)

	if _, err := service.Create(base, "too-large", oversized); err == nil {
		t.Fatal("expected oversized create to fail")
	}

	created, err := service.Create(base, "biology-review", "# Biology\n")
	if err != nil {
		t.Fatalf("create skill: %v", err)
	}
	if _, err := service.Update(created.Path, oversized); err == nil {
		t.Fatal("expected oversized update to fail")
	}
}
