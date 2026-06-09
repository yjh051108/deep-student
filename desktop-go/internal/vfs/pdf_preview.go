package vfs

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
)

const (
	pdfTextPreviewSource   = "pdf_text_layer_svg"
	pdfTextPreviewMimeType = "image/svg+xml"
	pdfTextPreviewWidth    = 816
	pdfTextPreviewHeight   = 1056
	maxPdfTextPreviewPages = 500
)

func (s *Service) generatePdfTextLayerPreviewMetadataLocked(resource Resource, effectiveMetadata map[string]any, now int64) map[string]any {
	if existing := firstNonEmptyMetadataString(effectiveMetadata, "previewJson", "preview_json"); existing != "" &&
		metadataString(effectiveMetadata, "previewSource", "") != pdfTextPreviewSource {
		return map[string]any{}
	}

	pageCount := metadataInt(effectiveMetadata, "pageCount", "page_count")
	pages := parseOcrPagesJSON(firstMetadataString(effectiveMetadata, "ocrPagesJson", "ocr_pages_json"))
	if len(pages) == 0 {
		text := firstNonEmptyMetadataString(effectiveMetadata, "extractedText", "extracted_text")
		if strings.TrimSpace(text) == "" {
			return map[string]any{
				"pageRenderingStatus": "no_text_layer",
				"pageRenderingSource": pdfTextPreviewSource,
				"pageRenderingError":  "No searchable PDF page text is available for generated previews.",
			}
		}
		pages = pdfTextLayerOcrPages(text, pageCount)
	}
	if len(pages) == 0 {
		return map[string]any{
			"pageRenderingStatus": "no_pages",
			"pageRenderingSource": pdfTextPreviewSource,
			"pageRenderingError":  "No PDF text pages could be estimated for generated previews.",
		}
	}

	totalPages := len(pages)
	if pageCount != nil && *pageCount > totalPages {
		totalPages = *pageCount
	}
	if totalPages < len(pages) {
		totalPages = len(pages)
	}

	renderPages := pages
	isTruncated := false
	if len(renderPages) > maxPdfTextPreviewPages {
		renderPages = renderPages[:maxPdfTextPreviewPages]
		isTruncated = true
	}
	if totalPages > len(renderPages) {
		isTruncated = true
	}

	stableID := resource.ID
	if resource.SourceID != nil && strings.TrimSpace(*resource.SourceID) != "" {
		stableID = strings.TrimSpace(*resource.SourceID)
	}
	hashSegment := resource.Hash
	if len(hashSegment) > 16 {
		hashSegment = hashSegment[:16]
	}
	previewDir := filepath.ToSlash(filepath.Join("pdf_previews", safeSegment(stableID, "resource"), safeSegment(hashSegment, "hash")))
	pageRefs := make([]map[string]any, 0, len(renderPages))
	for _, page := range renderPages {
		pageIndex := page.PageIndex
		if pageIndex < 0 {
			pageIndex = len(pageRefs)
		}
		relativePath := filepath.ToSlash(filepath.Join(previewDir, fmt.Sprintf("page-%04d.svg", pageIndex+1)))
		absolutePath, err := s.resolveLibraryPath(relativePath)
		if err != nil {
			return pdfTextPreviewFailedMetadata(err)
		}
		if err := os.MkdirAll(filepath.Dir(absolutePath), 0o700); err != nil {
			return pdfTextPreviewFailedMetadata(err)
		}
		svg := renderPdfTextLayerPreviewSVG(resourceNameForPreview(resource, effectiveMetadata), page, totalPages)
		if err := os.WriteFile(absolutePath, []byte(svg), 0o600); err != nil {
			return pdfTextPreviewFailedMetadata(err)
		}
		pageRefs = append(pageRefs, map[string]any{
			"pageIndex": pageIndex,
			"path":      relativePath,
			"width":     pdfTextPreviewWidth,
			"height":    pdfTextPreviewHeight,
			"mimeType":  pdfTextPreviewMimeType,
			"source":    pdfTextPreviewSource,
			"charCount": page.CharCount,
		})
	}

	previewJSON, err := json.Marshal(map[string]any{
		"pages":            pageRefs,
		"renderDpi":        0,
		"totalPages":       totalPages,
		"renderedAt":       formatMillis(now),
		"isTruncated":      isTruncated,
		"maxRenderedPages": len(renderPages),
		"source":           pdfTextPreviewSource,
		"mimeType":         pdfTextPreviewMimeType,
	})
	if err != nil {
		return pdfTextPreviewFailedMetadata(err)
	}

	return map[string]any{
		"previewJson":            string(previewJSON),
		"previewSource":          pdfTextPreviewSource,
		"previewGeneratedAt":     formatMillis(now),
		"previewPageCount":       len(renderPages),
		"previewMimeType":        pdfTextPreviewMimeType,
		"pageRenderingStatus":    "completed",
		"pageRenderingSource":    pdfTextPreviewSource,
		"pageRenderingError":     "",
		"pageRenderingTruncated": isTruncated,
	}
}

