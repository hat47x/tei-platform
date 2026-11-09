package generatorselection

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"testing"
)

// requirePOSIXShellForExternalCommands skips the external-command tests when
// the host has no POSIX shell at all.
//
// The fixture command used to be `cp <fixture> $TEI_SOURCE`, which only worked
// where bash could resolve the host's fixture path and read an exported
// variable. The bash that Windows provides is WSL bash: it cannot open a C:\...
// path and it expands $NAME from the Windows environment before bash parses the
// command, so both halves failed (#3461). externalSourceCommand names no host
// path, and GenerateWithCommand now runs the command in the directory the
// inputs are staged in, so these tests run on Windows too.
func requirePOSIXShellForExternalCommands(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("POSIX shell is unavailable: external command cannot be run on this host")
	}
}

// externalSourceCommand writes content to the file named by envVar without
// naming a host path, so the same command works under WSL bash, Git Bash and
// Linux bash. Content travels as base64 to keep quoting out of the picture.
func externalSourceCommand(t *testing.T, envVar string, content []byte) string {
	t.Helper()
	return fmt.Sprintf(`printf '%%s' %s | base64 -d > "$%s"`,
		base64.StdEncoding.EncodeToString(content), envVar)
}
