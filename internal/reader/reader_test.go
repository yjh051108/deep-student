package reader

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fumiama/go-docx"
	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// readFile 读整个文件为 []byte；测试专用。
func readFile(t *testing.T, path string) ([]byte, error) {
	t.Helper()
	return os.ReadFile(path)
}

func newSvc(t *testing.T, reg *llm.Registry) *Service {
	t.Helper()
	dir := t.TempDir()
	bs, _ := blob.New(filepath.Join(dir, "b"))
	fs := vfs.NewFS(bs)
	return New(fs, reg)
}

type fakeProv struct{ out string }

func (f *fakeProv) Name() string { return "openai" }
func (f *fakeProv) Chat(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: f.out}, nil
}
func (f *fakeProv) Stream(_ context.Context, _ llm.ChatRequest) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk, 1)
	ch <- llm.Chunk{Delta: f.out, Done: true}
	close(ch)
	return ch, nil
}
func (f *fakeProv) Embed(_ context.Context, _ llm.EmbedRequest) (*llm.EmbedResponse, error) {
	return &llm.EmbedResponse{Embeddings: [][]float32{{0}}}, nil
}

func TestOpenText(t *testing.T) {
	reg := llm.NewRegistry()
	s := newSvc(t, reg)
	uri := "vfs://note/x.md"
	if _, err := s.vfs.Put(uri, []byte("hello world"), map[string]string{"title": "x"}); err != nil {
		t.Fatal(err)
	}
	d, err := s.Open(uri)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Pages) != 1 || d.Pages[0].Content != "hello world" {
		t.Fatalf("pages=%+v", d.Pages)
	}
}

// makeOnePagePDF 用 pdfcpu 自己生成一个单页 PDF，确保能被 pdfcpu 解析回读。
// 内容是一个测试网格（CreateTestPageContent），不保证有可读文本，但能验证
// "整页流程": ReadContext → PageCount → ExtractPageContent 不出错。
func makeOnePagePDF(t *testing.T) []byte {
	t.Helper()
	xRefTable, err := pdfcpu.CreateDemoXRef()
	if err != nil {
		t.Fatalf("CreateDemoXRef: %v", err)
	}
	rootDict, err := xRefTable.Catalog()
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	mediaBox := types.RectForFormat("A4")
	p := model.Page{MediaBox: mediaBox, Fm: model.FontMap{}, Buf: new(bytes.Buffer)}
	if err := pdfcpu.AddPageTreeWithSamplePage(xRefTable, rootDict, p); err != nil {
		t.Fatalf("AddPageTreeWithSamplePage: %v", err)
	}
	var buf bytes.Buffer
	if err := api.Create(&bytes.Reader{}, &buf, &buf, model.NewDefaultConfiguration()); err != nil {
		// fallback: 用 CreatePDFFile 写临时文件再读
		tmp := filepath.Join(t.TempDir(), "t.pdf")
		if err2 := api.CreatePDFFile(xRefTable, tmp, model.NewDefaultConfiguration()); err2 != nil {
			t.Fatalf("CreatePDFFile: %v / Create: %v", err2, err)
		}
		data, _ := readFile(t, tmp)
		return data
	}
	return buf.Bytes()
}

func TestOpenPDF(t *testing.T) {
	reg := llm.NewRegistry()
	s := newSvc(t, reg)
	uri := "vfs://note/x.pdf"
	pdf := makeOnePagePDF(t)
	if _, err := s.vfs.Put(uri, pdf, map[string]string{"title": "x"}); err != nil {
		t.Fatal(err)
	}
	d, err := s.Open(uri)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if len(d.Pages) != 1 {
		t.Fatalf("pages=%d want 1, content[0]=%q", len(d.Pages), d.Pages[0].Content)
	}
}

// makeMultiPagePDF 拼 3 页合法 PDF：分别生成 1 页 PDF，再 MergeRaw 合并。
func makeMultiPagePDF(t *testing.T) []byte {
	t.Helper()
	mediaBox := types.RectForFormat("A4")
	pdfs := make([][]byte, 0, 3)
	for i := 0; i < 3; i++ {
		xRefTable, err := pdfcpu.CreateDemoXRef()
		if err != nil {
			t.Fatalf("CreateDemoXRef[%d]: %v", i, err)
		}
		rootDict, err := xRefTable.Catalog()
		if err != nil {
			t.Fatalf("Catalog[%d]: %v", i, err)
		}
		p := model.Page{MediaBox: mediaBox, Fm: model.FontMap{}, Buf: new(bytes.Buffer)}
		if err := pdfcpu.AddPageTreeWithSamplePage(xRefTable, rootDict, p); err != nil {
			t.Fatalf("AddPageTreeWithSamplePage[%d]: %v", i, err)
		}
		tmp := filepath.Join(t.TempDir(), "single.pdf")
		if err := api.CreatePDFFile(xRefTable, tmp, model.NewDefaultConfiguration()); err != nil {
			t.Fatalf("CreatePDFFile[%d]: %v", i, err)
		}
		data, _ := readFile(t, tmp)
		pdfs = append(pdfs, data)
	}
	// 把 3 个单页 PDF 装成 ReadSeeker 列表
	readers := make([]io.ReadSeeker, 0, len(pdfs))
	for _, d := range pdfs {
		readers = append(readers, bytes.NewReader(d))
	}
	var out bytes.Buffer
	if err := api.MergeRaw(readers, &out, false, model.NewDefaultConfiguration()); err != nil {
		t.Fatalf("MergeRaw: %v", err)
	}
	return out.Bytes()
}

