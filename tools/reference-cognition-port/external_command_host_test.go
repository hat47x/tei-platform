package cognitionport

import (
	"os/exec"
	"runtime"
	"testing"
)

// requireExternalCommandHost skips the tests that run an external command
// through `bash -c`. Those tests pass a `cp <fixture> $TEI_COGNITION_OUTPUT`
// script, and on Windows the fixture path contains a backslash or drive letter
// that neither WSL bash nor Git Bash resolves, so the command fails before the
// cognition boundary can observe it (#3461).
//
// Linux runs the reference behaviour, and the skip is logged so a reader does
// not mistake it for a failure of the implementation.
func requireExternalCommandHost(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("POSIX shell is unavailable: external command cannot be run on this host")
	}
	if runtime.GOOS == "windows" {
		t.Skip("Windows paths are not resolved by the bash -c cp fixture; Linux CI checks this (#3461)")
	}
}
