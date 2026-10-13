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

// Run the actual production Runtime Engine build CLI against a clean checkout
// and the exact Go compiler. A pinned executable wrapper modifies only the
// second successfully built engine. This is an injected mismatch, not a claim
// of natural nondeterminism or a test-only branch in the production CLI.
func TestExactRuntimeEngineRunnerRejectsAlteredSecondBuild(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("fault-injection executable wrapper requires the Linux shell")
	}
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go unavailable: %v", err)
	}
	goPath, err = filepath.Abs(goPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("Git unavailable: %v", err)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if err := requireCleanRepository(root); err != nil {
		t.Fatalf("actual Engine runner requires a clean committed checkout: %v", err)
	}

	dir := t.TempDir()
	cli := filepath.Join(dir, "engine-build")
	compile := exec.Command(goPath, "build", "-trimpath", "-buildvcs=false",
		"-mod=readonly", "-o", cli, "./tools/reference-transform-runtime-engine-build")
	compile.Dir = root
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile actual Engine CLI: %v: %s", err, output)
	}

	wrapperDir := filepath.Join(dir, "bin")
	if err := os.Mkdir(wrapperDir, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "modified-second-output")
	shellQuote := func(path string) string {
		return "'" + strings.ReplaceAll(path, "'", "'\\''") + "'"
	}
	script := strings.Join([]string{
		"#!/bin/sh",
		"set -eu",
		"real_go=" + shellQuote(goPath),
		"marker=" + shellQuote(marker),
		"if [ \"$1\" != build ]; then exec \"$real_go\" \"$@\"; fi",
		"output=''",
		"previous=''",
		"for arg in \"$@\"; do",
		"  if [ \"$previous\" = -o ]; then output=\"$arg\"; fi",
		"  previous=\"$arg\"",
		"done",
		"\"$real_go\" \"$@\"",
		"case \"$output\" in",
		"  */tei-transform-engine-build-*/engine-second)",
		"    printf '\\nTEI deliberate second-engine fault\\n' >> \"$output\"",
		"    printf '%s\\n' \"$output\" > \"$marker\"",
		"    ;;",
		"esac",
	}, "\n") + "\n"
	goWrapper := filepath.Join(wrapperDir, "go")
	if err := os.WriteFile(goWrapper, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	artifactOut := filepath.Join(dir, "must-not-export-engine")
	cmd := exec.Command(cli, "-repo", root, "-go", goWrapper, "-artifact-out", artifactOut)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOFLAGS=")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("mutated second actual Engine must fail with exit code 1: err=%v stdout=%q stderr=%q",
			err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "transform Runtime Engine build is not reproducible") {
		t.Fatalf("non-reproducibility diagnostic missing: %q", stderr.String())
	}

	secondBytes, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("second actual Go build was not modified: %v", err)
	}
	secondPath := strings.TrimSpace(string(secondBytes))
	if !strings.Contains(secondPath, "tei-transform-engine-build-") ||
		!strings.HasSuffix(secondPath, "engine-second") {
		t.Fatalf("wrong Go build output was modified: %q", secondPath)
	}
	if _, err := os.Stat(filepath.Dir(secondPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed engine runner did not clean its temporary build: %v", err)
	}
	if _, err := os.Stat(artifactOut); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("non-reproducible Engine exported an artifact: %v", err)
	}

	var evidence buildEvidence
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&evidence); err != nil {
		t.Fatalf("actual Engine CLI did not report valid evidence: %v; stdout=%q", err, stdout.String())
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		t.Fatalf("actual Engine CLI emitted trailing output: %v", err)
	}
	if evidence.Schema != evidenceSchema ||
		evidence.FirstArtifactDigest == evidence.SecondArtifactDigest ||
		!validE2EArtifactDigest(evidence.FirstArtifactDigest) ||
		!validE2EArtifactDigest(evidence.SecondArtifactDigest) ||
		evidence.Reproducible || evidence.ArtifactDigest != "" ||
		evidence.RuntimeEngine != nil {
		t.Fatalf("non-reproducible Engine was assigned a canonical artifact: %+v", evidence)
	}
	// Even failed reproducibility should retain verifiable actual-input
	// provenance and bind both observed output digests.
	if evidence.SourceClosureDigest == "" || evidence.Builder.ToolchainDigest == "" ||
		evidence.EvidenceFingerprint == "" {
		t.Fatalf("failure evidence lost provenance: %+v", evidence)
	}
	recomputed, err := evidenceFingerprint(evidence)
	if err != nil || recomputed != evidence.EvidenceFingerprint {
		t.Fatalf("failed build Evidence does not bind actual digests: got %q calculated %q (%v)",
			evidence.EvidenceFingerprint, recomputed, err)
	}
	t.Logf("actual Engine mismatch stayed unaccepted: first=%s second=%s",
		evidence.FirstArtifactDigest, evidence.SecondArtifactDigest)
}

func validE2EArtifactDigest(digest string) bool {
	if len(digest) != len("sha256:")+64 || !strings.HasPrefix(digest, "sha256:") {
		return false
	}
	for _, c := range strings.TrimPrefix(digest, "sha256:") {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}
