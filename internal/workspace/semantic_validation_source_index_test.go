package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestSemanticValidationLoadsAcrossNamespaceFilesAndHasExactSourceRange(t *testing.T) {
	root := t.TempDir()
	writeSemanticValidationWorkspaceFile(t, root, "tei.workspace.yaml", `workspace: semantic_validation_test
sources:
  - semantic
`)
	writeSemanticValidationWorkspaceFile(t, root, "semantic/sales/order.tei", `namespace sales

data OrderLine {
    attribute quantity type std.decimal
}
`)
	validationPath := "semantic/sales/validation.tei"
	validationSource := `namespace sales

validation sales.order_line.quantity_positive positive {
    attribute sales.order_line.quantity
}
`
	writeSemanticValidationWorkspaceFile(t, root, validationPath, validationSource)
	writeSemanticValidationWorkspaceFile(t, root, "semantic/std/types.tei", `namespace std
`)

	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("Workspaceの読み込みに失敗しました: %v", err)
	}

	var found bool
	for _, ns := range loaded.World.Semantics {
		if ns.ID != "sales" {
			continue
		}
		if len(ns.Data) != 1 || ns.Data[0].ID != "sales.order_line" {
			t.Fatalf("sales NamespaceのData統合結果が期待値と異なります: %#v", ns.Data)
		}
		if len(ns.Validations) != 1 || ns.Validations[0].ID != "sales.order_line.quantity_positive" {
			t.Fatalf("sales NamespaceのValidation統合結果が期待値と異なります: %#v", ns.Validations)
		}
		found = true
	}
	if !found {
		t.Fatal("sales Semantic Namespaceが見つかりません")
	}

	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("Source Indexの構築に失敗しました: %v", err)
	}
	subject := sourceindex.Subject{Kind: "semantic-validation", ID: "sales.order_line.quantity_positive"}
	refs := idx.ReferencesFor(subject)
	if len(refs) != 1 {
		t.Fatalf("Semantic Validationのsource references=%#v", refs)
	}
	if refs[0].ArtifactPath != validationPath {
		t.Fatalf("Semantic Validationのsource path=%q want=%q", refs[0].ArtifactPath, validationPath)
	}
	content := []byte(validationSource)
	rng := refs[0].Range
	if rng.Start.Offset == rng.End.Offset {
		t.Fatalf("Semantic Validationのsource rangeがpointのままです: %+v", rng)
	}
	if got := string(content[rng.Start.Offset:rng.End.Offset]); got != "sales.order_line.quantity_positive" {
		t.Fatalf("Semantic Validationのidentifier range=%q want=%q", got, "sales.order_line.quantity_positive")
	}
}

func writeSemanticValidationWorkspaceFile(t *testing.T, root, relativePath, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
