package main

import (
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func TestFormatSourceDiagnosticIncludesPathLineColumn(t *testing.T) {
	diagnostic := workspace.SourceDiagnostic{
		Code:     "PARSE_ERROR",
		Severity: validation.Error,
		Message:  "expected database or uses",
		Source: workspace.DirectSource{
			ArtifactPath: "applications/order-entry/application.tei",
			Range: sourceindex.Range{
				Start: sourceindex.Position{Offset: 26, Line: 1, Column: 27},
				End:   sourceindex.Position{Offset: 33, Line: 1, Column: 34},
			},
		},
	}
	got := formatSourceDiagnostic(diagnostic)
	want := "applications/order-entry/application.tei:1:27\terror\tPARSE_ERROR\texpected database or uses"
	if got != want {
		t.Fatalf("formatted diagnostic=%q want=%q", got, want)
	}
}

func TestFormatProjectedDiagnosticIncludesResolvedDefinition(t *testing.T) {
	projected := workspace.ProjectedDiagnostic{
		Diagnostic: validation.Diagnostic{
			Code:     "INVALID_DATABASE_ACCESS",
			Severity: validation.Error,
			Message:  "invalid access",
		},
		Sources: []sourceindex.Reference{{
			ArtifactPath: "applications/order-entry/application.tei",
			Range: sourceindex.Range{
				Start: sourceindex.Position{Offset: 12, Line: 1, Column: 13},
				End:   sourceindex.Position{Offset: 23, Line: 1, Column: 24},
			},
			Role: sourceindex.RoleDefinition,
		}},
	}
	got := formatProjectedDiagnostic(projected)
	want := "applications/order-entry/application.tei:1:13\terror\tINVALID_DATABASE_ACCESS\tinvalid access"
	if got != want {
		t.Fatalf("formatted diagnostic=%q want=%q", got, want)
	}
}

func TestFormatProjectedDiagnosticKeepsSourceLessFinding(t *testing.T) {
	projected := workspace.ProjectedDiagnostic{Diagnostic: validation.Diagnostic{
		Code:     "SYNTHETIC",
		Severity: validation.Warning,
		Message:  "derived finding",
	}}
	got := formatProjectedDiagnostic(projected)
	want := "warning\tSYNTHETIC\tderived finding"
	if got != want {
		t.Fatalf("formatted diagnostic=%q want=%q", got, want)
	}
}
