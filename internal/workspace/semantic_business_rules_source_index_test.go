package workspace_test

import (
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func TestSemanticBusinessRulesLoadIntoWorldAndExactSourceIndex(t *testing.T) {
	root := t.TempDir()
	writeSemanticValidationWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: business_rules_test\nsources:\n  - semantic\n")
	path := "semantic/sales/rules.tei"
	src := `namespace sales
data OrderLine {
    attribute orderId type std.identifier
    attribute lineNumber type std.integer
    attribute itemCode type std.code
    candidate-key businessLine {
        field sales.order_line.order_id
        field sales.order_line.line_number
    }
    functional-dependency itemByLine {
        determinant sales.order_line.order_id
        determinant sales.order_line.line_number
        dependent sales.order_line.item_code
    }
}
`
	writeSemanticValidationWorkspaceFile(t, root, path, src)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	var dataFound bool
	for _, ns := range loaded.World.Semantics {
		if ns.ID == "sales" {
			if len(ns.Data) != 1 {
				t.Fatalf("sales Data declarations=%+v", ns.Data)
			}
			dataFound = true
			data := ns.Data[0]
			if len(data.CandidateKeys) != 1 || len(data.FunctionalDependencies) != 1 ||
				data.CandidateKeys[0].ID != "sales.order_line.business_line" ||
				data.FunctionalDependencies[0].ID != "sales.order_line.item_by_line" {
				t.Fatalf("canonical key/FD declarations missing: %+v", data)
			}
		}
	}
	if !dataFound {
		t.Fatalf("sales semantic namespace was not loaded: %+v", loaded.World.Semantics)
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if strings.Contains(diagnostic.Code, "SEMANTIC_RULE") {
			t.Fatalf("unexpected business rule structural error: %+v", diagnostic)
		}
	}
	idx, read, err := workspace.IndexedLoadedSnapshot(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ kind, id, symbol string }{
		{"semantic-candidate-key", "sales.order_line.business_line", "businessLine"},
		{"semantic-functional-dependency", "sales.order_line.item_by_line", "itemByLine"},
	} {
		refs := idx.ReferencesFor(sourceindex.Subject{Kind: tc.kind, ID: tc.id})
		if len(refs) != 1 || refs[0].Role != sourceindex.RoleDefinition || refs[0].ArtifactPath != path {
			t.Fatalf("missing accepted-source semantic rule anchor for %s: %+v", tc.id, refs)
		}
		actual, err := read(refs[0].ArtifactPath)
		if err != nil {
			t.Fatal(err)
		}
		rng := refs[0].Range
		if rng.Start.Offset >= rng.End.Offset || string(actual[rng.Start.Offset:rng.End.Offset]) != tc.symbol {
			t.Fatalf("rule %s anchor is not its own declared identifier: %+v", tc.id, rng)
		}
	}
}

func TestSemanticBusinessRuleDoesNotAppearWithoutExplicitDeclaration(t *testing.T) {
	root := t.TempDir()
	writeSemanticValidationWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: business_rules_test\nsources:\n  - semantic\n")
	writeSemanticValidationWorkspaceFile(t, root, "semantic/sales/order.tei", `namespace sales
data OrderLine {
    attribute orderId type std.identifier
}
`)
	result, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	data := result.World.Semantics[0].Data[0]
	if len(data.CandidateKeys) != 0 || len(data.FunctionalDependencies) != 0 {
		t.Fatalf("physical or schema-only identity was inferred: %+v", data)
	}
}
