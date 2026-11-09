//go:build !js && !wasip1 && !plan9

package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

// openRootedSource uses Go's traversal-resistant os.Root. Relative symlinks
// inside the root work normally, but paths escaping via symlinks are rejected.
// Each opened file is pinned across renames. Load keeps one separate Root
// open across the manifest and all of its domain Source reads.
//
// Bind mounts and privileged filesystem constructs are outside os.Root's
// protection and are not treated as hostile-user-controlled symlinks.
func openWorkspaceRoot(root string) (*os.Root, error) {
	directory, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("open Workspace root: %w", err)
	}
	return directory, nil
}

func openRootedSource(root, path string) (*os.File, error) {
	return openRootedSourceWithFlags(root, path, os.O_RDONLY)
}

func openRootedSourceWithFlags(root, path string, flags int) (*os.File, error) {
	directory, err := openWorkspaceRoot(root)
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	file, err := directory.OpenFile(filepath.FromSlash(path), flags, 0)
	if err != nil {
		return nil, fmt.Errorf("open source %q in Workspace root: %w", path, err)
	}
	return file, nil
}
