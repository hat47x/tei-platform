package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func stateRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "tei-repro-state-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := removeWritableChecked(dir); err != nil {
			t.Errorf("remove isolated state %s: %v", dir, err)
		}
	})
	return dir
}

func removeWritableChecked(root string) error {
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return os.Chmod(path, 0o700)
		}
		return os.Chmod(path, 0o600)
	}); err != nil {
		return err
	}
	return os.RemoveAll(root)
}

func supplyRoot(t *testing.T) string {
	t.Helper()
	output, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		t.Skipf("go env unavailable: %v", err)
	}
	cache := filepath.Join(strings.TrimSpace(string(output)), "cache", "download")
	if _, err := os.Stat(cache); err != nil {
		t.Skipf("host download cache unavailable: %v", err)
	}
	return cache
}

// TestRunReportsReproducibilityForASelfContainedTarget checks the positive case
// with a target that needs no external module, so the test does not depend on
// any particular module being present in the supply.
func TestRunReportsReproducibilityForASelfContainedTarget(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("exact Go binary unavailable: %v", err)
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := Run(repo, "./tools/reference-build-authoring-intent", supplyRoot(t), "go", stateRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if !evidence.BothSucceeded || !evidence.DigestsEqual || !evidence.DistinctAttempts {
		t.Fatalf("two isolated builds were not reproducible: %+v", evidence.FailureCause)
	}
	if !evidence.Reproducible {
		t.Fatal("matching digests from two distinct attempts were not reported reproducible")
	}
	if len(evidence.Attempts) != 2 {
		t.Fatalf("expected two attempts, got %d", len(evidence.Attempts))
	}
	if evidence.Attempts[0].ArtifactDigest != evidence.Attempts[1].ArtifactDigest {
		t.Fatal("reported equal digests for unequal artifacts")
	}
	if evidence.Attempts[0].StateDir == evidence.Attempts[1].StateDir {
		t.Fatal("two attempts shared one isolated state directory")
	}
	if evidence.GoVersion == "" || evidence.Boundary == "" {
		t.Fatal("evidence lost its toolchain identity or scope statement")
	}
}

// TestRunReportsReproducibilityForCmdTei uses the real TEI CLI, which needs an
// external module supply. It skips if the supply cannot satisfy the target so a
// missing fixture is never read as a product claim.
func TestRunReportsReproducibilityForCmdTei(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("exact Go binary unavailable: %v", err)
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := Run(repo, "./cmd/tei", supplyRoot(t), "go", stateRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if !evidence.BothSucceeded {
		t.Skipf("cmd/tei did not build twice from this supply; reproducibility is NOT RUN: %s", evidence.FailureCause)
	}
	if !evidence.Reproducible {
		t.Fatalf("cmd/tei was not reproducible across two isolated attempts: %s", evidence.FailureCause)
	}
}

// TestRunSeparatesAttemptIdentityFromArtifactDigest pins the rule that replaying
// one attempt cannot stand in for a second run: the identity is random and is
// never derived from the bytes.
func TestRunSeparatesAttemptIdentityFromArtifactDigest(t *testing.T) {
	first, err := randomID()
	if err != nil {
		t.Fatal(err)
	}
	second, err := randomID()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("attempt identities repeated")
	}
	if len(first) != 64 {
		t.Fatalf("attempt identity is not 32 random bytes: %d chars", len(first))
	}
}

func TestFileURLKeepsVolumeOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("volume handling is Windows specific")
	}
	if got := fileURL(`C:\cache\download`); got != "file:///C:/cache/download" {
		t.Fatalf("file URL lost the volume: %s", got)
	}
}
