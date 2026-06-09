package vfs

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"strings"
	"testing"
)

func TestUploadFileExtractsPdfTextLayer(t *testing.T) {
	service := newTestService(t)
	pdfBytes := []byte("%PDF-1.4\n1 0 obj\n<< /Length 85 >>\nstream\nBT\n/F1 12 Tf\n(Chapter\\040One) Tj\n[(Go) 20 ( hybrid VFS)] TJ\nET\nendstream\nendobj\n%%EOF")

	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "chapter.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString(pdfBytes),
	})
	if err != nil {
		t.Fatalf("UploadFile(pdf) error = %v", err)
	}
	if uploaded.File.ExtractedText == nil || !strings.Contains(*uploaded.File.ExtractedText, "Chapter One") || !strings.Contains(*uploaded.File.ExtractedText, "hybrid VFS") {
		t.Fatalf("expected extracted PDF text, got %+v", uploaded.File.ExtractedText)
	}
	if uploaded.IndexStatus == nil || !uploaded.IndexStatus.Queued || uploaded.IndexStatus.UnitsCreated == 0 {
		t.Fatalf("expected indexed text units from PDF text layer, got %+v", uploaded.IndexStatus)
	}

	status, err := service.GetPdfProcessingStatus(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetPdfProcessingStatus() error = %v", err)
	}
	if !readyModesContain(status.ReadyModes, "text") || readyModesContain(status.ReadyModes, "image") || readyModesContain(status.ReadyModes, "ocr") || status.Stage != "completed_with_issues" {
		t.Fatalf("expected PDF text readiness with raster preview issue, got %+v", status)
	}

	ocrInfo, err := service.GetResourceOcrInfo(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetResourceOcrInfo() error = %v", err)
	}
	if ocrInfo.ActiveSource != "extracted" || ocrInfo.ExtractedText == nil || !strings.Contains(*ocrInfo.ExtractedText, "Chapter One") {
		t.Fatalf("expected extracted text OCR info, got %+v", ocrInfo)
	}

	search, err := service.RagSearch(VfsRagSearchInput{Query: "hybrid VFS", TopK: 5})
	if err != nil {
		t.Fatalf("RagSearch() error = %v", err)
	}
	if search.Count == 0 || !strings.Contains(search.Results[0].ChunkText, "hybrid VFS") {
		t.Fatalf("expected PDF extracted text to be searchable, got %+v", search)
	}
}

func TestUploadFileDetectsPdfPageCount(t *testing.T) {
	service := newTestService(t)
	pdfBytes := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Count 2 /Kids [3 0 R 4 0 R] >>\nendobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R >>\nendobj\n4 0 obj\n<< /Type /Page /Parent 2 0 R >>\nendobj\n%%EOF")

	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "two-pages.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString(pdfBytes),
	})
	if err != nil {
		t.Fatalf("UploadFile(pdf) error = %v", err)
	}
	if uploaded.File.PageCount == nil || *uploaded.File.PageCount != 2 {
		t.Fatalf("expected uploaded PDF page count 2, got %+v", uploaded.File.PageCount)
	}
	if uploaded.OcrStatus == nil || uploaded.OcrStatus.TotalPages != 2 {
		t.Fatalf("expected OCR status to expose PDF page count, got %+v", uploaded.OcrStatus)
	}

	status, err := service.GetPdfProcessingStatus(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetPdfProcessingStatus() error = %v", err)
	}
	if status.TotalPages == nil || *status.TotalPages != 2 || status.CurrentPage == nil || *status.CurrentPage != 2 {
		t.Fatalf("expected status to expose PDF page progress, got %+v", status)
	}
	if status.Progress.TotalPages == nil || *status.Progress.TotalPages != 2 || status.Progress.CurrentPage == nil || *status.Progress.CurrentPage != 2 {
		t.Fatalf("expected progress to mirror page count, got %+v", status.Progress)
	}
}

