//go:build !windows

package vfs

import "errors"

func renderPdfPagesWithPdfium(_ []byte, _ int) (pdfRasterRenderResult, error) {
	return pdfRasterRenderResult{}, errors.New("PDFium raster preview is unavailable on this platform")
}
