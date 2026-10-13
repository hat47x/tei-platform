//go:build linux

package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// openRootedSource opens every component relative to a pinned directory fd.
// O_NOFOLLOW rejects symlinks at every level, including the leaf; O_DIRECTORY
// ensures intermediate components cannot be swapped for regular files.
// The final fd remains pinned even when the directory tree is renamed.
func openRootedSource(root, path string) (*os.File, error) {
	fd, err := syscall.Open(root, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, fmt.Errorf("open Workspace root: %w", err)
	}
	parts := strings.Split(filepath.FromSlash(path), string(os.PathSeparator))
	for i, part := range parts {
		flags := syscall.O_RDONLY | syscall.O_CLOEXEC | syscall.O_NOFOLLOW
		if i < len(parts)-1 {
			flags |= syscall.O_DIRECTORY
		}
		next, openErr := syscall.Openat(fd, part, flags, 0)
		_ = syscall.Close(fd)
		if openErr != nil {
			return nil, fmt.Errorf("open source %q component %q: %w", path, part, openErr)
		}
		fd = next
	}
	return os.NewFile(uintptr(fd), path), nil
}
