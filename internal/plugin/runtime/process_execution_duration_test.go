package runtime_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	pruntime "tei/internal/plugin/runtime"
)

// This uses the real Go test binary as a child that emits no output and then
// sleeps for 30 seconds. The Host's own execution deadline must stop it even
// though stdout/stderr remain within their independent limits.
func TestStdioProcessLauncherStopsSilentChildAtExecutionLimit(t *testing.T) {
	launcher, command := helperLauncher(t, "wait")
	launcher.MaxExecutionDuration = 100 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	resp, err := launcher.Invoke(ctx, command, processRequest())
	if !errors.Is(err, context.DeadlineExceeded) ||
		!strings.Contains(err.Error(), "execution time limit exceeded") {
		t.Fatalf("silent child was not stopped by the Host deadline: %v", err)
	}
	if ctx.Err() != nil {
		t.Fatalf("internal execution limit canceled the parent context: %v", ctx.Err())
	}
	if resp.Status != "" || len(resp.Payload) != 0 {
		t.Fatal("timed-out process returned an untrusted response")
	}
	if elapsed := time.Since(start); elapsed > 4*time.Second {
		t.Fatalf("silent child retained Host execution too long: %s", elapsed)
	}
}

func TestStdioProcessLauncherRejectsInvalidExecutionLimit(t *testing.T) {
	launcher, command := helperLauncher(t, "echo")
	launcher.MaxExecutionDuration = -time.Second
	if _, err := launcher.Invoke(context.Background(), command, processRequest()); err == nil ||
		!strings.Contains(err.Error(), "execution limit must not be negative") {
		t.Fatalf("negative Host execution limit accepted: %v", err)
	}
}

func TestStdioProcessLauncherHonorsParentCancellationBeforeExecutionLimit(t *testing.T) {
	launcher, command := helperLauncher(t, "wait")
	launcher.MaxExecutionDuration = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := launcher.Invoke(ctx, command, processRequest())
	if !errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(err.Error(), "execution time limit exceeded") {
		t.Fatalf("parent cancellation was confused with the Host limit: %v", err)
	}
}

func TestStdioProcessLauncherConfiguredExecutionLimitAllowsValidProcess(t *testing.T) {
	launcher, command := helperLauncher(t, "echo")
	launcher.MaxExecutionDuration = 5 * time.Second
	resp, err := launcher.Invoke(context.Background(), command, processRequest())
	if err != nil || resp.Status != pruntime.ResponseOK {
		t.Fatalf("valid child rejected with execution limit: %v", err)
	}
}
