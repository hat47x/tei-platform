//go:build js || wasip1 || plan9

package workspace

import (
	"fmt"
	"os"
)

// The Go JS implementation of os.Root remains vulnerable to symlink races.
// WASI depends on the sandboxing of its host; Plan 9 cannot pin the directory
// across renames. Do not silently downgrade to a lexical path validation.
func openRootedSource(_, _ string) (*os.File, error) {
	return nil, fmt.Errorf("traversal-resistant Workspace source opening is unsupported on this target")
}
