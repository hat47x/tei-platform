package index

import (
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/semantic"
)

// bindingsWorld builds a World with one application, one capability and a
// custom Go binding that connects them.
func bindingsWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{
			ID: core.MustCanonicalID("security"),
			Types: []semantic.Type{{
				ID: core.MustCanonicalID("security.credential"), Symbol: "Credential", Kind: semantic.TypePrimitive,
			}},
			Capabilities: []semantic.Capability{{
				ID:     core.MustCanonicalID("security.authenticate"),
				Symbol: "authenticate",
				Inputs: []semantic.Parameter{{
					Name: "credential", Type: core.MustCanonicalID("security.credential"),
				}},
			}},
		}},
		Bindings: []realization.CustomGoBinding{{
			Application: core.MustCanonicalID("order_entry"),
			Capability:  core.MustCanonicalID("security.authenticate"),
		}},
	}
}

// A `realize X with custom go` is a declared edge from the application to the
// capability. Recording it only in CustomGoBindings left it out of the
// reference graph, so trace could not answer "what implements this capability?"
// and removing a binding changed no graph fingerprint (#3983).
func TestCustomGoBindingBecomesARealizesEdge(t *testing.T) {
	derived := Build(bindingsWorld())
	for _, dep := range derived.DeclaredDependencies {
		if dep.FromKind == "application" &&
			string(dep.From) == "order_entry" &&
			dep.Relation == "realizes" &&
			dep.ToKind == "semantic-capability" &&
			string(dep.To) == "security.authenticate" {
			return
		}
	}
	var got []string
	for _, dep := range derived.DeclaredDependencies {
		got = append(got, dep.FromKind+":"+string(dep.From)+"-"+dep.Relation+"->"+string(dep.To))
	}
	t.Fatalf("no realizes edge from order_entry; declared dependencies:\n  %v", got)
}

// The binding is still indexed on its own: adding the edge must not replace the
// adapter/hook record that other callers read.
func TestCustomGoBindingRemainsIndexed(t *testing.T) {
	derived := Build(bindingsWorld())
	if len(derived.CustomGoBindings) != 1 {
		t.Fatalf("CustomGoBindings = %d, want 1", len(derived.CustomGoBindings))
	}
	got := derived.CustomGoBindings[0]
	if got.Application != "order_entry" || got.Capability != "security.authenticate" {
		t.Fatalf("binding record changed: %#v", got)
	}
}

// An empty side would name no node, so it must not produce an edge that the
// projection would only report as unmaterialized.
func TestBindingWithAnEmptySideProducesNoEdge(t *testing.T) {
	world := bindingsWorld()
	world.Bindings = []realization.CustomGoBinding{{Capability: core.MustCanonicalID("security.authenticate")}}
	for _, dep := range Build(world).DeclaredDependencies {
		if dep.Relation == "realizes" {
			t.Fatalf("a binding with no application produced %q -> %q", dep.From, dep.To)
		}
	}
}
