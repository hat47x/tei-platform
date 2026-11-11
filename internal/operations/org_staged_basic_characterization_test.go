package operations

import (
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

// TestOrgStagedBasicSurface uses the dedicated org-staged-basic fixture to
// assert the simple-model acceptance criteria from the staged org modeling
// plan (WU-01). It does NOT change org-headless-workspace or any accepted
// source; the separate fixture keeps the new test from duplicating the
// existing native org characterization tests.
func TestOrgStagedBasicSurface(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-basic")
	surface, err := SurfaceWorkspace(root, "org_view")
	if err != nil {
		t.Fatalf("load staged basic workspace and derive operations: %v", err)
	}
	if surface.Effect != "read-only" || len(surface.AuthorityLimits) == 0 {
		t.Fatalf("operation surface must not claim Host authority: %+v", surface)
	}

	var list, read *Operation
	for i := range surface.Operations {
		op := &surface.Operations[i]
		switch op.ID {
		case "org_view.list.organization.membership":
			list = op
		case "org_view.read.organization.membership":
			read = op
		default:
			t.Fatalf("read-only application derived an unexpected operation: %+v", *op)
		}
	}
	if list == nil || read == nil || len(surface.Operations) != 2 {
		t.Fatalf("expected list and read of distinct Membership occurrences: %+v", surface.Operations)
	}
	if list.Kind != "list" || read.Kind != "read" || read.Autonomy != "unattended" {
		t.Fatalf("wrong operation semantics: list=%+v read=%+v", list, read)
	}
	if len(read.Inputs) != 1 || read.Inputs[0] != "organization.membership.membership_id" {
		t.Fatalf("read must use Membership occurrence identity, not Person/OrgUnit: %+v", read.Inputs)
	}
	if len(read.Effects) != 0 || read.Binding != "" || len(surface.Gaps) != 0 {
		t.Fatalf("read-only operation must not invent change authority: %+v", surface)
	}
}

// TestOrgStagedBasicSemanticSeparation asserts the staged basic fixture keeps
// Membership/Occupancy as separate relations and does not derive CaseRecord
// authorization from a semantic relation alone.
func TestOrgStagedBasicSemanticSeparation(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-basic")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("parse staged basic .tei source: %v", err)
	}

	var membership, occupancy bool
	for _, ns := range loaded.World.Semantics {
		for _, data := range ns.Data {
			switch data.ID {
			case "organization.membership", "organization.occupancy":
			default:
				continue
			}
			targets := map[string]bool{}
			for _, rel := range data.Relations {
				if rel.Cardinality != "one" || len(rel.Keys) != 1 {
					t.Fatalf("reference relation lost its explicitly authored target/key: %+v", rel)
				}
				targets[string(rel.Target)] = true
			}
			if data.ID == "organization.membership" {
				membership = targets["organization.person"] && targets["organization.org_unit"] && len(targets) == 2
			} else {
				occupancy = targets["organization.person"] && targets["organization.position"] && len(targets) == 2
			}
		}
	}
	if !membership || !occupancy {
		t.Fatal("staged basic semantic model collapsed Membership and Occupancy")
	}

	surface, err := SurfaceWorkspace(root, "org_view")
	if err != nil {
		t.Fatalf("derive read-only operation surface: %v", err)
	}
	if surface.Effect != "read-only" || len(surface.Operations) != 2 {
		t.Fatalf("unpersisted CaseRecord relationship must not create operations/grants: %+v", surface)
	}
	for _, op := range surface.Operations {
		if op.Target == "organization.case_record" || len(op.Effects) != 0 || op.Binding != "" {
			t.Fatalf("CaseRecord relation must not imply read, edit or Host authorization: %+v", op)
		}
	}
}
