package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func stateDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "tei-build-supply-state-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := removeWritable(dir); err != nil {
			t.Errorf("remove isolated state %s: %v", dir, err)
		}
	})
	return dir
}

func removeWritable(root string) error {
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

// TestBuildWithSupplyProducesAnArtifactAndReportsClosureSeparately runs one real
// build of the smallest external-module-free target and checks that the artifact
// digest and the source-closure verdict are independent facts.
func TestBuildWithSupplyProducesAnArtifactAndReportsClosureSeparately(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("exact Go binary unavailable: %v", err)
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := BuildWithSupply(repo, "go", "./tools/reference-build-authoring-intent",
		supplyRoot(t), filepath.Join(stateDir(t), "artifact"), stateDir(t))
	if err != nil {
		t.Fatal(err)
	}
	if !evidence.BuildSucceeded {
		t.Fatalf("build failed: %s", evidence.BuildError)
	}
	if !strings.HasPrefix(evidence.ArtifactDigest, "sha256:") {
		t.Fatalf("artifact digest missing: %q", evidence.ArtifactDigest)
	}
	if evidence.SourceClosureState != SourceClosureQualified {
		t.Fatalf("self-contained target was not closure-qualified: %s (%s)",
			evidence.SourceClosureState, evidence.SourceClosureError)
	}
	if evidence.Nature == "" || evidence.Boundary == "" {
		t.Fatal("evidence lost its scope statements")
	}
}

// TestBuildWithSupplyKeepsArtifactAndClosureApartForCmdTei records the X1
// negative: when cmd/tei builds from a verified supply but the existing source
// closure resolver still rejects the same target, the artifact digest must not
// be reported as a qualified closure.
func TestBuildWithSupplyKeepsArtifactAndClosureApartForCmdTei(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("exact Go binary unavailable: %v", err)
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := BuildWithSupply(repo, "go", "./cmd/tei",
		supplyRoot(t), filepath.Join(stateDir(t), "tei-artifact"), stateDir(t))
	if err != nil {
		t.Fatal(err)
	}
	if !evidence.BuildSucceeded {
		t.Skipf("cmd/tei did not build from this supply, so X1 is not exercised: %s", evidence.BuildError)
	}
	if evidence.SourceClosureState != SourceClosureRejected {
		t.Fatal("cmd/tei now passes source closure; the recorded X1 case is stale")
	}
	if evidence.ArtifactDigest == "" {
		t.Fatal("successful build reported no artifact digest")
	}
	// Two different stops produce the same `rejected` verdict, and the record
	// must not blur them: the closure resolver's own isolated state has no
	// module supply (`module lookup disabled`), and even with a supply it
	// refuses every external module root. Either message is acceptable here;
	// an unrelated failure is not.
	if !strings.Contains(evidence.SourceClosureError, "module lookup disabled") &&
		!strings.Contains(evidence.SourceClosureError, "external module") {
		t.Fatalf("closure rejection reason changed shape: %s", evidence.SourceClosureError)
	}
}

func TestPreloadEnvironmentRejectsHostModuleCache(t *testing.T) {
	env, err := preloadEnvironment(os.Environ(), stateDir(t), supplyRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	hostOutput, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		t.Skipf("go env unavailable: %v", err)
	}
	hostCache := strings.TrimSpace(string(hostOutput))
	for _, entry := range env {
		if strings.HasPrefix(entry, "GOMODCACHE=") && strings.Contains(entry, hostCache) {
			t.Fatalf("preload environment inherited the host module cache: %s", entry)
		}
		if strings.HasPrefix(entry, "GOPROXY=") && !strings.HasPrefix(entry, "GOPROXY=file://") {
			t.Fatalf("preload environment allows network resolution: %s", entry)
		}
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

// The reproducibility check asks the child which state directory it used, so a
// report that omits it turns a real build into an unperformed observation. This
// pins the field so that stays visible instead of silently degrading.
func TestBuildWithSupplyReportsTheStateDirectoryItUsed(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("exact Go binary unavailable: %v", err)
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(stateDir(t), "artifact-state")
	evidence, err := BuildWithSupply(repo, "go", "./tools/reference-build-authoring-intent",
		supplyRoot(t), filepath.Join(state, "artifact"), state)
	if err != nil {
		t.Fatal(err)
	}
	if !evidence.BuildSucceeded {
		t.Fatalf("build failed: %s", evidence.BuildError)
	}
	if evidence.StateDir != state {
		t.Fatalf("stateDir = %q, want %q", evidence.StateDir, state)
	}
}
