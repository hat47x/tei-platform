package derivedbridge

import (
	"strings"
	"testing"

	"tei/internal/index"
)

// An edge is identified by its endpoints and relation only, so two named
// outcomes that name the same payload become one edge. A caller reading a
// count as a number of business results has to be told that where the count
// is answered, not only in a comment.
func TestProjectionStatesThatSamePayloadOutcomesCollapse(t *testing.T) {
	projection := FromIndex(index.Derived{}, "full")
	joined := strings.Join(projection.AuthorityLimits, "\n")
	const statement = "edge-identity-is-from-relation-to-so-same-payload-outcomes-collapse"
	if !strings.Contains(joined, statement) {
		t.Fatalf("the projection does not state the edge identity limit: %v", projection.AuthorityLimits)
	}
}

// The statement is about edge identity; it must not read as a claim that
// outcomes themselves are lost.
func TestTheEdgeIdentityLimitDoesNotClaimOutcomesAreLost(t *testing.T) {
	projection := FromIndex(index.Derived{}, "full")
	for _, limit := range projection.AuthorityLimits {
		if strings.Contains(limit, "edge-identity") && strings.Contains(limit, "outcomes-lost") {
			t.Errorf("the limit claims outcomes are lost: %s", limit)
		}
	}
}
