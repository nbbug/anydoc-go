//go:build linux && amd64 && cgo && !dynamic

package anydoc

/*
#cgo LDFLAGS: -L${SRCDIR}/lib/linux_amd64 -lanydoc_go -lm -lstdc++ -ldl -lpthread
*/
import "C"

import _ "embed"

//go:embed lib/linux_amd64/libanydoc_go.a
var embeddedLib []byte

var embeddedLibName = "libanydoc_go.a"
