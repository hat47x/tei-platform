package cognitionport

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"testing"
)

// requireExternalCommandHost skips the tests that run an external command
// through `bash -c` when the host has no POSIX shell at all.
//
// The fixture command used to be `cp <fixture> $TEI_COGNITION_OUTPUT`, which
// only worked when the bash on PATH could resolve the host's fixture path: the
// bash that Windows provides is WSL bash, which needs /mnt/d/... rather than a
// drive letter (#3461). The command is now built by externalOutputCommand,
// which names no host path, so these tests run on Windows too.
func requireExternalCommandHost(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("POSIX shell is unavailable: external command cannot be run on this host")
	}
}

// externalOutputCommand writes content to the file named by envVar without
// naming a host path, so the same command works under WSL bash, Git Bash and
// Linux bash. Content travels as base64 to keep quoting out of the picture.
func externalOutputCommand(t *testing.T, envVar string, content []byte) string {
	t.Helper()
	return fmt.Sprintf(`printf '%%s' %s | base64 -d > "$%s"`,
		base64.StdEncoding.EncodeToString(content), envVar)
}
