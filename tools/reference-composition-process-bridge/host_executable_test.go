package compositionprocessbridge

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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

// temporaryBuildDir returns a directory the reference launcher accepts as the
// home of an executable. The launcher refuses an executable path that contains
// whitespace (internal/plugin/runtime/process_stdio.go), so a host whose
// temporary directory contains a space cannot run these tests. That is a
// property of the host, not of the operating system, so the tests skip only
// when the path actually has one (#3460).
func temporaryBuildDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if strings.ContainsAny(dir, " \t\r\n") {
		t.Skipf("the reference launcher refuses whitespace in an executable path and this host's temporary directory contains some: %q", dir)
	}
	return dir
}
