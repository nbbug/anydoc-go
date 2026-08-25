package anydoc

// This file exists so `go mod vendor` captures include/anydoc.h.
//
// The cgo preambles reference the header with `#include`, and the archive
// files with //go:embed directives inside the platform-specific files.
// Vendoring only copies subdirectories of a package when a go:embed directive
// in a buildable file names them: the platform files carry a `cgo` build
// constraint, so a vendor run with cgo disabled would drop the header (and
// stub builds need none of it). This untagged directive pins the header for
// every build configuration. The variable is never referenced — the linker
// eliminates the bytes from binaries.
//
// The archives themselves are captured by vendoring with the default
// cgo-enabled environment, which vendors the lib/<platform>/ trees for all
// platforms (see the README, "Vendoring").

import _ "embed"

//go:embed include/anydoc.h
var _vendorHeader []byte
