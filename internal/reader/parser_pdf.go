package reader

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// ParsePDF 用 pdfcpu 解析 PDF，按页提取文本内容。
//
// 设计：
//   - PDF 结构 / xref / page 树 / Flate 解码 全部交给 pdfcpu（工业级）
//   - 文本提取只解析 content stream 中的 Tj / TJ 文本操作符
//   - 对扫描型 PDF（无文本层）返回空文本，上层可降级到 OCR
func ParsePDF(data []byte) ([]Page, error) {
	if len(data) == 0 {
		return nil, errors.New("reader: empty pdf")
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return nil, errors.New("reader: not a pdf")
	}
	// 1) pdfcpu 读取 / 校验 PDF，得到带完整 xref 的 ctx
	ctx, err := api.ReadContext(bytes.NewReader(data), model.NewDefaultConfiguration())
	if err != nil {
		return nil, fmt.Errorf("reader: pdfcpu read: %w", err)
	}
	// ReadContext 不会自动计算 PageCount，需要 EnsurePageCount。
	if err := ctx.EnsurePageCount(); err != nil {
		return nil, fmt.Errorf("reader: ensure page count: %w", err)
	}
	pageCount := ctx.PageCount
	if pageCount <= 0 {
		return nil, errors.New("reader: pdf has no pages")
	}
	// 2) 逐页提取 content stream 并抽取文本
	out := make([]Page, 0, pageCount)
	for p := 1; p <= pageCount; p++ {
		r, err := pdfcpu.ExtractPageContent(ctx, p)
		if err != nil {
			if errors.Is(err, model.ErrNoContent) {
				// 页存在但无内容（纯图片/空白页）—— 保留空文本
				out = append(out, Page{Index: p, Content: ""})
				continue
			}
			return nil, fmt.Errorf("reader: extract page %d content: %w", p, err)
		}
		stream, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("reader: read page %d stream: %w", p, err)
		}
		text := extractTextFromContentStream(stream)
		out = append(out, Page{Index: p, Content: strings.TrimSpace(text)})
	}
	return out, nil
}

// extractTextFromContentStream 从 PDF 单页 content stream 提取文本。
// 支持的操作符：
//   - (string) Tj        直接文本
//   - [(str) (str) ...] TJ  数组文本（k 调整字距）
//   - <hex> Tj / TJ      hex 字符串
// 不支持：
//   - 字体子集 / CID 字形到 Unicode 的映射（扫描型 PDF 走 OCR）
//   - 复杂 text matrix（Tm）—— 仍能拿到原始字符串
var (
	tjRe  = regexp.MustCompile(`\(([^)]*)\)\s*Tj`)
	tjAR  = regexp.MustCompile(`\[(.*?)\]\s*TJ`)
	hexRe = regexp.MustCompile(`<([0-9a-fA-F]+)>`)
)

func extractTextFromContentStream(stream []byte) string {
	var sb strings.Builder
	// (text) Tj
	for _, m := range tjRe.FindAllSubmatch(stream, -1) {
		sb.Write(decodePDFString(m[1]))
		sb.WriteByte('\n')
	}
	// [array] TJ
	for _, m := range tjAR.FindAllSubmatch(stream, -1) {
		body := m[1]
		// 数组里的 (text)
		sub := regexp.MustCompile(`\(([^)]*)\)`)
		for _, sm := range sub.FindAllSubmatch(body, -1) {
			sb.Write(decodePDFString(sm[1]))
		}
		// 数组里的 <hex>
		for _, hm := range hexRe.FindAllSubmatch(body, -1) {
			dec, _ := decodeHexString(string(hm[1]))
			sb.Write(dec)
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func decodePDFString(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		c := b[i]
		if c == '\\' && i+1 < len(b) {
			next := b[i+1]
			switch next {
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			case '(':
				out = append(out, '(')
			case ')':
				out = append(out, ')')
			case '\\':
				out = append(out, '\\', '\\')
			default:
				out = append(out, next)
			}
			i++
			continue
		}
		out = append(out, c)
	}
	return out
}

func decodeHexString(s string) ([]byte, error) {
	if len(s)%2 == 1 {
		s += "0"
	}
	out := make([]byte, len(s)/2)
	for i := 0; i < len(out); i++ {
		hi, ok1 := hexNibble(s[2*i])
		lo, ok2 := hexNibble(s[2*i+1])
		if !ok1 || !ok2 {
			return nil, errors.New("invalid hex")
		}
		out[i] = hi<<4 | lo
	}
	return out, nil
}

func hexNibble(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}
