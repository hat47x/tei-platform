package modelref_test

import (
	"testing"

	"tei/internal/modelref"
)

// A label-only change must stop when an upstream edge does not depend on labels.
func TestDisplayRenameDoesNotLeakAcrossUnrelatedHop(t *testing.T) {
	before := fixture()
	before.Nodes[1].DisplayName = "Submit"
	after := fixture()
	after.Nodes[1].DisplayName = "Send"
	changes := modelref.Diff(before, after)
	affected := modelref.Impact(before, changes, rules())
	if len(affected) != 1 || affected[0].Consumer != key("form", "order-form") {
		t.Fatalf("label impact traversed non-display triggers edge: %+v", affected)
	}
}

// The source and derivation fingerprints must invalidate derived edges even
// if no logical key/version or relation type has changed.
func TestEdgeEvidenceFreshnessIsAChange(t *testing.T) {
	before := fixture()
	before.Edges[0].Evidence = []modelref.Evidence{{Authority: "derived", ArtifactFingerprint: "sha256:before", RuleFingerprint: "sha256:rule-a"}}
	after := fixture()
	after.Edges[0].Evidence = []modelref.Evidence{{Authority: "derived", ArtifactFingerprint: "sha256:after", RuleFingerprint: "sha256:rule-a"}}
	changes := modelref.Diff(before, after)
	if len(changes) != 1 || changes[0].Kind != "reference-evidence-changed" || changes[0].Key != key("function", "submit-order") {
		t.Fatalf("edge proof staleness not reported: %+v", changes)
	}
	affected := modelref.Impact(before, changes, rules())
	if len(affected) != 2 {
		t.Fatalf("downstream consumers should be reviewed: %+v", affected)
	}
	// Unordered evidence observations do not represent a semantic difference.
	before.Edges[0].Evidence = []modelref.Evidence{
		{Authority: "source-observation", SourceToken: "a", ArtifactFingerprint: "sha256:one"},
		{Authority: "source-observation", SourceToken: "b", ArtifactFingerprint: "sha256:one"},
	}
	after.Edges[0].Evidence = []modelref.Evidence{before.Edges[0].Evidence[1], before.Edges[0].Evidence[0]}
	if changed := modelref.Diff(before, after); len(changed) != 0 {
		t.Fatalf("evidence ordering cannot invalidate graph: %+v", changed)
	}
}

// A graph with a missing or ambiguous target must not invent extra impacts.
func TestImpactFailsClosedOnUnresolvedAndUnknownRelations(t *testing.T) {
	before := fixture()
	before.Edges[0].To.Subject.ID = "missing"
	change := modelref.Change{Key: key("data", "order-line"), Kind: "removed"}
	if got := modelref.Impact(before, []modelref.Change{change}, rules()); len(got) != 0 {
		t.Fatalf("missing reference created phantom consumers: %+v", got)
	}
	before = fixture()
	before.Edges[0].Relation = "unregistered-kind"
	if got := modelref.Impact(before, []modelref.Change{change}, rules()); len(got) != 0 {
		t.Fatalf("unregistered relation created phantom consumers: %+v", got)
	}
}
