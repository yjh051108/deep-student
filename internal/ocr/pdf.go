// OCR 的 PDF 文本层提取：复用 reader 包的 ParsePDF。

package ocr

import (
	"fmt"

	"github.com/helixnow/deep-student-go/internal/reader"
)

// parsePDFPages 解析 PDF 为分页文本。
func parsePDFPages(data []byte) ([]string, error) {
	pages, err := reader.ParsePDF(data)
	if err != nil {
		return nil, fmt.Errorf("ocr: parse pdf: %w", err)
	}
	out := make([]string, 0, len(pages))
	for _, p := range pages {
		out = append(out, p.Content)
	}
	return out, nil
}
