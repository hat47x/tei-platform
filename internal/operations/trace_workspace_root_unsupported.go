//go:build js || wasip1 || plan9

package operations

import (
	"errors"
	"os"
)

// These targets cannot pin a directory against symlink races, so the reader
// fails closed instead of degrading to a lexical check.
func openWorkspaceSourceRoot(string) (*os.Root, error) {
	return nil, errors.New("traversal-resistant Workspace source opening is unsupported on this target")
}
