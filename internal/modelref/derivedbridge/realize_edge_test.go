package derivedbridge

import (
	"testing"

	"tei/internal/core"
	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/subject"
)

// subjectID converts a literal to the Canonical ID type the index uses.
func subjectID(id string) core.CanonicalID { return core.CanonicalID(id) }

func bindingKey(scope, kind, id string) modelref.Key {
	return modelref.Key{Scope: scope, Subject: subject.Subject{Kind: kind, ID: id}}
}

// bindingDerived builds an index with one application, one capability and the
// realize binding between them.
func bindingDerived(application, capability string) index.Derived {
	d := index.Derived{}
	d.ApplicationPatterns = append(d.ApplicationPatterns,
		index.ApplicationPatternIndexEntry{Application: subjectID(application), Pattern: "pattern"})
	d.SemanticElements = append(d.SemanticElements,
		index.SemanticElementEntry{Kind: "capability", ID: subjectID(capability)})
	d.CustomGoBindings = append(d.CustomGoBindings,
		index.BindingIndexEntry{Application: subjectID(application), Capability: subjectID(capability),
			Adapter: "adapter.bind.go", Hook: "TeiBind"})
	return d
}

// TestRealizeBindingIsAGraphEdge pins #3983. A `realize X with custom go`
// declaration states that an application implements a capability. The index
// carried the binding, but the projection never made it an edge, so `trace`
// could not answer "what implements this capability".
func TestRealizeBindingIsAGraphEdge(t *testing.T) {
	p := FromIndex(bindingDerived("order_entry", "security.authenticate"), "c")

	from := bindingKey("c", "application", "order_entry")
	to := bindingKey("c", "semantic-capability", "security.authenticate")
	found := false
	for _, edge := range p.Graph.Edges {
		if edge.From != from || edge.Relation != "realizes" {
			continue
		}
		if edge.To.Subject != to.Subject {
			t.Fatalf("realizes edge points at %v, want %v", edge.To.Subject, to.Subject)
		}
		found = true
	}
	if !found {
		t.Fatalf("no realizes edge from %v; edges are %v", from, p.Graph.Edges)
	}

	// The rule must exist, or the edge is reported as invalid-edge-type.
	rule := modelref.Rule{Relation: "realizes", FromKind: "application", ToKind: "semantic-capability"}
	hasRule := false
	for _, r := range p.Rules {
		if r == rule {
			hasRule = true
		}
	}
	if !hasRule {
		t.Fatalf("the realizes rule is missing from %v", p.Rules)
	}
	if len(p.Unresolved) != 0 {
		t.Fatalf("unexpected unresolved entries: %v", p.Unresolved)
	}
}

// The adapter filename and hook name name a generated file, not a model subject,
// so they must not become graph nodes. Only the application and the capability
// are nodes.
func TestRealizeBindingDoesNotInventAnAdapterNode(t *testing.T) {
	p := FromIndex(bindingDerived("order_entry", "security.authenticate"), "c")
	// The fixture also declares an application pattern, so three kinds are
	// present before any binding is projected. An adapter node would be a fourth.
	allowed := map[string]bool{"application": true, "application-pattern": true, "semantic-capability": true}
	for _, node := range p.Graph.Nodes {
		if !allowed[node.Key.Subject.Kind] {
			t.Fatalf("binding introduced the node %v", node.Key)
		}
	}
	// And no edge may reference the adapter filename as a subject.
	for _, edge := range p.Graph.Edges {
		if edge.To.Subject.ID == "adapter.bind.go" || edge.From.Subject.ID == "adapter.bind.go" {
			t.Fatalf("the adapter filename became a graph subject: %v", edge)
		}
	}
}

// A binding with no declared capability is an incomplete declaration: it must be
// reported, not silently dropped.
func TestRealizeBindingWithoutCapabilityIsReported(t *testing.T) {
	d := bindingDerived("order_entry", "security.authenticate")
	d.SemanticElements = nil // drop the capability declaration
	p := FromIndex(d, "c")
	want := "unmaterialized-to:semantic-capability:security.authenticate"
	reported := false
	for _, entry := range p.Unresolved {
		if entry == want {
			reported = true
		}
	}
	if !reported {
		t.Fatalf("%q not reported; unresolved = %v", want, p.Unresolved)
	}
}

// A binding missing its application is also incomplete.
func TestRealizeBindingWithoutApplicationIsReported(t *testing.T) {
	d := bindingDerived("order_entry", "security.authenticate")
	d.ApplicationPatterns = nil
	p := FromIndex(d, "c")
	want := "unmaterialized-from:application:order_entry"
	reported := false
	for _, entry := range p.Unresolved {
		if entry == want {
			reported = true
		}
	}
	if !reported {
		t.Fatalf("%q not reported; unresolved = %v", want, p.Unresolved)
	}
}
