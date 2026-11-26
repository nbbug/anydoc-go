//go:build windows && arm64 && cgo

package anydoc

/*
#cgo LDFLAGS: -L${SRCDIR}/lib/windows_arm64 -lanydoc_go
*/
import "C"

// This build links the shared library from the GitHub release
// (anydoc-dynlib-windows_arm64.tar.gz). Extract the tarball so
// ${SRCDIR}/lib/windows_arm64 holds anydoc_go.dll, or point CGO_LDFLAGS at the
// directory that holds it:
//
//	CGO_LDFLAGS="-L/path/to/lib/windows_arm64" go build ./...
//
// At run time the DLL must be found too: place anydoc_go.dll next to
// the executable or on PATH.
//
// See the README's "Prebuilt libraries" section.
