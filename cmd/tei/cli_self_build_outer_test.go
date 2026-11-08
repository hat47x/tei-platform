package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests pin the §4 outer shape of `tei self-build` and its refusal
// conditions SC0-01〜SC0-05 (SB-C0, #2831). The primitive -child/-artifact form
// keeps its own tests; this file only covers the workspace form.

type selfBuildOuterResult struct {
	SchemaVersion string `json:"schemaVersion"`
	OK            bool   `json:"ok"`
	Qualified     bool   `json:"qualified"`
	Workspace     string `json:"workspace"`
	BuildState    string `json:"buildState"`
	Output        string `json:"output"`
	Boundary      string `json:"claimBoundary"`
	Error         *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	ChildRun *struct {
		CrossCheckOK bool   `json:"artifactCrossCheckMatched"`
		FailureCause string `json:"failureCause"`
	} `json:"childRun"`
}

func runOuterShape(t *testing.T, args ...string) (selfBuildOuterResult, int) {
	t.Helper()
	var out bytes.Buffer
	code, err := runSelfBuildCommand(args, &out)
	if err != nil {
		t.Fatalf("self-build returned an emitter error: %v", err)
	}
	var report selfBuildOuterResult
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("self-build did not emit one JSON object: %v\n%s", err, out.String())
	}
	if report.SchemaVersion != selfBuildEntrySchema {
		t.Fatalf("unexpected schemaVersion: %s", report.SchemaVersion)
	}
	if report.Boundary == "" {
		t.Fatal("report lost its boundary statement")
	}
	if report.Qualified {
		t.Fatal("report claimed qualification")
	}
	return report, code
}

func outerWorkspace(t *testing.T, root string) string {
	t.Helper()
	workspace := filepath.Join(root, "workspace")
	writeWorkspaceFile(t, workspace, "tei.workspace.yaml", "workspace: self_build_outer\nformatVersion: 1\nsources:\n  - semantic\n")
	return workspace
}

// SC0-01: without a workspace the entry refuses before anything starts.
func TestSelfBuildOuterRefusesWithoutWorkspace(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	report, code := runOuterShape(t,
		"-child", child,
		"-state", filepath.Join(dir, "state"),
		"-output", filepath.Join(dir, "artifact"))
	if code != 2 {
		t.Fatalf("missing -workspace was not a usage failure: code=%d", code)
	}
	if report.Error == nil || !strings.Contains(report.Error.Message, "-workspace is required") {
		t.Fatalf("missing -workspace had no stated reason: %+v", report.Error)
	}
	if report.ChildRun != nil {
		t.Fatal("a precondition failure started the child")
	}
}

// SC0-03 and SC0-04: a missing state and an existing output are refused before
// the child starts.
func TestSelfBuildOuterRefusesSharedStateAndExistingOutput(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")
	if err := os.WriteFile(output, []byte("already here"), 0o600); err != nil {
		t.Fatal(err)
	}

	report, code := runOuterShape(t,
		"-child", child,
		"-workspace", workspace,
		"-state", filepath.Join(dir, "state"),
		"-output", output)
	if code != 2 {
		t.Fatalf("existing output was not refused: code=%d", code)
	}
	if report.Error == nil || !strings.Contains(report.Error.Message, "does not overwrite") {
		t.Fatalf("existing output had no stated reason: %+v", report.Error)
	}
	if report.ChildRun != nil {
		t.Fatal("a refused entry started the child")
	}

	report, code = runOuterShape(t,
		"-child", child,
		"-workspace", workspace,
		"-output", filepath.Join(dir, "fresh"))
	if code != 2 {
		t.Fatalf("missing -state was not refused: code=%d", code)
	}
	if report.Error == nil || !strings.Contains(report.Error.Message, "shared state") {
		t.Fatalf("missing -state had no stated reason: %+v", report.Error)
	}
}

