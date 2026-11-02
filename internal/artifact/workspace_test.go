package artifact

import (
	"path/filepath"
	"testing"
)

func TestRenderWorkspaceRendersTheNegativeFixture(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "standard-crud", "workspace")

	set, err := RenderWorkspace(root, "ledger")
	if err != nil {
		t.Fatalf("render workspace: %v", err)
	}
	if set.InputRevision == "" {
		t.Fatal("the input revision must come from the workspace")
	}
	if len(set.Artifacts) != 4 {
		t.Fatalf("expected three table definitions and one function definition: %d", len(set.Artifacts))
	}
	kinds := map[string]int{}
	for _, artifact := range set.Artifacts {
		kinds[artifact.Kind]++
		if artifact.InputRevision != set.InputRevision {
			t.Fatalf("artifact %s is bound to another revision", artifact.Kind)
		}
	}
	if kinds[string(KindTableDefinition)] != 3 || kinds[string(KindFunctionDefinition)] != 1 {
		t.Fatalf("unexpected artifact kinds: %+v", kinds)
	}

	// sales.note has no key, so its table definition stays unresolved and the
	// set must not claim to be complete.
	if set.Manifest.Complete {
		t.Fatalf("the negative fixture must not be a complete set: %+v", set.Manifest)
	}
	var unresolved int
	for _, artifact := range set.Artifacts {
		unresolved += len(artifact.Unresolved)
	}
	if unresolved == 0 {
		t.Fatal("the negative fixture must carry unresolved entries")
	}
}

func TestRenderWorkspaceRefusesAnUnknownApplication(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "standard-crud", "workspace")
	if _, err := RenderWorkspace(root, "missing"); err == nil {
		t.Fatal("an application that the workspace does not declare must be refused")
	}
}
