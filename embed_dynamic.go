//go:build anydoc_dynamic && cgo && ((linux && (amd64 || arm64)) || (darwin && (amd64 || arm64)) || (windows && (amd64 || arm64)))

package anydoc

// Dynamic builds ship no embedded archive: the shared library comes from the
// GitHub release, and the cgo search path locates it (see the dynamic
// platform files and the README's dynamic-library section).

// EmbeddedLib returns nil in dynamic builds — nothing is embedded.
func EmbeddedLib() []byte {
	return nil
}

// ExtractLib has nothing to extract in dynamic builds.
func ExtractLib(dir string) (string, error) {
	return "", &UnavailableError{Reason: "anydoc_dynamic build: nothing is embedded; " +
		"download the shared library for your platform from the GitHub release " +
		"and see the README's dynamic-library section"}
}
