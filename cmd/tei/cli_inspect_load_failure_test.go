package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// loadFailureService returns the typed Load failure the operations layer
// produces, with text that names a real read problem.
type loadFailureService struct {
	root string
}

func (s loadFailureService) Inspect(sourceindex.Subject) (operations.SubjectInspection, error) {
	_, err := workspace.Load(s.root)
	if err != nil {
		return operations.SubjectInspection{}, &operations.ValidationFailure{
			Stage: operations.StageLoad, Err: err,
		}
	}
	return operations.SubjectInspection{}, nil
}

// TestInspectLoadFailureKeepsTheReadText pins #3420: a Load or Source Index
// failure happens while reading this process's own Workspace, so its text is
// safe to report and must match the MCP adapter's answer. Before this the CLI
// replaced it with the generic "workspace inspection failed".
func TestInspectLoadFailureKeepsTheReadText(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent-workspace")
	var out bytes.Buffer
	code, err := inspectWorkspaceJSONWithService(&out, missing, "", "semantic-data", "a.order", 0,
		loadFailureService{root: missing})
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if code != 2 {
		t.Fatalf("exit=%d, want 2", code)
	}
	failure := decodeInspectionFailure(t, out.Bytes())
	if failure.Code != "WORKSPACE_LOAD_FAILED" {
		t.Fatalf("code=%q", failure.Code)
	}
	if failure.Message == "workspace inspection failed" {
		t.Fatalf("the read failure was replaced by the generic placeholder")
	}
	if !strings.Contains(failure.Message, "absent-workspace") {
		t.Fatalf("the read failure text does not name what failed: %q", failure.Message)
	}
}

// A provider error from an unverified service is still masked: only the Load
// and Source Index stages expose their own text.
func TestInspectStillMasksUnverifiedProviderText(t *testing.T) {
	canonical := &operations.InspectionError{
		Code:    operations.InspectionUnsupportedKind,
		Message: `inspect does not support subject kind "x" at C:\private\token`,
	}
	var out bytes.Buffer
	if _, err := inspectWorkspaceJSONWithService(&out, "w", "", "x", "y", 0, maskingInspectionService{err: canonical}); err != nil {
		t.Fatalf("inspect: %v", err)
	}
	failure := decodeInspectionFailure(t, out.Bytes())
	if strings.Contains(failure.Message, "private") {
		t.Fatalf("unverified provider text leaked: %q", failure.Message)
	}
}

// The built binary must report the same Load failure text as the MCP adapter.
func TestInspectLoadFailureMatchesTheMCPAdapter(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI")
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	missing := filepath.Join(dir, "no-such-workspace")
	var out bytes.Buffer
	command := exec.Command(binary, "inspect", "-workspace", missing, "-subject-kind", "semantic-data",
		"-subject-id", "a.order")
	command.Stdout = &out
	err := command.Run()
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 2 {
		t.Fatalf("inspect: %v (%q)", err, out.String())
	}
	failure := decodeInspectionFailure(t, out.Bytes())
	if failure.Code != "WORKSPACE_LOAD_FAILED" || failure.Message == "workspace inspection failed" {
		t.Fatalf("load failure was masked: %+v", failure)
	}
	var body struct {
		Error *inspectionFailure `json:"error"`
	}
	if err := json.Unmarshal(out.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error == nil || !strings.Contains(body.Error.Message, "no-such-workspace") {
		t.Fatalf("load failure text is not the real one: %+v", body.Error)
	}
}
