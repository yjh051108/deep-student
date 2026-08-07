package mindmap

import (
	"strings"
	"testing"
)

func TestFromOutline(t *testing.T) {
	text := "Root\n  Child1\n    Leaf1\n  Child2\n"
	m := FromOutline("Test", text)
	if m.Root.Topic != "Test" {
		t.Fatal("root")
	}
	if len(m.Root.Children) != 2 {
		t.Fatalf("children=%d", len(m.Root.Children))
	}
	if len(m.Root.Children[0].Children) != 1 {
		t.Fatal("grandchild")
	}
}

func TestToOutline(t *testing.T) {
	m := &Map{Title: "T", Root: &Node{ID: "r", Topic: "Root", Children: []*Node{{ID: "a", Topic: "A"}}}}
	out := m.ToOutline()
	if !strings.Contains(out, "Root") || !strings.Contains(out, "A") {
		t.Fatal("outline missing")
	}
}
