package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Run the production CLI from a clean checkout with the genuine Go compiler.
// A temporary executable wrapper deliberately appends bytes to only the
// second output after building it. This does not prove natural nondeterminism.
func TestSelfBuildRunnerRejectsAlteredSecondBuilder(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("test wrapper uses the Linux shell")
	}
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go unavailable: %v", err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("Git unavailable: %v", err)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if err := requireCleanRepository(root); err != nil {
		t.Fatalf("self-build E2E requires a clean committed checkout: %v", err)
	}
	dir := t.TempDir()
	cli := filepath.Join(dir, "self-build")
	compile := exec.Command(goPath, "build", "-trimpath", "-buildvcs=false",
		"-mod=readonly", "-o", cli, "./tools/reference-plugin-self-build")
	compile.Dir = root
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile actual self-build CLI: %v: %s", err, output)
	}
	wrapperDir := filepath.Join(dir, "bin")
	if err := os.Mkdir(wrapperDir, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "altered-second-path")
	script := strings.Join([]string{
		"#!/bin/sh",
		"set -eu",
		"if [ \"$1\" != build ]; then exec \"$TEI_TEST_REAL_GO\" \"$@\"; fi",
		"output=''",
		"previous=''",
		"for arg in \"$@\"; do",
		"  if [ \"$previous\" = -o ]; then output=\"$arg\"; fi",
		"  previous=\"$arg\"",
		"done",
		"\"$TEI_TEST_REAL_GO\" \"$@\"",
		"case \"$output\" in",
		"  */tei-plugin-package-builder-*/builder-second)",
		"    printf '\\nTEI deliberate second-build fault\\n' >> \"$output\"",
		"    printf '%s\\n' \"$output\" > \"$TEI_TEST_FAULT_MARKER\"",
		"    ;;",
		"esac",
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(wrapperDir, "go"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(cli, "-repo", root)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"PATH="+wrapperDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"TEI_TEST_REAL_GO="+goPath,
		"TEI_TEST_FAULT_MARKER="+marker,
		"GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOFLAGS=",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("second actual builder mutation must exit 1: %v stdout=%q stderr=%q",
			err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "package-builder-binary is not reproducible") {
		t.Fatalf("failure diagnostic missing: %q", stderr.String())
	}
	b, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("second real build was not altered: %v", err)
	}
	secondPath := strings.TrimSpace(string(b))
	if !strings.Contains(secondPath, "tei-plugin-package-builder-") ||
		!strings.HasSuffix(secondPath, "builder-second") {
		t.Fatalf("wrong target for altered build output: %q", secondPath)
	}
	if _, err := os.Stat(filepath.Dir(secondPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed runner left a temporary build directory: %v", err)
	}
	var got selfBuildMismatchEvidence
	dec := json.NewDecoder(&stdout)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("actual CLI did not emit mismatch JSON: %v: %q", err, stdout.String())
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		t.Fatalf("actual CLI emitted more than one JSON: %v", err)
	}
	if got.Schema != selfBuildMismatchSchema ||
		got.Stage != "package-builder-binary" ||
		got.Outcome != "reproducibility-failure" ||
		got.FirstArtifactDigest == got.SecondArtifactDigest ||
		!validSelfBuildE2EDigest(got.FirstArtifactDigest) ||
		!validSelfBuildE2EDigest(got.SecondArtifactDigest) ||
		got.Reproducible || got.ArtifactDigest != "" ||
		got.ArtifactAccepted || got.ArtifactExportAllowed || got.ArtifactLockAllowed {
		t.Fatalf("unaccepted runner result promoted to success: %+v", got)
	}
	for _, key := range []string{"\"lockedBuild\"", "\"runtimePluginLock\"", "\"packageBuildPlan\""} {
		if strings.Contains(stdout.String(), key) {
			t.Fatalf("failed runner exported accepted lock or plan %s", key)
		}
	}
	t.Logf("full self-build runner rejected actual mismatch: first=%s second=%s",
		got.FirstArtifactDigest, got.SecondArtifactDigest)
}

func validSelfBuildE2EDigest(s string) bool {
	if !strings.HasPrefix(s, "sha256:") || len(s) != 71 {
		return false
	}
	for _, r := range s[len("sha256:"):] {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return false
		}
	}
	return true
}