func TestOpenPDFMultiPage(t *testing.T) {
	reg := llm.NewRegistry()
	s := newSvc(t, reg)
	uri := "vfs://note/multi.pdf"
	pdf := makeMultiPagePDF(t)
	if _, err := s.vfs.Put(uri, pdf, map[string]string{"title": "m"}); err != nil {
		t.Fatal(err)
	}
	d, err := s.Open(uri)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if len(d.Pages) != 3 {
		t.Fatalf("pages=%d want 3", len(d.Pages))
	}
	for i, p := range d.Pages {
		if p.Index != i+1 {
			t.Fatalf("page[%d].Index=%d", i, p.Index)
		}
	}
}

func TestParsePDFErrors(t *testing.T) {
	if _, err := ParsePDF(nil); err == nil {
		t.Fatal("want error for empty")
	}
	if _, err := ParsePDF([]byte("not a pdf")); err == nil {
		t.Fatal("want error for non-pdf")
	}
}

// makeOnePageDOCX 用 fumiama/go-docx 写一段真实 DOCX，验证 ParseDOCX 走库解析。
func makeOnePageDOCX(t *testing.T) []byte {
	t.Helper()
	w := docx.New().WithDefaultTheme()
	p := w.AddParagraph()
	p.AddText("hello docx world")
	var buf bytes.Buffer
	if _, err := w.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	return buf.Bytes()
}

// makeMultiPageDOCX 写带分页符的 DOCX，应切出 2 页。
func makeMultiPageDOCX(t *testing.T) []byte {
	t.Helper()
	w := docx.New().WithDefaultTheme()
	p1 := w.AddParagraph()
	p1.AddText("page one")
	w.AddParagraph().AddPageBreaks()
	p2 := w.AddParagraph()
	p2.AddText("page two")
	var buf bytes.Buffer
	if _, err := w.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	return buf.Bytes()
}

func TestOpenDOCX(t *testing.T) {
	reg := llm.NewRegistry()
	s := newSvc(t, reg)
	uri := "vfs://note/x.docx"
	data := makeOnePageDOCX(t)
	if _, err := s.vfs.Put(uri, data, map[string]string{"title": "x"}); err != nil {
		t.Fatal(err)
	}
	d, err := s.Open(uri)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if len(d.Pages) == 0 {
		t.Fatal("no pages parsed")
	}
	var all string
	for _, p := range d.Pages {
		all += p.Content + "\n"
	}
	if !strings.Contains(all, "hello docx world") {
		t.Fatalf("missing text, got pages=%+v", d.Pages)
	}
}

func TestOpenDOCXMultiPage(t *testing.T) {
	reg := llm.NewRegistry()
	s := newSvc(t, reg)
	uri := "vfs://note/multi.docx"
	data := makeMultiPageDOCX(t)
	if _, err := s.vfs.Put(uri, data, map[string]string{"title": "m"}); err != nil {
		t.Fatal(err)
	}
	d, err := s.Open(uri)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if len(d.Pages) != 2 {
		t.Fatalf("pages=%d want 2, pages=%+v", len(d.Pages), d.Pages)
	}
	if !strings.Contains(d.Pages[0].Content, "page one") {
		t.Fatalf("page0=%q", d.Pages[0].Content)
	}
	if !strings.Contains(d.Pages[1].Content, "page two") {
		t.Fatalf("page1=%q", d.Pages[1].Content)
	}
}

func TestParseDOCXErrors(t *testing.T) {
	if _, err := ParseDOCX(nil); err == nil {
		t.Fatal("want error for empty")
	}
	if _, err := ParseDOCX([]byte("not a zip")); err == nil {
		t.Fatal("want error for non-docx")
	}
}

func TestInjectToChat(t *testing.T) {
	s, _ := newSvc(t, nil), llm.NewRegistry()
	d := &Document{URI: "vfs://x", Title: "T", Pages: []Page{{Index: 1, Content: "a"}, {Index: 2, Content: "b"}, {Index: 3, Content: "c"}}}
	prompt := s.InjectToChat(d, 2, 3, "select")
	if !strings.Contains(prompt, "Selected:") || !strings.Contains(prompt, "Page 2") || !strings.Contains(prompt, "Page 3") {
		t.Fatalf("prompt=%q", prompt)
	}
}

func TestSummarize(t *testing.T) {
	reg := llm.NewRegistry()
	reg.Register(&fakeProv{out: "summary"})
	s := newSvc(t, reg)
	d := &Document{URI: "vfs://x", Title: "T", Pages: []Page{{Index: 1, Content: "a"}}}
	out, err := s.Summarize(context.Background(), d, 1)
	if err != nil {
		t.Fatal(err)
	}
	if out != "summary" {
		t.Fatalf("out=%q", out)
	}
}

func TestLinesAndReadAll(t *testing.T) {
	ls := Lines("a\nb\nc")
	if len(ls) != 3 {
		t.Fatalf("lines=%d", len(ls))
	}
	s, _ := ReadAll(strings.NewReader("hi"))
	if s != "hi" {
		t.Fatalf("readall=%q", s)
	}
}
