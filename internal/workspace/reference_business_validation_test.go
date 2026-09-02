package workspace_test

import (
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestReferenceBusinessValidationsAreSemanticAndTraceable(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	loaded, err := workspace.Load(ws)
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}

	quantity := semanticValidationByID(t, loaded.World.Semantics, "sales.order_line.quantity_positive")
	if quantity.Kind != semantic.ValidationPositive || quantity.Attribute != "sales.order_line.quantity" || quantity.Relation != "" {
		t.Fatalf("quantity_positiveのCanonicalな意味が期待値と異なります: %#v", quantity)
	}

	lineNumber := semanticValidationByID(t, loaded.World.Semantics, "sales.order.lines.line_number_unique")
	if lineNumber.Kind != semantic.ValidationRelationUnique || lineNumber.Relation != "sales.order.lines" || lineNumber.Attribute != "sales.order_line.line_number" {
		t.Fatalf("line_number_uniqueのCanonicalな意味が期待値と異なります: %#v", lineNumber)
	}

	idx, err := workspace.BuildSourceIndex(ws, loaded)
	if err != nil {
		t.Fatalf("Source Indexの構築に失敗しました: %v", err)
	}
	for _, id := range []string{
		"sales.order_line.quantity_positive",
		"sales.order.lines.line_number_unique",
	} {
		refs := idx.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: id})
		if len(refs) != 1 {
			t.Fatalf("%sのsource references=%#v", id, refs)
		}
		if refs[0].ArtifactPath != "semantic/sales/validation.tei" {
			t.Fatalf("%sのsource path=%q want=%q", id, refs[0].ArtifactPath, "semantic/sales/validation.tei")
		}
	}
}

func semanticValidationByID(t *testing.T, namespaces []semantic.Namespace, id core.CanonicalID) semantic.Validation {
	t.Helper()
	for _, ns := range namespaces {
		for _, validation := range ns.Validations {
			if validation.ID == id {
				return validation
			}
		}
	}
	t.Fatalf("Semantic Validation %sが見つかりません", id)
	return semantic.Validation{}
}
