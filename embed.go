//go:build cgo && ((linux && (amd64 || arm64)) || (darwin && (amd64 || arm64)) || (windows && amd64))

package anydoc

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// EmbeddedLib returns the bytes of the static archive packaged for the
// current platform.
//
// The archive is embedded so the Go module toolchain treats it as a declared
// dependency of the package: `go mod vendor`, module zips, and `go list`
// cannot silently drop it. Linking itself happens at build time through the
// cgo LDFLAGS in the platform-specific files; this accessor exists for
// diagnostics, redistribution audits, and tooling that wants the archive on
// disk. Callers that never call it pay nothing: the linker eliminates the
// unreferenced ~30 MB of bytes from the final binary.
func EmbeddedLib() []byte {
	return embeddedLib
}

// ExtractLib writes the embedded archive into dir and returns the path of the
// file it created. The file name records the module version and target
// platform, so several platforms can be collected in one directory.
func ExtractLib(dir string) (string, error) {
	if len(embeddedLib) == 0 {
		return "", &UnavailableError{Reason: fmt.Sprintf(
			"no embedded static library for %s/%s", runtime.GOOS, runtime.GOARCH)}
	}
	name := fmt.Sprintf("anydoc-%s-%s_%s-%s", Version, runtime.GOOS, runtime.GOARCH, embeddedLibName)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, embeddedLib, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
