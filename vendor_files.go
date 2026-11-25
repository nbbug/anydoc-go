package anydoc

// This file exists so `go mod vendor` captures include/anydoc.h.
//
// The cgo preambles reference the header with `#include`, and vendoring only
// copies subdirectories of a package when a //go:embed directive in a
// buildable file names them. The platform files carry a `cgo` build
// constraint, so a vendor run with cgo disabled would drop the header (and
// stub builds need none of it). This untagged directive pins the header for
// every build configuration. The variable is never referenced — the linker
// eliminates the bytes from binaries.
//
// Static and shared libraries are downloaded from the GitHub release, not
// embedded or vendored. Extract one kind into lib/<platform>/ under the
// vendored module, or point CGO_LDFLAGS at it (see the README, "Prebuilt
// libraries").

import _ "embed"

//go:embed include/anydoc.h
var _vendorHeader []byte
