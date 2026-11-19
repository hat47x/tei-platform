//go:build !windows && !tei_order_only && !tei_repair_only

package main

import (
	"io"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// startDocumentedGuideWeb executes the handbook's actual POSIX shell block.
// go run creates a compiled child process; cancelling its wrapper alone can
// leave that Web server alive. Run the complete command in a dedicated group.
func startDocumentedGuideWeb(t *testing.T, script, directory string, environment []string) <-chan error {
	t.Helper()
	cmd := exec.Command("sh", "-eu", "-c", script)
	cmd.Dir = directory
	cmd.Env = environment
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard // Never expose a PostgreSQL DSN in test failure output.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("step 5: cannot start the documented go-run shell: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	t.Cleanup(func() {
		// The shell, go wrapper and generated executable inherit this group.
		// Kill children even if the original shell has already exited.
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("step 5: documented go-run process did not terminate")
		}
	})
	return done
}
