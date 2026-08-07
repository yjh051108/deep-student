// Package paper 论文检索：arXiv / OpenAlex、批量下载、去重、引用格式、DOI 解析。
package paper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Source 论文元数据。
type Source struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Authors   []string  `json:"authors"`
	Abstract  string    `json:"abstract"`
	Year      int       `json:"year"`
	Venue     string    `json:"venue"`
	URL       string    `json:"url"`
	PDFURL    string    `json:"pdf_url"`
	DOI       string    `json:"doi"`
	Source    string    `json:"source"` // arxiv | openalex
	CreatedAt time.Time `json:"created_at"`
}

// Service 论文服务。
type Service struct {
	vfs   *vfs.FS
	store *store.Store
	http  *http.Client
	mu    sync.Mutex
	seen  map[string]string // sha256 -> id
}

// New 创建 Service。
func New(fs *vfs.FS, st *store.Store) *Service {
	return &Service{vfs: fs, store: st, http: &http.Client{Timeout: 60 * time.Second}, seen: map[string]string{}}
}

// SearchArXiv 调 arXiv API。
func (s *Service) SearchArXiv(ctx context.Context, q string, max int) ([]Source, error) {
	url := "http://export.arxiv.org/api/query?search_query=" + urlQueryEscape(q) + "&max_results=" + fmt.Sprintf("%d", max)
	resp, err := s.http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return parseArXiv(body), nil
}

