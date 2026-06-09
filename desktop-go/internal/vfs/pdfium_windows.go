//go:build windows

package vfs

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	fpdfRenderFlagNone = 0
	fpdfBitmapBGRA     = 4
)

type pdfiumDLL struct {
	dll                *windows.DLL
	initLibrary        *windows.Proc
	loadMemDocument    *windows.Proc
	loadMemDocument64  *windows.Proc
	closeDocument      *windows.Proc
	getPageCount       *windows.Proc
	loadPage           *windows.Proc
	closePage          *windows.Proc
	getPageSizeByIndex *windows.Proc
	bitmapCreateEx     *windows.Proc
	bitmapDestroy      *windows.Proc
	bitmapFillRect     *windows.Proc
	renderPageBitmap   *windows.Proc
	getLastError       *windows.Proc
}

var (
	pdfiumLoadMu sync.Mutex
	pdfiumLoaded *pdfiumDLL
	pdfiumMu     sync.Mutex
)

func renderPdfPagesWithPdfium(pdfBytes []byte, maxPages int) (pdfRasterRenderResult, error) {
	if len(pdfBytes) == 0 {
		return pdfRasterRenderResult{}, errors.New("PDF bytes are empty")
	}
	if maxPages <= 0 {
		maxPages = 1
	}

	pdfiumMu.Lock()
	defer pdfiumMu.Unlock()

	lib, err := loadPdfium()
	if err != nil {
		return pdfRasterRenderResult{}, err
	}

	document, err := lib.loadDocumentFromBytes(pdfBytes)
	defer runtime.KeepAlive(pdfBytes)
	if err != nil {
		return pdfRasterRenderResult{}, err
	}
	if document == 0 {
		return pdfRasterRenderResult{}, fmt.Errorf("PDFium document load failed: %s", lib.lastError())
	}
	defer lib.closeDocument.Call(document)

	pageCountValue, _, _ := lib.getPageCount.Call(document)
	totalPages := int(pageCountValue)
	if totalPages <= 0 {
		return pdfRasterRenderResult{}, fmt.Errorf("FPDF_GetPageCount returned %d", totalPages)
	}
	renderPages := totalPages
	if renderPages > maxPages {
		renderPages = maxPages
	}

	result := pdfRasterRenderResult{
		TotalPages: totalPages,
		Pages:      make([]pdfRasterRenderedPage, 0, renderPages),
	}
	for pageIndex := 0; pageIndex < renderPages; pageIndex++ {
		page, err := lib.renderPage(document, pageIndex)
		if err != nil {
			return pdfRasterRenderResult{}, err
		}
		result.Pages = append(result.Pages, page)
	}
	return result, nil
}

func loadPdfium() (*pdfiumDLL, error) {
	pdfiumLoadMu.Lock()
	defer pdfiumLoadMu.Unlock()
	if pdfiumLoaded != nil {
		return pdfiumLoaded, nil
	}
	loaded, err := openPdfium()
	if err != nil {
		return nil, err
	}
	pdfiumLoaded = loaded
	return loaded, nil
}

func openPdfium() (*pdfiumDLL, error) {
	candidates, candidateFailures := pdfiumCandidatePaths()
	failures := append([]string{}, candidateFailures...)
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		absolute, err := normalizePdfiumDLLPath(candidate)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", candidate, err))
			continue
		}
		info, err := os.Stat(absolute)
		if err != nil || info.IsDir() {
			if err != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", absolute, err))
			}
			continue
		}
		dll, err := loadPdfiumDLL(absolute)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", absolute, err))
			continue
		}
		loaded, err := bindPdfiumDLL(dll)
		if err != nil {
			_ = dll.Release()
			failures = append(failures, fmt.Sprintf("%s: %v", absolute, err))
			continue
		}
		loaded.initLibrary.Call()
		return loaded, nil
	}
	return nil, fmt.Errorf("pdfium.dll unavailable; tried %d path(s): %s", len(candidates), strings.Join(failures, "; "))
}

func loadPdfiumDLL(absolutePath string) (*windows.DLL, error) {
	handle, err := windows.LoadLibraryEx(
		absolutePath,
		0,
		windows.LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR|windows.LOAD_LIBRARY_SEARCH_SYSTEM32,
	)
	if err != nil {
		return nil, err
	}
	return &windows.DLL{Name: absolutePath, Handle: handle}, nil
}

