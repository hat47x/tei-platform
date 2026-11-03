//go:build unix

package pluginbuildsupply

import (
	"os"
	"os/exec"
	"syscall"
)

// setProcessGroup puts the builder and every process it starts (go build,
// the compiler, the linker) into one group, so a kill reaches all of them.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return os.ErrProcessDone
	}
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

// termination reads the exit status the operating system reported.
func termination(state *os.ProcessState) Termination {
	if ws, ok := state.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return Termination{Exited: false, ExitCode: -1, Signal: signalName(ws.Signal())}
	}
	return Termination{Exited: true, ExitCode: state.ExitCode()}
}

func signalName(sig syscall.Signal) string {
	switch sig {
	case syscall.SIGKILL:
		return "SIGKILL"
	case syscall.SIGTERM:
		return "SIGTERM"
	case syscall.SIGINT:
		return "SIGINT"
	case syscall.SIGSEGV:
		return "SIGSEGV"
	case syscall.SIGABRT:
		return "SIGABRT"
	default:
		return "SIGNAL"
	}
}
