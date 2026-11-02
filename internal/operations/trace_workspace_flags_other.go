//go:build !unix

package operations

import "os"

const workspaceSourceOpenFlags = os.O_RDONLY
