package gobuilder

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// BuildPackage runs the selected Go toolchain with the same fixed invocation
// and deny-by-default environment used by both self-build References.
// It does not decide artifact acceptance, publish a binary, or create
// domain-specific Evidence; callers must verify actual outputs separately.
func BuildPackage(repo, goExecutable, packagePath, output, stateDir string) error {
	cmd := exec.Command(
		goExecutable,
		"build",
		"-trimpath",
		"-buildvcs=false",
		"-mod=readonly",
		"-o",
		output,
		packagePath,
	)
	cmd.Dir = repo
	env, err := Environment(os.Environ(), stateDir, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	cmd.Env = env
	combined, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build %s: %w: %s", packagePath, err, strings.TrimSpace(string(combined)))
	}
	return nil
}

// ResolveReproducibility is strictly a byte-digest comparison decision.
// The matching digest becomes a candidate only; no Build/Host authority is
// conferred. Caller-owned Evidence keeps distinct Plugin and Engine schemas.
func ResolveReproducibility(first, second string) (bool, string) {
	if first != second {
		return false, ""
	}
	return true, first
}
