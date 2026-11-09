package typereorg

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

func reprojectionWorld() ir.World {
	w := world()
	w.Semantics[0].Validations = []semantic.Validation{
		{ID: "s.order.cust.valid", Symbol: "custValid", Attribute: "s.order.cust"},
	}
	w.Semantics[0].Capabilities = []semantic.Capability{
		{ID: "s.cap", Symbol: "cap", Inputs: []semantic.Parameter{{Name: "cust", Type: "s.cust_code"}}},
	}
	return w
}

func TestMergeReprojectionCoversEveryView(t *testing.T) {
	imp, err := Analyze(reprojectionWorld(), Merge{From: []string{"s.cust_code"}, Into: "s.client_code"})
	if err != nil {
		t.Fatal(err)
	}
	views := map[string]bool{}
	for _, entry := range imp.Reprojection {
		views[entry.View] = true
		if entry.TypeAfter != "s.client_code" {
			t.Errorf("%s/%s: after = %q, want the target type", entry.View, entry.Element, entry.TypeAfter)
		}
	}
	for _, want := range []string{"attribute", "validation", "capability", "key", "persistence"} {
		if !views[want] {
			t.Errorf("reprojection is missing view %q: %+v", want, imp.Reprojection)
		}
	}
	if !imp.ConsistentAfter {
		t.Error("every affected view should agree on the target type")
	}
	if imp.Applied || !imp.ReviewRequired {
		t.Errorf("reprojection must stay a reviewable plan: applied=%v review=%v", imp.Applied, imp.ReviewRequired)
	}
}

func TestMergeReprojectionUnknownTargetIsInconsistent(t *testing.T) {
	imp, err := Analyze(world(), Merge{From: []string{"s.cust_code"}, Into: "s.unknown"})
	if err != nil {
		t.Fatal(err)
	}
	if imp.ConsistentAfter {
		t.Error("an unknown target type cannot make every view agree")
	}
	if !codes(imp.Diagnostics)["unknown-target-type"] {
		t.Fatalf("diagnostics = %+v", imp.Diagnostics)
	}
}

func TestSplitReprojectionUsesReassignment(t *testing.T) {
	imp, err := SplitImpact(world(), Split{
		From:         "s.cust_code",
		Into:         []string{"s.client_code", "s.item_code"},
		Reassignment: map[string]string{"s.order.cust": "s.client_code"},
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range imp.Reprojection {
		if entry.View == "attribute" && entry.Element == "s.order.cust" {
			found = true
			if entry.TypeAfter != "s.client_code" {
				t.Errorf("attribute after = %q, want s.client_code", entry.TypeAfter)
			}
		}
		if entry.TypeAfter == "" {
			t.Errorf("unresolved reprojection entry: %+v", entry)
		}
	}
	if !found {
		t.Fatalf("reprojection is missing s.order.cust: %+v", imp.Reprojection)
	}
	if !imp.ConsistentAfter {
		t.Error("every reassigned view should agree on a target type")
	}
}

func TestSplitReprojectionUnassignedIsInconsistent(t *testing.T) {
	imp, err := SplitImpact(world(), Split{From: "s.cust_code", Into: []string{"s.client_code"}})
	if err != nil {
		t.Fatal(err)
	}
	if imp.ConsistentAfter {
		t.Error("an unassigned attribute cannot make every view agree")
	}
	if !codes(imp.Diagnostics)["split-attribute-unassigned"] {
		t.Fatalf("diagnostics = %+v", imp.Diagnostics)
	}
}
