//go:build !linux

package compositionprocessbridge

import (
	"context"
	"errors"
	"os/exec"
)

// No hash-then-launch fallback is offered for an unverified platform.
func preparePersistentCommand(context.Context, string, string, []string) (*exec.Cmd, func() error, error) {
	return nil, nil, errors.New("verified persistent reference launch requires Linux")
}
