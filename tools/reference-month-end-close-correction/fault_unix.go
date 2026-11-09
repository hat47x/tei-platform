//go:build unix

package monthendclose

import "syscall"

func stopSelf() { _ = syscall.Kill(syscall.Getpid(), syscall.SIGSTOP) }
