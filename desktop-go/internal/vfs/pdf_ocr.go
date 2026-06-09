package vfs

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type pdfOCRRunnerFunc func(service *Service, resource Resource, metadata map[string]any, now int64) (pdfOCRResult, error)

type pdfOCRResult struct {
	Pages  []OcrPageInfo
	Source string
}

func (r pdfOCRResult) text() string {
	parts := make([]string, 0, len(r.Pages))
	for _, page := range r.Pages {
		text := strings.TrimSpace(page.Text)
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n\n")
}

func defaultPdfOCRRunner(_ *Service, _ Resource, _ map[string]any, _ int64) (pdfOCRResult, error) {
	return pdfOCRResult{}, errors.New("no OCR runner is configured for the lean Go VFS pipeline")
}

func (s *Service) generatePdfOCRMetadata(resource Resource, now int64) map[string]any {
	s.eventMu.RLock()
	runner := s.pdfOCRRunner
	s.eventMu.RUnlock()
	if runner == nil {
		runner = defaultPdfOCRRunner
	}

	result, err := runner(s, resource, normalizeMetadata(resource.Metadata), now)
	if err != nil {
		return pdfOCRFailedMetadata("unavailable", err, now)
	}
	if len(result.Pages) == 0 {
		return pdfOCRFailedMetadata("no_pages", errors.New("OCR runner returned no pages"), now)
	}

	pages := make([]OcrPageInfo, 0, len(result.Pages))
	for index, page := range result.Pages {
		if page.PageIndex < 0 {
			page.PageIndex = index
		}
		page.Text = strings.TrimSpace(page.Text)
		if page.CharCount <= 0 {
			page.CharCount = len([]rune(page.Text))
		}
		if page.Text == "" && !page.IsFailed {
			page.IsFailed = true
		}
		pages = append(pages, page)
	}
	pagesJSON, err := json.Marshal(pages)
	if err != nil {
		return pdfOCRFailedMetadata("serialize_failed", err, now)
	}
	source := strings.TrimSpace(result.Source)
	if source == "" {
		source = "ocr"
	}
	text := strings.TrimSpace((pdfOCRResult{Pages: pages}).text())
	if text == "" {
		return pdfOCRFailedMetadata("empty_text", errors.New("OCR runner returned empty page text"), now)
	}

	failedCount := 0
	for _, page := range pages {
		if page.IsFailed {
			failedCount++
		}
	}
	status := "completed"
	if failedCount > 0 {
		status = "completed_with_issues"
	}
	return map[string]any{
		"ocrText":         text,
		"ocrPagesJson":    string(pagesJSON),
		"ocrPagesSource":  source,
		"ocrStatus":       status,
		"ocrError":        "",
		"ocrUpdatedAt":    formatMillis(now),
		"ocrCompletedAt":  formatMillis(now),
		"ocrPageCount":    len(pages),
		"ocrFailedPages":  failedCount,
		"textIndexState":  "indexed",
		"indexStatus":     "indexed",
		"processingError": "",
	}
}

func pdfOCRFailedMetadata(status string, err error, now int64) map[string]any {
	message := ""
	if err != nil {
		message = err.Error()
	}
	return map[string]any{
		"ocrStatus":       status,
		"ocrError":        message,
		"ocrUpdatedAt":    formatMillis(now),
		"processingError": fmt.Sprintf("OCR unavailable: %s", message),
	}
}
