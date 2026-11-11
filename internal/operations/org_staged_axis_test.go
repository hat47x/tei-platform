package operations

import (
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

// TestOrgStagedAxisLoadsAndValidates asserts the staged axis workspace is
// accepted by the existing loader without new Core kinds.
func TestOrgStagedAxisLoadsAndValidates(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	_, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load staged axis workspace: %v", err)
	}
}

// TestOrgStagedAxisStructureDoesNotGrantOperations asserts the staged axis
// fixture keeps OrgUnit axis relations as semantic structure only, and does
// not derive read/write/Policy/Grant from them alone.
func TestOrgStagedAxisStructureDoesNotGrantOperations(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	surface, err := SurfaceWorkspace(root, "org_axes_view")
	if err != nil {
		t.Fatalf("derive operation surface for axis fixture: %v", err)
	}
	if surface.Effect != "read-only" || len(surface.AuthorityLimits) == 0 {
		t.Fatalf("operation surface must not claim Host authority: %+v", surface)
	}
	for _, op := range surface.Operations {
		if op.Target == "organization.org_unit" || len(op.Effects) != 0 || op.Binding != "" {
			t.Fatalf("OrgUnit axis relations must not imply read, edit or Host authorization: %+v", op)
		}
	}
}
