package deliverybundle

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStartAndStopService(t *testing.T) {
	statePath := activeState(t)
	receipt, err := Start(statePath, "exec sleep 5", filepath.Join(t.TempDir(), "service.log"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = Stop(statePath) })
	if receipt.PID <= 0 || receipt.Log == "" {
		t.Fatalf("unexpected start receipt: %+v", receipt)
	}
	if _, err := os.Stat(pidFile(statePath)); err != nil {
		t.Fatal(err)
	}
	stopped, err := Stop(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if stopped.PID != receipt.PID {
		t.Fatalf("stopped pid=%d want %d", stopped.PID, receipt.PID)
	}
	if _, err := os.Stat(pidFile(statePath)); !os.IsNotExist(err) {
		t.Fatalf("pid file was not removed: %v", err)
	}
}

func TestStartTwiceIsRejected(t *testing.T) {
	statePath := activeState(t)
	if _, err := Start(statePath, "exec sleep 5", ""); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = Stop(statePath) })
	if _, err := Start(statePath, "exec sleep 5", ""); err == nil {
		t.Fatal("second start was accepted")
	}
}

func TestStopWithoutRecordIsRejected(t *testing.T) {
	statePath := activeState(t)
	if _, err := Stop(statePath); err == nil {
		t.Fatal("stop without a record was accepted")
	}
}

func TestStopGracefulWaitsForTermination(t *testing.T) {
	statePath := activeState(t)
	if _, err := Start(statePath, "exec sleep 5", ""); err != nil {
		t.Fatal(err)
	}
	receipt, err := StopGraceful(statePath, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.PID <= 0 {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
	if _, err := os.Stat(pidFile(statePath)); !os.IsNotExist(err) {
		t.Fatalf("pid file was not removed: %v", err)
	}
}

func TestRestartReplacesProcess(t *testing.T) {
	statePath := activeState(t)
	first, err := Start(statePath, "exec sleep 5", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = Stop(statePath) })
	second, err := Restart(statePath, "exec sleep 5", "", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if second.PID == first.PID {
		t.Fatalf("restart kept the same pid: %d", second.PID)
	}
	if _, err := os.Stat(pidFile(statePath)); err != nil {
		t.Fatal(err)
	}
}

func TestRestartWithoutRunningServiceIsRejected(t *testing.T) {
	statePath := activeState(t)
	if _, err := Restart(statePath, "exec sleep 5", "", time.Second); err == nil {
		t.Fatal("restart without a running service was accepted")
	}
}
