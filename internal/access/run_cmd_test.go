package access_test

import "os/exec"

// runCmd runs a command and returns its combined output. It exists so the
// junction helper can be written without pulling os/exec into every test file.
func runCmd(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}
