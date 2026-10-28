//go:build !unix

package pluginbuildsupply

import (
	"os"
	"os/exec"
)

// Process groups and signals need a Unix platform; the through tests run on
// Linux. Elsewhere a kill stops only the builder process itself.
func setProcessGroup(*exec.Cmd) {}

func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return os.ErrProcessDone
	}
	return cmd.Process.Kill()
}

func termination(state *os.ProcessState) Termination {
	if state.ExitCode() < 0 {
		return Termination{Exited: false, ExitCode: -1, Signal: "SIGNAL"}
	}
	return Termination{Exited: true, ExitCode: state.ExitCode()}
}
