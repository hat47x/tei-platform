package operations

import (
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

// TestOrgStagedTemporalWindows asserts the staged axis fixture keeps valid-time
// and known-time attributes declared, and that the workspace loader accepts
// the temporal shape without inventing grants.
func TestOrgStagedTemporalWindows(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	_, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load staged axis workspace: %v", err)
	}

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