func TestUploadFileBuildsPdfTextLayerPageInfo(t *testing.T) {
	service := newTestService(t)
	pdfBytes := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Count 2 /Kids [3 0 R 4 0 R] >>\nendobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R >>\nendobj\n4 0 obj\n<< /Type /Page /Parent 2 0 R >>\nendobj\n5 0 obj\n<< /Length 90 >>\nstream\nBT\n(Alpha page text.) Tj\n(Beta page text.) Tj\nET\nendstream\nendobj\n%%EOF")

	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "paged-text.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString(pdfBytes),
	})
	if err != nil {
		t.Fatalf("UploadFile(pdf) error = %v", err)
	}
	if uploaded.File.OcrPagesJSON == nil || !strings.Contains(*uploaded.File.OcrPagesJSON, "Alpha page text") {
		t.Fatalf("expected text-layer page JSON on uploaded file, got %+v", uploaded.File.OcrPagesJSON)
	}
	resource, err := service.GetResource(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetResource(uploaded) error = %v", err)
	}
	if resource == nil || metadataString(resource.Metadata, "previewSource", "") != pdfTextPreviewSource || firstMetadataString(resource.Metadata, "previewJson") == nil {
		t.Fatalf("expected generated PDF text-layer preview metadata, got %+v", resource)
	}
	pageImage, err := service.GetPdfPageImage(uploaded.SourceID, 0)
	if err != nil {
		t.Fatalf("GetPdfPageImage(generated text preview) error = %v", err)
	}
	if pageImage.MimeType != pdfTextPreviewMimeType || pageImage.Size == 0 {
		t.Fatalf("expected generated SVG page preview, got %+v", pageImage)
	}
	decodedPreview, err := base64.StdEncoding.DecodeString(pageImage.Base64)
	if err != nil {
		t.Fatalf("generated SVG preview base64 decode error: %v", err)
	}
	if !strings.Contains(string(decodedPreview), "Alpha page text") || !strings.Contains(string(decodedPreview), pdfTextPreviewSource) {
		t.Fatalf("generated SVG preview should include page text and source marker, got %q", decodedPreview)
	}

	ocrInfo, err := service.GetResourceOcrInfo(uploaded.SourceID)
	if err != nil {
		t.Fatalf("GetResourceOcrInfo() error = %v", err)
	}
	if ocrInfo.HasOcr || ocrInfo.ActiveSource != "extracted" || len(ocrInfo.OcrPages) != 2 {
		t.Fatalf("expected extracted text-layer pages without claiming OCR, got %+v", ocrInfo)
	}
	combined := ocrInfo.OcrPages[0].Text + "\n" + ocrInfo.OcrPages[1].Text
	if !strings.Contains(combined, "Alpha page text") || !strings.Contains(combined, "Beta page text") {
		t.Fatalf("expected page text chunks to contain extracted text, got %+v", ocrInfo.OcrPages)
	}
	if ocrInfo.ExtractedText == nil || !strings.Contains(*ocrInfo.ExtractedText, "Alpha page text") {
		t.Fatalf("expected extracted text to remain active source, got %+v", ocrInfo)
	}
}

func TestDetectPdfPageCountFallsBackToPageTreeCount(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Count 7 >>\nendobj\n%%EOF")
	count := detectPdfPageCount(pdfBytes)
	if count == nil || *count != 7 {
		t.Fatalf("expected fallback page tree count 7, got %+v", count)
	}
}

