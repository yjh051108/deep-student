package skills

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

func newSvc(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	bs, _ := blob.New(filepath.Join(dir, "b"))
	fs := vfs.NewFS(bs)
	st, err := store.Open(filepath.Join(dir, "x.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return New(fs, st, llm.NewRegistry(), nil)
}

func TestBuiltinSkills(t *testing.T) {
	s := newSvc(t)
	if len(s.Skills()) < 12 {
		t.Fatalf("expected 12 builtin, got %d", len(s.Skills()))
	}
}

func TestRegisterToolAndCall(t *testing.T) {
	s := newSvc(t)
	s.RegisterTool(ToolBinding{
		Name: "echo", Desc: "echo back", Schema: json.RawMessage(`{}`),
		Handler: func(ctx context.Context, args json.RawMessage) (any, error) {
			return string(args), nil
		},
	})
	out, err := s.Tool(context.Background(), "echo", json.RawMessage(`"hi"`))
	if err != nil {
		t.Fatal(err)
	}
	if out.(string) != `"hi"` {
		t.Fatal("echo")
	}
}

func TestLoadBuiltinSkillsDir(t *testing.T) {
	dir := t.TempDir()
	if err := writeSkillFile(filepath.Join(dir, "alpha.md"), "alpha", "alpha skill"); err != nil {
		t.Fatal(err)
	}
	s := newSvc(t)
	if err := s.LoadBuiltinSkillsDir(dir); err != nil {
		t.Fatal(err)
	}
	if s.Skill("alpha") == nil {
		t.Fatal("alpha not loaded")
	}
}

func writeSkillFile(path, name, body string) error {
	content := "---\nname: " + name + "\n---\n" + body
	return writeFile(path, content)
}

func writeFile(path, content string) error {
	return writeFileBytes(path, []byte(content))
}

func writeFileBytes(path string, b []byte) error {
	return osWriteFile(path, b)
}

// 桥接 os.WriteFile
func osWriteFile(path string, b []byte) error {
	return osIOWriteFile(path, b, 0o644)
}
