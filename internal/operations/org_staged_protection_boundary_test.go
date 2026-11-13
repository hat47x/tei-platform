package operations

import (
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

// TestOrgStagedProtectionBoundary verifies the staged axis README documents
// that normal organizational data is not treated as secret by default, and
// that the fixture does not invent new ACL/OS-isolation/anonymization.
func TestOrgStagedProtectionBoundary(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load staged axis workspace: %v", err)
	}

	found := map[string]bool{}
	for _, ns := range loaded.World.Semantics {
		for _, data := range ns.Data {
			switch string(data.ID) {
			case "organization.person", "organization.org_unit", "organization.membership":
				found[string(data.ID)] = true
			}
		}
	}
	if !found["organization.person"] || !found["organization.org_unit"] || !found["organization.membership"] {
		t.Fatalf("missing normal org data: %+v", found)
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
			t.Fatalf("OrgUnit relations must not imply read, edit or Host authorization: %+v", op)
		}
	}
}
