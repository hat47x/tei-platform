package threeelement

import (
	"testing"
)

func TestModelFingerprintIsStable(t *testing.T) {
	model := NewThreeElementModel("order-system", "Order system")
	first := ModelFingerprint(model)
	if first == "" {
		t.Fatal("expected non-empty fingerprint")
	}
	if got := ModelFingerprint(NewThreeElementModel("order-system", "Order system")); got != first {
		t.Fatalf("fingerprint changed for identical model: %s != %s", got, first)
	}
}

func TestValidateDetectsUnknownRelations(t *testing.T) {
	model := NewThreeElementModel("probe", "Probe")
	model.DataElements = []DataElement{
		{ID: "order", Name: "Order", Attributes: []DataAttribute{{ID: "orderId", Name: "Order ID", Type: "text"}}, Relations: []DataRelation{{ID: "r1", TargetID: "missing"}}},
	}
	diags := model.Validate()
	if len(diags) == 0 {
		t.Fatal("expected diagnostics for unknown relation target")
	}
}

func TestUnusedDataReportsUnreferencedData(t *testing.T) {
	model := NewThreeElementModel("probe", "Probe")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit", Operations: []FunctionOperation{{DataID: "order", Operations: "CRUD"}}}}
	if got := model.UnusedData(); len(got) != 0 {
		t.Fatalf("expected no unused data, got %v", got)
	}
}

func TestSameNameConflicts(t *testing.T) {
	model := NewThreeElementModel("probe", "Probe")
	model.DataElements = []DataElement{{ID: "order1", Name: "Order"}, {ID: "order2", Name: "Order"}}
	conflicts := model.SameNameConflicts()
	if len(conflicts) == 0 {
		t.Fatal("expected same-name conflicts")
	}
}
