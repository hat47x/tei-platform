package operations

import (
	"encoding/json"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// directResult builds the operation result a first-syntax-error run produces.
func directResult(path string) WorkspaceValidation {
	return WorkspaceValidation{Direct: &workspace.SourceDiagnostic{
		Code:     "PARSE_ERROR",
		Severity: validation.Error,
		Message:  "expected \"namespace\"",
		Source: workspace.DirectSource{
			ArtifactPath: path,
			Range:        sourceindex.Range{},
		},
	}}
}

// TestStoppedRunSaysTheRunIsIncomplete pins #4178. `validate` stops at the first
// syntax error, so a single diagnostic was indistinguishable from "there is one
// problem". The report now names the truncation and the file it stopped at.
func TestStoppedRunSaysTheRunIsIncomplete(t *testing.T) {
	const path = "semantic/a.tei"
	report := WorkspaceValidationReportFrom(directResult(path))

	if report.Incomplete == nil {
		t.Fatal("a run that stopped reports no incompleteness")
	}
	if report.Incomplete.Reason != "load-stopped-at-first-syntax-error" {
		t.Fatalf("reason = %q", report.Incomplete.Reason)
	}
	if report.Incomplete.StoppedAt != path {
		t.Fatalf("stoppedAt = %q, want %q", report.Incomplete.StoppedAt, path)
	}
	if report.Incomplete.Detail == "" {
		t.Fatal("no detail: a caller reading only the reason cannot tell what was skipped")
	}
	// The existing fields must be untouched: one diagnostic, as before.
	if len(report.Diagnostics) != 1 || report.Diagnostics[0].Diagnostic.Code != "PARSE_ERROR" {
		t.Fatalf("diagnostics changed shape: %+v", report.Diagnostics)
	}
	if report.WorldFingerprint != "" || report.SourceIndexFingerprint != "" {
		t.Fatal("a stopped run claims fingerprints it never computed")
	}
}

// A run that reaches the end must not claim incompleteness, or the new field
// stops meaning anything.
func TestCompleteRunDoesNotClaimIncompleteness(t *testing.T) {
	report := WorkspaceValidationReportFrom(WorkspaceValidation{
		Diagnostics: []workspace.ProjectedDiagnostic{{
			Diagnostic: validation.Diagnostic{Code: "SOME_WARNING", Severity: "warning", Message: "note"},
		}},
		WorldFingerprint:       "sha256:aa",
		SourceIndexFingerprint: "sha256:bb",
	})
	if report.Incomplete != nil {
		t.Fatalf("a complete run reported incompleteness: %+v", report.Incomplete)
	}
	if len(report.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %d, want 1", len(report.Diagnostics))
	}
}

// The stopped-at path must come from the diagnostic, so a future change to how
// the direct finding is built cannot leave a stale file name.
func TestStoppedAtFollowsTheDiagnostic(t *testing.T) {
	const path = "applications/x/application.tei"
	report := WorkspaceValidationReportFrom(directResult(path))
	if report.Incomplete == nil || report.Incomplete.StoppedAt != path {
		t.Fatalf("stoppedAt = %+v, want %q", report.Incomplete, path)
	}
}

// The wire name must stay incomplete, or a caller's parser stops seeing it.
func TestIncompleteFieldNameIsWireStable(t *testing.T) {
	report := WorkspaceValidationReportFrom(directResult("semantic/a.tei"))
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	value, ok := body["incomplete"]
	if !ok {
		t.Fatalf("no incomplete field in %s", raw)
	}
	var decoded WorkspaceValidationIncomplete
	if err := json.Unmarshal(value, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Reason != "load-stopped-at-first-syntax-error" {
		t.Fatalf("reason over the wire = %q", decoded.Reason)
	}
}
