package capabilitytransport

import (
	"path/filepath"
	"runtime"
)

// hostExecutable returns the path for a Go-built helper binary. `go build -o
// <path>` appends `.exe` on Windows when the given name has no extension, and
// Windows will not execute an extensionless PE file, so every test that builds
// and launches a helper must use this instead of a bare filepath.Join.
//
// The tests run this helper from the Go test process, which looks up the exact
// path; a missing `.exe` shows up as
// `executable file not found in %PATH%` (#3149 Windows survey).
func hostExecutable(dir, name string) string {
	if runtime.GOOS == "windows" && filepath.Ext(name) == "" {
		name += ".exe"
	}
	return filepath.Join(dir, name)
}