func TestUploadFileExtractsFlatePdfTextLayer(t *testing.T) {
	service := newTestService(t)
	content := []byte("BT\n<005700610069006c007300200074006500780074> Tj\nET")
	var compressed bytes.Buffer
	zipper := zlib.NewWriter(&compressed)
	if _, err := zipper.Write(content); err != nil {
		t.Fatalf("zlib write error = %v", err)
	}
	if err := zipper.Close(); err != nil {
		t.Fatalf("zlib close error = %v", err)
	}
	pdfBytes := []byte("%PDF-1.4\n1 0 obj\n<< /Filter /FlateDecode /Length 1 >>\nstream\n")
	pdfBytes = append(pdfBytes, compressed.Bytes()...)
	pdfBytes = append(pdfBytes, []byte("\nendstream\nendobj\n%%EOF")...)

	uploaded, err := service.UploadFile(UploadFileInput{
		Name:          "flate.pdf",
		MimeType:      "application/pdf",
		Base64Content: base64.StdEncoding.EncodeToString(pdfBytes),
	})
	if err != nil {
		t.Fatalf("UploadFile(flate pdf) error = %v", err)
	}
	if uploaded.File.ExtractedText == nil || *uploaded.File.ExtractedText != "Wails text" {
		t.Fatalf("expected UTF-16BE Flate PDF text, got %+v", uploaded.File.ExtractedText)
	}
}

func TestStartPdfProcessingExtractsTextForExistingPdfResource(t *testing.T) {
	service := newTestService(t)
	sourceID := "file_existing_pdf"
	pdfBytes := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Count 1 /Kids [3 0 R] >>\nendobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R >>\nendobj\n4 0 obj\n<< /Length 44 >>\nstream\nBT\n(Retry processing text layer) Tj\nET\nendstream\nendobj\n%%EOF")
	_, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "file",
		Data:     base64.StdEncoding.EncodeToString(pdfBytes),
		SourceID: &sourceID,
		Metadata: map[string]any{
			"fileRecord": true,
			"name":       "legacy.pdf",
			"mimeType":   "application/pdf",
			"status":     "active",
		},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse(existing pdf) error = %v", err)
	}
	before, err := service.GetResource(sourceID)
	if err != nil {
		t.Fatalf("GetResource(before) error = %v", err)
	}
	if before == nil || resourceTextContent(*before) != nil {
		t.Fatalf("test setup should not have extracted text yet: %+v", before)
	}

	stage := "text_extraction"
	if err := service.StartPdfProcessing(sourceID, &stage); err != nil {
		t.Fatalf("StartPdfProcessing() error = %v", err)
	}
	after, err := service.GetResource(sourceID)
	if err != nil {
		t.Fatalf("GetResource(after) error = %v", err)
	}
	if after == nil || metadataString(after.Metadata, "textExtractionStatus", "") != "completed" {
		t.Fatalf("expected completed text extraction metadata, got %+v", after)
	}
	if count := metadataInt(after.Metadata, "pageCount"); count == nil || *count != 1 {
		t.Fatalf("expected retry processing to detect page count, got %+v", count)
	}
	extracted := metadataString(after.Metadata, "extractedText", "")
	if !strings.Contains(extracted, "Retry processing text layer") {
		t.Fatalf("expected retry processing to extract PDF text, got %q", extracted)
	}

	status, err := service.GetPdfProcessingStatus(sourceID)
	if err != nil {
		t.Fatalf("GetPdfProcessingStatus(after) error = %v", err)
	}
	if !readyModesContain(status.ReadyModes, "text") || readyModesContain(status.ReadyModes, "image") || readyModesContain(status.ReadyModes, "ocr") || status.Stage != "completed_with_issues" {
		t.Fatalf("expected processed PDF to report text readiness with raster preview issue, got %+v", status)
	}
	if status.TotalPages == nil || *status.TotalPages != 1 {
		t.Fatalf("expected processed PDF to report page count, got %+v", status)
	}
	ocrInfo, err := service.GetResourceOcrInfo(sourceID)
	if err != nil {
		t.Fatalf("GetResourceOcrInfo(after) error = %v", err)
	}
	if len(ocrInfo.OcrPages) != 1 || !strings.Contains(ocrInfo.OcrPages[0].Text, "Retry processing text layer") || ocrInfo.HasOcr {
		t.Fatalf("expected processing to build extracted text page info without claiming OCR, got %+v", ocrInfo)
	}
	image, err := service.GetPdfPageImage(sourceID, 0)
	if err != nil {
		t.Fatalf("GetPdfPageImage(after processing) error = %v", err)
	}
	if image.MimeType != pdfTextPreviewMimeType {
		t.Fatalf("expected generated SVG preview after processing, got %+v", image)
	}
}

