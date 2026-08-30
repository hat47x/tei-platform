package workspace_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func TestSemanticValidationCarriesPathIndependentSubject(t *testing.T) {
	world := ir.World{
		Applications: []application.Application{{
			ID: "order_entry",
			Databases: []application.DatabaseRequirement{{
				Alias:  core.LocalName("sales"),
				Access: database.Access("invalid"),
			}},
		}},
	}

	diagnostic, ok := diagnosticByCode(validation.Validate(world), "INVALID_DATABASE_ACCESS")
	if !ok {
		t.Fatal("missing INVALID_DATABASE_ACCESS diagnostic")
	}
	if diagnostic.Subject == nil {
		t.Fatal("diagnostic has no semantic subject")
	}
	want := subject.Subject{Kind: "application", ID: "order_entry"}
	if *diagnostic.Subject != want {
		t.Fatalf("subject=%+v want=%+v", *diagnostic.Subject, want)
	}
}

func TestProjectDiagnosticsResolvesDefinitionWithoutMessageParsing(t *testing.T) {
	s := subject.Subject{Kind: "application", ID: "order_entry"}
	diagnostic := validation.Diagnostic{
		Code:     "INVALID_DATABASE_ACCESS",
		Severity: validation.Error,
		Message:  "this text is intentionally unrelated to source lookup",
		Subject:  &s,
	}
	definition := sourceindex.Reference{
		Subject:      s,
		ArtifactPath: "applications/order-entry/application.tei",
		Range: sourceindex.Range{
			Start: sourceindex.Position{Offset: 12, Line: 1, Column: 13},
			End:   sourceindex.Position{Offset: 23, Line: 1, Column: 24},
		},
		Role: sourceindex.RoleDefinition,
	}
	implementation := sourceindex.Reference{
		Subject:      s,
		ArtifactPath: "applications/order-entry/custom.go",
		Role:         sourceindex.RoleImplementation,
	}
	index := sourceindex.Index{References: []sourceindex.Reference{implementation, definition}}
	index.Normalize()

	got := workspace.ProjectDiagnostics([]validation.Diagnostic{diagnostic}, index)
	if len(got) != 1 || len(got[0].Sources) != 1 {
		t.Fatalf("projection=%#v", got)
	}
	if got[0].Sources[0] != definition {
		t.Fatalf("source=%+v want definition=%+v", got[0].Sources[0], definition)
	}
	if got[0].Diagnostic.Message != diagnostic.Message {
		t.Fatal("projection changed diagnostic content")
	}
}

func TestProjectDiagnosticsFollowsSourceMoveBySubject(t *testing.T) {
	s := subject.Subject{Kind: "semantic-data", ID: "sales.order"}
	diagnostic := validation.Diagnostic{Code: "TEST", Severity: validation.Error, Message: "same finding", Subject: &s}

	before := sourceindex.Index{References: []sourceindex.Reference{{
		Subject:      s,
		ArtifactPath: "semantic/sales/order.tei",
		Role:         sourceindex.RoleDefinition,
	}}}
	after := sourceindex.Index{References: []sourceindex.Reference{{
		Subject:      s,
		ArtifactPath: "semantic/domain/order.tei",
		Role:         sourceindex.RoleDefinition,
	}}}

	beforeProjected := workspace.ProjectDiagnostics([]validation.Diagnostic{diagnostic}, before)
	afterProjected := workspace.ProjectDiagnostics([]validation.Diagnostic{diagnostic}, after)
	if beforeProjected[0].Diagnostic.Subject == nil || afterProjected[0].Diagnostic.Subject == nil {
		t.Fatal("semantic subject was lost")
	}
	if *beforeProjected[0].Diagnostic.Subject != *afterProjected[0].Diagnostic.Subject {
		t.Fatal("semantic subject changed after source move")
	}
	if beforeProjected[0].Sources[0].ArtifactPath != "semantic/sales/order.tei" {
		t.Fatalf("before path=%q", beforeProjected[0].Sources[0].ArtifactPath)
	}
	if afterProjected[0].Sources[0].ArtifactPath != "semantic/domain/order.tei" {
		t.Fatalf("after path=%q", afterProjected[0].Sources[0].ArtifactPath)
	}
}

func TestProjectDiagnosticsKeepsFindingWhenSourceIsUnavailable(t *testing.T) {
	s := subject.Subject{Kind: "synthetic", ID: "derived.target"}
	diagnostic := validation.Diagnostic{Code: "SYNTHETIC", Severity: validation.Warning, Message: "derived finding", Subject: &s}

	got := workspace.ProjectDiagnostics([]validation.Diagnostic{diagnostic}, sourceindex.Index{})
	if len(got) != 1 {
		t.Fatalf("projection=%#v", got)
	}
	if len(got[0].Sources) != 0 {
		t.Fatalf("unexpected sources=%#v", got[0].Sources)
	}
	if got[0].Diagnostic.Code != diagnostic.Code {
		t.Fatal("source-less finding was dropped")
	}
}

func diagnosticByCode(diagnostics []validation.Diagnostic, code string) (validation.Diagnostic, bool) {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return diagnostic, true
		}
	}
	return validation.Diagnostic{}, false
}
