package operations

import (
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

// TestOrgStagedEntitySeparation verifies the staged axis fixture keeps
// Employment, Membership, and Affiliation as separate semantic Data and
// does not derive Tenant/Grant from them alone.
func TestOrgStagedEntitySeparation(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load staged axis workspace: %v", err)
	}

	// Employment, Membership, and Affiliation are separate Data.
	found := map[string]bool{}
	for _, ns := range loaded.World.Semantics {
		for _, data := range ns.Data {
			switch string(data.ID) {
			case "organization.employment", "organization.membership", "organization.affiliation":
				found[string(data.ID)] = true
			}
		}
	}
	if !found["organization.employment"] || !found["organization.membership"] || !found["organization.affiliation"] {
		t.Fatalf("missing entity data: %+v", found)
	}

	// OrgUnit axis relations must not imply read/write/Policy/Grant.
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
