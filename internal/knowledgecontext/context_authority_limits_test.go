package knowledgecontext

import (
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

// A caller reading `outgoing` as a number of business results is wrong: two
// named outcomes that name the same payload are one edge. The context has to
// say that where it answers the count.
func TestContextStatesThatSamePayloadOutcomesCollapse(t *testing.T) {
	ctx := Build(ir.World{}, sourceindex.Index{}, subject.Subject{Kind: "semantic-capability", ID: "equip.loan_register"})
	joined := strings.Join(ctx.AuthorityLimits, "\n")
	const statement = "edge-identity-is-from-relation-to-so-same-payload-outcomes-collapse"
	if !strings.Contains(joined, statement) {
		t.Fatalf("the context does not state the edge identity limit: %v", ctx.AuthorityLimits)
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
