//go:build unix

package workspace

import (
	"os"
	"syscall"
)

// openRootedSourceForRead avoids blocking while opening FIFOs. ReadSource
// rejects nonregular files after opening them through the pinned os.Root.
func openRootedSourceForRead(root, path string) (*os.File, error) {
	return openRootedSourceWithFlags(root, path, os.O_RDONLY|syscall.O_NONBLOCK)
}