func TestStartPdfProcessingBuildsSvgPreviewFromExistingOcrPages(t *testing.T) {
	service := newTestService(t)
	sourceID := "file_ocr_pages_only"
	ocrPagesJSON := `[{"pageIndex":0,"text":"Migrated OCR page text","charCount":22,"isFailed":false}]`
	_, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "file",
		Data:     base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n")),
		SourceID: &sourceID,
		Metadata: map[string]any{
			"fileRecord":   true,
			"name":         "migrated.pdf",
			"mimeType":     "application/pdf",
			"status":       "active",
			"pageCount":    1,
			"ocrPagesJson": ocrPagesJSON,
		},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse(existing ocr pages pdf) error = %v", err)
	}

	if err := service.StartPdfProcessing(sourceID, nil); err != nil {
		t.Fatalf("StartPdfProcessing(ocr pages only) error = %v", err)
	}
	status, err := service.GetPdfProcessingStatus(sourceID)
	if err != nil {
		t.Fatalf("GetPdfProcessingStatus(ocr pages only) error = %v", err)
	}
	if !readyModesContain(status.ReadyModes, "ocr") || readyModesContain(status.ReadyModes, "text") || readyModesContain(status.ReadyModes, "image") || status.Stage != "completed_with_issues" {
		t.Fatalf("expected OCR readiness without image/text for OCR-page migrated preview, got %+v", status)
	}
	image, err := service.GetPdfPageImage(sourceID, 0)
	if err != nil {
		t.Fatalf("GetPdfPageImage(ocr pages only) error = %v", err)
	}
	if image.MimeType != pdfTextPreviewMimeType {
		t.Fatalf("expected generated SVG preview from OCR pages, got %+v", image)
	}
	decodedPreview, err := base64.StdEncoding.DecodeString(image.Base64)
	if err != nil {
		t.Fatalf("generated OCR-page SVG preview base64 decode error: %v", err)
	}
	if !strings.Contains(string(decodedPreview), "Migrated OCR page text") {
		t.Fatalf("generated OCR-page SVG preview should include page text, got %q", decodedPreview)
	}
}

func TestStartPdfProcessingOcrRunnerPersistsRealOcrMetadata(t *testing.T) {
	service := newTestService(t)
	sourceID := "file_scanned_ocr"
	service.SetPDFOCRRunner(func(_ *Service, _ Resource, _ map[string]any, _ int64) (pdfOCRResult, error) {
		return pdfOCRResult{
			Source: "test_ocr",
			Pages: []OcrPageInfo{{
				PageIndex: 0,
				Text:      "Scanned OCR text",
			}},
		}, nil
	})

	_, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "file",
		Data:     base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n")),
		SourceID: &sourceID,
		Metadata: map[string]any{
			"fileRecord": true,
			"name":       "scanned.pdf",
			"mimeType":   "application/pdf",
			"status":     "active",
			"pageCount":  1,
		},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse(scanned pdf) error = %v", err)
	}

	stage := "ocr_processing"
	if err := service.StartPdfProcessing(sourceID, &stage); err != nil {
		t.Fatalf("StartPdfProcessing(ocr runner) error = %v", err)
	}
	resource, err := service.GetResource(sourceID)
	if err != nil {
		t.Fatalf("GetResource(ocr runner) error = %v", err)
	}
	if resource == nil || metadataString(resource.Metadata, "ocrStatus", "") != "completed" || metadataString(resource.Metadata, "ocrPagesSource", "") != "test_ocr" {
		t.Fatalf("expected persisted OCR metadata, got %+v", resource)
	}
	if !strings.Contains(metadataString(resource.Metadata, "ocrText", ""), "Scanned OCR text") {
		t.Fatalf("expected OCR text in metadata, got %+v", resource.Metadata)
	}
	status, err := service.GetPdfProcessingStatus(sourceID)
	if err != nil {
		t.Fatalf("GetPdfProcessingStatus(ocr runner) error = %v", err)
	}
	if !readyModesContain(status.ReadyModes, "ocr") || readyModesContain(status.ReadyModes, "text") {
		t.Fatalf("expected OCR readiness without native text readiness, got %+v", status)
	}
	info, err := service.GetResourceOcrInfo(sourceID)
	if err != nil {
		t.Fatalf("GetResourceOcrInfo(ocr runner) error = %v", err)
	}
	if !info.HasOcr || info.ActiveSource != "ocr" || info.OcrText == nil || !strings.Contains(*info.OcrText, "Scanned OCR text") || len(info.OcrPages) != 1 {
		t.Fatalf("expected active real OCR info, got %+v", info)
	}
	refs, err := service.GetResourceRefs(GetResourceRefsInput{SourceIDs: []string{sourceID}})
	if err != nil {
		t.Fatalf("GetResourceRefs(ocr runner) error = %v", err)
	}
	resolved, err := service.ResolveResourceRefs(refs.Refs)
	if err != nil {
		t.Fatalf("ResolveResourceRefs(ocr runner) error = %v", err)
	}
	if len(resolved) != 1 || len(resolved[0].MultimodalBlocks) != 1 || resolved[0].MultimodalBlocks[0].Text == nil || !strings.Contains(*resolved[0].MultimodalBlocks[0].Text, "Scanned OCR text") {
		t.Fatalf("expected real OCR text multimodal block, got %+v", resolved)
	}
}

