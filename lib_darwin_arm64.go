//go:build darwin && arm64 && cgo

package anydoc

/*
#cgo LDFLAGS: -L${SRCDIR}/lib/darwin_arm64 -lanydoc_go -Wl,-rpath,${SRCDIR}/lib/darwin_arm64
*/
import "C"

// This build links the shared library from the GitHub release
// (anydoc-dynlib-darwin_arm64.tar.gz). Extract the tarball so
// ${SRCDIR}/lib/darwin_arm64 holds libanydoc_go.dylib, or point CGO_LDFLAGS at the
// directory that holds it:
//
//	CGO_LDFLAGS="-L/path/to/lib/darwin_arm64" go build ./...
//
// The link line records an rpath to ${SRCDIR}/lib/darwin_arm64, so a library
// extracted inside the module is found at run time too. A copy shipped
// elsewhere must be on DYLD_LIBRARY_PATH or in a standard loader path.
//
// See the README's "Prebuilt libraries" section.
