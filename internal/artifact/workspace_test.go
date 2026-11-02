package artifact

import (
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/operations"
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
	if len(set.Artifacts) != 5 {
		t.Fatalf("expected three table definitions, one function definition and one operation surface: %d", len(set.Artifacts))
	}
	kinds := map[string]int{}
	for _, artifact := range set.Artifacts {
		kinds[artifact.Kind]++
		if artifact.InputRevision != set.InputRevision {
			t.Fatalf("artifact %s is bound to another revision", artifact.Kind)
		}
	}
	if kinds[string(KindTableDefinition)] != 3 || kinds[string(KindFunctionDefinition)] != 1 || kinds[string(KindOperationSurface)] != 1 {
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
		if artifact.Kind != string(KindOperationSurface) {
			continue
		}
		if err := artifact.Verify(); err != nil {
			t.Fatalf("the surface artifact must verify: %v", err)
		}
		if !strings.Contains(artifact.Content, operations.OperationSurfaceSchemaVersion) {
			t.Fatalf("the surface artifact must carry the surface schema:\n%s", artifact.Content)
		}
		// The fixture has four gaps: note read/create, order list conflict and
		// order retire. Each gap becomes one unresolved entry.
		if len(artifact.Unresolved) != 4 {
			t.Fatalf("every gap must be carried as unresolved: %+v", artifact.Unresolved)
		}
	}
	if unresolved == 0 {
		t.Fatal("the negative fixture must carry unresolved entries")
	}
	// Generation runs CheckSet itself, so an inconsistent set can not be
	// handed to a caller as if it were consistent.
	if len(set.Findings) != 0 {
		t.Fatalf("a consistently rendered set must carry no findings: %+v", set.Findings)
	}
}

func TestRenderWorkspaceRefusesAnUnknownApplication(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "standard-crud", "workspace")
	if _, err := RenderWorkspace(root, "missing"); err == nil {
		t.Fatal("an application that the workspace does not declare must be refused")
	}
}