func TestStartPdfProcessingDefaultOcrRunnerDoesNotClaimOcr(t *testing.T) {
	service := newTestService(t)
	sourceID := "file_ocr_unavailable"
	_, err := service.CreateOrReuse(CreateResourceInput{
		Type:     "file",
		Data:     base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n")),
		SourceID: &sourceID,
		Metadata: map[string]any{
			"fileRecord": true,
			"name":       "ocr-unavailable.pdf",
			"mimeType":   "application/pdf",
			"status":     "active",
		},
	})
	if err != nil {
		t.Fatalf("CreateOrReuse(ocr unavailable pdf) error = %v", err)
	}

	stage := "ocr_processing"
	if err := service.StartPdfProcessing(sourceID, &stage); err != nil {
		t.Fatalf("StartPdfProcessing(default OCR runner) error = %v", err)
	}
	resource, err := service.GetResource(sourceID)
	if err != nil {
		t.Fatalf("GetResource(default OCR runner) error = %v", err)
	}
	if resource == nil || metadataString(resource.Metadata, "ocrStatus", "") != "unavailable" || metadataString(resource.Metadata, "ocrText", "") != "" {
		t.Fatalf("expected unavailable OCR metadata without OCR text, got %+v", resource)
	}
	info, err := service.GetResourceOcrInfo(sourceID)
	if err != nil {
		t.Fatalf("GetResourceOcrInfo(default OCR runner) error = %v", err)
	}
	if info.HasOcr || info.ActiveSource != "none" || info.OcrText != nil || len(info.OcrPages) != 0 {
		t.Fatalf("default OCR runner should not claim OCR, got %+v", info)
	}
	status, err := service.GetPdfProcessingStatus(sourceID)
	if err != nil {
		t.Fatalf("GetPdfProcessingStatus(default OCR runner) error = %v", err)
	}
	if readyModesContain(status.ReadyModes, "ocr") {
		t.Fatalf("default OCR runner should not expose OCR readiness, got %+v", status)
	}
}

func readyModesContain(modes []string, mode string) bool {
	for _, existing := range modes {
		if existing == mode {
			return true
		}
	}
	return false
}

func failedStagesContain(stages []PdfProcessingFailedStage, stage string) bool {
	for _, existing := range stages {
		if existing.Stage == stage && strings.TrimSpace(existing.Message) != "" {
			return true
		}
	}
	return false
}
