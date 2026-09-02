package workspace_test

import (
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func TestReferenceBusinessValidationsKeepRulesInSemanticAndApplicationUsesIdentity(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	result, err := workspace.Load(ws)
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}

	if diagnostics := validation.Validate(result.World); len(diagnostics) != 0 {
		t.Fatalf("Reference WorkspaceにDiagnosticがあります: %#v", diagnostics)
	}

	quantity := semanticValidationByID(t, result.World.Semantics, "sales.order_line.quantity_positive")
	if quantity.Kind != semantic.ValidationAttributePositive || quantity.Attribute != "sales.order_line.quantity" || quantity.Relation != "" {
		t.Fatalf("数量ValidationのCanonical契約が期待値と異なります: %#v", quantity)
	}

	unique := semanticValidationByID(t, result.World.Semantics, "sales.order.lines_line_number_unique")
	if unique.Kind != semantic.ValidationRelationAttributeUnique || unique.Relation != "sales.order.lines" || unique.Attribute != "sales.order_line.line_number" {
		t.Fatalf("明細番号ValidationのCanonical契約が期待値と異なります: %#v", unique)
	}

	app := result.World.Applications[0]
	want := []core.CanonicalID{
		"sales.order.lines_line_number_unique",
		"sales.order_line.quantity_positive",
	}
	if len(app.Validations) != len(want) {
		t.Fatalf("ApplicationのValidation参照=%#v", app.Validations)
	}
	for i := range want {
		if app.Validations[i] != want[i] {
			t.Fatalf("ApplicationのValidation参照[%d]=%s want=%s", i, app.Validations[i], want[i])
		}
	}

	idx, err := workspace.BuildSourceIndex(ws, result)
	if err != nil {
		t.Fatalf("Source Indexの構築に失敗しました: %v", err)
	}
	for _, id := range want {
		refs := idx.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: string(id)})
		if len(refs) != 1 {
			t.Fatalf("Semantic Validation %sのsource references=%#v", id, refs)
		}
		if refs[0].ArtifactPath != "semantic/sales/validation.tei" {
			t.Fatalf("Semantic Validation %sのsource path=%q", id, refs[0].ArtifactPath)
		}
	}
}

func semanticValidationByID(t *testing.T, namespaces []semantic.Namespace, id core.CanonicalID) semantic.Validation {
	t.Helper()
	for _, namespace := range namespaces {
		for _, rule := range namespace.Validations {
			if rule.ID == id {
				return rule
			}
		}
	}
	t.Fatalf("Semantic Validation %sが見つかりません", id)
	return semantic.Validation{}
}
