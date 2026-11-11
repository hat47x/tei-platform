package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

var (
	cliBinaryOnce sync.Once
	cliBinaryPath string
	cliBinaryErr  error
)

// sharedCLIBinary builds the CLI once per test binary and returns its path.
//
// The failure-classification cases each invoke a real process, because the point
// is the exit code and the JSON document, not an in-process call. Building once
// per case meant eleven `go build` runs for one test: cmd/tei took 564s alone and
// hit the 10-minute limit under load (round 65). The binary does not change
// between cases within one run, so it is built once and reused.
func sharedCLIBinary(t *testing.T) string {
	t.Helper()
	cliBinaryOnce.Do(func() {
		dir, err := os.MkdirTemp("", "tei-cli-shared-")
		if err != nil {
			cliBinaryErr = err
			return
		}
		path := filepath.Join(dir, "tei")
		if os.PathSeparator == '\\' {
			path += ".exe"
		}
		out, err := exec.Command("go", "build", "-o", path, ".").CombinedOutput()
		if err != nil {
			cliBinaryErr = err
			t.Logf("build CLI: %v\n%s", err, out)
			return
		}
		cliBinaryPath = path
	})
	if cliBinaryErr != nil {
		t.Fatalf("build CLI once: %v", cliBinaryErr)
	}
	return cliBinaryPath
}
