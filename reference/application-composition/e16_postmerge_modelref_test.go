// E16 checks the already-integrated typed impact path against one real World.
// These source and impact observations are not an acceptance or runtime grant.
package main

import (
	"path/filepath"
	"testing"

	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/subject"
	"tei/internal/workspace"
)

func e16Key(kind, id string) modelref.Key {
	return modelref.Key{Scope: "order-entry-reference", Subject: subject.Subject{Kind: kind, ID: id}}
}

func TestE16ActualWorkspaceRelationChangeReachesOwners(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load reference workspace: %v", err)
	}
	sources, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("build Source Index from same loaded Workspace: %v", err)
	}
	report := inspectE5(loaded.World, sources, "order_entry")
	if report.Status != "observed" || len(report.Issues) != 0 {
		t.Fatalf("application definitions unresolved: %d issues", len(report.Issues))
	}
	for _, anchor := range []struct{ kind, id, path string }{
		{"semantic-relation", "sales.order.customer", "semantic/sales/order.tei"},
		{"application-pattern", "order_entry.customer_reference", "applications/order-entry/patterns/customer-reference.tei"},
	} {
		if !e14UniqueDefinition(sources, anchor.kind, anchor.id, anchor.path) {
			t.Fatalf("typed reference has no unique definition: %s:%s", anchor.kind, anchor.id)
		}
	}

	// Reuse the existing Derived Index and modelref: no second dependency graph.
	projection := derivedbridge.FromIndex(index.Build(loaded.World), "order-entry-reference")
	if len(projection.Unresolved) != 0 {
		t.Fatalf("typed reference projection unresolved (%d)", len(projection.Unresolved))
	}
	if diagnostics := modelref.Validate(projection.Graph, projection.Rules); len(diagnostics) != 0 {
		t.Fatalf("typed reference validation has %d diagnostics", len(diagnostics))
	}

	relation := e16Key("semantic-relation", "sales.order.customer")
	pattern := e16Key("application-pattern", "order_entry.customer_reference")
	application := e16Key("application", "order_entry")
	before := projection.Graph
	if changes := modelref.Diff(before, before); len(changes) != 0 {
		t.Fatalf("unchanged graph produced %d changes", len(changes))
	}

	// This is an artificial version change, not a two-revision Source comparison.
	// It checks the actual parsed World -> typed Graph -> Diff -> Impact chain.
	after := before
	after.Nodes = append([]modelref.Node(nil), before.Nodes...)
	found := 0
	for i := range after.Nodes {
		if after.Nodes[i].Key == relation {
			after.Nodes[i].Version = "e16-synthetic-next-version"
			found++
		}
	}
	if found != 1 {
		t.Fatalf("expected one typed relation node; found %d", found)
	}
	changes := modelref.Diff(before, after)
	if len(changes) != 1 || changes[0].Key != relation || changes[0].Kind != "version-changed" {
		t.Fatalf("unexpected typed change classification: %+v", changes)
	}
	seen := map[modelref.Key]bool{}
	for _, impact := range modelref.Impact(before, changes, projection.Rules) {
		seen[impact.Consumer] = true
	}
	if !seen[pattern] || !seen[application] {
		t.Fatalf("real Workspace impact lost Pattern/Application: pattern=%v application=%v", seen[pattern], seen[application])
	}
}

func TestE16UnknownScopeCannotBorrowActualWorkspaceRelations(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load reference workspace: %v", err)
	}
	projection := derivedbridge.FromIndex(index.Build(loaded.World), "order-entry-reference")
	if len(projection.Unresolved) != 0 {
		t.Fatalf("typed reference projection unresolved (%d)", len(projection.Unresolved))
	}
	alien := e16Key("semantic-relation", "sales.order.customer")
	alien.Scope = "other-workspace"
	changes := []modelref.Change{{Key: alien, Kind: "version-changed"}}
	if impacts := modelref.Impact(projection.Graph, changes, projection.Rules); len(impacts) != 0 {
		t.Fatalf("a foreign Workspace scope borrowed local impact: %d candidates", len(impacts))
	}
}