func bindPdfiumDLL(dll *windows.DLL) (*pdfiumDLL, error) {
	type requiredProc struct {
		name string
		dst  **windows.Proc
	}
	loaded := &pdfiumDLL{dll: dll}
	required := []requiredProc{
		{"FPDF_InitLibrary", &loaded.initLibrary},
		{"FPDF_LoadMemDocument", &loaded.loadMemDocument},
		{"FPDF_CloseDocument", &loaded.closeDocument},
		{"FPDF_GetPageCount", &loaded.getPageCount},
		{"FPDF_LoadPage", &loaded.loadPage},
		{"FPDF_ClosePage", &loaded.closePage},
		{"FPDF_GetPageSizeByIndex", &loaded.getPageSizeByIndex},
		{"FPDFBitmap_CreateEx", &loaded.bitmapCreateEx},
		{"FPDFBitmap_Destroy", &loaded.bitmapDestroy},
		{"FPDFBitmap_FillRect", &loaded.bitmapFillRect},
		{"FPDF_RenderPageBitmap", &loaded.renderPageBitmap},
	}
	for _, item := range required {
		proc, err := dll.FindProc(item.name)
		if err != nil {
			return nil, err
		}
		*item.dst = proc
	}
	loaded.loadMemDocument64, _ = dll.FindProc("FPDF_LoadMemDocument64")
	loaded.getLastError, _ = dll.FindProc("FPDF_GetLastError")
	return loaded, nil
}

func pdfiumCandidatePaths() ([]string, []string) {
	configured := os.Getenv("DEEP_STUDENT_PDFIUM_PATH")
	executable := ""
	if value, err := os.Executable(); err == nil {
		executable = value
	}
	cwd := ""
	if value, err := os.Getwd(); err == nil {
		cwd = value
	}
	return pdfiumCandidatePathsFor(configured, executable, cwd, os.Getenv("DEEP_STUDENT_ENABLE_DEV_PDFIUM_PATHS") == "1")
}

func pdfiumCandidatePathsFor(configured string, executable string, cwd string, devPathsEnabled bool) ([]string, []string) {
	seen := map[string]struct{}{}
	out := []string{}
	failures := []string{}
	add := func(path string) {
		if strings.TrimSpace(path) == "" {
			return
		}
		cleaned, err := normalizePdfiumDLLPath(path)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", path, err))
			return
		}
		key := strings.ToLower(cleaned)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, cleaned)
	}

	if configured != "" {
		add(configured)
	}
	if executable != "" {
		dir := filepath.Dir(executable)
		add(filepath.Join(dir, "pdfium.dll"))
		add(filepath.Join(dir, "..", "Resources", "pdfium.dll"))
	}
	if repoRoot, ok := pdfiumDevRepoRootFrom(cwd, devPathsEnabled); ok {
		add(filepath.Join(repoRoot, "pdfium.dll"))
		add(filepath.Join(repoRoot, "src-tauri", "resources", "pdfium", "pdfium.dll"))
	}
	return out, failures
}

func normalizePdfiumDLLPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("path is empty")
	}
	if !filepath.IsAbs(trimmed) {
		return "", errors.New("path must be absolute")
	}
	absolute, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(filepath.Base(absolute), "pdfium.dll") {
		return "", errors.New("path must point to pdfium.dll")
	}
	return filepath.Clean(absolute), nil
}

func pdfiumDevRepoRoot() (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	return pdfiumDevRepoRootFrom(cwd, os.Getenv("DEEP_STUDENT_ENABLE_DEV_PDFIUM_PATHS") == "1")
}

func pdfiumDevRepoRootFrom(cwd string, enabled bool) (string, bool) {
	if !enabled || strings.TrimSpace(cwd) == "" {
		return "", false
	}
	for dir := cwd; dir != ""; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "desktop-go", "go.mod")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", false
}

func (lib *pdfiumDLL) loadDocumentFromBytes(pdfBytes []byte) (uintptr, error) {
	if len(pdfBytes) == 0 {
		return 0, errors.New("PDF bytes are empty")
	}
	if lib.loadMemDocument64 != nil {
		document, _, _ := lib.loadMemDocument64.Call(
			uintptr(unsafe.Pointer(&pdfBytes[0])),
			uintptr(len(pdfBytes)),
			0,
		)
		return document, nil
	}
	if len(pdfBytes) > math.MaxInt32 {
		return 0, fmt.Errorf("PDF is too large for FPDF_LoadMemDocument: %d bytes", len(pdfBytes))
	}
	document, _, _ := lib.loadMemDocument.Call(
		uintptr(unsafe.Pointer(&pdfBytes[0])),
		uintptr(len(pdfBytes)),
		0,
	)
	return document, nil
}

