package anydoc

// OcrMode decides what happens to a PDF whose pages need OCR, mirroring the
// Node and Python bindings' `ocr` option. anydoc itself never does OCR.
type OcrMode string

const (
	// OcrReject is the default: the conversion fails with a needs_ocr error
	// naming the pages that need OCR. The document never leaves the machine.
	OcrReject OcrMode = "reject"

	// OcrHosted sends the whole document to Firecrawl Parse instead, which
	// extracts the text. The document leaves the machine; see the README
	// security notes.
	OcrHosted OcrMode = "hosted"
)

// Options controls conversion behavior beyond format selection. The nil or
// zero value is the default: PDFs that need OCR are rejected with needs_ocr.
type Options struct {
	// Ocr decides what happens to a PDF whose pages need OCR: OcrReject
	// (the zero value) fails with a needs_ocr error naming the pages;
	// OcrHosted sends the document to Firecrawl Parse.
	Ocr OcrMode

	// APIKey authenticates the Firecrawl Parse request, raising the keyless
	// rate limits. Empty falls back to the FIRECRAWL_API_KEY environment
	// variable, then keyless.
	APIKey string

	// APIURL overrides the Firecrawl Parse endpoint. Empty falls back to
	// the FIRECRAWL_API_URL environment variable, then
	// https://api.firecrawl.dev.
	APIURL string
}
