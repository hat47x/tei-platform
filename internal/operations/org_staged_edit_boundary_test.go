package operations

import (
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

// TestOrgStagedEditBoundary verifies the staged axis fixture keeps
// irreversible projections read-only and that the existing operation
// surface is the only edit-gate path.
func TestOrgStagedEditBoundary(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load staged axis workspace: %v", err)
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}
	catalog := CatalogWorld(loaded.World, idx, "", "wf", "sf")
	mapped := MapWorld(loaded.World, catalog)
	if mapped.Effect != "read-only" {
		t.Fatalf("projection must remain read-only: %+v", mapped)
	}
	if mapped.Coverage != "partial-projection" {
		t.Fatalf("projection coverage must declare its limits: %+v", mapped)
	}
	if len(mapped.AuthorityLimits) == 0 {
		t.Fatalf("projection must declare authority limits: %+v", mapped)
	}
}
