package operations

import (
    "path/filepath"
    "testing"

    "tei/internal/workspace"
)

// This intentionally consumes actual .tei source and the existing TEI loader,
// rather than constructing an independent JSON-only organization toy model.
// It is NOT a test of SQL execution, temporal rules, Host grants or data effects.
func TestOrgHeadlessAuthoringToOperations(t *testing.T) {
    root := filepath.Join("..", "..", "reference", "org-headless-workspace")
    surface, err := SurfaceWorkspace(root, "org_view")
    if err != nil {
        t.Fatalf("load native organization workspace and derive operations: %v", err)
    }
    if surface.WorldFingerprint == "" || surface.ModelRevision == "" {
        t.Fatalf("missing actual workspace-derived Source/World identity: %+v", surface)
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

func TestOrgHeadlessSeparateMembershipFromOccupancy(t *testing.T) {
    root := filepath.Join("..", "..", "reference", "org-headless-workspace")
    loaded, err := workspace.Load(root)
    if err != nil {
        t.Fatalf("parse native organization .tei source: %v", err)
    }
    var membership, occupancy bool
    for _, ns := range loaded.World.Semantics {
        for _, data := range ns.Data {
            if data.ID != "organization.membership" && data.ID != "organization.occupancy" {
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
        t.Fatal("native semantic model collapsed Membership and Occupancy")
    }
}
