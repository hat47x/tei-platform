package modelref_test

import (
	"testing"

	"tei/internal/modelref"
)

// A plugin conflict cannot depend on registration order to promote a
// display-only modification into a downstream Business/Function dependency.
func TestImpactReviewRequiresUnambiguousRules(t *testing.T) {
	before := fixture()
	change := modelref.Change{Key: key("data", "order-line"), Kind: "display-renamed"}
	policy := append(rules(), modelref.Rule{
		Relation: "reads", FromKind: "function", ToKind: "data", DisplayAffects: true,
	})
	if got := modelref.Impact(before, []modelref.Change{change}, policy); len(got) != 0 {
		t.Fatalf("conflicting rule invented consumers: %+v", got)
	}
	review := modelref.ReviewImpact(before, []modelref.Change{change}, policy)
	if review.GraphValid || !hasCode(review.Diagnostics, "conflicting-rule") || len(review.Affected) != 0 {
		t.Fatalf("invalid policy was treated as a complete impact review: %+v", review)
	}
	// Valid rules explicitly do not propagate display labels across reads.
	review = modelref.ReviewImpact(before, []modelref.Change{change}, rules())
	if !review.GraphValid || len(review.Diagnostics) != 0 || len(review.Affected) != 0 {
		t.Fatalf("clean graph should yield a reliable empty candidate list: %+v", review)
	}
}

func TestImpactReviewReportsIncompleteGraphWithoutClaimingNoEffects(t *testing.T) {
	before := fixture()
	change := []modelref.Change{{Key: key("data", "order-line"), Kind: "version-changed"}}
	before.Edges[0].To.Subject.ID = "missing"
	review := modelref.ReviewImpact(before, change, rules())
	if review.GraphValid || !hasCode(review.Diagnostics, "missing-reference") || len(review.Affected) != 0 {
		t.Fatalf("broken graph hid unresolved references: %+v", review)
	}
	before = fixture()
	before.Edges[0].Evidence = []modelref.Evidence{{Authority: "derived", ArtifactFingerprint: "sha256:source"}}
	review = modelref.ReviewImpact(before, change, rules())
	if review.GraphValid || !hasCode(review.Diagnostics, "unbound-derivation-evidence") {
		t.Fatalf("unverified provenance promoted: %+v", review)
	}
	if got := modelref.Impact(before, change, rules()); len(got) != 0 {
		t.Fatalf("malformed evidence was traversed: %+v", got)
	}
}

func TestImpactReviewKeepsTransitiveCandidatesAndGuardsChangeKind(t *testing.T) {
	before := fixture()
	change := []modelref.Change{{Key: key("data", "order-line"), Kind: "version-changed"}}
	review := modelref.ReviewImpact(before, change, rules())
	if !review.GraphValid || len(review.Affected) != 3 || review.RuleFingerprint != modelref.RuleFingerprint(rules()) {
		t.Fatalf("valid three-view graph lost impact candidates: %+v", review)
	}
	change[0].Kind = "future-unknown-change"
	review = modelref.ReviewImpact(before, change, rules())
	if review.GraphValid || !hasCode(review.Diagnostics, "unknown-change-kind") || len(review.Affected) != 0 {
		t.Fatalf("unknown change was silently accepted: %+v", review)
	}
	change[0] = modelref.Change{Key: key("data", "absent"), Kind: "removed"}
	review = modelref.ReviewImpact(before, change, rules())
	if review.GraphValid || !hasCode(review.Diagnostics, "unresolved-change-key") || len(review.Affected) != 0 {
		t.Fatalf("unbound changed identity was silently accepted: %+v", review)
	}
}
