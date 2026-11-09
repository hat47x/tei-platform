package architecturecheck

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// trackedExecutableExtensions are build outputs that must not be tracked.
// A source tree can rebuild them; committing them makes every clone heavier
// and hides the source they came from (#2947).
var trackedExecutableExtensions = []string{
	".exe", ".dll", ".so", ".dylib", ".class", ".jar", ".wasm", ".o", ".a",
}

// TestNoTrackedBuildOutputs lists tracked files from git itself rather than
// the working tree, so an ignored-but-tracked file is still caught.
func TestNoTrackedBuildOutputs(t *testing.T) {
	root := filepath.Join("..", "..")
	cmd := exec.Command("git", "-C", root, "ls-files", "-z")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git ls-files: %v\n%s", err, stderr.String())
	}
	var offenders []string
	for _, name := range strings.Split(strings.TrimRight(string(out), "\x00"), "\x00") {
		if name == "" {
			continue
		}
		lower := strings.ToLower(name)
		for _, extension := range trackedExecutableExtensions {
			if strings.HasSuffix(lower, extension) {
				// A fixture that is intentionally executable must be called out
				// here by path; do not add a broad directory exemption.
				if allowedTrackedFixture(name) {
					continue
				}
				offenders = append(offenders, name)
				break
			}
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("build outputs must not be tracked: %s", strings.Join(offenders, ", "))
	}
}

// allowedTrackedFixture is empty on purpose. Adding a path requires an
// explicit comment explaining why the binary is source, not build output.
func allowedTrackedFixture(string) bool { return false }
