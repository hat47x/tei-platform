package operations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

// TestOrgStagedDelegationMisinference verifies the staged axis fixture keeps
// employment/affiliation/directive as structure only and does not derive
// Tenant read/write Grant from them.
func TestOrgStagedDelegationMisinference(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load staged axis workspace: %v", err)
	}

	found := map[string]bool{}
	for _, ns := range loaded.World.Semantics {
		for _, data := range ns.Data {
			switch string(data.ID) {
			case "organization.employment", "organization.affiliation", "organization.directive":
				found[string(data.ID)] = true
			}
		}
	}
	if !found["organization.employment"] || !found["organization.affiliation"] || !found["organization.directive"] {
		t.Fatalf("missing delegation scenario data: %+v", found)
	}

	surface, err := SurfaceWorkspace(root, "org_axes_view")
	if err != nil {
		t.Fatalf("derive operation surface for axis fixture: %v", err)
	}
	if surface.Effect != "read-only" || len(surface.AuthorityLimits) == 0 {
		t.Fatalf("operation surface must not claim Host authority: %+v", surface)
	}
	for _, op := range surface.Operations {
		if op.Target == "organization.employment" || op.Target == "organization.affiliation" || op.Target == "organization.directive" ||
			len(op.Effects) != 0 || op.Binding != "" {
			t.Fatalf("delegation relations must not imply effects or Host authorization: %+v", op)
		}
	}

	scenariosPath := filepath.Join(root, "scenarios", "delegation-misinference.json")
	raw, err := os.ReadFile(scenariosPath)
	if err != nil {
		t.Fatalf("read delegation scenarios: %v", err)
	}
	var scenarios struct {
		Scenarios    []json.RawMessage `json:"scenarios"`
		Expectations map[string]string `json:"expectations"`
	}
	if err := json.Unmarshal(raw, &scenarios); err != nil {
		t.Fatalf("parse delegation scenarios: %v", err)
	}
	if len(scenarios.Scenarios) == 0 {
		t.Fatal("expected at least one delegation scenario")
	}
}
