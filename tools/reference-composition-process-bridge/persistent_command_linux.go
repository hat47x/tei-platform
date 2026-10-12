package compositionprocessbridge

import (
	"context"
	"os"
	"os/exec"
)

// Pin and execute the same private inode. A mutable source path is metadata,
// never the launch target. Closing the parent descriptor after Start does not
// close the descriptor inherited by the child; both lifetimes are accounted for.
func preparePersistentCommand(ctx context.Context, path, digest string, args []string) (*exec.Cmd, func() error, error) {
	image, err := pinReferenceExecutable(ctx, path, digest)
	if err != nil {
		return nil, nil, err
	}
	cmd := exec.Command("/proc/self/fd/3", args...)
	cmd.ExtraFiles = []*os.File{image.executable}
	return cmd, image.Close, nil
}
