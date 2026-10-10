package modelref_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"tei/internal/modelref"
)

func rekeyFixture() (modelref.Graph, modelref.Graph, modelref.Transition, modelref.EdgeRetarget) {
	before := fixture()
	after := fixture()
	after.Nodes[0].Key = key("data", "order-detail")
	after.Edges[0].To = ref(key("data", "order-detail"), "1")
	transition := modelref.Transition{Kind: "rekey", From: []modelref.Key{key("data", "order-line")}, To: []modelref.Key{key("data", "order-detail")}}
	move := modelref.EdgeRetarget{From: key("function", "submit-order"), Relation: "reads", Old: ref(key("data", "order-line"), "1"), New: ref(key("data", "order-detail"), "1")}
	return before, after, transition, move
}
func TestRekeyReviewExplicitlyPreservesThreeViewReferences(t *testing.T) {
	before, after, transition, move := rekeyFixture()
	previous, _ := json.Marshal(before)
	out := modelref.ReviewMigration(before, after, transition, []modelref.EdgeRetarget{move}, rules(), "sha256:base")
	if !out.Consistent || len(out.Diagnostics) != 0 {
		t.Fatalf("valid rekey review: %+v", out.Diagnostics)
	}
	if len(out.Affected) != 5 {
		t.Fatalf("missing downstream Business/Form consumers: %+v", out.Affected)
	}
	now, _ := json.Marshal(before)
	if !reflect.DeepEqual(previous, now) {
		t.Fatal("preview mutated the accepted baseline")
	}
	if out.RuleFingerprint != modelref.RuleFingerprint(rules()) {
		t.Fatal("lost policy fingerprint")
	}
}
func TestRekeyMissingExplicitRetargetFailsClosed(t *testing.T) {
	before, after, transition, move := rekeyFixture()
	for _, tc := range []struct {
		name     string
		moves    []modelref.EdgeRetarget
		baseline string
		want     string
	}{
		{"no-move", nil, "sha256:base", "unreviewed-incoming-reference"},
		{"no-base", []modelref.EdgeRetarget{move}, "", "unbound-baseline-revision"},
		{"double-move", []modelref.EdgeRetarget{move, move}, "sha256:base", "duplicate-retarget"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			review := modelref.ReviewMigration(before, after, transition, tc.moves, rules(), tc.baseline)
			if review.Consistent || !hasCode(review.Diagnostics, tc.want) {
				t.Fatalf("expected %s, got %+v", tc.want, review.Diagnostics)
			}
		})
	}
}
func TestSplitRequiresIndividualConsumerChoices(t *testing.T) {
	before, after, transition, move := rekeyFixture()
	transition.Kind = "split"
	transition.To = append(transition.To, key("data", "order-history"))
	after.Nodes = append(after.Nodes, node("data", "order-history", "1"))
	yes := modelref.ReviewMigration(before, after, transition, []modelref.EdgeRetarget{move}, rules(), "revision-1")
	if !yes.Consistent {
		t.Fatalf("explicit split candidate: %+v", yes.Diagnostics)
	}
	no := modelref.ReviewMigration(before, after, transition, nil, rules(), "revision-1")
	if no.Consistent || !hasCode(no.Diagnostics, "unreviewed-incoming-reference") {
		t.Fatalf("split inferred target without review: %+v", no.Diagnostics)
	}
}
func TestUnrelatedEdgeRemovalIsNotHiddenByRekey(t *testing.T) {
	before, after, transition, move := rekeyFixture()
	after.Edges = after.Edges[:2] // unrelated form->function edge silently lost
	out := modelref.ReviewMigration(before, after, transition, []modelref.EdgeRetarget{move}, rules(), "revision-1")
	if out.Consistent || !hasCode(out.Diagnostics, "unrelated-reference-changed") {
		t.Fatalf("hidden unrelated edge change: %+v", out.Diagnostics)
	}
}
