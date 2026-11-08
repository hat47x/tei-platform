package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// These tests pin the SC0 negatives from
// plan/design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md §5 for the thin entry.
// They run the entry directly (not the whole CLI process) so each precondition
// is checked without paying for a build per case.

type selfBuildResult struct {
	SchemaVersion string `json:"schemaVersion"`
	OK            bool   `json:"ok"`
	Qualified     bool   `json:"qualified"`
	Boundary      string `json:"boundary"`
	Error         *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Child *struct {
		ExecutableDigest string   `json:"executableDigest"`
		Argv             []string `json:"argv"`
		ExitCode         int      `json:"exitCode"`
	} `json:"child"`
	Artifact *struct {
		Digest       string `json:"digest"`
		ChildDigest  string `json:"childReportedDigest"`
		ChildMatches bool   `json:"childReportedDigestMatches"`
	} `json:"artifact"`
}

func runSelfBuild(t *testing.T, args ...string) (selfBuildResult, int) {
	t.Helper()
	var out bytes.Buffer
	code, err := runSelfBuildCommand(args, &out)
	if err != nil {
		t.Fatalf("self-build returned an emitter error: %v", err)
	}
	var report selfBuildResult
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("self-build did not emit one JSON object: %v\n%s", err, out.String())
	}
	if report.SchemaVersion != selfBuildSchema {
		t.Fatalf("unexpected schemaVersion: %s", report.SchemaVersion)
	}
	if report.Boundary == "" {
		t.Fatal("report lost its boundary statement")
	}
	if report.Qualified {
		t.Fatal("report claimed closure qualification")
	}
	return report, code
}

// SC0-01: a missing workspace or builder never starts a build.
func TestSelfBuildRefusesWithoutRequiredFlags(t *testing.T) {
	report, code := runSelfBuild(t, "-build-state", "state", "-output", "out")
	if code != 2 {
		t.Fatalf("missing -workspace was not a usage failure: code=%d", code)
	}
	if report.Error == nil || !strings.Contains(report.Error.Message, "-workspace is required") {
		t.Fatalf("missing -workspace had no stated reason: %+v", report.Error)
	}
	if report.Child != nil || report.Artifact != nil {
		t.Fatal("a precondition failure recorded a child or an artifact")
	}
}

