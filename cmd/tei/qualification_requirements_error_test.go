package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// errorCodeOfEnvelope reads error.code from one CLI envelope document.
func errorCodeOfEnvelope(t *testing.T, raw []byte) string {
	t.Helper()
	var body struct {
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("not one JSON document: %v (%q)", err, string(raw))
	}
	if body.Error == nil {
		return ""
	}
	return body.Error.Code
}

// TestUnreadableRequirementsFileIsAnArgumentError pins #3943: a path the caller
// named but cannot be read is an argument problem, not a Workspace whose
// knowledge failed to build. Reporting KNOWLEDGE_BUILD_FAILED made a mistyped
// path look like a broken model, and a reader could not tell the two apart.
func TestUnreadableRequirementsFileIsAnArgumentError(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	missing := filepath.Join(t.TempDir(), "no-such-requirements.json")

	var out bytes.Buffer
	code, err := knowledgeQualifyJSON(&out, reference, "semantic-data",
		"security.authenticated_subject", missing, 0)
	if err != nil {
		t.Fatalf("knowledge-qualify: %v", err)
	}
	if code != 2 {
		t.Fatalf("exit=%d, want 2", code)
	}
	if got := errorCodeOfEnvelope(t, out.Bytes()); got != resultCodeInvalidArgument {
		t.Fatalf("error.code=%q, want %q (body=%s)", got, resultCodeInvalidArgument, out.String())
	}
	if !strings.Contains(out.String(), "-qualification-requirements") {
		t.Fatalf("the message does not name the flag: %s", out.String())
	}

	// The same for the other command that reads this file.
	var handoff bytes.Buffer
	if _, err := knowledgeReviewHandoffJSON(&handoff, reference, "semantic-data",
		"security.authenticated_subject", missing, "", 0); err != nil {
		t.Fatalf("knowledge-review-handoff: %v", err)
	}
	if got := errorCodeOfEnvelope(t, handoff.Bytes()); got != resultCodeInvalidArgument {
		t.Fatalf("knowledge-review-handoff error.code=%q, want %q", got, resultCodeInvalidArgument)
	}
}

// A file that opens but does not hold the expected document is also an argument
// problem: the caller supplied the wrong file.
func TestWrongRequirementsDocumentIsAnArgumentError(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	path := filepath.Join(t.TempDir(), "requirements.json")
	if err := os.WriteFile(path, []byte(`{"qualifications":[{"id":"owner-approved"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if _, err := knowledgeQualifyJSON(&out, reference, "semantic-data",
		"security.authenticated_subject", path, 0); err != nil {
		t.Fatalf("knowledge-qualify: %v", err)
	}
	if got := errorCodeOfEnvelope(t, out.Bytes()); got != resultCodeInvalidArgument {
		t.Fatalf("error.code=%q, want %q (body=%s)", got, resultCodeInvalidArgument, out.String())
	}
	if !strings.Contains(out.String(), "expected JSON document") {
		t.Fatalf("the message does not say the document was wrong: %s", out.String())
	}
}

// The distinction must hold in both directions: a readable requirements file
// with a Workspace that cannot be loaded is still a build failure, not an
// argument error.
func TestGenuineBuildFailureKeepsItsOwnCode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "requirements.json")
	if err := os.WriteFile(path, []byte(`{"expectedKind":"semantic-data"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	missingWorkspace := filepath.Join(t.TempDir(), "no-such-workspace")
	var out bytes.Buffer
	if _, err := knowledgeQualifyJSON(&out, missingWorkspace, "semantic-data",
		"security.authenticated_subject", path, 0); err != nil {
		t.Fatalf("knowledge-qualify: %v", err)
	}
	if got := errorCodeOfEnvelope(t, out.Bytes()); got != resultCodeKnowledgeFailed {
		t.Fatalf("error.code=%q, want %q (body=%s)", got, resultCodeKnowledgeFailed, out.String())
	}
}

// A valid file against a real Workspace still succeeds, so the guards did not
// turn the flag into a failure path.
func TestReadableRequirementsFileStillSucceeds(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	path := filepath.Join(t.TempDir(), "requirements.json")
	if err := os.WriteFile(path, []byte(`{"expectedKind":"semantic-data"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	code, err := knowledgeQualifyJSON(&out, reference, "semantic-data",
		"security.authenticated_subject", path, 0)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v (%s)", code, err, out.String())
	}
	var body struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(out.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.OK {
		t.Fatalf("ok=false: %s", out.String())
	}
}
