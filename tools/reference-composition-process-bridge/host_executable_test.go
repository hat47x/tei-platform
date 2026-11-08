package compositionprocessbridge

import (
	"path/filepath"
	"runtime"
)

// hostExecutable returns the path for a Go-built helper binary. `go build -o
// <path>` appends `.exe` on Windows when the given name has no extension, and
// Windows will not execute an extensionless PE file. A test that builds and
// launches a helper must therefore use this helper; otherwise the launch fails
// with `executable file not found in %PATH%` even though the build succeeded
// (#3149 Windows survey).
func hostExecutable(dir, name string) string {
	if runtime.GOOS == "windows" && filepath.Ext(name) == "" {
		name += ".exe"
	}
	return filepath.Join(dir, name)
}
