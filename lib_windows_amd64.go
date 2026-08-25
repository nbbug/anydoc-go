//go:build windows && amd64 && cgo

package anydoc

/*
#cgo LDFLAGS: -L${SRCDIR}/lib/windows_amd64 -lanydoc_go
*/
import "C"

import _ "embed"

//go:embed lib/windows_amd64/libanydoc_go.a
var embeddedLib []byte
