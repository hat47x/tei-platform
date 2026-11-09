package generatorselection

import (
	"os"
	"runtime"
	"time"
)

// ProcessMeasurement describes a completed harness invocation. Callers measure
// the process interval, including I/O, stream handling and any execution wrapper,
// after compilation has finished.
// CPU times are the OS accounting exposed by os.ProcessState (which may include
// children); they are not a resource limit or a process-tree monitor.
type ProcessMeasurement struct {
	ElapsedNanos     int64  `json:"elapsedNanos"`
	UserCPUNanos     int64  `json:"userCpuNanos"`
	SystemCPUNanos   int64  `json:"systemCpuNanos"`
	GOOS             string `json:"goos"`
	GOARCH           string `json:"goarch"`
	HarnessGoVersion string `json:"harnessGoVersion"`
}

// MeasureProcess returns no measurement when the command did not yield process
// state. A CPU time of zero is a valid observation at the OS clock resolution.
func MeasureProcess(state *os.ProcessState, elapsed time.Duration) *ProcessMeasurement {
	if state == nil {
		return nil
	}
	return &ProcessMeasurement{
		ElapsedNanos: elapsed.Nanoseconds(),
		UserCPUNanos: state.UserTime().Nanoseconds(), SystemCPUNanos: state.SystemTime().Nanoseconds(),
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, HarnessGoVersion: runtime.Version(),
	}
}
