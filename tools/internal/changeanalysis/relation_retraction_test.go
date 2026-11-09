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

// worldWithoutRelationID returns the same Reference Workspace twice, with the
// named Relation removed from the after World only. This is the in-memory shape
// of a Relation deletion or of the removal half of a rename / split.
func worldWithoutRelationID(t *testing.T, id core.CanonicalID) (before, after ir.World) {
	t.Helper()
	before = loadWorld(t, referenceWorkspaceCopy(t)).World
	after = loadWorld(t, referenceWorkspaceCopy(t)).World
	removed := false
	for i := range after.Semantics {
		for d := range after.Semantics[i].Data {
			var kept []semantic.Relation
			for _, r := range after.Semantics[i].Data[d].Relations {
				if r.ID == id {
					removed = true
					continue
				}
				kept = append(kept, r)
			}
			after.Semantics[i].Data[d].Relations = kept
		}
	}
	if !removed {
		t.Fatalf("relation %s was not found to remove", id)
	}
	return before, after
}

func assertRelationImpactReadOnly(t *testing.T, before, after ir.World, changes []SubjectChange, edges []ImpactEdge, unresolved []Unresolved) {
	t.Helper()
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		t.Fatal(err)
	}
	again, unAgain := ImpactWithBefore(before, after, changes)
	if !reflect.DeepEqual(edges, again) || !reflect.DeepEqual(unresolved, unAgain) {
		t.Fatal("read-only Relation impact analysis was non-deterministic")
	}
	beforeAgain, _ := json.Marshal(before)
	afterAgain, _ := json.Marshal(after)
	if !reflect.DeepEqual(beforeJSON, beforeAgain) || !reflect.DeepEqual(afterJSON, afterAgain) {
		t.Fatal("read-only Relation impact analysis mutated canonical worlds")
	}
}

func TestRemovedRelationKeepsPreviousEdgesInsteadOfReportingNoImpact(t *testing.T) {
	before, after := worldWithoutRelationID(t, "sales.order.lines")
	changes, err := Diff(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := keys(changes), []string{"removed relation:sales.order.lines"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("diff=%v want %v", got, want)
	}

	afterOnly, _ := Impact(after, changes)
	for _, e := range afterOnly {
		if strings.HasPrefix(e.From, "relation:") {
			t.Fatalf("after-only impact unexpectedly knows the removed Relation: %+v", e)
		}
	}

	edges, unresolved := ImpactWithBefore(before, after, changes)
	got := viaSet(edges)
	for _, want := range []string{
		"relation:sales.order.lines -> data:sales.order via previous.semantic.data.relation",
		"relation:sales.order.lines -> data:sales.order_line via previous.semantic.relation.target",
		"relation:sales.order.lines -> persistence-mapping:sales_core.order_line via previous.persistence.relation-storage",
		"relation:sales.order.lines -> application-pattern:order_entry.order_lines via previous.application.pattern.relation",
	} {
		if !got[want] {
			t.Fatalf("missing previous Relation edge %q: %+v", want, edges)
		}
	}

	byCode := unresolvedByCode(unresolved)
	if _, ok := byCode["RELATION_RETRACTION_REQUIRES_REVIEW"]; !ok {
		t.Fatalf("removed Relation was not flagged for human review: %+v", unresolved)
	}
	if _, ok := byCode["RELATION_PREVIOUS_DECLARATION_MISSING"]; ok {
		t.Fatalf("a resolvable previous Relation was reported as missing: %+v", unresolved)
	}
	assertRelationImpactReadOnly(t, before, after, changes, edges, unresolved)
}

func TestChangedRelationShowsBothPreviousAndCurrentEndpoints(t *testing.T) {
	before := loadWorld(t, referenceWorkspaceCopy(t)).World
	after := loadWorld(t, referenceWorkspaceCopy(t)).World
	changed := false
	for i := range after.Semantics {
		for d := range after.Semantics[i].Data {
			for r := range after.Semantics[i].Data[d].Relations {
				if after.Semantics[i].Data[d].Relations[r].ID == "sales.order.lines" {
					after.Semantics[i].Data[d].Relations[r].Cardinality = semantic.CardinalityOne
					changed = true
				}
			}
		}
	}
	if !changed {
		t.Fatal("relation sales.order.lines was not found to change")
	}

	changes, err := Diff(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := keys(changes), []string{"changed relation:sales.order.lines"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("diff=%v want %v", got, want)
	}

	edges, unresolved := ImpactWithBefore(before, after, changes)
	got := viaSet(edges)
	if !got["relation:sales.order.lines -> data:sales.order_line via semantic.relation.target"] {
		t.Fatalf("current Relation edge missing after the change: %+v", edges)
	}
	if !got["relation:sales.order.lines -> data:sales.order_line via previous.semantic.relation.target"] {
		t.Fatalf("previous Relation edge missing after the change: %+v", edges)
	}
	if _, ok := unresolvedByCode(unresolved)["RELATION_RETRACTION_REQUIRES_REVIEW"]; !ok {
		t.Fatalf("changed Relation was not flagged for review: %+v", unresolved)
	}
	assertRelationImpactReadOnly(t, before, after, changes, edges, unresolved)
}

func TestUnverifiableOldRelationIdentityNeverBecomesSilentNoImpact(t *testing.T) {
	world := loadWorld(t, referenceWorkspaceCopy(t)).World
	edges, unresolved := ImpactWithBefore(ir.World{}, world, []SubjectChange{{
		Kind: "relation", ID: "sales.order.unseen", Change: "removed",
	}})
	if len(edges) != 0 {
		t.Fatalf("fabricated previous Relation linkage: %+v", edges)
	}
	byCode := unresolvedByCode(unresolved)
	if _, ok := byCode["RELATION_PREVIOUS_DECLARATION_MISSING"]; !ok {
		t.Fatalf("unresolvable previous Relation was not reported: %+v", unresolved)
	}
	if _, ok := byCode["RELATION_RETRACTION_REQUIRES_REVIEW"]; !ok {
		t.Fatalf("removed Relation was not flagged for review: %+v", unresolved)
	}
}
