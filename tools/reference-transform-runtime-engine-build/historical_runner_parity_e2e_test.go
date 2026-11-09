package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// The exact pre-extraction reference commit still has its own direct
// exec.Command("go", "build", ...) implementation; #1525 extracted that
// execution into tools/internal/gobuilder.BuildPackage. A fixed commit avoids
// silently changing the historical comparison baseline with main.
const preSharedGoBuildRunnerRevision = "5726a3475d8930e49f41c907ecfd352c9936192f"

// TestHistoricalRuntimeEngineRunnerParity builds TWO different generations of
// the actual Engine builder CLI, then runs both against the SAME clean current
// checkout using the SAME exact Go executable.
//
// Both runners produce independent engine artifacts and evidence. This tests
// the historical *runner implementation*, not merely a hand-copied Go recipe.
// It does not compare artifacts from two different application source trees.
func TestHistoricalRuntimeEngineRunnerParity(t *testing.T) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go unavailable: %v", err)
	}
	// The Windows CI can identify the physical Go installation behind the
	// hosted toolcache junction. Pin that absolute executable so the pinned
	// historical SelectExecutable (which cannot be modified) receives a
	// path whose EvalSymlinks is supported by the current Windows runner.
	if physical := os.Getenv("TEI_HISTORICAL_GO_EXECUTABLE"); physical != "" {
		goPath = physical
	}
	goPath, err = filepath.Abs(goPath)
	if err != nil {
		t.Fatal(err)
	}
	// Windows hosted Go toolcache may contain directory junctions. The
	// absolute executable selected by LookPath is sufficient; resolving
	// junctions via EvalSymlinks is not portable across Windows agents.
	goInfo, err := os.Stat(goPath)
	if err != nil || !goInfo.Mode().IsRegular() {
		t.Fatalf("selected Go executable is not a regular file: %q (%v)", goPath, err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("Git unavailable: %v", err)
	}
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	repo, err = filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	// The comparison must run against committed source, not a working tree that
	// another package in the same `go test ./...` run may be writing into
	// (#3637). Rather than requiring the whole live checkout to be clean — a
	// precondition this test cannot control and that made the result depend on
	// scheduling — both sides build from their own detached worktree at a
	// pinned revision. A worktree created from HEAD is clean by construction, so
	// the comparison keeps its meaning without reading the live working tree.
	// A shallow clone lacks the pinned revision, and the later `git show` would
	// fail with a misleading "path ... not in <rev>" message. Fail with the
	// real cause instead of skipping, so a missing history is never a pass.
	if out, err := exec.Command("git", "-C", repo, "cat-file", "-e", preSharedGoBuildRunnerRevision+"^{commit}").CombinedOutput(); err != nil {
		t.Fatalf("pinned historical revision %s is not available in this clone (shallow clone?); fetch full history, e.g. `git fetch --unshallow`: %v: %s", preSharedGoBuildRunnerRevision, err, out)
	}
	git := func(dir string, args ...string) []byte {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return bytes.TrimSpace(out)
	}
	// Historical history must exist locally: a shallow clone is NOT a valid
	// success case. CI intentionally checks out a full Git history.
	oldRunnerSource := git(repo, "show", preSharedGoBuildRunnerRevision+":tools/reference-transform-runtime-engine-build/main.go")

	tmp := t.TempDir()
	if runtime.GOOS == "windows" {
		// Legacy builder output export uses filepath.Rel(root, out) and
		// requires both paths to be on the same Windows volume. The hosted
		// runner normally puts t.TempDir on C: but checks out source on D:.
		// Keep all comparison artifacts OUTSIDE the Git checkout while using
		// its drive, without modifying the pinned historical Runner.
		tmp, err = os.MkdirTemp(filepath.Dir(repo), "tei-historical-runner-")
		if err != nil {
			t.Fatalf("allocate isolated same-volume comparison directory: %v", err)
		}
		t.Cleanup(func() { os.RemoveAll(tmp) })
	}
	oldCheckout := filepath.Join(tmp, "historical-checkout")
	add := exec.Command("git", "-C", repo, "worktree", "add", "--detach", oldCheckout, preSharedGoBuildRunnerRevision)
	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("create exact historical detached checkout: %v: %s", err, out)
	}
	t.Cleanup(func() {
		cmd := exec.Command("git", "-C", repo, "worktree", "remove", "--force", oldCheckout)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("remove historical checkout: %v: %s", err, out)
		}
	})
	if got := string(git(oldCheckout, "rev-parse", "HEAD")); got != preSharedGoBuildRunnerRevision {
		t.Fatalf("historical revision changed: got %s want %s", got, preSharedGoBuildRunnerRevision)
	}
	if got := git(oldCheckout, "status", "--porcelain=v1", "--untracked-files=all"); len(got) != 0 {
		t.Fatalf("historical checkout is not clean: %s", got)
	}
	// The current side also runs from its own worktree, so a change another
	// package left in the live working tree cannot alter what is compared
	// (#3637). The revision compared is HEAD, the same commit a clean checkout
	// would have.
	currentCheckout := filepath.Join(tmp, "current-checkout")
	addCurrent := exec.Command("git", "-C", repo, "worktree", "add", "--detach", currentCheckout, "HEAD")
	if out, err := addCurrent.CombinedOutput(); err != nil {
		t.Fatalf("create current checkout at HEAD: %v: %s", err, out)
	}
	t.Cleanup(func() {
		cmd := exec.Command("git", "-C", repo, "worktree", "remove", "--force", currentCheckout)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("remove current checkout: %v: %s", err, out)
		}
	})
	if got := git(currentCheckout, "status", "--porcelain=v1", "--untracked-files=all"); len(got) != 0 {
		t.Fatalf("current checkout at HEAD is not clean: %s", got)
	}
	currentRunnerSource, err := os.ReadFile(filepath.Join(currentCheckout, "tools", "reference-transform-runtime-engine-build", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(oldRunnerSource, []byte("cmd := exec.Command(")) ||
		!bytes.Contains(oldRunnerSource, []byte("\"-mod=readonly\"")) ||
		bytes.Contains(oldRunnerSource, []byte("return gobuilder.BuildPackage(repo, goPath, packagePath, output, stateDir)")) ||
		!bytes.Contains(currentRunnerSource, []byte("return gobuilder.BuildPackage(repo, goPath, packagePath, output, stateDir)")) {
		t.Fatal("historical/current builder implementations no longer represent opposite sides of Go BuildPackage extraction")
	}

	// Compile both REAL runner executables independently. They are different
	// programs from different Git commits; neither is a simulated adapter.
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	binDir := tmp
	if runtime.GOOS == "windows" {
		// Binaries executed from the repository drive are denied access to the
		// user temp directory on this host. Build them on the system temp drive;
		// artifacts still stay on the repository volume for filepath.Rel.
		binDir, err = os.MkdirTemp("", "tei-historical-runner-bin-")
		if err != nil {
			t.Fatalf("allocate runner binary directory: %v", err)
		}
		t.Cleanup(func() { os.RemoveAll(binDir) })
	}
	oldCLI := filepath.Join(binDir, "old-engine-builder"+ext)
	newCLI := filepath.Join(binDir, "new-engine-builder"+ext)
	buildRunner := func(label, checkout, executable string) {
		t.Helper()
		cmd := exec.Command(goPath, "build", "-trimpath", "-buildvcs=false",
			"-mod=readonly", "-o", executable, "./tools/reference-transform-runtime-engine-build")
		cmd.Dir = checkout
		cmd.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=", "GOTOOLCHAIN=local", "GOPROXY=off")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("compile %s runner at %s: %v: %s", label, checkout, err, out)
		}
	}
	buildRunner("historical", oldCheckout, oldCLI)
	buildRunner("current", currentCheckout, newCLI)

	type output struct {
		evidence buildEvidence
		artifact []byte
	}
	runRunner := func(label, cli string) output {
		t.Helper()
		artifactPath := filepath.Join(tmp, label+"-engine"+ext)
		cmd := exec.Command(cli, "-repo", currentCheckout, "-go", goPath, "-artifact-out", artifactPath)
		cmd.Dir = currentCheckout
		cmd.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=", "GOTOOLCHAIN=local", "GOPROXY=off")
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("%s real runner failed: %v; stdout=%q stderr=%q", label, err, stdout.String(), stderr.String())
		}
		decoder := json.NewDecoder(&stdout)
		decoder.DisallowUnknownFields()
		var ev buildEvidence
		if err := decoder.Decode(&ev); err != nil {
			t.Fatalf("%s evidence is not valid: %v: %s", label, err, stdout.String())
		}
		var trailing any
		if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
			t.Fatalf("%s evidence contains trailing data: %v", label, err)
		}
		if !ev.Reproducible || ev.ArtifactDigest == "" ||
			ev.ArtifactDigest != ev.FirstArtifactDigest ||
			ev.ArtifactDigest != ev.SecondArtifactDigest ||
			ev.RuntimeEngine == nil ||
			ev.RuntimeEngine.ImplementationDigest != ev.ArtifactDigest {
			t.Fatalf("%s runner did not create accepted reproducible evidence: %+v", label, ev)
		}
		recomputed, err := evidenceFingerprint(ev)
		if err != nil || recomputed != ev.EvidenceFingerprint {
			t.Fatalf("%s Evidence fingerprint mismatch: actual %q recalculated %q (%v)",
				label, ev.EvidenceFingerprint, recomputed, err)
		}
		raw, err := os.ReadFile(artifactPath)
		if err != nil {
			t.Fatalf("%s runner did not publish independently verified engine: %v", label, err)
		}
		sum := sha256.Sum256(raw)
		actualDigest := "sha256:" + hex.EncodeToString(sum[:])
		if len(raw) == 0 || actualDigest != ev.ArtifactDigest {
			t.Fatalf("%s published engine digest does not match evidence: %s vs %s", label, actualDigest, ev.ArtifactDigest)
		}
		return output{evidence: ev, artifact: raw}
	}
	historical := runRunner("historical", oldCLI)
	current := runRunner("current", newCLI)

	if historical.evidence.RepositoryRevision != string(git(currentCheckout, "rev-parse", "HEAD")) ||
		current.evidence.RepositoryRevision != historical.evidence.RepositoryRevision {
		t.Fatal("runners did not build the same current repository revision")
	}
	if historical.evidence.SourceClosureDigest != current.evidence.SourceClosureDigest ||
		!reflect.DeepEqual(historical.evidence.SourceClosure, current.evidence.SourceClosure) ||
		historical.evidence.Builder != current.evidence.Builder {
		t.Fatal("selected source files or Go toolchain identity differs between historical and modern runner")
	}
	if !bytes.Equal(historical.artifact, current.artifact) {
		t.Fatalf("different generated engine bytes: historical=%s current=%s source=%s",
			historical.evidence.ArtifactDigest, current.evidence.ArtifactDigest,
			current.evidence.SourceClosureDigest)
	}
	// The Evidence schema, full content and fingerprint must also remain
	// stable. Both inputs have the same revision; runner build revisions differ.
	if !reflect.DeepEqual(historical.evidence, current.evidence) {
		a, _ := json.Marshal(historical.evidence)
		b, _ := json.Marshal(current.evidence)
		t.Fatalf("build Evidence meaning drifted between historical and current runner:\nhistorical=%s\ncurrent=%s", a, b)
	}
	// The comparison reads only its own detached worktrees, so the invariant
	// worth asserting is that those worktrees are still exactly as committed.
	// The live checkout is not an input any more, and failing on it would make
	// this test depend on what other packages did (#3637).
	for label, checkout := range map[string]string{"historical": oldCheckout, "current": currentCheckout} {
		if status := git(checkout, "status", "--porcelain=v1", "--untracked-files=all"); len(status) != 0 {
			t.Fatalf("comparison modified its own %s checkout: %s", label, status)
		}
	}
	t.Log(fmt.Sprintf("historical runner %s and current runner agree: source=%s engine=%s evidence=%s",
		preSharedGoBuildRunnerRevision,
		current.evidence.SourceClosureDigest,
		current.evidence.ArtifactDigest,
		current.evidence.EvidenceFingerprint))
}
