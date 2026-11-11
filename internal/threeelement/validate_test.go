package threeelement

import (
	"testing"
)

func TestValidateModelAggregatesDiagnostics(t *testing.T) {
	model := NewThreeElementModel("probe", "Probe")
	model.DataElements = []DataElement{
		{ID: "order", Name: "Order", Attributes: []DataAttribute{{ID: "orderId"}}, Relations: []DataRelation{{ID: "r1", TargetID: "missing"}}},
	}
	report := ValidateModel(model)
	if !report.OK {
		t.Fatalf("expected validation to pass with warnings, got errors: %+v", report.Diagnostics)
	}
	if report.WarningCount == 0 {
		t.Fatal("expected at least one warning")
	}
	if len(report.UnusedData) != 1 || report.UnusedData[0] != "order" {
		t.Fatalf("expected unused data order, got %v", report.UnusedData)
	}
}

func TestContradictoryCRUDs(t *testing.T) {
	model := NewThreeElementModel("probe", "Probe")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit"}}
	model.CRUDDefinitions = []CRUDDefinition{
		{ID: "c1", FunctionID: "submit", DataID: "order", Create: true, Read: false},
		{ID: "c2", FunctionID: "submit", DataID: "order", Create: false, Read: true},
	}
	report := ValidateModel(model)
	if len(report.ContradictoryCRUDs) == 0 {
		t.Fatal("expected contradictory CRUDs")
	}
}
