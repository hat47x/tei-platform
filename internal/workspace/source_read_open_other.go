//go:build !unix

package workspace

import "os"

// Non-Unix targets retain the existing rooted open behavior. Targets without
// traversal-resistant os.Root support continue to fail closed there.
func openRootedSourceForRead(root, path string) (*os.File, error) {
	return openRootedSource(root, path)
}
