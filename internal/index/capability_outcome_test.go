package index

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// capabilityWorld builds a World with two capabilities where the first declares
// an outcome that the second takes as an input.
func capabilityWorld() ir.World {
	return ir.World{Semantics: []semantic.Namespace{{
		ID: core.MustCanonicalID("security"),
		Types: []semantic.Type{{
			ID: core.MustCanonicalID("security.authenticated_subject"), Symbol: "AuthenticatedSubject", Kind: semantic.TypePrimitive,
		}},
		Capabilities: []semantic.Capability{
			{
				ID:     core.MustCanonicalID("security.authenticate"),
				Symbol: "authenticate",
				Outcomes: []semantic.Outcome{{
					Name: "authenticated", Payload: core.MustCanonicalID("security.authenticated_subject"),
				}},
			},
			{
				ID:     core.MustCanonicalID("security.authorize"),
				Symbol: "authorize",
				Inputs: []semantic.Parameter{{
					Name: "subject", Type: core.MustCanonicalID("security.authenticated_subject"),
				}},
			},
		},
	}}}
}

// A capability's outcomes are results it produces, and a later capability may
// take one as an input (security.authorize takes security.authenticated_subject,
// which security.authenticate declares as an outcome). The derived graph used to
// omit outcome edges, so the chain looked like a consumer with no producer
// (#3884).
func TestCapabilityOutcomesBecomeEdges(t *testing.T) {
	derived := Build(capabilityWorld())
	var edges []string
	var found bool
	for _, dep := range derived.SemanticDependencies {
		edges = append(edges, string(dep.From)+"-"+dep.Relation+"->"+string(dep.To))
		if dep.FromKind == "semantic-capability" &&
			string(dep.From) == "security.authenticate" &&
			dep.Relation == "outcome" &&
			string(dep.To) == "security.authenticated_subject" {
			found = true
		}
	}
	if !found {
		t.Fatalf("no outcome edge from security.authenticate; derived edges:\n  %s",
			strings.Join(edges, "\n  "))
	}
}

// An outcome with no payload declares nothing to resolve, so it must not
// produce an edge that would then read as unresolved.
func TestOutcomeWithoutPayloadProducesNoEdge(t *testing.T) {
	world := capabilityWorld()
	world.Semantics[0].Capabilities[0].Outcomes = []semantic.Outcome{{Name: "anonymous"}}
	for _, dep := range Build(world).SemanticDependencies {
		if dep.Relation == "outcome" && string(dep.From) == "security.authenticate" {
			t.Fatalf("an outcome with no payload produced %q", dep.To)
		}
	}
}
