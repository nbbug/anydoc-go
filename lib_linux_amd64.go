//go:build linux && amd64 && cgo && !anydoc_dynamic

package anydoc

/*
#cgo LDFLAGS: -L${SRCDIR}/lib/linux_amd64 -lanydoc_go -lm -lstdc++ -ldl -lpthread
*/
import "C"

// Static build: the archive is not embedded in the module. Download
// anydoc-static-linux_amd64.tar.gz from the GitHub release and extract it so
// the ${SRCDIR}/lib/linux_amd64 search path above finds libanydoc_go.a, or point
// CGO_LDFLAGS at the directory that holds it:
//
//   CGO_LDFLAGS="-L/path/to/lib/linux_amd64" go build ./...
//
// Extract only one kind (static or shared) into a given lib directory: when
// both are present, the linker prefers the shared library.
//
// See the README's "Prebuilt libraries" section.
