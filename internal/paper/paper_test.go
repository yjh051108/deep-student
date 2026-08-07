package paper

import (
	"strings"
	"testing"
)

func TestCiteBibtex(t *testing.T) {
	s := Source{Title: "A Study", Authors: []string{"Alice", "Bob"}, Year: 2020, DOI: "10.1/abc"}
	c := Cite(s, "bibtex")
	if !strings.Contains(c, "@article{") || !strings.Contains(c, "A Study") {
		t.Fatalf("cite=%q", c)
	}
}

func TestCiteAPA(t *testing.T) {
	s := Source{Title: "T", Authors: []string{"A"}, Year: 2021, DOI: "x"}
	c := Cite(s, "apa")
	if !strings.Contains(c, "(2021)") || !strings.Contains(c, "https://doi.org/x") {
		t.Fatalf("cite=%q", c)
	}
}

func TestCiteGB7714(t *testing.T) {
	s := Source{Title: "T", Authors: []string{"A"}, Year: 2022, DOI: "x", Venue: "J"}
	c := Cite(s, "gb7714")
	if !strings.Contains(c, "[J]") || !strings.Contains(c, "DOI:x") {
		t.Fatalf("cite=%q", c)
	}
}

func TestCiteDefault(t *testing.T) {
	s := Source{Title: "T"}
	if Cite(s, "unknown") != "T" {
		t.Fatal("default cite")
	}
}

func TestParseArXiv(t *testing.T) {
	body := `<?xml version="1.0"?>
<feed>
  <entry>
    <id>http://arxiv.org/abs/2401.01234</id>
    <title>Title One</title>
  </entry>
  <entry>
    <id>http://arxiv.org/abs/2401.05678</id>
    <title>Title Two</title>
  </entry>
</feed>`
	xs := parseArXiv([]byte(body))
	if len(xs) != 2 {
		t.Fatalf("got %d", len(xs))
	}
	if xs[0].Title != "Title One" {
		t.Fatalf("title=%q", xs[0].Title)
	}
	if !strings.HasSuffix(xs[0].PDFURL, ".pdf") {
		t.Fatalf("pdf=%q", xs[0].PDFURL)
	}
}

func TestInvertAbstract(t *testing.T) {
	idx := map[string][]int{
		"hello": {0, 3},
		"world": {1, 4},
		"again": {2},
	}
	s := invertAbstract(idx)
	if s != "hello world again hello world" {
		t.Fatalf("got=%q", s)
	}
}

func TestInvertAbstractEmpty(t *testing.T) {
	if invertAbstract(nil) != "" {
		t.Fatal("expected empty")
	}
}

func TestExtractTag(t *testing.T) {
	if extractTag("<a><title>Hello</title></a>", "title") != "Hello" {
		t.Fatal("extract tag")
	}
	if extractTag("<a></a>", "title") != "" {
		t.Fatal("missing tag")
	}
}
