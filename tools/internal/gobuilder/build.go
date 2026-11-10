package gobuilder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// BuildPackage runs the selected Go toolchain with the same fixed invocation
// and deny-by-default environment used by both self-build References.
// It does not decide artifact acceptance, publish a binary, or create
// domain-specific Evidence; callers must verify actual outputs separately.
func BuildPackage(repo, goExecutable, packagePath, output, stateDir string) error {
	return buildPackage(repo, goExecutable, packagePath, output, stateDir, "")
}

// BuildPackageWithSupply is BuildPackage with one difference: module downloads
// come from a named supply directory instead of being disabled. The isolation is
// otherwise identical, so a caller that needs dependencies records which supply
// it used rather than inheriting the host's.
func BuildPackageWithSupply(repo, goExecutable, packagePath, output, stateDir, supply string) error {
	if strings.TrimSpace(supply) == "" {
		return fmt.Errorf("a supply directory is required")
	}
	absolute, err := filepath.Abs(supply)
	if err != nil {
		return err
	}
	if info, statErr := os.Stat(absolute); statErr != nil || !info.IsDir() {
		return fmt.Errorf("supply root is not a readable directory: %s", absolute)
	}
	return buildPackage(repo, goExecutable, packagePath, output, stateDir, absolute)
}

func buildPackage(repo, goExecutable, packagePath, output, stateDir, supply string) error {
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
	if supply != "" {
		// GOPROXY is replaced rather than appended: the fixed settings disable it,
		// and a file supply is the only source this build may use.
		env = replaceEnv(env, "GOPROXY", "file://"+filepath.ToSlash(supply))
	}
	cmd.Env = env
	combined, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build %s: %w: %s", packagePath, err, strings.TrimSpace(string(combined)))
	}
	return nil
}

// replaceEnv returns env with key set to value, dropping any earlier entry. The
// last duplicate would win in practice, but leaving both would make the recorded
// environment ambiguous.
func replaceEnv(env []string, key, value string) []string {
	out := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if name, _, ok := strings.Cut(entry, "="); ok && strings.EqualFold(name, key) {
			continue
		}
		out = append(out, entry)
	}
	return append(out, key+"="+value)
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
