//go:build !cgo || !((linux && (amd64 || arm64)) || (darwin && (amd64 || arm64)) || (windows && amd64))

package anydoc

import (
	"strings"
	"testing"
)

// TestStubUnavailable runs on every build where the packaged library cannot
// link (cgo disabled or unsupported platform). The package must compile and
// every function must fail with a helpful UnavailableError instead of
// panicking or linking against a missing archive.
func TestStubUnavailable(t *testing.T) {
	if _, err := ToMarkdown("anything"); !IsUnavailable(err) {
		t.Errorf("ToMarkdown: expected UnavailableError, got %v", err)
	}
	if _, err := ToMarkdownBytes([]byte("x"), nil); !IsUnavailable(err) {
		t.Errorf("ToMarkdownBytes: expected UnavailableError, got %v", err)
	}
	if _, err := ToMarkdownWithAssetLinks([]byte("x"), nil); !IsUnavailable(err) {
		t.Errorf("ToMarkdownWithAssetLinks: expected UnavailableError, got %v", err)
	}
	if doc, err := ToDocument([]byte("x"), nil); doc != nil || !IsUnavailable(err) {
		t.Errorf("ToDocument: expected (nil, UnavailableError), got (%v, %v)", doc, err)
	}
	if f, ok := FormatFromBytes([]byte("x")); f != "" || ok {
		t.Errorf("FormatFromBytes: expected zero value and ok=false, got %q, %v", f, ok)
	}
	if f, ok := FormatFromExtension("docx"); f != "" || ok {
		t.Errorf("FormatFromExtension: expected zero value and ok=false, got %q, %v", f, ok)
	}
	if f, ok := FormatFromPath("/a/b.docx"); f != "" || ok {
		t.Errorf("FormatFromPath: expected zero value and ok=false, got %q, %v", f, ok)
	}
	if lib := EmbeddedLib(); lib != nil {
		t.Errorf("EmbeddedLib: expected nil in stub builds, got %d bytes", len(lib))
	}
	if _, err := ExtractLib(t.TempDir()); !IsUnavailable(err) {
		t.Errorf("ExtractLib: expected UnavailableError, got %v", err)
	}

	// The message should name the fix, so users are not left guessing.
	_, err := ToMarkdown("anything")
	if !strings.Contains(err.Error(), "CGO_ENABLED=1") {
		t.Errorf("error should mention CGO_ENABLED=1, got: %v", err)
	}
}
