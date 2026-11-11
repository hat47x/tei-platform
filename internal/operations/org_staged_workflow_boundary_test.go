package operations

import (
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

// TestOrgStagedWorkflowBoundary verifies the staged axis fixture connects
// business procedure to existing Action/Pattern/Capability/EffectRule/RunPlan
// without introducing new Workflow Engine, Status Enum, Workflow Host, or
// Proof Store concepts.
func TestOrgStagedWorkflowBoundary(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load staged axis workspace: %v", err)
	}

	found := false
	for _, ns := range loaded.World.Semantics {
		for _, cap := range ns.Capabilities {
			if string(cap.ID) == "organization.approve_membership_change" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("organization.approve_membership_change capability is missing")
	}

	surface, err := SurfaceWorkspace(root, "org_axes_view")
	if err != nil {
		t.Fatalf("derive operation surface for axis fixture: %v", err)
	}
	if surface.Effect != "read-only" || len(surface.AuthorityLimits) == 0 {
		t.Fatalf("operation surface must not claim Host authority: %+v", surface)
	}
	for _, op := range surface.Operations {
		if op.Target == "organization.approve_membership_change" || len(op.Effects) != 0 || op.Binding != "" {
			t.Fatalf("capability must not imply effects or Host authorization: %+v", op)
		}
	}
}
