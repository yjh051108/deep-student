// Package reader 文档解析：PDF 走 pdfcpu，DOCX 走 fumiama/go-docx。
package reader

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/fumiama/go-docx"
)

// ParseDOCX 解析 .docx（本质是 zip + word/document.xml）。
//
// 走 fumiama/go-docx 拿到 paragraph / table 结构，按 <w:br w:type="page"/>
// 分页符切分 Page。
//
// 解析失败时降级到正则抽取（极少数损坏文件）。
func ParseDOCX(data []byte) ([]Page, error) {
	if len(data) == 0 {
		return nil, errors.New("reader: empty docx")
	}
	// fumiama 的 Parse 接受 io.ReaderAt + size；bytes.Reader 满足。
	doc, err := docx.Parse(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		// zip 解析失败 → 兜底
		paras := fallbackParagraphs(data)
		if len(paras) == 0 {
			return nil, fmt.Errorf("reader: open docx zip: %w", err)
		}
		return splitDOCXByPages(paras), nil
	}
	if doc == nil {
		return nil, errors.New("reader: docx parsed nil")
	}
	paras := extractParagraphs(doc)
	if len(paras) == 0 {
		// 兜底：fumiama 解析不到任何内容时回退到正则（兼容损坏文件）
		if xmlBytes, xerr := readDocxDocumentXML(data); xerr == nil {
			paras = fallbackParagraphs(xmlBytes)
		}
	}
	if len(paras) == 0 {
		return nil, errors.New("reader: docx has no content")
	}
	return splitDOCXByPages(paras), nil
}

// extractParagraphs 从 fumiama 解析后的 Docx 抽取按顺序的段落文本。
// 段落和表格都按 Document.Body.Items 顺序拼接；表格用 markdown 形式输出。
// 分页符（<w:br w:type="page"/>）在文本里以 \x0c (form feed) 形式插入，
// 上层 splitDOCXByPages 据此切页。
func extractParagraphs(d *docx.Docx) []string {
	if d == nil {
		return nil
	}
	var out []string
	for _, item := range d.Document.Body.Items {
		switch o := item.(type) {
		case *docx.Paragraph:
			if o == nil {
				continue
			}
			text := paragraphTextWithPageBreaks(o)
			if text == "" {
				continue
			}
			out = append(out, text)
		case *docx.Table:
			if o == nil {
				continue
			}
			text := o.String()
			if text == "" {
				continue
			}
			out = append(out, text)
		}
	}
	return out
}

// paragraphTextWithPageBreaks 走 Paragraph 的 children / runs，把分页符转成 \x0c。
// fumiama 自带的 Paragraph.String() 把 BarterRabbet 写为 \n，会丢失分页语义。
func paragraphTextWithPageBreaks(p *docx.Paragraph) string {
	if p == nil {
		return ""
	}
	var sb strings.Builder
	for _, c := range p.Children {
		switch r := c.(type) {
		case *docx.Run:
			if r == nil {
				continue
			}
			for _, rc := range r.Children {
				switch x := rc.(type) {
				case *docx.Text:
					if x != nil {
						sb.WriteString(x.Text)
					}
				case *docx.Tab:
					sb.WriteByte('\t')
				case *docx.BarterRabbet:
					if x != nil && x.Type == "page" {
						sb.WriteByte('\x0c')
					} else if x != nil {
						sb.WriteByte('\n')
					}
				}
			}
		case *docx.Hyperlink:
			// 超链接内嵌的 Run 文本也带上
			for _, rc := range r.Run.Children {
				switch x := rc.(type) {
				case *docx.Text:
					if x != nil {
						sb.WriteString(x.Text)
					}
				case *docx.Tab:
					sb.WriteByte('\t')
				case *docx.BarterRabbet:
					if x != nil && x.Type == "page" {
						sb.WriteByte('\x0c')
					} else if x != nil {
						sb.WriteByte('\n')
					}
				}
			}
		}
	}
	return strings.TrimRight(sb.String(), "\n\t ")
}

// readDocxDocumentXML 从原始 zip 中直接抽 word/document.xml，做兜底解析用。
func readDocxDocumentXML(data []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, errors.New("reader: document.xml not found in docx")
}

var (
	docxTextRe = regexp.MustCompile(`<w:t[^>]*>([^<]*)</w:t>`)
	docxPRe    = regexp.MustCompile(`<w:p[ >]`)
	docxPgRe   = regexp.MustCompile(`<w:br\s+w:type="page"\s*/>`)
)

// fallbackParagraphs 旧版正则兜底；fumiama 解析失败 / 拿不到段落时使用。
// 仅在确实发现 <w:p> 标签时才切分；空输入返回 nil。
func fallbackParagraphs(xml []byte) []string {
	if len(xml) == 0 {
		return nil
	}
	indices := docxPRe.FindAllIndex(xml, -1)
	if len(indices) == 0 {
		// 整段没有任何 <w:p>，不是 docx → 返回 nil 让上层报错
		return nil
	}
	var paras []string
	last := 0
	for _, m := range indices {
		piece := xml[last:m[0]]
		var sb strings.Builder
		for _, tm := range docxTextRe.FindAllSubmatch(piece, -1) {
			sb.Write(tm[1])
		}
		paras = append(paras, strings.TrimSpace(sb.String()))
		last = m[0]
	}
	var sb strings.Builder
	for _, tm := range docxTextRe.FindAllSubmatch(xml[last:], -1) {
		sb.Write(tm[1])
	}
	paras = append(paras, strings.TrimSpace(sb.String()))
	return paras
}

// splitDOCXByPages 按分页符 <w:br w:type="page"/> 把段落切到不同页。
// 若无分页则整篇为一页。
func splitDOCXByPages(paras []string) []Page {
	groups := [][]string{{}}
	for _, p := range paras {
		if !strings.Contains(p, "\x0c") {
			groups[len(groups)-1] = append(groups[len(groups)-1], p)
			continue
		}
		// 段落内可能含分页符（罕见）；按 \f 切。
		parts := strings.Split(p, "\x0c")
		for i, part := range parts {
			if i > 0 {
				groups = append(groups, []string{})
			}
			if part != "" {
				groups[len(groups)-1] = append(groups[len(groups)-1], part)
			}
		}
	}
	out := make([]Page, 0, len(groups))
	for i, g := range groups {
		text := strings.TrimSpace(strings.Join(g, "\n"))
		out = append(out, Page{Index: i + 1, Content: text})
	}
	if len(out) == 0 {
		out = append(out, Page{Index: 1, Content: ""})
	}
	return out
}
