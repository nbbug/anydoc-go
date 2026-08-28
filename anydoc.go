//go:build cgo && ((linux && (amd64 || arm64)) || (darwin && (amd64 || arm64)) || (windows && amd64))

package anydoc

/*
#cgo CFLAGS: -I${SRCDIR}/include
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include "include/anydoc.h"
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

// call runs one ABI entry point and turns its status code into an error.
//
// The goroutine is pinned for the duration because the ABI reports the error
// message through a thread-local slot that a *second* call
// (`anydoc_last_error`) reads. Go is free to resume a goroutine on a different
// OS thread after a cgo call returns, and a conversion is long enough for that
// to happen: the message would then be missing, or belong to another
// conversion that ran on the thread we landed on.
func call(entry func() C.int) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if code := entry(); code != errOK {
		return convertError(code)
	}
	return nil
}

// ToMarkdown converts a document file to Markdown. The format is detected
// from the file content; the extension is the fallback for signature-less
// formats (CSV) and unrecognizable containers.
func ToMarkdown(path string) (string, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var out *C.char
	var outLen C.uintptr_t
	if err := call(func() C.int { return C.anydoc_to_markdown(cpath, &out, &outLen) }); err != nil {
		return "", err
	}
	defer C.anydoc_string_free(out)
	markdown, err := cStringN(out, outLen)
	if err != nil {
		return "", err
	}
	return markdown, nil
}

// ToMarkdownBytes converts an in-memory document to Markdown. Pass a Format
// to select the parser, or nil to detect it from the content, which
// signature-less formats (CSV) have to name explicitly.
func ToMarkdownBytes(data []byte, format *Format) (string, error) {
	if len(data) == 0 {
		return "", &ConvertError{Kind: "unsupported", Detail: "empty input"}
	}
	tag := C.int(formatNone)
	if format != nil {
		tag = C.int(formatToTag(*format))
		if tag == C.int(formatNone) {
			return "", &ConvertError{Kind: "unknown_format", Detail: "unknown format: " + string(*format)}
		}
	}
	var out *C.char
	var outLen C.uintptr_t
	if err := call(func() C.int {
		return C.anydoc_to_markdown_bytes(
			(*C.uint8_t)(unsafe.Pointer(&data[0])), C.uintptr_t(len(data)), tag, &out, &outLen,
		)
	}); err != nil {
		return "", err
	}
	defer C.anydoc_string_free(out)
	markdown, err := cStringN(out, outLen)
	if err != nil {
		return "", err
	}
	return markdown, nil
}

// ToMarkdownWithAssetLinks converts an in-memory document to Markdown with
// embedded images rewritten as `![alt](images/image-N.ext)` so they keep
// their original positions. The Markdown itself is produced by anydoc's
// official GFM serializer. PDF has no document model and is converted the
// same way as ToMarkdownBytes.
func ToMarkdownWithAssetLinks(data []byte, format *Format) (string, error) {
	if len(data) == 0 {
		return "", &ConvertError{Kind: "unsupported", Detail: "empty input"}
	}
	tag := C.int(formatNone)
	if format != nil {
		tag = C.int(formatToTag(*format))
		if tag == C.int(formatNone) {
			return "", &ConvertError{Kind: "unknown_format", Detail: "unknown format: " + string(*format)}
		}
	}
	var out *C.char
	var outLen C.uintptr_t
	if err := call(func() C.int {
		return C.anydoc_to_markdown_with_asset_links(
			(*C.uint8_t)(unsafe.Pointer(&data[0])), C.uintptr_t(len(data)), tag, &out, &outLen,
		)
	}); err != nil {
		return "", err
	}
	defer C.anydoc_string_free(out)
	markdown, err := cStringN(out, outLen)
	if err != nil {
		return "", err
	}
	return markdown, nil
}

// ToDocument parses an in-memory document into the document model, which also
// carries the embedded assets. Pass a Format to select the parser, or nil to
// detect it from the content.
//
// Unsupported for PDF: PDF conversion produces Markdown directly and has no
// document-model form; use ToMarkdownBytes.
func ToDocument(data []byte, format *Format) (*Document, error) {
	if len(data) == 0 {
		return nil, &ConvertError{Kind: "unsupported", Detail: "empty input"}
	}
	tag := C.int(formatNone)
	if format != nil {
		tag = C.int(formatToTag(*format))
		if tag == C.int(formatNone) {
			return nil, &ConvertError{Kind: "unknown_format", Detail: "unknown format: " + string(*format)}
		}
	}
	var buf *C.uint8_t
	var bufLen C.uintptr_t
	if err := call(func() C.int {
		return C.anydoc_to_document(
			(*C.uint8_t)(unsafe.Pointer(&data[0])), C.uintptr_t(len(data)), tag, &buf, &bufLen,
		)
	}); err != nil {
		return nil, err
	}
	defer C.anydoc_buffer_free(buf, bufLen)
	if buf == nil || bufLen == 0 {
		return &Document{}, nil
	}
	// Copy the C buffer into a Go-owned slice before decoding, so the C
	// buffer can be freed immediately rather than tying its lifetime to the
	// decoded Document.
	raw, err := cGoBytes(unsafe.Pointer(buf), bufLen)
	if err != nil {
		return nil, err
	}
	return decodeDocument(raw)
}

// FormatFromBytes detects the format from the content itself: the signature
// and identity each container specification designates (PDF header, RTF open
// group, OLE stream names, ZIP package mimetype/content types). Plain-text
// formats (CSV) carry no signature and return the zero Format and ok=false;
// so does anything unrecognized.
func FormatFromBytes(data []byte) (f Format, ok bool) {
	if len(data) == 0 {
		return "", false
	}
	var tag C.int
	code := C.anydoc_format_from_bytes((*C.uint8_t)(unsafe.Pointer(&data[0])), C.uintptr_t(len(data)), &tag)
	if code != C.int(errOK) {
		return "", false
	}
	return tagToFormat(int(tag))
}

// FormatFromExtension maps a bare extension (no leading dot) to a Format,
// matched case-insensitively. ok is false for anything unrecognized.
func FormatFromExtension(ext string) (f Format, ok bool) {
	cext := C.CString(ext)
	defer C.free(unsafe.Pointer(cext))
	var tag C.int
	code := C.anydoc_format_from_extension(cext, &tag)
	if code != C.int(errOK) {
		return "", false
	}
	return tagToFormat(int(tag))
}

// FormatFromPath maps a path's extension to a Format. ok is false when the
// path has no extension or names nothing recognized.
func FormatFromPath(path string) (f Format, ok bool) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var tag C.int
	code := C.anydoc_format_from_path(cpath, &tag)
	if code != C.int(errOK) {
		return "", false
	}
	return tagToFormat(int(tag))
}

// convertError builds a ConvertError from an ABI error code, pulling the
// detail message from the thread-local last-error slot. Only call it from
// inside call(), where the OS thread is pinned to the one the ABI entry ran
// on.
func convertError(code C.int) error {
	if code == errOK {
		return nil
	}
	kind := errorKind(code)
	var needsOcr *NeedsOcrError
	if code == errNeedsOcr {
		// Before lastError(): the pages are borrowed from the same slot that
		// lastError takes, and must be copied while this thread holds it.
		needsOcr = lastErrorOcrPages()
	}
	detail := lastError()
	return &ConvertError{Kind: kind, Detail: detail, NeedsOcr: needsOcr}
}

func errorKind(code C.int) string {
	switch code {
	case errUnsupported:
		return "unsupported"
	case errMalformed:
		return "malformed"
	case errEncrypted:
		return "encrypted"
	case errResourceLimit:
		return "resource_limit"
	case errMissingPart:
		return "missing_part"
	case errIO:
		return "io"
	case errPDFNoModel:
		return "pdf_no_model"
	case errInvalidArg:
		return "invalid_argument"
	case errUnknownFormat:
		return "unknown_format"
	case errNeedsOcr:
		return "needs_ocr"
	default:
		return fmt.Sprintf("unknown_error(%d)", int(code))
	}
}

// lastError returns the human-readable message the last ABI call stashed on
// the current OS thread, or "" when there was none. The C side returns a
// freshly allocated string the caller must free.
func lastError() string {
	ptr := C.anydoc_last_error()
	if ptr == nil {
		return ""
	}
	defer C.anydoc_string_free(ptr)
	return cString(ptr)
}

// lastErrorOcrPages copies the pages needing OCR out of the thread-local slot
// after an ABI call returned errNeedsOcr. Only call it from inside call(),
// where the OS thread is pinned to the one the ABI entry ran on, and before
// lastError(): the pages pointer is borrowed from the slot that lastError
// takes, so the copy must happen first.
func lastErrorOcrPages() *NeedsOcrError {
	var pages *C.uint32_t
	var n C.uintptr_t
	var pageCount C.uint32_t
	if C.anydoc_last_error_ocr_pages(&pages, &n, &pageCount) != errOK || pages == nil || n == 0 {
		return nil
	}
	pageNums := make([]uint32, int(n))
	for i, page := range unsafe.Slice(pages, int(n)) {
		pageNums[i] = uint32(page)
	}
	return &NeedsOcrError{Pages: pageNums, PageCount: uint32(pageCount)}
}

// cStringN reads a length-prefixed C buffer as a Go string. The bytes are
// copied, so the caller may free the C buffer immediately after.
func cStringN(s *C.char, n C.uintptr_t) (string, error) {
	if s == nil || n == 0 {
		return "", nil
	}
	b, err := cGoBytes(unsafe.Pointer(s), n)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func cGoBytes(ptr unsafe.Pointer, n C.uintptr_t) ([]byte, error) {
	// C.GoBytes accepts a C int length even on 64-bit Go builds.
	if uint64(n) > 1<<31-1 {
		return nil, &ConvertError{Kind: "resource_limit", Detail: fmt.Sprintf("FFI output is too large for Go: %d bytes", n)}
	}
	return C.GoBytes(ptr, C.int(n)), nil
}

// cString reads a NUL-terminated C string as a Go string.
func cString(s *C.char) string {
	if s == nil {
		return ""
	}
	return C.GoString(s)
}
