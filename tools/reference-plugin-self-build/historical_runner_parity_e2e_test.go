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
	"sort"
	"strings"
	"testing"
)

// Pin the last qualified pre-extraction revision, before the actual Go build
// invocation was moved into gobuilder.BuildPackage in PR #1525.
// The runner binaries are compiled independently from different Git trees,
// but must both build the same current source revision and exact Go toolchain.
const historicalPluginGoBuildRunnerRevision = "5726a3475d8930e49f41c907ecfd352c9936192f"

func TestHistoricalProcessPluginRunnerEvidenceParity(t *testing.T) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go executable unavailable: %v", err)
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
	// Use the absolute Go executable without resolving Windows toolcache
	// junctions, which are valid but not always EvalSymlinks-compatible.
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
	// precondition the test cannot control and that made the result depend on
	// scheduling — both sides build from their own detached worktree at a
	// pinned revision. A worktree created from HEAD is clean by construction, so
	// the comparison keeps its meaning without reading the live working tree.
	// A shallow clone lacks the pinned revision, and the later `git show` would
	// fail with a misleading "path ... not in <rev>" message. Fail with the
	// real cause instead of skipping, so a missing history is never a pass.
	if out, err := exec.Command("git", "-C", repo, "cat-file", "-e", historicalPluginGoBuildRunnerRevision+"^{commit}").CombinedOutput(); err != nil {
		t.Fatalf("pinned historical revision %s is not available in this clone (shallow clone?); fetch full history, e.g. `git fetch --unshallow`: %v: %s", historicalPluginGoBuildRunnerRevision, err, out)
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

	temp := t.TempDir()
	if runtime.GOOS == "windows" {
		// Legacy builder output export uses filepath.Rel(root, out) and
		// requires both paths to be on the same Windows volume. The hosted
		// runner normally puts t.TempDir on C: but checks out source on D:.
		// Keep all comparison artifacts OUTSIDE the Git checkout while using
		// its drive, without modifying the pinned historical Runner.
		temp, err = os.MkdirTemp(filepath.Dir(repo), "tei-historical-runner-")
		if err != nil {
			t.Fatalf("allocate isolated same-volume comparison directory: %v", err)
		}
		t.Cleanup(func() { os.RemoveAll(temp) })
	}
	// The current side also runs from its own worktree, so a change another
	// package left in the live working tree cannot alter what is compared
	// (#3637). The revision compared is HEAD, the same commit a clean checkout
	// would have.
	currentTree := filepath.Join(temp, "current-worktree")
	addCurrent := exec.Command("git", "-C", repo, "worktree", "add", "--detach", currentTree, "HEAD")
	if out, err := addCurrent.CombinedOutput(); err != nil {
		t.Fatalf("create current checkout at HEAD: %v: %s", err, out)
	}
	t.Cleanup(func() {
		cmd := exec.Command("git", "-C", repo, "worktree", "remove", "--force", currentTree)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("remove current checkout: %v: %s", err, output)
		}
	})
	if output := git("-C", currentTree, "status", "--porcelain=v1", "--untracked-files=all"); len(output) != 0 {
		t.Fatalf("current checkout at HEAD is not clean: %s", output)
	}
	currentSource, err := os.ReadFile(filepath.Join(currentTree, "tools", "reference-plugin-self-build", "main.go"))
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

	// The historical runner pins a manifest parser that does not know
	// formatVersion. Rather than keeping the live Workspaces on the old format
	// forever, the comparison runs both runners against a frozen commit whose
	// two manifests are restored from testdata (#1920 Owner decision).
	frozenTree := filepath.Join(temp, "frozen-worktree")
	addFrozen := exec.Command("git", "-C", repo, "worktree", "add", "--detach", frozenTree, "HEAD")
	if out, err := addFrozen.CombinedOutput(); err != nil {
		t.Fatalf("create frozen checkout: %v: %s", err, out)
	}
	t.Cleanup(func() {
		cmd := exec.Command("git", "-C", repo, "worktree", "remove", "--force", frozenTree)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("remove frozen checkout: %v: %s", err, output)
		}
	})
	frozenManifests := map[string]string{
		filepath.Join("reference", "workspace", "tei.workspace.yaml"):                      "tools/reference-plugin-self-build/testdata/frozen-workspace/package.tei.workspace.yaml",
		filepath.Join("reference", "plugin-self-build", "workspace", "tei.workspace.yaml"): "tools/reference-plugin-self-build/testdata/frozen-workspace/reference.tei.workspace.yaml",
	}
	for target, source := range frozenManifests {
		raw, err := os.ReadFile(filepath.Join(currentTree, filepath.FromSlash(source)))
		if err != nil {
			t.Fatalf("read frozen manifest %s: %v", source, err)
		}
		if err := os.WriteFile(filepath.Join(frozenTree, target), raw, 0o644); err != nil {
			t.Fatalf("write frozen manifest %s: %v", target, err)
		}
	}
	commit := exec.Command("git", "-C", frozenTree,
		"-c", "user.name=tei-frozen-workspace",
		"-c", "user.email=tei-frozen-workspace@example.invalid",
		"commit", "-am", "freeze pre-formatVersion workspace manifests for runner parity")
	if out, err := commit.CombinedOutput(); err != nil {
		t.Fatalf("commit frozen checkout: %v: %s", err, out)
	}
	frozenRevision := string(git("-C", frozenTree, "rev-parse", "HEAD"))
	// The frozen tree must differ from the live revision only by the two
	// manifests that the pinned parser cannot read in the new format.
	liveRevision := string(git("rev-parse", "HEAD"))
	changedPaths := strings.Fields(string(git("-C", repo, "diff", "--name-only", liveRevision, frozenRevision)))
	sort.Strings(changedPaths)
	wantChanged := []string{
		"reference/plugin-self-build/workspace/tei.workspace.yaml",
		"reference/workspace/tei.workspace.yaml",
	}
	if !reflect.DeepEqual(changedPaths, wantChanged) {
		t.Fatalf("frozen comparison changed unexpected paths: got %v want %v", changedPaths, wantChanged)
	}

	// Build two distinct, real plugin self-build runner executables.
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	oldCLI := filepath.Join(temp, "historical-plugin-builder"+ext)
	newCLI := filepath.Join(temp, "current-plugin-builder"+ext)
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
	compile("current", currentTree, newCLI)

	run := func(label, executable, checkout string) evidence {
		t.Helper()
		cmd := exec.Command(executable, "-repo", checkout, "-go", goPath)
		cmd.Dir = checkout
		// The frozen comparison tree intentionally predates formatVersion, so
		// the current runner needs the explicit frozen-artifact opt-in. The
		// historical runner ignores the variable.
		cmd.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=", "GOTOOLCHAIN=local", "GOPROXY=off",
			"TEI_WORKSPACE_ALLOW_UNSPECIFIED_FORMAT=1")
		if runtime.GOOS == "windows" {
			// The runners create isolated build state with os.MkdirTemp("", ...).
			// A Windows host that runs the executable from a non-system drive
			// cannot create directories under the system temp directory
			// (observed as `mkdir ...: Access is denied`), while the same
			// executable can when TMP/TEMP point at a directory on its own
			// volume. Point the inherited temp variables at the comparison
			// directory, which is already allocated on that volume.
			cmd.Env = append(cmd.Env, "TMP="+temp, "TEMP="+temp, "TMPDIR="+temp)
		}
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
		rev := string(git("-C", checkout, "rev-parse", "HEAD"))
		if ev.RepositoryRevision != rev {
			t.Fatalf("%s built different source revision: got %s want %s", label, ev.RepositoryRevision, rev)
		}
		return ev
	}
	oldEvidence := run("historical", oldCLI, frozenTree)
	newEvidence := run("current", newCLI, frozenTree)
	if oldEvidence.RepositoryRevision != frozenRevision || newEvidence.RepositoryRevision != frozenRevision {
		t.Fatalf("runners did not build the frozen comparison revision %s: old=%s new=%s",
			frozenRevision, oldEvidence.RepositoryRevision, newEvidence.RepositoryRevision)
	}

	// The comparison covers the actual build inputs and outputs. Repository
	// identity fields are excluded by the frozen-commit design (#1920), so a
	// manifest migration does not need to keep the live Workspaces frozen.
	if oldEvidence.GoVersion != newEvidence.GoVersion ||
		oldEvidence.ProcessSourceClosureDigest != newEvidence.ProcessSourceClosureDigest ||
		!reflect.DeepEqual(oldEvidence.ProcessSourceClosure, newEvidence.ProcessSourceClosure) ||
		oldEvidence.GoBinaryDigest != newEvidence.GoBinaryDigest ||
		oldEvidence.GoToolchainDigest != newEvidence.GoToolchainDigest ||
		oldEvidence.PackageBuilderFirstDigest != newEvidence.PackageBuilderFirstDigest ||
		oldEvidence.PackageBuilderSecondDigest != newEvidence.PackageBuilderSecondDigest ||
		oldEvidence.PackageBuildPlanFingerprint != newEvidence.PackageBuildPlanFingerprint ||
		oldEvidence.PackageFirstArtifactDigest != newEvidence.PackageFirstArtifactDigest ||
		oldEvidence.PackageSecondArtifactDigest != newEvidence.PackageSecondArtifactDigest ||
		oldEvidence.PackageLockFingerprint != newEvidence.PackageLockFingerprint ||
		oldEvidence.NativeSourceFingerprint != newEvidence.NativeSourceFingerprint ||
		oldEvidence.NativeDefinitionFingerprint != newEvidence.NativeDefinitionFingerprint ||
		oldEvidence.NativeBuilderFirstDigest != newEvidence.NativeBuilderFirstDigest ||
		oldEvidence.NativeBuilderSecondDigest != newEvidence.NativeBuilderSecondDigest ||
		oldEvidence.NativeBuildPlanFingerprint != newEvidence.NativeBuildPlanFingerprint ||
		oldEvidence.NativeFirstArtifactDigest != newEvidence.NativeFirstArtifactDigest ||
		oldEvidence.NativeSecondArtifactDigest != newEvidence.NativeSecondArtifactDigest ||
		oldEvidence.NativeLockFingerprint != newEvidence.NativeLockFingerprint ||
		oldEvidence.FirstArtifactDigest != newEvidence.FirstArtifactDigest ||
		oldEvidence.SecondArtifactDigest != newEvidence.SecondArtifactDigest ||
		oldEvidence.LockedBuild.ArtifactDigest != newEvidence.LockedBuild.ArtifactDigest {
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
		t.Fatalf("historical/current Plugin self-build Evidence drift: old=sha256:%s new=sha256:%s\nold=%s\nnew=%s",
			hex.EncodeToString(oldSum[:]), hex.EncodeToString(newSum[:]), oldJSON, newJSON)
	}
	// The comparison builds both sides from detached worktrees, so the invariant
	// worth asserting is that those worktrees are still exactly as committed.
	// The live checkout is not an input any more, and failing on it would make
	// this test depend on what other packages did (#3637).
	for label, checkout := range map[string]string{"historical": historicalTree, "current": currentTree} {
		if status := git("-C", checkout, "status", "--porcelain=v1", "--untracked-files=all"); len(status) != 0 {
			t.Fatalf("comparison modified its own %s checkout: %s", label, status)
		}
	}
	newSum := sha256.Sum256([]byte(newEvidence.ProcessSourceClosureDigest + newEvidence.FirstArtifactDigest + newEvidence.NativeFirstArtifactDigest))
	t.Logf("historical/current self-build build evidence equal on frozen revision %s: selectedSource=%s processPlugin=%s combinedSHA256=%s",
		frozenRevision, newEvidence.ProcessSourceClosureDigest, newEvidence.FirstArtifactDigest, hex.EncodeToString(newSum[:]))
}
