//go:build darwin && arm64 && cgo && !dynamic

package anydoc

/*
#cgo LDFLAGS: -L${SRCDIR}/lib/darwin_arm64 -lanydoc_go -lm
*/
import "C"

import _ "embed"

//go:embed lib/darwin_arm64/libanydoc_go.a
var embeddedLib []byte

var embeddedLibName = "libanydoc_go.a"
