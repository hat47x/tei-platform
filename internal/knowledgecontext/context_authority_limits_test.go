package knowledgecontext

import (
	"strings"
	"testing"

	"tei/internal/index"
	"tei/internal/ir"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

// Compare both projections of the same declaration, not merely a string
// emitted for an empty world. The original #5426 case has two named
// outcomes with the same payload and one input referencing that payload.
func TestContextRetainsSamePayloadOutcomeOccurrences(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "equip",
		Data: []semantic.Data{{ID: "equip.loan", Symbol: "Loan"}},
		Capabilities: []semantic.Capability{{
			ID: "equip.loan_register", Symbol: "register",
			Inputs: []semantic.Parameter{{Name: "loan", Type: "equip.loan"}},
			Outcomes: []semantic.Outcome{
				{Name: "created", Payload: "equip.loan"},
				{Name: "overlapping", Payload: "equip.loan"},
			},
		}},
	}}}
	target := subject.Subject{Kind: "semantic-capability", ID: "equip.loan_register"}
	for _, n := range []int{2, 1} {
		world.Semantics[0].Capabilities[0].Outcomes = world.Semantics[0].Capabilities[0].Outcomes[:n]
		ctx := Build(world, sourceindex.Index{}, target)
		joined := strings.Join(ctx.AuthorityLimits, "\n")
		if !strings.Contains(joined, "dependency-occurrences-are-retained-without-outcome-names") ||
			strings.Contains(joined, "same-payload-outcomes-collapse") {
			t.Fatalf("context misstates dependency occurrences: %v", ctx.AuthorityLimits)
		}
		var inputs, outcomes int
		for _, edge := range ctx.Outgoing {
			if edge.To.ID != "equip.loan" {
				t.Fatalf("unexpected dependency: %+v", edge)
			}
			switch edge.Relation {
			case "input":
				inputs++
			case "outcome":
				outcomes++
			}
		}
		if len(ctx.Outgoing) != n+1 || inputs != 1 || outcomes != n {
			t.Fatalf("%d declared outcomes: context has %d edges, %d inputs, %d outcomes", n, len(ctx.Outgoing), inputs, outcomes)
		}
		projection := derivedbridge.FromIndex(index.Build(world), "test")
		var projectedInputs, projectedOutcomes int
		for _, edge := range projection.Graph.Edges {
			if edge.From.Subject != target {
				continue
			}
			switch edge.Relation {
			case "input":
				projectedInputs++
			case "outcome":
				projectedOutcomes++
			}
		}
		if projectedInputs != 1 || projectedOutcomes != 1 {
			t.Fatalf("%d declared outcomes: trace projection has %d inputs and %d outcomes", n, projectedInputs, projectedOutcomes)
		}
		if !strings.Contains(strings.Join(projection.AuthorityLimits, "\n"), "edge-identity-is-from-relation-to-so-same-payload-outcomes-collapse") {
			t.Fatalf("trace projection omits its edge identity limit: %v", projection.AuthorityLimits)
		}
	}
}

// The four limits that were already there must stay, so an addition does not
// silently replace an existing statement.
func TestTheExistingContextLimitsAreKept(t *testing.T) {
	ctx := Build(ir.World{}, sourceindex.Index{}, subject.Subject{Kind: "semantic-capability", ID: "x"})
	for _, want := range []string{
		"derived-context-is-not-canonical-authority",
		"dependency-edge-is-not-semantic-applicability",
		"dependency-edge-is-not-write-or-runtime-authority",
		"projection-or-runtime-identity-is-not-business-data-identity",
	} {
		if !strings.Contains(strings.Join(ctx.AuthorityLimits, "\n"), want) {
			t.Errorf("the existing limit %q was dropped: %v", want, ctx.AuthorityLimits)
		}
	}
}
