package typereorg

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

func splitWorld() ir.World {
	return ir.World{Semantics: []semantic.Namespace{{ID: "sales", Types: []semantic.Type{
		{ID: "sales.order_id", Symbol: "orderID", Kind: "alias", Underlying: "std.identifier"},
		{ID: "sales.order_status", Symbol: "orderStatus", Kind: "alias", Underlying: "std.code"},
	}, Data: []semantic.Data{
		{ID: "sales.order", Symbol: "Order", Attributes: []semantic.Attribute{
			{ID: "sales.order.id", Symbol: "id", TypeRef: "sales.order_id"},
			{ID: "sales.order.status", Symbol: "status", TypeRef: "sales.order_status"},
		}},
	}}}}
}

func TestSplitImpactListsUsesAndFlagsUnassigned(t *testing.T) {
	w := splitWorld()
	imp, err := SplitImpact(w, Split{From: "sales.order_id", Into: []string{"sales.order_status"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(imp.Attributes) != 1 || imp.Attributes[0].Attribute != "sales.order.id" {
		t.Fatalf("attributes = %+v", imp.Attributes)
	}
	// unassigned attribute -> diagnostic
	found := false
	for _, d := range imp.Diagnostics {
		if d.Code == "split-attribute-unassigned" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected split-attribute-unassigned diagnostic, got %+v", imp.Diagnostics)
	}
}

func TestSplitImpactWithReassignment(t *testing.T) {
	w := splitWorld()
	imp, err := SplitImpact(w, Split{
		From:         "sales.order_id",
		Into:         []string{"sales.order_status"},
		Reassignment: map[string]string{"sales.order.id": "sales.order_status"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(imp.Attributes) != 1 || imp.Attributes[0].Target != "sales.order_status" {
		t.Fatalf("attributes = %+v", imp.Attributes)
	}
	for _, d := range imp.Diagnostics {
		if d.Code == "split-attribute-unassigned" {
			t.Fatalf("unexpected unassigned diagnostic: %+v", d)
		}
	}
}

func TestSplitImpactUnknownTarget(t *testing.T) {
	w := splitWorld()
	imp, err := SplitImpact(w, Split{From: "sales.order_id", Into: []string{"sales.missing"}})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range imp.Diagnostics {
		if d.Code == "unknown-target-type" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected unknown-target-type diagnostic, got %+v", imp.Diagnostics)
	}
}