func pdfTextPreviewFailedMetadata(err error) map[string]any {
	return map[string]any{
		"pageRenderingStatus": "failed",
		"pageRenderingSource": pdfTextPreviewSource,
		"pageRenderingError":  err.Error(),
	}
}

func resourceNameForPreview(resource Resource, metadata map[string]any) string {
	if name := firstNonEmptyMetadataString(metadata, "title", "name", "fileName", "file_name"); name != "" {
		return name
	}
	return resourceName(resource)
}

func renderPdfTextLayerPreviewSVG(title string, page OcrPageInfo, totalPages int) string {
	lines := wrapPdfPreviewText(page.Text, 78, 34)
	if len(lines) == 0 {
		lines = []string{"No text was estimated for this page."}
	}
	pageNumber := page.PageIndex + 1
	if pageNumber <= 0 {
		pageNumber = 1
	}
	if totalPages < pageNumber {
		totalPages = pageNumber
	}

	var body strings.Builder
	for index, line := range lines {
		y := 172 + index*24
		body.WriteString(fmt.Sprintf(`<text x="72" y="%d" class="line">%s</text>`, y, html.EscapeString(line)))
		body.WriteByte('\n')
	}

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">
<rect width="100%%" height="100%%" fill="#f8fafc"/>
<rect x="48" y="44" width="720" height="968" rx="10" fill="#ffffff" stroke="#cbd5e1"/>
<text x="72" y="92" class="title">%s</text>
<text x="72" y="124" class="meta">Page %d of %d - generated from searchable PDF text</text>
<line x1="72" y1="144" x2="744" y2="144" stroke="#e2e8f0"/>
%s<text x="72" y="972" class="footer">Deep Student Go hybrid VFS - %s</text>
<style>
.title{font:600 24px sans-serif;fill:#0f172a}
.meta{font:400 15px sans-serif;fill:#64748b}
.line{font:400 18px ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;fill:#1e293b}
.footer{font:400 13px sans-serif;fill:#94a3b8}
</style>
</svg>`, pdfTextPreviewWidth, pdfTextPreviewHeight, pdfTextPreviewWidth, pdfTextPreviewHeight, html.EscapeString(title), pageNumber, totalPages, body.String(), pdfTextPreviewSource)
}

func wrapPdfPreviewText(text string, maxColumns int, maxLines int) []string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if text == "" {
		return []string{}
	}
	runes := []rune(text)
	lines := []string{}
	start := 0
	for start < len(runes) && len(lines) < maxLines {
		end := start + maxColumns
		if end >= len(runes) {
			lines = append(lines, string(runes[start:]))
			break
		}
		breakAt := end
		for i := end; i > start+maxColumns/2; i-- {
			if runes[i-1] == ' ' || isPageTextBreakRune(runes[i-1]) {
				breakAt = i
				break
			}
		}
		lines = append(lines, strings.TrimSpace(string(runes[start:breakAt])))
		start = breakAt
		for start < len(runes) && runes[start] == ' ' {
			start++
		}
	}
	if start < len(runes) && len(lines) == maxLines {
		last := strings.TrimRight(lines[len(lines)-1], ". ")
		lines[len(lines)-1] = last + "..."
	}
	return lines
}
