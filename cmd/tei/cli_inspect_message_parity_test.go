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
)

// maskingInspectionService stands in for a Host-supplied implementation whose
// error text cannot be trusted. It must keep the masked message.
type maskingInspectionService struct {
	err error
}

func (s maskingInspectionService) Inspect(sourceindex.Subject) (operations.SubjectInspection, error) {
	return operations.SubjectInspection{}, s.err
}

// canonicalInspectionService declares that it forwards the canonical
// Operation's own errors, the way the local reference Host does.
type canonicalInspectionService struct {
	err error
}

func (s canonicalInspectionService) Inspect(sourceindex.Subject) (operations.SubjectInspection, error) {
	return operations.SubjectInspection{}, s.err
}

func (s canonicalInspectionService) CanonicalInspectionMessages() bool { return true }

// TestInspectShowsTheCanonicalMessageForAVerifiedService pins #3420: the normal
// CLI path binds a Host-selected Inspector, not the in-process builtin. Before
// this the CLI replaced every failure message with "inspection service reported
// <CODE>" while the MCP adapter showed the canonical text, so the same call
// produced two different documents.
func TestInspectShowsTheCanonicalMessageForAVerifiedService(t *testing.T) {
	canonical := &operations.InspectionError{
		Code:    operations.InspectionUnsupportedKind,
		Message: `inspect does not support subject kind "bogus"`,
	}
	var out bytes.Buffer
	code, err := inspectWorkspaceJSONWithService(&out, "w", "", "bogus", "x", 0, canonicalInspectionService{err: canonical})
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if code != 2 {
		t.Fatalf("exit=%d, want 2", code)
	}
	failure := decodeInspectionFailure(t, out.Bytes())
	if failure.Code != operations.InspectionUnsupportedKind {
		t.Fatalf("code=%q", failure.Code)
	}
	if failure.Message != canonical.Message {
		t.Fatalf("message was rewritten: %q", failure.Message)
	}
}

// An implementation the Host did not verify keeps its text hidden: a third-party
// service could put host-private paths or credentials into an error message.
func TestInspectMasksTheMessageForAnUnverifiedService(t *testing.T) {
	canonical := &operations.InspectionError{
		Code:    operations.InspectionUnsupportedKind,
		Message: `inspect does not support subject kind "bogus" at C:\secret\path`,
	}
	var out bytes.Buffer
	if _, err := inspectWorkspaceJSONWithService(&out, "w", "", "bogus", "x", 0, maskingInspectionService{err: canonical}); err != nil {
		t.Fatalf("inspect: %v", err)
	}
	failure := decodeInspectionFailure(t, out.Bytes())
	if failure.Code != operations.InspectionUnsupportedKind {
		t.Fatalf("code=%q", failure.Code)
	}
	if strings.Contains(failure.Message, "secret") {
		t.Fatalf("unverified service text leaked: %q", failure.Message)
	}
	if !strings.Contains(failure.Message, "inspection service reported") {
		t.Fatalf("masked message lost its shape: %q", failure.Message)
	}
}

// The builtin in-process service is canonical by construction.
func TestBuiltinInspectionServiceIsCanonical(t *testing.T) {
	if !canonicalInspectionMessages(builtinCLIInspectionService{root: "w"}) {
		t.Fatal("the builtin service was treated as unverified")
	}
	if canonicalInspectionMessages(nil) {
		t.Fatal("a missing service was treated as canonical")
	}
	if canonicalInspectionMessages(maskingInspectionService{}) {
		t.Fatal("an undeclared service was treated as canonical")
	}
}

// The built binary must agree with the MCP adapter for the same input, which is
// the property #3420 asks for.
func TestInspectFailureMatchesTheMCPAdapter(t *testing.T) {
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
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	var out bytes.Buffer
	command := exec.Command(binary, "inspect", "-workspace", reference, "-subject-kind", "bogus", "-subject-id", "x")
	command.Stdout = &out
	err := command.Run()
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 2 {
		t.Fatalf("inspect: %v (%q)", err, out.String())
	}
	failure := decodeInspectionFailure(t, out.Bytes())
	if failure.Code != operations.InspectionUnsupportedKind {
		t.Fatalf("code=%q", failure.Code)
	}
	// The canonical operation's own text, not the transport's placeholder.
	if !strings.Contains(failure.Message, "does not support subject kind") {
		t.Fatalf("message is the masked placeholder: %q", failure.Message)
	}
	if strings.Contains(failure.Message, "inspection service reported") {
		t.Fatalf("message was not the canonical one: %q", failure.Message)
	}
}

type inspectionFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func decodeInspectionFailure(t *testing.T, raw []byte) inspectionFailure {
	t.Helper()
	var body struct {
		Error *inspectionFailure `json:"error"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("not one JSON document: %v (%q)", err, string(raw))
	}
	if body.Error == nil {
		t.Fatalf("no error document: %s", raw)
	}
	return *body.Error
}
