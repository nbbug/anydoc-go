package anydoc

import (
	"errors"
)

// ABI error codes mirroring include/anydoc.h (ERR_*). Stable across versions.
// They are plain Go constants here — the C values are pinned by the committed
// header, and TestWireConstsMatchHeader verifies the two still agree whenever
// a cgo build runs — so the error and model types stay pure Go and compile on
// every platform, including stub builds.
const (
	errOK            = 0
	errUnsupported   = 1
	errMalformed     = 2
	errEncrypted     = 3
	errResourceLimit = 4
	errMissingPart   = 5
	errIO            = 6
	errPDFNoModel    = 7
	errInvalidArg    = 8
	errUnknownFormat = 9
	errNeedsOcr      = 10
)

// ConvertError is the typed error every conversion function returns. It
// carries the same variant kind the Node and Python bindings expose
// ("unsupported", "malformed", "encrypted", ...) plus the crate's
// human-readable detail.
type ConvertError struct {
	// Kind is the lowercase variant name, matching the Node and Python
	// bindings: "unsupported", "malformed", "encrypted", "resource_limit",
	// "missing_part", "io", "pdf_no_model", "needs_ocr". Go also reports
	// "unknown_format" for an invalid explicit Format.
	Kind string
	// Detail is the crate's Display output for the error, e.g.
	// "pages 2, 5-7 of 12 need OCR".
	Detail string
	// NeedsOcr is non-nil exactly when Kind == "needs_ocr": some pages of a
	// PDF are scanned or image-only and were not converted. It mirrors the
	// Node binding's `pages`/`pageCount` error fields.
	NeedsOcr *NeedsOcrError
}

// NeedsOcrError names the pages of a PDF that need OCR. Pages are 1-indexed,
// matching anydoc's reporting and the Node binding.
type NeedsOcrError struct {
	// Pages are the 1-indexed page numbers that need OCR.
	Pages []uint32
	// PageCount is the number of pages in the document.
	PageCount uint32
}

func (e *ConvertError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Detail == "" {
		return "anydoc: " + e.Kind
	}
	return "anydoc: " + e.Kind + ": " + e.Detail
}

// UnavailableError reports that the static library cannot be used in the
// current build: cgo was disabled, or the platform has no packaged archive.
// Every exported function returns it (or a nil/zero result) in that case, so
// the package always compiles — the stub simply fails at run time with an
// explanation.
type UnavailableError struct {
	// Reason describes what is missing and how to fix it.
	Reason string
}

func (e *UnavailableError) Error() string {
	return "anydoc: unavailable: " + e.Reason
}

// IsUnavailable reports whether err is an UnavailableError. Use it to detect
// stub builds (for example to fall back to another converter) without string
// matching.
func IsUnavailable(err error) bool {
	var ue *UnavailableError
	return errors.As(err, &ue)
}

func unavailableError(reason string) error {
	return &UnavailableError{Reason: reason}
}

// unavailableDetail is the message stub builds return. It names the supported
// matrix so the failure is self-explanatory.
const unavailableDetail = "no prebuilt anydoc static library for this build: " +
	"the module packages archives for linux/amd64, linux/arm64, darwin/amd64, " +
	"darwin/arm64, and windows/amd64, and requires CGO_ENABLED=1. " +
	"Build with cgo on one of these platforms, or build the archive from " +
	"source for another target (see scripts/build-all.sh)."