// SearchOpenAlex 调 OpenAlex API。
func (s *Service) SearchOpenAlex(ctx context.Context, q string, max int) ([]Source, error) {
	url := "https://api.openalex.org/works?search=" + urlQueryEscape(q) + "&per-page=" + fmt.Sprintf("%d", max)
	resp, err := s.http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var raw struct {
		Results []struct {
			ID              string `json:"id"`
			DisplayName     string `json:"display_name"`
			Title           string `json:"title"`
			PublicationYear int    `json:"publication_year"`
			Doi             string `json:"doi"`
			Authorships     []struct {
				Author struct {
					DisplayName string `json:"display_name"`
				} `json:"author"`
			} `json:"authorships"`
			PrimaryLocation struct {
				Source *struct {
					DisplayName string `json:"display_name"`
				} `json:"source"`
				PDFURL string `json:"pdf_url"`
			} `json:"primary_location"`
			AbstractInvertedIndex map[string][]int `json:"abstract_inverted_index"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]Source, 0, len(raw.Results))
	for _, r := range raw.Results {
		out = append(out, Source{
			ID: r.ID, Title: r.Title, Authors: extractAuthors(r.Authorships),
			Year: r.PublicationYear, DOI: strings.TrimPrefix(r.Doi, "https://doi.org/"),
			PDFURL: r.PrimaryLocation.PDFURL, Source: "openalex",
			Abstract:  invertAbstract(r.AbstractInvertedIndex),
			CreatedAt: time.Now(),
		})
	}
	return out, nil
}

// Download 下载 PDF 并入库（带 SHA256 去重）。
func (s *Service) Download(ctx context.Context, src Source) (string, error) {
	if src.PDFURL == "" {
		return "", fmt.Errorf("paper: no pdf url")
	}
	resp, err := s.http.Get(src.PDFURL)
	if err != nil {
		// fallback: unpaywall
		if src.DOI != "" {
			alt := "https://unpaywall.org/" + src.DOI + "?email=anonymous@example.com"
			resp, err = s.http.Get(alt)
		}
		if err != nil {
			return "", err
		}
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	sum := sha256.Sum256(data)
	ref := hex.EncodeToString(sum[:])
	s.mu.Lock()
	if id, ok := s.seen[ref]; ok {
		s.mu.Unlock()
		return id, nil
	}
	s.mu.Unlock()

	uri := fmt.Sprintf("vfs://paper/%s", uuid.NewString())
	meta := map[string]string{
		"title": src.Title, "tags": "paper," + src.Source,
		"doi": src.DOI, "year": fmt.Sprintf("%d", src.Year),
	}
	entry, err := s.vfs.Put(uri, data, meta)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.seen[entry.BlobRef] = uri
	s.mu.Unlock()
	return uri, nil
}

// Cite 输出引用格式。
func Cite(src Source, format string) string {
	authors := strings.Join(src.Authors, ", ")
	switch format {
	case "bibtex":
		key := strings.ReplaceAll(strings.ToLower(strings.Split(authors, ",")[0]), " ", "") + fmt.Sprintf("%d", src.Year)
		return fmt.Sprintf(`@article{%s,
  title  = {%s},
  author = {%s},
  year   = {%d},
  doi    = {%s},
}`, key, src.Title, authors, src.Year, src.DOI)
	case "apa":
		return fmt.Sprintf("%s (%d). %s. %s. https://doi.org/%s", authors, src.Year, src.Title, src.Venue, src.DOI)
	case "gb7714":
		return fmt.Sprintf("%s. %s[J]. %s, %d. DOI:%s.", authors, src.Title, src.Venue, src.Year, src.DOI)
	default:
		return src.Title
	}
}

// ResolveDOI DOI → 开放链接。
func (s *Service) ResolveDOI(ctx context.Context, doi string) (string, error) {
	resp, err := s.http.Get("https://api.openalex.org/works/doi:" + doi)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("paper: resolve doi %d", resp.StatusCode)
	}
	var raw struct {
		PrimaryLocation struct {
			PDFURL string `json:"pdf_url"`
		} `json:"primary_location"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return "", err
	}
	if raw.PrimaryLocation.PDFURL != "" {
		return raw.PrimaryLocation.PDFURL, nil
	}
	return "https://doi.org/" + doi, nil
}

func extractAuthors(a []struct {
	Author struct {
		DisplayName string `json:"display_name"`
	} `json:"author"`
}) []string {
	out := make([]string, 0, len(a))
	for _, x := range a {
		out = append(out, x.Author.DisplayName)
	}
	return out
}

func invertAbstract(idx map[string][]int) string {
	if len(idx) == 0 {
		return ""
	}
	// 简易还原：按出现位置最大下标重建
	maxPos := 0
	for _, poss := range idx {
		for _, p := range poss {
			if p > maxPos {
				maxPos = p
			}
		}
	}
	arr := make([]string, maxPos+1)
	for word, poss := range idx {
		for _, p := range poss {
			arr[p] = word
		}
	}
	return strings.Join(arr, " ")
}

func parseArXiv(body []byte) []Source {
	// 极简解析：仅提取 <entry> 块中的 <title> 与 <id>
	var out []Source
	xml := string(body)
	for {
		i := strings.Index(xml, "<entry>")
		if i < 0 {
			break
		}
		j := strings.Index(xml[i:], "</entry>")
		if j < 0 {
			break
		}
		entry := xml[i : i+j+len("</entry>")]
		title := extractTag(entry, "title")
		id := extractTag(entry, "id")
		src := Source{Title: title, URL: id, PDFURL: id, Source: "arxiv", CreatedAt: time.Now()}
		if !strings.HasSuffix(id, ".pdf") {
			src.PDFURL = strings.ReplaceAll(id, "abs", "pdf") + ".pdf"
		}
		out = append(out, src)
		xml = xml[i+j+len("</entry>"):]
	}
	return out
}

func extractTag(s, tag string) string {
	start := "<" + tag + ">"
	end := "</" + tag + ">"
	i := strings.Index(s, start)
	if i < 0 {
		return ""
	}
	i += len(start)
	j := strings.Index(s[i:], end)
	if j < 0 {
		return ""
	}
	return strings.TrimSpace(s[i : i+j])
}

func urlQueryEscape(s string) string {
	r := strings.NewReplacer(" ", "+", "#", "%23", "&", "%26", "?", "%3F", "/", "%2F")
	return r.Replace(s)
}
