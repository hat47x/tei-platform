package workspace_test

import (
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestReferenceCommandPatternUsesCapabilityContractWithoutDuplicatingIt(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	result, err := workspace.Load(ws)
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}

	const patternID = "order_entry.submit_order"
	const patternPath = "applications/order-entry/patterns/submit-order.tei"

	app := referenceApplicationByID(t, result.World.Applications, "order_entry")
	if !containsCanonicalID(app.SemanticUses, "sales") {
		t.Fatalf("order_entryがsales Semantic Namespaceを利用していません: uses=%#v", app.SemanticUses)
	}

	pattern := applicationPatternByID(t, app.Patterns, patternID)
	if pattern.Kind != "command" || pattern.Capability != "sales.order.submit" {
		t.Fatalf("command Patternの意味が期待値と異なります: %#v", pattern)
	}
	if pattern.Data != "" || pattern.Relation != "" || pattern.Query != "" || pattern.Datasource != "" {
		t.Fatalf("Capabilityが持つ意味や実装資源がcommand Patternへ重複記述されています: %#v", pattern)
	}

	capability := semanticCapabilityByID(t, result.World.Semantics, pattern.Capability)
	if len(capability.Inputs) != 1 {
		t.Fatalf("sales.order.submitのinput=%#v", capability.Inputs)
	}
	if capability.Inputs[0].Name != "order" || capability.Inputs[0].Type != "sales.order" {
		t.Fatalf("sales.order.submitのinput contractが期待値と異なります: %#v", capability.Inputs[0])
	}
	if capability.Output != "sales.order" {
		t.Fatalf("sales.order.submitの既存outputが変わっています: %q", capability.Output)
	}

	idx, err := workspace.BuildSourceIndex(ws, result)
	if err != nil {
		t.Fatalf("Source Indexの構築に失敗しました: %v", err)
	}
	refs := idx.ReferencesFor(sourceindex.Subject{Kind: "application-pattern", ID: patternID})
	if len(refs) != 1 {
		t.Fatalf("command Patternのsource references=%#v", refs)
	}
	if refs[0].ArtifactPath != patternPath {
		t.Fatalf("command Patternのsource path=%q want=%q", refs[0].ArtifactPath, patternPath)
	}
}

func semanticCapabilityByID(t *testing.T, namespaces []semantic.Namespace, id core.CanonicalID) semantic.Capability {
	t.Helper()
	for _, namespace := range namespaces {
		for _, capability := range namespace.Capabilities {
			if capability.ID == id {
				return capability
			}
		}
	}
	t.Fatalf("Semantic Capability %sが見つかりません", id)
	return semantic.Capability{}
}
