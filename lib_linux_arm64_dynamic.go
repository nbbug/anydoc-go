//go:build anydoc_dynamic && linux && arm64 && cgo

package anydoc

/*
#cgo LDFLAGS: -L${SRCDIR}/dynlib/linux_arm64 -lanydoc_go
*/
import "C"

// Dynamic build: no archive is embedded. The shared library must be reachable
// through the cgo search path — either placed at dynlib/linux_arm64 inside a vendored
// copy of the module, or via CGO_LDFLAGS="-L<your-dir>/linux_arm64" (cgo appends the
// environment flags after this file's). See the README's dynamic-library
// section.
