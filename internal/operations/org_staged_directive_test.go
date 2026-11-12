package operations

import (
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

// TestOrgStagedDirectiveIsNotCurrentAuthority verifies Directive is a separate
// semantic Data and does not become current membership or execution authority
// by itself.
func TestOrgStagedDirectiveIsNotCurrentAuthority(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load staged axis workspace: %v", err)
	}

	found := false
	for _, ns := range loaded.World.Semantics {
		for _, data := range ns.Data {
			if string(data.ID) == "organization.directive" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("organization.directive data is missing")
	}

	surface, err := SurfaceWorkspace(root, "org_axes_view")
	if err != nil {
		t.Fatalf("derive operation surface for axis fixture: %v", err)
	}
	if surface.Effect != "read-only" || len(surface.AuthorityLimits) == 0 {
		t.Fatalf("operation surface must not claim Host authority: %+v", surface)
	}
	for _, op := range surface.Operations {
		if op.Target == "organization.directive" || len(op.Effects) != 0 || op.Binding != "" {
			t.Fatalf("Directive must not imply effects or Host authorization: %+v", op)
		}
	}
}
