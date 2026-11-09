package modelref_test

import (
	"testing"

	"tei/internal/modelref"
)

func TestBeforeOnlyImpactCannotCertifyNewNode(t *testing.T) {
	changes := []modelref.Change{{Key: key("data", "new-order"), Kind: "added"}}
	review := modelref.ReviewImpact(fixture(), changes, rules())
	if review.GraphValid || !hasCode(review.Diagnostics, "after-snapshot-required") || len(review.Affected) != 0 {
		t.Fatalf("before-only review falsely certified new subject: %+v", review)
	}
}

func TestImpactDiffAccountsForNewConsumerReferences(t *testing.T) {
	before, after := fixture(), fixture()
	newData := node("data", "order-audit", "1")
	after.Nodes = append(after.Nodes, newData)
	after.Edges = append(after.Edges, modelref.Edge{
		From: key("function", "submit-order"),
		To:   ref(newData.Key, "1"), Relation: "reads",
	})
	review := modelref.ReviewImpactDiff(before, after, rules())
	if !review.GraphValid || len(review.Diagnostics) > 0 || len(review.Changes) == 0 {
		t.Fatalf("valid two-snapshot review failed: %+v", review)
	}
	seen := map[modelref.Key]bool{}
	for _, candidate := range review.Affected {
		if candidate.Changed == newData.Key && candidate.Reason == "added" {
			seen[candidate.Consumer] = true
		}
	}
	for _, expected := range []modelref.Key{
		key("function", "submit-order"), key("task", "confirm-order"), key("form", "order-form"),
	} {
		if !seen[expected] {
			t.Fatalf("new subject lost consumer %+v: %+v", expected, review.Affected)
		}
	}
	if len(seen) != 3 {
		t.Fatalf("duplicate or invented consumers for added subject: %+v", review.Affected)
	}
}

func TestImpactDiffChecksBothSnapshotsAndDoesNotInferRenames(t *testing.T) {
	before, after := fixture(), fixture()
	after.Nodes[0].Key = key("data", "order-detail")
	after.Edges[0].To = ref(key("data", "order-detail"), "1")
	review := modelref.ReviewImpactDiff(before, after, rules())
	if !review.GraphValid || len(review.Changes) == 0 {
		t.Fatalf("explicit key replacement not described: %+v", review)
	}
	added, removed := false, false
	for _, change := range review.Changes {
		if change.Kind == "added" && change.Key == key("data", "order-detail") {
			added = true
		}
		if change.Kind == "removed" && change.Key == key("data", "order-line") {
			removed = true
		}
	}
	if !added || !removed {
		t.Fatalf("implicit rename concealed add/remove semantics: %+v", review.Changes)
	}
	after.Edges[0].To.Subject.ID = "stale"
	review = modelref.ReviewImpactDiff(before, after, rules())
	if review.GraphValid || !hasCode(review.Diagnostics, "missing-reference") || len(review.Affected) != 0 {
		t.Fatalf("broken after graph was promoted to complete impact: %+v", review)
	}
}

func TestImpactDiffIncludesNewIncomingEdgesOnStableChangedIdentity(t *testing.T) {
	before, after := fixture(), fixture()
	after.Nodes[0].Version = "2"
	after.Edges[0].To.Version = "2"
	after.Nodes = append(after.Nodes, node("function", "audit", "1"))
	after.Edges = append(after.Edges, modelref.Edge{
		From: key("function", "audit"), To: ref(key("data", "order-line"), "2"), Relation: "reads",
	})
	review := modelref.ReviewImpactDiff(before, after, rules())
	if !review.GraphValid {
		t.Fatalf("valid after graph rejected: %+v", review.Diagnostics)
	}
	found := false
	for _, impact := range review.Affected {
		if impact.Changed == key("data", "order-line") && impact.Consumer == key("function", "audit") && impact.Reason == "version-changed" {
			found = true
		}
	}
	if !found {
		t.Fatalf("new consumer omitted from changed identity: %+v", review.Affected)
	}
}
