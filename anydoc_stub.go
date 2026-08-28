//go:build !cgo || !((linux && (amd64 || arm64)) || (darwin && (amd64 || arm64)) || (windows && amd64))

package anydoc

// This file is the fallback for builds that cannot link the packaged static
// library: cgo disabled, or a platform with no prebuilt archive. Every
// exported function keeps its signature and fails with a friendly
// UnavailableError instead of a compile error, so code that imports the
// package always builds. Check IsUnavailable to detect this at run time.

// ToMarkdown converts a document file to Markdown.
func ToMarkdown(path string) (string, error) {
	return "", unavailableError(unavailableDetail)
}

// ToMarkdownBytes converts an in-memory document to Markdown.
func ToMarkdownBytes(data []byte, format *Format) (string, error) {
	return "", unavailableError(unavailableDetail)
}

// ToMarkdownWithOptions converts a document file to Markdown with extended
// behavior; see the cgo implementation.
func ToMarkdownWithOptions(path string, opts *Options) (string, error) {
	return "", unavailableError(unavailableDetail)
}

// ToMarkdownBytesWithOptions converts an in-memory document to Markdown with
// extended behavior; see the cgo implementation.
func ToMarkdownBytesWithOptions(data []byte, format *Format, opts *Options) (string, error) {
	return "", unavailableError(unavailableDetail)
}

// ToMarkdownWithAssetLinks converts an in-memory document to Markdown with
// embedded images rewritten as `![alt](images/image-N.ext)`.
func ToMarkdownWithAssetLinks(data []byte, format *Format) (string, error) {
	return "", unavailableError(unavailableDetail)
}

// ToDocument parses an in-memory document into the document model.
func ToDocument(data []byte, format *Format) (*Document, error) {
	return nil, unavailableError(unavailableDetail)
}

// FormatFromBytes detects the format from the content itself.
func FormatFromBytes(data []byte) (f Format, ok bool) {
	return "", false
}

// FormatFromExtension maps a bare extension to a Format.
func FormatFromExtension(ext string) (f Format, ok bool) {
	return "", false
}

// FormatFromPath maps a path's extension to a Format.
func FormatFromPath(path string) (f Format, ok bool) {
	return "", false
}

// EmbeddedLib returns the bytes of the static archive for the current
// platform. Stub builds package no archive.
func EmbeddedLib() []byte {
	return nil
}

// ExtractLib writes the embedded archive into dir. Stub builds have no
// archive to write.
func ExtractLib(dir string) (string, error) {
	return "", unavailableError(unavailableDetail)
}
