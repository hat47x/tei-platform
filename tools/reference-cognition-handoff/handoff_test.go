package cognitionhandoff

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func proposalPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "tools", "reference-cognition-port", "testdata", "proposals", "validation.json")
}

func TestHandoffCreatesSharedAcceptanceCandidate(t *testing.T) {
	candidate, err := Handoff(proposalPath(t), filepath.Join(repoRoot(t), "reference", "workspace"), "local-rules-a", "deterministic-rules", "1", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Schema != CandidateSchema || candidate.ID != "sales.candidate_obs_1" {
		t.Fatalf("unexpected candidate: %+v", candidate)
	}
	if candidate.Origin.Kind != "ai" || candidate.Origin.Actor != "local-rules-a/deterministic-rules@1" {
		t.Fatalf("unexpected origin: %+v", candidate.Origin)
	}
	if candidate.BaseModelRevision == "" || len(candidate.Delta) != 1 {
		t.Fatalf("incomplete candidate: %+v", candidate)
	}
	operation := candidate.Delta[0]
	if operation.Op != "add-validation" || operation.Namespace != "sales" ||
		operation.Kind != "positive" || operation.Attribute != "sales.order_line.quantity" {
		t.Fatalf("unexpected operation: %+v", operation)
	}
}

func TestHandoffRejectsUnsupportedCandidateKind(t *testing.T) {
	raw := `{
  "schema": "tei.reference.cognition-proposal/v0",
  "state": "proposed",
  "candidates": [{"id":"x","kind":"diagnosis","subject":"sales.order_line.quantity","payload":{},"basis":[]}]
}`
	path := filepath.Join(t.TempDir(), "proposal.json")
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Handoff(path, filepath.Join(repoRoot(t), "reference", "workspace"), "p", "m", "1", "", ""); err == nil {
		t.Fatal("unsupported candidate kind was accepted")
	}
}

func TestHandoffRejectsUnsupportedExpression(t *testing.T) {
	raw := `{
  "schema": "tei.reference.cognition-proposal/v0",
  "state": "proposed",
  "candidates": [{"id":"x","kind":"validation-candidate","subject":"sales.order_line.quantity","payload":{"expression":"quantity < 0"},"basis":[]}]
}`
	path := filepath.Join(t.TempDir(), "proposal.json")
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Handoff(path, filepath.Join(repoRoot(t), "reference", "workspace"), "p", "m", "1", "", ""); err == nil {
		t.Fatal("unsupported expression was accepted")
	}
}
