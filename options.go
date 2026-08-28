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

	// OcrCustom hands the whole document to the OcrHandler this Options
	// carries, so callers can plug in their own OCR: a local model (for
	// example ONNX), a local OCR service, or any other API. OcrCustom with
	// a nil OcrHandler reports an unsupported error.
	OcrCustom OcrMode = "custom"
)

// OcrHandler extracts Markdown from a PDF whose pages need OCR. Implement it
// to plug in a local OCR model or any OCR service of your own; OcrHandlerFunc
// adapts a plain function. The handler receives the whole PDF — the same
// document the hosted mode uploads — and the errors it returns are passed
// through unchanged.
type OcrHandler interface {
	OcrMarkdown(pdf []byte) (string, error)
}

// OcrHandlerFunc adapts a plain function to OcrHandler.
type OcrHandlerFunc func(pdf []byte) (string, error)

// OcrMarkdown implements OcrHandler.
func (f OcrHandlerFunc) OcrMarkdown(pdf []byte) (string, error) {
	return f(pdf)
}

// Options controls conversion behavior beyond format selection. The nil or
// zero value is the default: PDFs that need OCR are rejected with needs_ocr.
type Options struct {
	// Ocr decides what happens to a PDF whose pages need OCR: OcrReject
	// (the zero value) fails with a needs_ocr error naming the pages;
	// OcrHosted sends the document to Firecrawl Parse; OcrCustom hands it
	// to OcrHandler below.
	Ocr OcrMode

	// OcrHandler extracts the Markdown when Ocr is OcrCustom. Ignored for
	// the other modes.
	OcrHandler OcrHandler

	// APIKey authenticates the Firecrawl Parse request, raising the keyless
	// rate limits. Empty falls back to the FIRECRAWL_API_KEY environment
	// variable, then keyless.
	APIKey string

	// APIURL overrides the Firecrawl Parse endpoint. Empty falls back to
	// the FIRECRAWL_API_URL environment variable, then
	// https://api.firecrawl.dev.
	APIURL string
}
