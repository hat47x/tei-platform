package operations

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
)

// This is a read-only cross-operation consistency check on a REAL public
// Workspace source. Both operations load independently: matching fingerprints
// do not prove one atomic Source read, current Policy or write authority.
func TestTEARealWorkspaceMapAndLocateAgreeOnQualifiedRelation(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	relation := sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.lines"}
	parent := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	target := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order_line"}

	mapped, err := MapWorkspace(root)
	if err != nil {
		t.Fatalf("map public Workspace: %v", err)
	}
	located, err := LocateWorkspace(root, relation, 1)
	if err != nil {
		t.Fatalf("locate public Workspace relation: %v", err)
	}
	if mapped.Effect != "read-only" || located.Effect != "read-only" ||
		mapped.Coverage != "partial-projection" {
		t.Fatalf("incorrect effect or coverage: map=%q/%q locate=%q", mapped.Effect, mapped.Coverage, located.Effect)
	}
	if mapped.Catalog.Workspace == nil || located.Workspace == nil ||
		!reflect.DeepEqual(mapped.Catalog.Workspace, located.Workspace) {
		t.Fatalf("Workspace identities differ or are absent: map=%+v locate=%+v", mapped.Catalog.Workspace, located.Workspace)
	}
	if mapped.Catalog.WorldFingerprint == "" || mapped.Catalog.SourceIndexFingerprint == "" ||
		mapped.Catalog.ModelRevision == "" ||
		mapped.Catalog.WorldFingerprint != located.WorldFingerprint ||
		mapped.Catalog.SourceIndexFingerprint != located.SourceIndexFingerprint ||
		mapped.Catalog.ModelRevision != located.ModelRevision {
		t.Fatalf("separate Map/Locate reads disagree: map=%+v locate=%+v", mapped.Catalog, located)
	}
	// Existing #4392 owns MapWorkspace's Catalog.Validation qualification:
	// on this baseline it is zero-valued even when Locate validated the
	// real Workspace. This test must not mistake "not checked" for "failed"
	// or duplicate the validator fix in a separate TEA lane.
	if !located.Validation.OK || located.Validation.ErrorCount != 0 {
		t.Fatalf("valid public Workspace did not validate in Locate: %+v", located.Validation)
	}
	if located.Subject.Subject != relation || located.Subject.Ambiguous ||
		located.Subject.SourceStatus != "indexed" ||
		located.Subject.Parent == nil || *located.Subject.Parent != parent {
		t.Fatalf("relation identity/source/parent lost: %+v", located.Subject)
	}
	if len(located.Path) == 0 || located.Path[len(located.Path)-1].Subject != parent {
		t.Fatalf("parent path missing: %+v", located.Path)
	}

	linkCount := func(kind string, from, to sourceindex.Subject) int {
		var count int
		for _, link := range mapped.Links {
			if link.Relation == kind && link.From == from && link.To == to {
				if link.Resolution != "resolved" {
					t.Fatalf("qualified navigation relation is not structurally resolved: %+v", link)
				}
				count++
			}
		}
		return count
	}
	if n := linkCount("contains", parent, relation); n != 1 {
		t.Fatalf("expected one parent→relation link, got %d", n)
	}
	if n := linkCount("refers-to", relation, target); n != 1 {
		t.Fatalf("expected one relation→target link, got %d", n)
	}
	var targetLinks int
	for _, link := range located.Outgoing {
		if link.Role == LinkRoleRelationTarget && link.From == relation && link.To == target {
			if !link.Resolved {
				t.Fatalf("relation target declared but unresolved: %+v", link)
			}
			targetLinks++
		}
	}
	if targetLinks != 1 {
		t.Fatalf("Locate target is not the same typed relation as Map: count=%d outgoing=%+v", targetLinks, located.Outgoing)
	}
	var unknownQueryDependencies bool
	for _, family := range located.Coverage.NotDerived {
		if family == "query-data-dependency" {
			unknownQueryDependencies = true
		}
	}
	if !unknownQueryDependencies {
		t.Fatalf("bounded locate falsely implies complete query/Data relation coverage: %+v", located.Coverage)
	}
	// No attempt to turn "resolved" or an indexed Source into TEA Role,
	// Disclosure, Action-specific CRUD, Effect, or Owner acceptance is made.
}
