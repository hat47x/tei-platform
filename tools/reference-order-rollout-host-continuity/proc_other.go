//go:build !unix

package orderrollouthost

import "errors"

var errNoSignals = errors.New("process stop/continue needs a Unix platform")

// StopProcess is not available on this platform; the real-DB tests run on Unix.
func StopProcess(int) error { return errNoSignals }

// ContinueProcess is not available on this platform.
func ContinueProcess(int) error { return errNoSignals }

// KillProcess is not available on this platform.
func KillProcess(int) error { return errNoSignals }
