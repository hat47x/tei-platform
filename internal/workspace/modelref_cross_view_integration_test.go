package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/modelref/sourcebridge"
	"tei/internal/workspace"
	"tei/internal/subject"
)

func TestIssue1120ReadOnlyReferenceWorldToSourceAndImpact(t *testing.T) {
	root := filepath.Join(repoRoot(t), "reference", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load authoritative reference workspace: %v", err)
	}
	sources, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("source index from same workspace snapshot: %v", err)
	}
	const scope = "order-entry-reference"
	projection := derivedbridge.FromIndex(index.Build(loaded.World), scope)
	if len(projection.Graph.Nodes) == 0 || len(projection.Graph.Edges) == 0 {
		t.Fatal("read-only typed projection is empty")
	}
	// This Graph describes existing Data and Application Patterns, not an
	// accepted Business Procedure, actual CRUD, or generated UI.
	key := func(kind, id string) modelref.Key {
		return modelref.Key{Scope: scope, Subject: subject.Subject{Kind: kind, ID: id}}
	}
	pattern := key("application-pattern", "order_entry.customer_reference")
	relation := key("semantic-relation", "sales.order.customer")
	customer := key("semantic-data", "customer.customer")
	app := key("application", "order_entry")
	containsEdge := func(from, to modelref.Key) bool {
		for _, edge := range projection.Graph.Edges {
			if edge.From == from && edge.To.Scope == to.Scope && edge.To.Subject == to.Subject {
				return true
			}
		}
		return false
	}
	if !containsEdge(pattern, relation) || !containsEdge(relation, customer) || !containsEdge(app, pattern) {
		t.Fatalf("Data/Function/Relation cross-reference lost: %v", projection.Unresolved)
	}
	for _, diag := range modelref.Validate(projection.Graph, projection.Rules) {
		// The adapter must not return references invalid under its own
		// observed-kind rules. Gaps are reported separately as Unresolved.
		t.Errorf("invalid observed typed reference: %s at %s", diag.Code, diag.At)
	}
	read := func(relative string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	}
	// Real Workspace consumers reuse one validated snapshot for all visual
	// projections instead of repeatedly reading every Source artifact.
	resolver, err := sourcebridge.NewResolver(projection.Graph, sources, read)
	if err != nil {
		t.Fatalf("source snapshot validation: %v", err)
	}
	targets := []struct {
		key  modelref.Key
		path string
	}{
		{pattern, "applications/order-entry/patterns/customer-reference.tei"},
		{relation, "semantic/sales/order.tei"},
	}
	for _, item := range targets {
		got := resolver.Locate(item.key)
		if got.Status != "located" {
			t.Fatalf("cannot navigate %v to source: %+v", item.key, got)
		}
		found := false
		for _, ref := range got.References {
			if ref.ArtifactPath == item.path {
				found = true
			}
		}
		if !found {
			t.Fatalf("source navigation mismatch for %v: %+v", item.key, got)
		}
	}
	affected := modelref.Impact(projection.Graph, []modelref.Change{{Key: relation, Kind: "version-changed"}}, projection.Rules)
	seenPattern, seenApp := false, false
	for _, consumer := range affected {
		if consumer.Consumer == pattern {
			seenPattern = true
		}
		if consumer.Consumer == app {
			seenApp = true
		}
	}
	if !seenPattern || !seenApp {
		t.Fatalf("relation must surface direct Pattern and transitive Application review candidates: %+v", affected)
	}
	// A legacy Business step cannot gain Canonical identity just by joining
	// the Application Pattern. The Business authoring lane owns that meaning.
	for _, node := range projection.Graph.Nodes {
		if node.Key.Subject.Kind == "business-step" || node.Key.Subject.Kind == "task" {
			t.Fatalf("fabricated accepted Business identity: %+v", node)
		}
	}
}
