package vault

import "testing"

func TestExtractLinks(t *testing.T) {
	body := "看 [[笔记A]] 和 [[笔记B|别名]] 还有 [[笔记C#^block]] 以及 [[笔记A]] 重复"
	links := ExtractLinks(body)
	if len(links) != 3 {
		t.Fatalf("links=%v", links)
	}
	if links[0] != "笔记A" || links[1] != "笔记B" || links[2] != "笔记C" {
		t.Fatalf("links=%v", links)
	}
}

func TestResolveLinks(t *testing.T) {
	entries := []ScannedEntry{
		{URI: "vfs://note/a", ID: "a", Title: "Alpha"},
		{URI: "vfs://note/b", ID: "b", Title: "Beta"},
	}
	contents := map[string]string{
		"vfs://note/a": "参见 [[Beta]] 和 [[a]]",
		"vfs://note/b": "无链接",
	}
	links := ResolveLinks(entries, contents)
	if len(links) != 2 {
		t.Fatalf("links=%+v", links)
	}
	// 第一条：按标题命中 Beta
	if links[0].Target != "Beta" || links[0].TargetURI != "vfs://note/b" || links[0].TargetTitle != "Beta" {
		t.Fatalf("link0=%+v", links[0])
	}
	// 第二条：按 ds_id 命中 a
	if links[1].Target != "a" || links[1].TargetURI != "vfs://note/a" || links[1].TargetTitle != "Alpha" {
		t.Fatalf("link1=%+v", links[1])
	}
}

func TestNormalizeLinkTarget(t *testing.T) {
	cases := map[string]string{
		"笔记A":        "笔记A",
		"笔记B|别名":     "笔记B",
		"笔记C#^block": "笔记C",
		" 空格 ":      "空格",
	}
	for in, want := range cases {
		if got := normalizeLinkTarget(in); got != want {
			t.Fatalf("normalize(%q)=%q want %q", in, got, want)
		}
	}
}
