//go:build unix

package operations

import (
	"os"
	"syscall"
)

// workspaceSourceOpenFlags avoids blocking while opening a FIFO; the reader
// rejects nonregular files after the open, on the same handle.
const workspaceSourceOpenFlags = os.O_RDONLY | syscall.O_NONBLOCK
