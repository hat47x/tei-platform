package generatorselection

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestMeasurementProcessHelper(t *testing.T) {
	if os.Getenv("TEI_MEASUREMENT_HELPER") != "1" {
		return
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
	switch os.Args[len(os.Args)-1] {
	case "nonzero":
		os.Exit(7)
	case "malformed":
		fmt.Print("!")
	case "timeout":
		time.Sleep(5 * time.Second)
	default:
		fmt.Print(`{"state":"present","amount":"1.00","currency":"JPY"}`)
	}
	os.Exit(0)
}

func TestRunProgramRecordsMeasurementsIncludingFailedExecutions(t *testing.T) {
	t.Setenv("TEI_MEASUREMENT_HELPER", "1")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"success", "nonzero", "malformed", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			if mode == "timeout" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Second)
				defer cancel()
			}
			_, evidence, err := runProgram(ctx, executable, []string{"-test.run=^TestMeasurementProcessHelper$", "--", mode}, []byte(`{}`), ExecutionOptions{})
			if (err == nil) != (mode == "success") {
				t.Fatalf("unexpected error: %v", err)
			}
			m := evidence.Measurement
			if m == nil || m.ElapsedNanos <= 0 || m.UserCPUNanos < 0 || m.SystemCPUNanos < 0 {
				t.Fatalf("missing process measurement: %+v", m)
			}
			if m.GOOS != runtime.GOOS || m.GOARCH != runtime.GOARCH || m.HarnessGoVersion != runtime.Version() {
				t.Fatalf("missing harness environment: %+v", m)
			}
			if mode == "timeout" && !evidence.TimedOut {
				t.Fatal("timeout was not recorded")
			}
		})
	}
}

func TestRunProgramDoesNotInventMeasurementWhenStartFails(t *testing.T) {
	_, evidence, err := runProgram(context.Background(), filepath.Join(t.TempDir(), "missing"), nil, []byte(`{}`), ExecutionOptions{})
	if err == nil || evidence.Measurement != nil {
		t.Fatalf("start failure has a process measurement: %+v, %v", evidence, err)
	}
}
