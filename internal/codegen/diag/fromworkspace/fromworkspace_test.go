package fromworkspace_test

import (
	"reflect"
	"testing"

	"tei/internal/codegen/diag"
	"tei/internal/codegen/diag/fromworkspace"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func span(start, startLine, startColumn, end, endLine, endColumn int) sourceindex.Range {
	return sourceindex.Range{
		Start: sourceindex.Position{Offset: start, Line: startLine, Column: startColumn},
		End:   sourceindex.Position{Offset: end, Line: endLine, Column: endColumn},
	}
}

func TestSourceDiagnosticKeepsPathOffsetsAndStart(t *testing.T) {
	got := fromworkspace.SourceDiagnostic(workspace.SourceDiagnostic{
		Code: "PARSE_ERROR", Severity: validation.Error, Message: "unexpected token",
		Source: workspace.DirectSource{ArtifactPath: "semantic/sales.tei", Range: span(42, 3, 7, 45, 3, 10)},
	})
	want := diag.Record{
		Code: "PARSE_ERROR", Severity: "error", File: "semantic/sales.tei",
		Range: &diag.Range{Start: 42, End: 45}, Line: 3, Column: 7,
		Message: "unexpected token",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if err := got.Validate(); err != nil {
		t.Fatal(err)
	}
}

// The projection is made by the Workspace's own ProjectDiagnostics, so the
// converter is tested on what the loader really gives.
func TestProjectedDiagnosticUsesTheFirstDefinitionAndRelatesTheRest(t *testing.T) {
	quantity := sourceindex.Subject{Kind: "semantic", ID: "sales.order_line.quantity"}
	index := sourceindex.Index{References: []sourceindex.Reference{
		{Subject: quantity, ArtifactPath: "semantic/a.tei", Range: span(10, 2, 1, 18, 2, 9), Role: sourceindex.RoleDefinition},
		{Subject: quantity, ArtifactPath: "semantic/b.tei", Range: span(0, 1, 1, 8, 1, 9), Role: sourceindex.RoleDefinition},
		{Subject: quantity, ArtifactPath: "app/form.tei", Range: span(5, 1, 6, 9, 1, 10), Role: sourceindex.RoleAuthoring},
	}}
	projected := workspace.ProjectDiagnostics([]validation.Diagnostic{{
		Code: "SEMANTIC_ATTRIBUTE_ROLE_UNSUPPORTED", Severity: validation.Warning,
		Message: "role is not used", Subject: &quantity,
	}}, index)
	if len(projected) != 1 {
		t.Fatalf("projection gave %d diagnostics", len(projected))
	}
	got := fromworkspace.ProjectedDiagnostic(projected[0])
	want := diag.Record{
		Code: "SEMANTIC_ATTRIBUTE_ROLE_UNSUPPORTED", Severity: "warning",
		File: "semantic/a.tei", Range: &diag.Range{Start: 10, End: 18}, Line: 2, Column: 1,
		Element: "sales.order_line.quantity", Message: "role is not used",
		Related: []diag.Related{{
			File: "semantic/b.tei", Range: &diag.Range{Start: 0, End: 8},
			Element: "sales.order_line.quantity", Role: "definition",
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if err := got.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestProjectedDiagnosticWithoutSubjectOrSourceHasNoPlace(t *testing.T) {
	got := fromworkspace.ProjectedDiagnostic(workspace.ProjectedDiagnostic{
		Diagnostic:              validation.Diagnostic{Code: "DUPLICATE_SEMANTIC_ELEMENT", Severity: validation.Error, Message: "duplicate"},
		UnresolvedSourceOrigins: []string{"standard"},
	})
	want := diag.Record{Code: "DUPLICATE_SEMANTIC_ELEMENT", Severity: "error", Message: "duplicate"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
