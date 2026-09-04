//go:build linux && arm64 && cgo && !anydoc_dynamic

package anydoc

/*
#cgo LDFLAGS: -L${SRCDIR}/lib/linux_arm64 -lanydoc_go -lm -lstdc++ -ldl -lpthread
*/
import "C"

import _ "embed"

//go:embed lib/linux_arm64/libanydoc_go.a
var embeddedLib []byte

var embeddedLibName = "libanydoc_go.a"
