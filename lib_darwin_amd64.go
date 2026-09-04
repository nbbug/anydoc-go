//go:build darwin && amd64 && cgo && !anydoc_dynamic

package anydoc

/*
#cgo LDFLAGS: -L${SRCDIR}/lib/darwin_amd64 -lanydoc_go -lm
*/
import "C"

import _ "embed"

//go:embed lib/darwin_amd64/libanydoc_go.a
var embeddedLib []byte

var embeddedLibName = "libanydoc_go.a"
