package operations

import (
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

// TestOrgStagedApprovalCandidateSeparation verifies the staged axis fixture
// does not derive Capability/EffectRule/RunPlan from RoleAssignment or
// Responsibility structure relations.
func TestOrgStagedApprovalCandidateSeparation(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load staged axis workspace: %v", err)
	}

	found := map[string]bool{}
	for _, ns := range loaded.World.Semantics {
		for _, data := range ns.Data {
			switch string(data.ID) {
			case "organization.role_assignment", "organization.responsibility":
				found[string(data.ID)] = true
			}
		}
	}
	if !found["organization.role_assignment"] || !found["organization.responsibility"] {
		t.Fatalf("missing approval candidate data: %+v", found)
	}

	surface, err := SurfaceWorkspace(root, "org_axes_view")
	if err != nil {
		t.Fatalf("derive operation surface for axis fixture: %v", err)
	}
	if surface.Effect != "read-only" || len(surface.AuthorityLimits) == 0 {
		t.Fatalf("operation surface must not claim Host authority: %+v", surface)
	}
	for _, op := range surface.Operations {
		if op.Target == "organization.role_assignment" || op.Target == "organization.responsibility" ||
			len(op.Effects) != 0 || op.Binding != "" {
			t.Fatalf("role relations must not imply effects or Host authorization: %+v", op)
		}
	}
}
