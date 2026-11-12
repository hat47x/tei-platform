package operations

import (
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

// TestOrgStagedProjectionFitGap asserts the existing read-only operations can
// navigate the staged axis Source, while explicit coverage/authority limits
// record the missing org-specific projections.
func TestOrgStagedProjectionFitGap(t *testing.T) {
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
	for _, limit := range mapped.AuthorityLimits {
		if limit == "standard-custom-extension-qualification-is-not-observed" {
			return
		}
	}
	t.Fatalf("missing projection authority limit: %+v", mapped.AuthorityLimits)
}
