package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "tei-two-stage-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := removeWritableTree(dir); err != nil {
			t.Errorf("remove isolated state %s: %v", dir, err)
		}
	})
	return dir
}

func removeWritableTree(root string) error {
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

// TestJudgeKeepsTheThreeVerdictsSeparate pins the precedence: an invoker
// mismatch is not rescued by equal artifacts, and equal artifacts from a
// different invoker are two unrelated builds.
func TestJudgeKeepsTheThreeVerdictsSeparate(t *testing.T) {
	base := StageRecord{
		ChildDigest:    "sha256:child",
		TargetPackage:  "./target",
		SupplyRoot:     "/supply",
		ArtifactDigest: "sha256:a1",
	}
	stage1 := base
	stage1.InvokerDigest = "sha256:seed"

	cases := []struct {
		name             string
		stage2           StageRecord
		wantInvoker      bool
		wantRecipe       bool
		wantArtifacts    bool
		wantCauseMatches string
	}{
		{
			name: "true self build",
			stage2: StageRecord{
				InvokerDigest:  "sha256:a1",
				ChildDigest:    "sha256:child",
				TargetPackage:  "./target",
				SupplyRoot:     "/supply",
				ArtifactDigest: "sha256:a1",
			},
			wantInvoker: true, wantRecipe: true, wantArtifacts: true,
		},
		{
			name: "same artifact but stage1 did not invoke stage2",
			stage2: StageRecord{
				InvokerDigest:  "sha256:seed",
				ChildDigest:    "sha256:child",
				TargetPackage:  "./target",
				SupplyRoot:     "/supply",
				ArtifactDigest: "sha256:a1",
			},
			wantInvoker: false, wantRecipe: true, wantArtifacts: true,
			wantCauseMatches: "not invoked by the stage1 artifact",
		},
		{
			name: "different recipe",
			stage2: StageRecord{
				InvokerDigest:  "sha256:a1",
				ChildDigest:    "sha256:other-child",
				TargetPackage:  "./target",
				SupplyRoot:     "/supply",
				ArtifactDigest: "sha256:a1",
			},
			wantInvoker: true, wantRecipe: false, wantArtifacts: true,
			wantCauseMatches: "same child and recipe",
		},
		{
			name: "non converging artifacts",
			stage2: StageRecord{
				InvokerDigest:  "sha256:a1",
				ChildDigest:    "sha256:child",
				TargetPackage:  "./target",
				SupplyRoot:     "/supply",
				ArtifactDigest: "sha256:a2",
			},
			wantInvoker: true, wantRecipe: true, wantArtifacts: false,
			wantCauseMatches: "artifacts differ",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			invoker, recipe, artifacts, cause := Judge(stage1, tc.stage2)
			if invoker != tc.wantInvoker || recipe != tc.wantRecipe || artifacts != tc.wantArtifacts {
				t.Fatalf("verdicts = invoker:%v recipe:%v artifacts:%v, want %v/%v/%v",
					invoker, recipe, artifacts, tc.wantInvoker, tc.wantRecipe, tc.wantArtifacts)
			}
			if tc.wantCauseMatches == "" {
				if cause != "" {
					t.Fatalf("successful case reported a cause: %s", cause)
				}
				return
			}
			if !strings.Contains(cause, tc.wantCauseMatches) {
				t.Fatalf("cause %q does not mention %q", cause, tc.wantCauseMatches)
			}
		})
	}
}

func TestJudgeRejectsAStage2InvokerThatIsNotTheStage1Artifact(t *testing.T) {
	stage1 := StageRecord{InvokerDigest: "sha256:seed", ArtifactDigest: "sha256:a1", ChildDigest: "sha256:c", TargetPackage: "p", SupplyRoot: "s"}
	stage2 := StageRecord{InvokerDigest: "sha256:a9", ArtifactDigest: "sha256:a1", ChildDigest: "sha256:c", TargetPackage: "p", SupplyRoot: "s"}
	invoker, _, artifacts, cause := Judge(stage1, stage2)
	if invoker {
		t.Fatal("an unrelated invoker was accepted")
	}
	if !artifacts {
		t.Fatal("the artifact comparison was skipped")
	}
	if cause == "" {
		t.Fatal("mismatch had no stated cause")
	}
}

// TestRunTwoStageUsesTheStage1ArtifactAsStage2Invoker runs the real experiment
// with a Go child. It skips when the toolchain or supply is unavailable, so a
// missing fixture is never reported as a result.
func TestRunTwoStageUsesTheStage1ArtifactAsStage2Invoker(t *testing.T) {
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
	evidence, err := RunTwoStage(repo, child, "./tools/reference-go-build-stage-driver", supply, filepath.Join(work, "run"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence.Stages) != 2 {
		t.Fatalf("expected two stages, got %d", len(evidence.Stages))
	}
	if !evidence.Stage1DrivesStage2 {
		t.Fatalf("stage1 artifact did not invoke stage2: %s", evidence.FailureCause)
	}
	if !evidence.ArtifactsEqual {
		t.Fatalf("stage1 and stage2 artifacts differ: %s", evidence.FailureCause)
	}
	if !evidence.L3Claimed {
		t.Fatalf("converged experiment did not claim L3: %s", evidence.FailureCause)
	}
	if evidence.Boundary == "" {
		t.Fatal("evidence lost its boundary statement")
	}

	// The record must survive a JSON round trip, because callers read it.
	raw, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	var decoded TwoStageEvidence
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Stages) != 2 || !decoded.L3Claimed {
		t.Fatalf("round trip lost the comparison: %+v", decoded)
	}
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
