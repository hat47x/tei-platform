package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestRunTwoStageRepeatsAcrossIndependentWorkDirectories closes the NOT RUN
// item "two independent attempts": the same child and target are run twice in
// separate work directories, and both runs must reach L3 with the same stage2
// artifact digest. Two runs in one directory would share state and could not
// count as independent.
func TestRunTwoStageRepeatsAcrossIndependentWorkDirectories(t *testing.T) {
	if testing.Short() {
		t.Skip("the repeated two-stage run builds four binaries; skip in -short")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	work := tempDir(t)
	child := filepath.Join(work, "child")
	if runtime.GOOS == "windows" {
		child += ".exe"
	}
	build := exec.Command("go", "build", "-trimpath", "-o", child, "./tools/reference-go-build-with-supply")
	build.Dir = repo
	if out, err := build.CombinedOutput(); err != nil {
		t.Skipf("child build unavailable: %v: %s", err, out)
	}
	supply := supplyRoot(t)
	const target = "./tools/reference-go-build-stage-driver"

	first, err := RunTwoStage(repo, child, target, supply, filepath.Join(work, "run-1"), "")
	if err != nil {
		t.Fatal(err)
	}
	if !first.L3Claimed {
		t.Fatalf("first run did not reach L3: %s", first.FailureCause)
	}
	second, err := RunTwoStage(repo, child, target, supply, filepath.Join(work, "run-2"), "")
	if err != nil {
		t.Fatal(err)
	}
	if !second.L3Claimed {
		t.Fatalf("second run did not reach L3: %s", second.FailureCause)
	}

	firstStage2 := first.Stages[1]
	secondStage2 := second.Stages[1]
	if firstStage2.ArtifactDigest != secondStage2.ArtifactDigest {
		t.Fatalf("independent runs produced different stage2 artifacts: %s vs %s",
			firstStage2.ArtifactDigest, secondStage2.ArtifactDigest)
	}
	// The invoker of each run is the other run's stage1 artifact only if the
	// whole chain is deterministic; check that directly instead of assuming it.
	if firstStage2.InvokerDigest != first.Stages[0].ArtifactDigest {
		t.Fatal("first run's stage2 invoker is not its stage1 artifact")
	}
	if secondStage2.InvokerDigest != second.Stages[0].ArtifactDigest {
		t.Fatal("second run's stage2 invoker is not its stage1 artifact")
	}
	if first.Stages[0].ArtifactDigest != second.Stages[0].ArtifactDigest {
		t.Fatalf("stage1 artifacts differ across runs: %s vs %s",
			first.Stages[0].ArtifactDigest, second.Stages[0].ArtifactDigest)
	}
	if first.SeedGoDigest != second.SeedGoDigest {
		t.Fatalf("runs used different seed toolchains: %s vs %s", first.SeedGoDigest, second.SeedGoDigest)
	}
}
