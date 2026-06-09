package vfs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	pdfRasterPreviewSource      = "pdfium_raster"
	pdfRasterPreviewMimeType    = "image/png"
	pdfRasterPreviewTargetWidth = 816
	pdfRasterPreviewMaxHeight   = 1600
	maxPdfRasterPreviewPages    = 12
)

type pdfRasterPreviewRendererFunc func(service *Service, resource Resource, effectiveMetadata map[string]any, now int64) (map[string]any, bool)

var pdfRasterPreviewRenderer pdfRasterPreviewRendererFunc = renderPdfRasterPreviewWithPdfium

type pdfPreviewRenderJob struct {
	Resource Resource
	Now      int64
	Enabled  bool
}

func pdfPreviewRenderJobForResource(resource Resource, now int64) pdfPreviewRenderJob {
	return pdfPreviewRenderJob{
		Resource: resource,
		Now:      now,
		Enabled:  mediaTypeForResource(resource) == "pdf" && !metadataHasPreviewJSON(resource.Metadata),
	}
}

func (s *Service) renderAndCommitPdfPreview(job pdfPreviewRenderJob) error {
	if !job.Enabled {
		return nil
	}
	metadata := s.generatePdfPreviewMetadata(job.Resource, job.Now)
	if len(metadata) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := s.findResourceIndexByAnyIDLocked(job.Resource.ID)
	if !ok || resourceIsDeleted(s.state.Resources[index]) {
		return nil
	}
	current := s.state.Resources[index]
	if current.Hash != job.Resource.Hash || metadataHasPreviewJSON(current.Metadata) {
		return nil
	}
	s.state.Resources[index].Metadata = mergeMetadata(current.Metadata, metadata)
	s.state.Resources[index].UpdatedAt = nowMillis()
	return s.flushLocked()
}

func (s *Service) generatePdfPreviewMetadata(resource Resource, now int64) map[string]any {
	effectiveMetadata := normalizeMetadata(resource.Metadata)
	metadata := s.generatePdfRasterPreviewMetadataLocked(resource, effectiveMetadata, now)
	effectiveMetadata = mergeMetadata(effectiveMetadata, metadata)
	return mergeMetadata(metadata, s.generatePdfTextLayerPreviewMetadataLocked(resource, effectiveMetadata, now))
}

func (s *Service) generatePdfRasterPreviewMetadataLocked(resource Resource, effectiveMetadata map[string]any, now int64) map[string]any {
	if existing := firstNonEmptyMetadataString(effectiveMetadata, "previewJson", "preview_json"); existing != "" {
		return map[string]any{}
	}
	metadata, _ := pdfRasterPreviewRenderer(s, resource, effectiveMetadata, now)
	return metadata
}

func renderPdfRasterPreviewWithPdfium(service *Service, resource Resource, _ map[string]any, now int64) (map[string]any, bool) {
	pdfBytes, err := service.readResourceBytes(resource)
	if err != nil {
		return pdfRasterPreviewFailedMetadata("read_failed", err), false
	}
	rendered, err := renderPdfPagesWithPdfium(pdfBytes, maxPdfRasterPreviewPages)
	if err != nil {
		return pdfRasterPreviewFailedMetadata("unavailable", err), false
	}
	if rendered.TotalPages <= 0 || len(rendered.Pages) == 0 {
		return pdfRasterPreviewFailedMetadata("no_pages", fmt.Errorf("PDFium returned no rendered pages")), false
	}

	stableID := resource.ID
	if resource.SourceID != nil && strings.TrimSpace(*resource.SourceID) != "" {
		stableID = strings.TrimSpace(*resource.SourceID)
	}
	hashSegment := resource.Hash
	if len(hashSegment) > 16 {
		hashSegment = hashSegment[:16]
	}
	previewDir := filepath.ToSlash(filepath.Join("pdf_raster_previews", safeSegment(stableID, "resource"), safeSegment(hashSegment, "hash")))
	pageRefs := make([]map[string]any, 0, len(rendered.Pages))
	for _, page := range rendered.Pages {
		relativePath := filepath.ToSlash(filepath.Join(previewDir, fmt.Sprintf("page-%04d.png", page.PageIndex+1)))
		absolutePath, err := service.resolveLibraryPath(relativePath)
		if err != nil {
			return pdfRasterPreviewFailedMetadata("write_failed", err), false
		}
		if err := os.MkdirAll(filepath.Dir(absolutePath), 0o700); err != nil {
			return pdfRasterPreviewFailedMetadata("write_failed", err), false
		}
		if err := os.WriteFile(absolutePath, page.PNG, 0o600); err != nil {
			return pdfRasterPreviewFailedMetadata("write_failed", err), false
		}
		pageRefs = append(pageRefs, map[string]any{
			"pageIndex": page.PageIndex,
			"path":      relativePath,
			"width":     page.Width,
			"height":    page.Height,
			"mimeType":  pdfRasterPreviewMimeType,
			"source":    pdfRasterPreviewSource,
			"byteSize":  len(page.PNG),
		})
	}

	isTruncated := rendered.TotalPages > len(rendered.Pages)
	previewJSON, err := json.Marshal(map[string]any{
		"pages":            pageRefs,
		"renderDpi":        0,
		"totalPages":       rendered.TotalPages,
		"renderedAt":       formatMillis(now),
		"isTruncated":      isTruncated,
		"maxRenderedPages": len(rendered.Pages),
		"source":           pdfRasterPreviewSource,
		"mimeType":         pdfRasterPreviewMimeType,
	})
	if err != nil {
		return pdfRasterPreviewFailedMetadata("serialize_failed", err), false
	}

	return map[string]any{
		"previewJson":            string(previewJSON),
		"previewSource":          pdfRasterPreviewSource,
		"previewGeneratedAt":     formatMillis(now),
		"previewPageCount":       len(rendered.Pages),
		"previewMimeType":        pdfRasterPreviewMimeType,
		"pageCount":              rendered.TotalPages,
		"pageRenderingStatus":    "completed",
		"pageRenderingSource":    pdfRasterPreviewSource,
		"pageRenderingError":     "",
		"pageRenderingTruncated": isTruncated,
		"rasterPreviewStatus":    "completed",
		"rasterPreviewSource":    pdfRasterPreviewSource,
		"rasterPreviewError":     "",
	}, true
}

func pdfRasterPreviewFailedMetadata(status string, err error) map[string]any {
	message := ""
	if err != nil {
		message = err.Error()
	}
	return map[string]any{
		"rasterPreviewStatus": status,
		"rasterPreviewSource": pdfRasterPreviewSource,
		"rasterPreviewError":  message,
	}
}

type pdfRasterRenderResult struct {
	TotalPages int
	Pages      []pdfRasterRenderedPage
}

type pdfRasterRenderedPage struct {
	PageIndex int
	Width     int
	Height    int
	PNG       []byte
}
