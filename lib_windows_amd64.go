//go:build windows && amd64 && cgo && !anydoc_dynamic

package anydoc

/*
#cgo LDFLAGS: -L${SRCDIR}/lib/windows_amd64 -lanydoc_go
*/
import "C"

// Static build: the archive is not embedded in the module. Download
// anydoc-static-windows_amd64.tar.gz from the GitHub release and extract it so
// the ${SRCDIR}/lib/windows_amd64 search path above finds anydoc_go.lib, or point
// CGO_LDFLAGS at the directory that holds it:
//
//   CGO_LDFLAGS="-L/path/to/lib/windows_amd64" go build ./...
//
// Extract only one kind (static or shared) into a given lib directory: when
// both are present, the linker prefers the shared library.
//
// Windows uses the MSVC toolchain (x86_64-pc-windows-msvc or
// aarch64-pc-windows-msvc), so the static archive is the COFF
// anydoc_go.lib rather than a GNU .a. Go's cgo still links with mingw
// gcc, whose GNU ld reads MSVC .lib archives fine; users need mingw-w64,
// not Visual Studio.
//
// See the README's "Prebuilt libraries" section.
