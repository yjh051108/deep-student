package vfs

import (
	"bytes"
	"errors"
	"fmt"
)

type PDFiumSmokeResult struct {
	TotalPages      int `json:"totalPages"`
	RenderedPages   int `json:"renderedPages"`
	FirstPageWidth  int `json:"firstPageWidth"`
	FirstPageHeight int `json:"firstPageHeight"`
	FirstPageBytes  int `json:"firstPageBytes"`
}

func SmokePDFiumRasterPreview() (PDFiumSmokeResult, error) {
	rendered, err := renderPdfPagesWithPdfium(minimalPDFiumSmokeDocument(), 1)
	if err != nil {
		return PDFiumSmokeResult{}, err
	}
	if rendered.TotalPages <= 0 || len(rendered.Pages) == 0 {
		return PDFiumSmokeResult{}, fmt.Errorf("PDFium smoke rendered no pages: %+v", rendered)
	}
	page := rendered.Pages[0]
	if page.Width <= 0 || page.Height <= 0 {
		return PDFiumSmokeResult{}, fmt.Errorf("PDFium smoke rendered invalid page size %dx%d", page.Width, page.Height)
	}
	if !bytes.HasPrefix(page.PNG, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return PDFiumSmokeResult{}, errors.New("PDFium smoke rendered bytes without a PNG signature")
	}
	return PDFiumSmokeResult{
		TotalPages:      rendered.TotalPages,
		RenderedPages:   len(rendered.Pages),
		FirstPageWidth:  page.Width,
		FirstPageHeight: page.Height,
		FirstPageBytes:  len(page.PNG),
	}, nil
}

func minimalPDFiumSmokeDocument() []byte {
	return []byte("%PDF-1.0\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n3 0 obj<</Type/Page/MediaBox[0 0 612 792]/Parent 2 0 R>>endobj\nxref\n0 4\n0000000000 65535 f \n0000000009 00000 n \n0000000056 00000 n \n0000000107 00000 n \ntrailer<</Size 4/Root 1 0 R>>\nstartxref\n172\n%%EOF")
}
