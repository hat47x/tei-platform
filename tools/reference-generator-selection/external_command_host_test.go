package generatorselection

import (
	"os/exec"
	"runtime"
	"testing"
)

// requirePOSIXShellForExternalCommands skips the external-command tests when
// the host cannot run a POSIX shell command. The tools pass a `cp A B` script
// to `bash -c`; on Windows the fixture path contains a backslash or a drive
// letter that neither WSL bash nor Git Bash resolves, so the command fails
// before it can be observed (#3461).
//
// The test still runs on Linux, which is where the reference behaviour is
// checked. Skipping is explicit rather than silently passing: a reader sees the
// reason in the test log.
func requirePOSIXShellForExternalCommands(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("bash"); err != nil {
			t.Skip("POSIX shell is unavailable: external command cannot be run on this host")
		}
		t.Skip("Windows paths are not resolved by the bash -c cp fixture; Linux CI checks this (#3461)")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("POSIX shell is unavailable: external command cannot be run on this host")
	}
}
