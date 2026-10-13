//go:build !linux

package workspace

import (
	"fmt"
	"os"
)

// A portable EvalSymlinks/Open sequence does not establish read-time
// containment. Until a native handle-relative implementation is provided,
// unsupported targets reject source reads rather than silently downgrade.
func openRootedSource(_, _ string) (*os.File, error) {
	return nil, fmt.Errorf("race-free Workspace source opening is unsupported on this target")
}
