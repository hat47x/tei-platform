package targetprofile

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestCPUWorkloadHelper(t *testing.T) {
	if os.Getenv("TEI_CPU_LIMIT_HELPER") != "1" {
		return
	}
	var x uint64 = 1
	for {
		for i := 0; i < 100000; i++ {
			x = x*1664525 + 1013904223
		}
		var usage syscall.Rusage
		if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
			os.Exit(2)
		}
		if usage.Utime.Nano()+usage.Stime.Nano() >= int64(2*time.Second) {
			fmt.Println(x)
			os.Exit(0)
		}
	}
}

func TestCPULimitStopsWorkloadBeforeWallTimeout(t *testing.T) {
	e := requireCPU(t)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEI_CPU_LIMIT_HELPER", "1")
	for _, limited := range []bool{false, true} {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		args := []string{"-test.run=^TestCPUWorkloadHelper$"}
		cmd := exec.CommandContext(ctx, executable, args...)
		if limited {
			cmd = e.Execution.Command(ctx, executable, args...)
		}
		output, err := cmd.CombinedOutput()
		timedOut := ctx.Err() != nil
		cancel()
		if timedOut || cmd.ProcessState == nil {
			t.Fatalf("wall timeout or unavailable process state: limited=%v err=%v", limited, err)
		}
		cpu := cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()
		if !limited {
			if err != nil || cpu < 2*time.Second || len(output) == 0 {
				t.Fatalf("positive control failed: cpu=%v err=%v output=%q", cpu, err, output)
			}
		} else {
			status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus)
			if err == nil || !ok || !status.Signaled() || (status.Signal() != syscall.SIGKILL && status.Signal() != syscall.SIGXCPU) || cpu >= 2*time.Second {
				t.Fatalf("CPU limit did not stop workload: cpu=%v state=%v err=%v", cpu, cmd.ProcessState, err)
			}
		}
		t.Logf("limited=%v CPU=%v exit=%v wallTimeout=%v", limited, cpu, err, timedOut)
	}
}