func (lib *pdfiumDLL) renderPage(document uintptr, pageIndex int) (pdfRasterRenderedPage, error) {
	var pageWidth float64
	var pageHeight float64
	ok, _, _ := lib.getPageSizeByIndex.Call(
		document,
		uintptr(pageIndex),
		uintptr(unsafe.Pointer(&pageWidth)),
		uintptr(unsafe.Pointer(&pageHeight)),
	)
	if ok == 0 || pageWidth <= 0 || pageHeight <= 0 {
		return pdfRasterRenderedPage{}, fmt.Errorf("FPDF_GetPageSizeByIndex(%d) failed: %s", pageIndex, lib.lastError())
	}
	targetWidth := pdfRasterPreviewTargetWidth
	targetHeight := int(math.Round(pageHeight / pageWidth * float64(targetWidth)))
	if targetHeight <= 0 {
		targetHeight = pdfTextPreviewHeight
	}
	if targetHeight > pdfRasterPreviewMaxHeight {
		scale := float64(pdfRasterPreviewMaxHeight) / float64(targetHeight)
		targetHeight = pdfRasterPreviewMaxHeight
		targetWidth = int(math.Round(float64(targetWidth) * scale))
		if targetWidth <= 0 {
			targetWidth = pdfRasterPreviewTargetWidth
		}
	}

	page, _, _ := lib.loadPage.Call(document, uintptr(pageIndex))
	if page == 0 {
		return pdfRasterRenderedPage{}, fmt.Errorf("FPDF_LoadPage(%d) failed: %s", pageIndex, lib.lastError())
	}
	defer lib.closePage.Call(page)

	stride := targetWidth * 4
	buffer := make([]byte, stride*targetHeight)
	bitmap, _, _ := lib.bitmapCreateEx.Call(
		uintptr(targetWidth),
		uintptr(targetHeight),
		fpdfBitmapBGRA,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(stride),
	)
	if bitmap == 0 {
		return pdfRasterRenderedPage{}, fmt.Errorf("FPDFBitmap_CreateEx(%dx%d) failed: %s", targetWidth, targetHeight, lib.lastError())
	}
	defer func() {
		lib.bitmapDestroy.Call(bitmap)
		runtime.KeepAlive(buffer)
	}()

	if ok, _, _ := lib.bitmapFillRect.Call(bitmap, 0, 0, uintptr(targetWidth), uintptr(targetHeight), uintptr(0xffffffff)); ok == 0 {
		return pdfRasterRenderedPage{}, fmt.Errorf("FPDFBitmap_FillRect(%dx%d) failed: %s", targetWidth, targetHeight, lib.lastError())
	}
	lib.renderPageBitmap.Call(
		bitmap,
		page,
		0,
		0,
		uintptr(targetWidth),
		uintptr(targetHeight),
		0,
		fpdfRenderFlagNone,
	)

	pngBytes, err := bitmapToPNG(buffer, targetWidth, targetHeight, stride)
	if err != nil {
		return pdfRasterRenderedPage{}, fmt.Errorf("encode rendered page %d failed: %w", pageIndex, err)
	}
	return pdfRasterRenderedPage{
		PageIndex: pageIndex,
		Width:     targetWidth,
		Height:    targetHeight,
		PNG:       pngBytes,
	}, nil
}

func bitmapToPNG(source []byte, width int, height int, stride int) ([]byte, error) {
	if width <= 0 || height <= 0 || stride < width*4 || len(source) < stride*height {
		return nil, fmt.Errorf("invalid PDFium bitmap width=%d height=%d stride=%d bytes=%d", width, height, stride, len(source))
	}

	imageRGBA := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		row := source[y*stride:]
		for x := 0; x < width; x++ {
			src := x * 4
			dst := y*imageRGBA.Stride + x*4
			imageRGBA.Pix[dst+0] = row[src+2]
			imageRGBA.Pix[dst+1] = row[src+1]
			imageRGBA.Pix[dst+2] = row[src+0]
			alpha := row[src+3]
			if alpha == 0 {
				alpha = 0xff
			}
			imageRGBA.Pix[dst+3] = alpha
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, imageRGBA); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (lib *pdfiumDLL) lastError() string {
	if lib == nil || lib.getLastError == nil {
		return "unknown"
	}
	value, _, _ := lib.getLastError.Call()
	if value == 0 {
		return "unknown"
	}
	return fmt.Sprintf("code %d", value)
}