// SC0-05: a workspace that cannot be loaded stops the entry.
func TestSelfBuildOuterRefusesUnloadableWorkspace(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	report, code := runOuterShape(t,
		"-child", child,
		"-workspace", filepath.Join(dir, "missing-workspace"),
		"-state", filepath.Join(dir, "state"),
		"-output", filepath.Join(dir, "out"))
	if code != 2 {
		t.Fatalf("unloadable workspace was not refused: code=%d", code)
	}
	if report.Error == nil || !strings.Contains(report.Error.Message, "workspace load failed") {
		t.Fatalf("workspace load failure had no stated reason: %+v", report.Error)
	}
	if report.ChildRun != nil {
		t.Fatal("an unloadable workspace started the child")
	}
}

// SC0-09: a missing child is refused, because the entry never resolves PATH.
func TestSelfBuildOuterRefusesMissingChild(t *testing.T) {
	dir := t.TempDir()
	workspace := outerWorkspace(t, dir)
	report, code := runOuterShape(t,
		"-workspace", workspace,
		"-state", filepath.Join(dir, "state"),
		"-output", filepath.Join(dir, "out"))
	if code != 2 {
		t.Fatalf("missing -child was not refused: code=%d", code)
	}
	if report.Error == nil || !strings.Contains(report.Error.Message, "never resolves a builder from PATH") {
		t.Fatalf("missing -child had no stated reason: %+v", report.Error)
	}
}

// SC0-06: a successful child produces an artifact, and the entry still says
// qualified=false and compares the child's claim against the bytes on disk.
func TestSelfBuildOuterRecordsTheChildRunAndTheArtifact(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")
	report, code := runOuterShape(t,
		"-child", child,
		"-workspace", workspace,
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-intent", "intent.sample",
		"-child-arg", childArgs(t, "-artifact", output, "-digest", "sha256:deadbeef"))
	if code != 1 {
		// The fake child reports a wrong digest on purpose, so the cross-check
		// fails and the entry reports a child failure while still recording
		// both sides.
		t.Fatalf("mismatched child report was not reported as a child failure: code=%d %+v", code, report.Error)
	}
	if report.ChildRun == nil {
		t.Fatal("no child run was recorded")
	}
	if report.ChildRun.CrossCheckOK {
		t.Fatal("entry trusted the child's digest instead of the artifact bytes")
	}
	if report.ChildRun.FailureCause == "" {
		t.Fatal("mismatch had no stated cause")
	}
	if report.Qualified {
		t.Fatal("successful artifact claimed qualification")
	}
	if report.Workspace == "" || report.BuildState == "" || report.Output == "" {
		t.Fatalf("the §4 fields were not recorded: %+v", report)
	}
}

// A matching child report is recorded with ok=true and still qualified=false.
func TestSelfBuildOuterSucceedsOnAMatchingReport(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")
	// Compute the digest the fake child will create, so the report can match.
	if err := os.WriteFile(output, []byte("artifact-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := digestOfFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(output); err != nil {
		t.Fatal(err)
	}

	report, code := runOuterShape(t,
		"-child", child,
		"-workspace", workspace,
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output, "-digest", digest))
	if code != 0 {
		t.Fatalf("a matching child report was not accepted: code=%d %+v", code, report.Error)
	}
	if !report.OK {
		t.Fatal("successful run did not report ok")
	}
	if !report.ChildRun.CrossCheckOK || report.ChildRun.FailureCause != "" {
		t.Fatalf("matching cross-check reported a failure: %+v", report.ChildRun)
	}
	if report.Qualified {
		t.Fatal("matching artifact claimed qualification")
	}
}

// The primitive form must keep working: a caller that names no §4 flag still
// reaches the -child/-artifact grammar.
func TestSelfBuildKeepsThePrimitiveForm(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "artifact")
	child := buildSelfBuildChild(t, dir)
	var out bytes.Buffer
	code, err := runSelfBuildCommand([]string{
		"-child", child,
		"-child-arg", childArgs(t, "-artifact", artifact, "-digest", "sha256:deadbeef"),
		"-artifact", artifact,
	}, &out)
	if code != 0 || err != nil {
		t.Fatalf("primitive form stopped working: code=%d err=%v", code, err)
	}
	var evidence selfBuildEntryEvidence
	if err := json.Unmarshal(out.Bytes(), &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.Invoked.ExecutableDigest == "" {
		t.Fatal("primitive form lost the invoked identity")
	}
	if report := evidence; report.CrossCheckOK {
		t.Fatal("primitive form trusted the child's digest")
	}
}
