package changeanalysis

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func retractionWorld() ir.World {
	return ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Data: []semantic.Data{{
			ID: "sales.line", Symbol: "Line",
			Attributes: []semantic.Attribute{
				{ID: "sales.line.order_id", Symbol: "orderId", TypeRef: "std.identifier"},
				{ID: "sales.line.line_no", Symbol: "lineNo", TypeRef: "std.integer"},
				{ID: "sales.line.item_id", Symbol: "itemId", TypeRef: "std.identifier"},
			},
			CandidateKeys: []semantic.CandidateKey{{
				ID: "sales.line.business_key", Symbol: "businessKey",
				Attributes: []core.CanonicalID{"sales.line.order_id", "sales.line.line_no"},
			}},
			FunctionalDependencies: []semantic.FunctionalDependency{{
				ID: "sales.line.item_by_line", Symbol: "itemByLine",
				Determinants: []core.CanonicalID{"sales.line.order_id", "sales.line.line_no"},
				Dependents:   []core.CanonicalID{"sales.line.item_id"},
			}},
		}},
	}}}
}

func testRuleImpacts(t *testing.T, before, after ir.World, expected []string, wantReviewCount int) {
	t.Helper()
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		t.Fatal(err)
	}
	changes, err := Diff(before, after)
	if err != nil {
		t.Fatal(err)
	}
	edges, unresolved := ImpactWithBefore(before, after, changes)
	found := map[string]bool{}
	for _, e := range edges {
		found[e.From+" -> "+e.To+" via "+e.Via] = true
	}
	for _, want := range expected {
		if !found[want] {
			t.Errorf("missing old/new rule edge %s: %+v", want, edges)
		}
	}
	seenReview := 0
	for _, u := range unresolved {
		if u.Code == "LOGICAL_RULE_RETRACTION_REQUIRES_REVIEW" {
			seenReview++
		}
	}
	if seenReview != wantReviewCount {
		t.Fatalf("expected %d explicit retraction warnings, got %d: %+v", wantReviewCount, seenReview, unresolved)
	}
	again, unAgain := ImpactWithBefore(before, after, changes)
	if !reflect.DeepEqual(edges, again) || !reflect.DeepEqual(unresolved, unAgain) {
		t.Fatal("read-only rule retraction impact was non-deterministic")
	}
	afterBeforeJSON, _ := json.Marshal(before)
	afterAfterJSON, _ := json.Marshal(after)
	if !reflect.DeepEqual(beforeJSON, afterBeforeJSON) || !reflect.DeepEqual(afterJSON, afterAfterJSON) {
		t.Fatal("read-only impact analysis mutated canonical worlds")
	}
}

func TestRuleRetractionPreservesOldEdgesInsteadOfReportingNoImpact(t *testing.T) {
	before := retractionWorld()
	after := retractionWorld()
	after.Semantics[0].Data[0].CandidateKeys = nil
	after.Semantics[0].Data[0].FunctionalDependencies = nil
	testRuleImpacts(t, before, after, []string{
		"candidate-key:sales.line.business_key -> data:sales.line via previous.semantic.data.candidate-key",
		"candidate-key:sales.line.business_key -> attribute:sales.line.order_id via previous.semantic.candidate-key.field",
		"candidate-key:sales.line.business_key -> attribute:sales.line.line_no via previous.semantic.candidate-key.field",
		"functional-dependency:sales.line.item_by_line -> attribute:sales.line.order_id via previous.semantic.functional-dependency.determinant",
		"functional-dependency:sales.line.item_by_line -> attribute:sales.line.item_id via previous.semantic.functional-dependency.dependent",
	}, 2)
	changes, err := Diff(before, after)
	if err != nil || len(changes) != 2 {
		t.Fatalf("missing removal diff: %+v %v", changes, err)
	}
	newEdges, _ := Impact(after, changes)
	for _, e := range newEdges {
		if strings.HasPrefix(e.From, "candidate-key:") || strings.HasPrefix(e.From, "functional-dependency:") {
			t.Fatalf("after-only impact unexpectedly knows old rule: %+v", e)
		}
	}
}

func TestRuleAmendmentShowsBothPreviousAndNewDeterminants(t *testing.T) {
	before := retractionWorld()
	after := retractionWorld()
	after.Semantics[0].Data[0].CandidateKeys = []semantic.CandidateKey{{
		ID: "sales.line.business_key", Symbol: "businessKey",
		Attributes: []core.CanonicalID{"sales.line.item_id"},
	}}
	after.Semantics[0].Data[0].FunctionalDependencies = []semantic.FunctionalDependency{{
		ID: "sales.line.item_by_line", Symbol: "itemByLine",
		Determinants: []core.CanonicalID{"sales.line.item_id"},
		Dependents:   []core.CanonicalID{"sales.line.order_id"},
	}}
	testRuleImpacts(t, before, after, []string{
		"candidate-key:sales.line.business_key -> attribute:sales.line.order_id via previous.semantic.candidate-key.field",
		"candidate-key:sales.line.business_key -> attribute:sales.line.item_id via semantic.candidate-key.field",
		"functional-dependency:sales.line.item_by_line -> attribute:sales.line.item_id via previous.semantic.functional-dependency.dependent",
		"functional-dependency:sales.line.item_by_line -> attribute:sales.line.order_id via semantic.functional-dependency.dependent",
	}, 2)
}

func TestNewKeyAddsNoHistoricalRetractionWarning(t *testing.T) {
	before := retractionWorld()
	before.Semantics[0].Data[0].CandidateKeys = nil
	before.Semantics[0].Data[0].FunctionalDependencies = nil
	after := retractionWorld()
	testRuleImpacts(t, before, after, []string{
		"candidate-key:sales.line.business_key -> data:sales.line via semantic.data.candidate-key",
		"functional-dependency:sales.line.item_by_line -> attribute:sales.line.item_id via semantic.functional-dependency.dependent",
	}, 0)
}

func TestUnverifiableOldRuleIdentityNeverBecomesSilentNoImpact(t *testing.T) {
	world := retractionWorld()
	edges, unresolved := ImpactWithBefore(ir.World{}, world, []SubjectChange{{
		Kind: "candidate-key", ID: "sales.line.unseen", Change: "removed",
	}})
	if len(edges) != 0 {
		t.Fatalf("fabricated old linkage: %+v", edges)
	}
	seenMissing, seenReview := false, false
	for _, u := range unresolved {
		if u.Code == "LOGICAL_RULE_PREVIOUS_DECLARATION_MISSING" {
			seenMissing = true
		}
		if u.Code == "LOGICAL_RULE_RETRACTION_REQUIRES_REVIEW" {
			seenReview = true
		}
	}
	if !seenMissing || !seenReview {
		t.Fatalf("old rule identity silently lost: %+v", unresolved)
	}
}
