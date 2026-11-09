//go:build unix

package orderrollouthost

import "syscall"

// StopProcess pauses every thread of the process (SIGSTOP), heartbeats and
// lease renewals included, until ContinueProcess.
func StopProcess(pid int) error { return syscall.Kill(pid, syscall.SIGSTOP) }

// ContinueProcess resumes a stopped process (SIGCONT).
func ContinueProcess(pid int) error { return syscall.Kill(pid, syscall.SIGCONT) }

// KillProcess terminates a process without letting it clean up (SIGKILL).
func KillProcess(pid int) error { return syscall.Kill(pid, syscall.SIGKILL) }