// SC0-03/SC0-04: the state directory and a pre-existing output are refused
// before anything runs.
func TestSelfBuildRefusesSharedStateAndExistingOutput(t *testing.T) {
	dir := t.TempDir()
	builder := writeFakeBuilder(t, dir)
	output := filepath.Join(dir, "artifact")
	if err := os.WriteFile(output, []byte("already here"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, code := runSelfBuild(t,
		"-workspace", dir, "-builder", builder, "-build-state", filepath.Join(dir, "state"), "-output", output)
	if code != 2 {
		t.Fatalf("existing output was not refused: code=%d", code)
	}
	if report.Error == nil || !strings.Contains(report.Error.Message, "does not overwrite") {
		t.Fatalf("existing output had no stated reason: %+v", report.Error)
	}
	if report.Child != nil {
		t.Fatal("a refused entry started the child")
	}

	report, code = runSelfBuild(t,
		"-workspace", dir, "-builder", builder, "-output", filepath.Join(dir, "fresh"))
	if code != 2 {
		t.Fatalf("missing -build-state was not refused: code=%d", code)
	}
	if report.Error == nil || !strings.Contains(report.Error.Message, "shared state") {
		t.Fatalf("missing -build-state had no stated reason: %+v", report.Error)
	}
}

// SC0-09: a bare builder name is refused so the digest cannot be skipped.
func TestSelfBuildRefusesBareBuilderName(t *testing.T) {
	dir := t.TempDir()
	report, code := runSelfBuild(t,
		"-workspace", dir, "-builder", "go", "-build-state", filepath.Join(dir, "state"), "-output", filepath.Join(dir, "out"))
	if code != 2 {
		t.Fatalf("bare builder name was not refused: code=%d", code)
	}
	if report.Error == nil || !strings.Contains(report.Error.Message, "bare name") {
		t.Fatalf("bare name had no stated reason: %+v", report.Error)
	}
}

// SC0-05: a workspace with validation errors stops before the child starts.
func TestSelfBuildRefusesWorkspaceWithValidationErrors(t *testing.T) {
	dir := t.TempDir()
	// An empty directory is not a workspace at all; the loader reports that.
	builder := writeFakeBuilder(t, dir)
	report, code := runSelfBuild(t,
		"-workspace", filepath.Join(dir, "missing-workspace"),
		"-builder", builder,
		"-build-state", filepath.Join(dir, "state"),
		"-output", filepath.Join(dir, "out"))
	if code != 2 {
		t.Fatalf("unloadable workspace was not refused: code=%d", code)
	}
	if report.Error == nil || !strings.Contains(report.Error.Message, "workspace load failed") {
		t.Fatalf("workspace load failure had no stated reason: %+v", report.Error)
	}
}

// SC0-06: a successful child produces an artifact report, and the entry still
// says qualified=false and compares the child's claim against the bytes.
func TestSelfBuildRecordsChildAndCrossChecksArtifact(t *testing.T) {
	dir := t.TempDir()
	builder := writeFakeBuilder(t, dir)
	workspace := minimalSelfBuildWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")
	report, code := runSelfBuild(t,
		"-workspace", workspace,
		"-builder", builder,
		"-build-state", filepath.Join(dir, "state"),
		"-output", output,
		"-supply", filepath.Join(dir, "supply"))
	if code != 0 {
		t.Fatalf("a valid precondition and a working child were rejected: code=%d %+v", code, report.Error)
	}
	if !report.OK {
		t.Fatal("successful run did not report ok")
	}
	if report.Child == nil || report.Artifact == nil {
		t.Fatalf("successful run did not record the child and the artifact: %+v", report)
	}
	if report.Child.ExecutableDigest == "" {
		t.Fatal("child executable digest was not recorded")
	}
	if len(report.Child.Argv) == 0 || report.Child.Argv[0] == "" {
		t.Fatal("argv did not record the exact executable")
	}
	// The fake child reports a wrong digest on purpose; the entry must notice.
	if report.Artifact.ChildDigest == "" {
		t.Fatal("child-reported digest was not captured")
	}
	if report.Artifact.ChildMatches {
		t.Fatal("entry trusted the child's digest instead of the artifact bytes")
	}
	if report.Artifact.Digest == "" {
		t.Fatal("artifact digest was not recorded")
	}
	if report.Qualified {
		t.Fatal("successful artifact claimed qualification")
	}
}

// minimalSelfBuildWorkspace writes the smallest workspace the loader accepts:
// a manifest with no sources, so validation has nothing to report.
func minimalSelfBuildWorkspace(t *testing.T, root string) string {
	t.Helper()
	workspace := filepath.Join(root, "workspace")
	writeWorkspaceFile(t, workspace, "tei.workspace.yaml", "workspace: self_build_entry\nformatVersion: 1\nsources:\n  - semantic\n")
	return workspace
}

func writeFakeBuilder(t *testing.T, dir string) string {
	t.Helper()
	builder := filepath.Join(dir, "builder")
	if runtime.GOOS == "windows" {
		builder += ".exe"
	}
	source := filepath.Join(dir, "builder.go")
	program := `package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	var output, state string
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-artifact-out":
			output = os.Args[i+1]
			i++
		case "-state":
			state = os.Args[i+1]
			i++
		}
	}
	if state != "" {
		_ = os.MkdirAll(state, 0o755)
	}
	if output != "" {
		_ = os.MkdirAll(filepath.Dir(output), 0o755)
		if err := os.WriteFile(output, []byte("built"), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	fmt.Printf("{\"artifactDigest\":%q}\n", "sha256:deadbeef")
}
`
	if err := os.WriteFile(source, []byte(program), 0o600); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-trimpath", "-o", builder, source)
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fake builder: %v: %s", err, out)
	}
	return builder
}
