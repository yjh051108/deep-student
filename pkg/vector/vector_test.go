package vector

import (
	"math/rand"
	"testing"
)

func TestCosineSearch(t *testing.T) {
	ix := New()
	dim := 64
	for i := 0; i < 100; i++ {
		v := make([]float32, dim)
		for j := range v {
			v[j] = rand.Float32()
		}
		ix.Add(Vector{ID: randID(i), Dim: dim, Embed: v})
	}
	q := make([]float32, dim)
	for j := range q {
		q[j] = rand.Float32()
	}
	hits, err := ix.Search(q, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 5 {
		t.Fatalf("expected 5 hits, got %d", len(hits))
	}
	for i := 1; i < len(hits); i++ {
		if hits[i].Score > hits[i-1].Score {
			t.Fatal("hits not sorted desc")
		}
	}
}

func TestEncodeDecode(t *testing.T) {
	v := []float32{1, 2.5, -3, 0.001}
	b := Encode(v)
	back := Decode(b)
	if len(back) != len(v) {
		t.Fatal("len mismatch")
	}
	for i := range v {
		if back[i] != v[i] {
			t.Fatalf("idx=%d %f vs %f", i, v[i], back[i])
		}
	}
}

func randID(i int) string {
	return "v" + string(rune('a'+(i%26)))
}
