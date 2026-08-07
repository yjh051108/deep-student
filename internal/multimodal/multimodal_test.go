package multimodal

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/store/blob"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// mockProv 提供嵌入能力。
type mockProv struct{}

func (m *mockProv) Name() string { return "openai" }

func (m *mockProv) Chat(context.Context, llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: "ok"}, nil
}

func (m *mockProv) Stream(context.Context, llm.ChatRequest) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk, 1)
	ch <- llm.Chunk{Delta: "ok", Done: true}
	close(ch)
	return ch, nil
}

func (m *mockProv) Embed(_ context.Context, req llm.EmbedRequest) (*llm.EmbedResponse, error) {
	out := make([][]float32, len(req.Input))
	for i := range out {
		// 确定性伪向量：按文本内容哈希
		v := make([]float32, 8)
		sum := 0
		for _, c := range req.Input[i] {
			sum += int(c)
		}
		for j := range v {
			v[j] = float32((sum + j*7) % 100) / 100
		}
		out[i] = v
	}
	return &llm.EmbedResponse{Embeddings: out}, nil
}

// newSvc 构造测试服务。
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
	reg := llm.NewRegistry()
	reg.Register(&mockProv{})
	return New(st, reg, fs)
}

func TestIndexAndKeywordSearch(t *testing.T) {
	s := newSvc(t)
	ctx := context.Background()
	n, err := s.IndexResource(ctx, "vfs://note/n1", "这是关于机器学习的笔记，包含神经网络与反向传播。")
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("chunks=%d", n)
	}
	// 关键词检索
	results, err := s.Search(ctx, "神经网络", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("no keyword results")
	}
	if results[0].URI != "vfs://note/n1" {
		t.Fatalf("result=%+v", results[0])
	}
}

func TestDeleteAndStats(t *testing.T) {
	s := newSvc(t)
	ctx := context.Background()
	s.IndexResource(ctx, "vfs://note/n1", "内容一")
	s.IndexResource(ctx, "vfs://note/n2", "内容二")
	st, err := s.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if st.TotalUnits < 2 || st.IndexedURIs != 2 {
		t.Fatalf("stats=%+v", st)
	}
	if err := s.Delete("vfs://note/n1"); err != nil {
		t.Fatal(err)
	}
	st2, _ := s.Stats()
	if st2.IndexedURIs != 1 {
		t.Fatalf("after delete=%+v", st2)
	}
}

func TestChunkText(t *testing.T) {
	chunks := chunkText(strings.Repeat("a", 2500), 800)
	if len(chunks) != 4 {
		t.Fatalf("chunks=%d", len(chunks))
	}
	if chunkText("", 800) != nil {
		t.Fatal("empty should be nil")
	}
}
