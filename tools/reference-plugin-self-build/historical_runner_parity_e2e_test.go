package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// Pin the last qualified pre-extraction revision, before the actual Go build
// invocation was moved into gobuilder.BuildPackage in PR #1525.
// The runner binaries are compiled independently from different Git trees,
// but must both build the same current source revision and exact Go toolchain.
const historicalPluginGoBuildRunnerRevision = "5726a3475d8930e49f41c907ecfd352c9936192f"

func TestHistoricalProcessPluginRunnerEvidenceParity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("historical Git-worktree CLI qualification needs a separate Windows runner")
	}
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go executable unavailable: %v", err)
	}
	goPath, err = filepath.Abs(goPath)
	if err != nil {
		t.Fatal(err)
	}
	goPath, err = filepath.EvalSymlinks(goPath)
	if err != nil {
		t.Fatal(err)
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
	if err := requireCleanRepository(repo); err != nil {
		t.Fatalf("comparison requires the same clean committed source revision: %v", err)
	}
	git := func(args ...string) []byte {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, data)
		}
		return bytes.TrimSpace(data)
	}
	oldSource := git("show", historicalPluginGoBuildRunnerRevision+":tools/reference-plugin-self-build/main.go")
	currentSource, err := os.ReadFile(filepath.Join(repo, "tools", "reference-plugin-self-build", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	before := []byte("cmd := exec.Command(")
	extracted := []byte("return gobuilder.BuildPackage(repo, goPath, packagePath, output, stateDir)")
	if !bytes.Contains(oldSource, before) ||
		!bytes.Contains(oldSource, []byte("\"-mod=readonly\"")) ||
		bytes.Contains(oldSource, extracted) ||
		!bytes.Contains(currentSource, extracted) {
		t.Fatal("pinned historical and current source are not opposite sides of the exact Go builder extraction")
	}

	temp := t.TempDir()
	historicalTree := filepath.Join(temp, "historic-worktree")
	add := exec.Command("git", "-C", repo, "worktree", "add", "--detach", historicalTree, historicalPluginGoBuildRunnerRevision)
	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("create historical checkout from full Git history: %v: %s", err, out)
	}
	t.Cleanup(func() {
		cmd := exec.Command("git", "-C", repo, "worktree", "remove", "--force", historicalTree)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("remove historical checkout: %v: %s", err, output)
		}
	})
	if got := git("-C", historicalTree, "rev-parse", "HEAD"); string(got) != historicalPluginGoBuildRunnerRevision {
		t.Fatalf("historical revision changed: got %s", got)
	}
	if output := git("-C", historicalTree, "status", "--porcelain=v1", "--untracked-files=all"); len(output) != 0 {
		t.Fatalf("historical checkout unclean: %s", output)
	}
	// Build two distinct, real plugin self-build runner executables.
	oldCLI := filepath.Join(temp, "historical-plugin-builder")
	newCLI := filepath.Join(temp, "current-plugin-builder")
	compile := func(label, checkout, output string) {
		t.Helper()
		cmd := exec.Command(goPath, "build", "-trimpath", "-buildvcs=false",
			"-mod=readonly", "-o", output, "./tools/reference-plugin-self-build")
		cmd.Dir = checkout
		cmd.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=", "GOTOOLCHAIN=local", "GOPROXY=off")
		if raw, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("compile %s plugin builder: %v: %s", label, err, raw)
		}
	}
	compile("historical", historicalTree, oldCLI)
	compile("current", repo, newCLI)

	run := func(label, executable string) evidence {
		t.Helper()
		cmd := exec.Command(executable, "-repo", repo, "-go", goPath)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=", "GOTOOLCHAIN=local", "GOPROXY=off")
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("%s self-build failed: %v stdout=%s stderr=%s", label, err, stdout.String(), stderr.String())
		}
		var ev evidence
		decoder := json.NewDecoder(&stdout)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&ev); err != nil {
			t.Fatalf("%s self-build Evidence invalid: %v: %s", label, err, stdout.String())
		}
		var trailing any
		if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
			t.Fatalf("%s runner emitted trailing Evidence: %v", label, err)
		}
		if !ev.Reproducible ||
			ev.FirstArtifactDigest == "" ||
			ev.FirstArtifactDigest != ev.SecondArtifactDigest ||
			ev.ProcessSourceClosureDigest == "" ||
			ev.GoToolchainDigest == "" ||
			ev.PackageBuilderFirstDigest != ev.PackageBuilderSecondDigest ||
			ev.NativeBuilderFirstDigest != ev.NativeBuilderSecondDigest ||
			ev.PackageFirstArtifactDigest != ev.PackageSecondArtifactDigest ||
			ev.NativeFirstArtifactDigest != ev.NativeSecondArtifactDigest ||
			ev.LockedBuild.ArtifactDigest != ev.FirstArtifactDigest {
			t.Fatalf("%s did not accept a fully converged self-build: %+v", label, ev)
		}
		rev := string(git("rev-parse", "HEAD"))
		if ev.RepositoryRevision != rev {
			t.Fatalf("%s built different source revision: got %s want %s", label, ev.RepositoryRevision, rev)
		}
		return ev
	}
	oldEvidence := run("historical", oldCLI)
	newEvidence := run("current", newCLI)

	// Go compiler's actual selected files, compiler closure identity, all
	// nested Plugin BuildPlan/Lock and transform convergence must be equal.
	if oldEvidence.ProcessSourceClosureDigest != newEvidence.ProcessSourceClosureDigest ||
		!reflect.DeepEqual(oldEvidence.ProcessSourceClosure, newEvidence.ProcessSourceClosure) ||
		oldEvidence.GoBinaryDigest != newEvidence.GoBinaryDigest ||
		oldEvidence.GoToolchainDigest != newEvidence.GoToolchainDigest {
		t.Fatal("historical/current runners used different actual Go build inputs")
	}
	oldJSON, err := json.Marshal(oldEvidence)
	if err != nil {
		t.Fatal(err)
	}
	newJSON, err := json.Marshal(newEvidence)
	if err != nil {
		t.Fatal(err)
	}
	oldSum := sha256.Sum256(oldJSON)
	newSum := sha256.Sum256(newJSON)
	if !bytes.Equal(oldJSON, newJSON) {
		t.Fatalf("historical/current Plugin self-build Evidence drift: old=sha256:%s new=sha256:%s\nold=%s\nnew=%s",
			hex.EncodeToString(oldSum[:]), hex.EncodeToString(newSum[:]), oldJSON, newJSON)
	}
	if status := git("status", "--porcelain=v1", "--untracked-files=all"); len(status) != 0 {
		t.Fatalf("comparison modified current source checkout: %s", status)
	}
	t.Logf("historical/current self-build Evidence equal: selectedSource=%s processPlugin=%s fullEvidenceSHA256=%s",
		newEvidence.ProcessSourceClosureDigest, newEvidence.FirstArtifactDigest, hex.EncodeToString(newSum[:]))
}
