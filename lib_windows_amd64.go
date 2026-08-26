//go:build windows && amd64 && cgo

package anydoc

/*
#cgo LDFLAGS: -L${SRCDIR}/lib/windows_amd64 -lanydoc_go
*/
import "C"

import _ "embed"

// The archive is built with the MSVC toolchain (x86_64-pc-windows-msvc), so
// it is a COFF .lib rather than a GNU .a. Go's cgo links with mingw gcc,
// whose GNU ld reads MSVC .lib archives fine — this is the same pairing the
// WeKnora binding ships.
//
//go:embed lib/windows_amd64/anydoc_go.lib
var embeddedLib []byte

var embeddedLibName = "anydoc_go.lib"
