//go:build dynamic && linux && amd64 && cgo

package anydoc

/*
#cgo LDFLAGS: -L${SRCDIR}/dynlib/linux_amd64 -lanydoc_go
*/
import "C"

// Dynamic build: no archive is embedded. The shared library must be reachable
// through the cgo search path — either placed at dynlib/linux_amd64 inside a vendored
// copy of the module, or via CGO_LDFLAGS="-L<your-dir>/linux_amd64" (cgo appends the
// environment flags after this file's). See the README's dynamic-library
// section.
