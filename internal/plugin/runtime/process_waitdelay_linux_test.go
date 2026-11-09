//go:build linux

package runtime_test

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
)

func TestStdioProcessLauncherBoundsInheritedPipeWait(t *testing.T) {
	launcher, command := helperLauncher(t, "exit-with-inherited-pipes")
	launcher.MaxExecutionDuration = 6 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	start := time.Now()
	_, err := launcher.Invoke(ctx, command, processRequest())
	if !errors.Is(err, exec.ErrWaitDelay) {
		t.Fatalf("expected bounded inherited-pipe wait, got %v", err)
	}
	if ctx.Err() != nil {
		t.Fatalf("parent deadline elapsed before pipe wait ended: %v", ctx.Err())
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("inherited pipe wait exceeded budget: %s", elapsed)
	}
}
