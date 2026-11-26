//go:build linux && amd64 && cgo

package anydoc

/*
#cgo LDFLAGS: -L${SRCDIR}/lib/linux_amd64 -lanydoc_go -Wl,-rpath,${SRCDIR}/lib/linux_amd64
*/
import "C"

// This build links the shared library from the GitHub release
// (anydoc-dynlib-linux_amd64.tar.gz). Extract the tarball so
// ${SRCDIR}/lib/linux_amd64 holds libanydoc_go.so, or point CGO_LDFLAGS at the
// directory that holds it:
//
//	CGO_LDFLAGS="-L/path/to/lib/linux_amd64" go build ./...
//
// The link line records an rpath to ${SRCDIR}/lib/linux_amd64, so a library
// extracted inside the module is found at run time too. A copy shipped
// elsewhere must be on LD_LIBRARY_PATH or in a standard loader path.
//
// See the README's "Prebuilt libraries" section.
