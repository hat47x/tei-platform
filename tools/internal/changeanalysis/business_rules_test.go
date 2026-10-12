package changeanalysis

import (
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func TestLogicalKeysAndFDsBecomeIndependentSemanticDiffSubjects(t *testing.T) {
	before := ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Data: []semantic.Data{{
			ID: "sales.order_line", Symbol: "OrderLine",
			Attributes: []semantic.Attribute{
				{ID: "sales.order_line.order_line_id", Symbol: "orderLineId", TypeRef: "std.identifier"},
				{ID: "sales.order_line.line_number", Symbol: "lineNumber", TypeRef: "std.integer"},
				{ID: "sales.order_line.item_code", Symbol: "itemCode", TypeRef: "std.code"},
			},
		}},
	}}}
	after := ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Data: []semantic.Data{{
			ID: "sales.order_line", Symbol: "OrderLine",
			Attributes: before.Semantics[0].Data[0].Attributes,
			CandidateKeys: []semantic.CandidateKey{{
				ID: "sales.order_line.business_line", Symbol: "businessLine",
				Attributes: []core.CanonicalID{"sales.order_line.order_line_id", "sales.order_line.line_number"},
			}},
			FunctionalDependencies: []semantic.FunctionalDependency{{
				ID: "sales.order_line.item_by_line", Symbol: "itemByLine",
				Determinants: []core.CanonicalID{"sales.order_line.order_line_id", "sales.order_line.line_number"},
				Dependents: []core.CanonicalID{"sales.order_line.item_code"},
			}},
		}},
	}}}
	changes, err := Diff(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 {
		t.Fatalf("expected exactly two typed rule changes: %+v", changes)
	}
	if got, want := []string{
		changes[0].Change + " " + changes[0].Key(),
		changes[1].Change + " " + changes[1].Key(),
	}, []string{
		"added candidate-key:sales.order_line.business_line",
		"added functional-dependency:sales.order_line.item_by_line",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("diff=%+v", changes)
	}
	edges, unresolved := Impact(after, changes)
	want := []string{
		"candidate-key:sales.order_line.business_line -> data:sales.order_line",
		"functional-dependency:sales.order_line.item_by_line -> attribute:sales.order_line.item_code",
	}
	for _, expected := range want {
		found := false
		for _, edge := range edges {
			if edge.From+" -> "+edge.To == expected {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing logical impact %s: %+v", expected, edges)
		}
	}
	foundUnverified := false
	for _, item := range unresolved {
		if item.Code == "LOGICAL_BUSINESS_RULE_UNVERIFIED" {
			foundUnverified = true
		}
	}
	if !foundUnverified {
		t.Fatalf("logic was falsely reported as verified: %+v", unresolved)
	}
	missingFD := after
	missingFD.Semantics = append([]semantic.Namespace(nil), after.Semantics...)
	missingFD.Semantics[0].Data = append([]semantic.Data(nil), after.Semantics[0].Data...)
	missingFD.Semantics[0].Data[0].FunctionalDependencies = nil
	removed, err := Diff(after, missingFD)
	if err != nil || len(removed) != 1 || removed[0].Change != "removed" ||
		removed[0].Key() != "functional-dependency:sales.order_line.item_by_line" {
		t.Fatalf("FD deletion not visible: %+v err=%v", removed, err)
	}
}
