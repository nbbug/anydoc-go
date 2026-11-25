//go:build anydoc_dynamic && linux && amd64 && cgo

package anydoc

/*
#cgo LDFLAGS: -L${SRCDIR}/lib/linux_amd64 -lanydoc_go
*/
import "C"

// Dynamic build: link the shared library downloaded from the GitHub release
// (anydoc-dynlib-linux_amd64.tar.gz). The library must be reachable through
// the cgo search path: either extract it into lib/linux_amd64 inside a
// checkout or vendored module, or point CGO_LDFLAGS at the directory that
// holds it:
//
//   CGO_LDFLAGS="-L/path/to/lib/linux_amd64" go build -tags anydoc_dynamic ./...
//
// Extract only one kind (static or shared) into a given lib/linux_amd64
// directory: when both are present, the linker prefers the shared library.
//
// At run time, put the directory on LD_LIBRARY_PATH (Linux) or
// DYLD_LIBRARY_PATH (macOS), or place the DLL next to the executable or on
// PATH (Windows). See the README's "Prebuilt libraries" section.
