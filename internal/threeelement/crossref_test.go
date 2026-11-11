package threeelement

import (
	"testing"
)

func TestGenerateBusinessDataCrossReference(t *testing.T) {
	model := NewThreeElementModel("probe", "Probe")
	model.BusinessElements = []BusinessElement{{ID: "sales", Name: "Sales", Activities: []BusinessActivity{{ID: "a1", Name: "Enter order", FunctionIDs: []string{"submit"}}}}}
	model.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit", Operations: []FunctionOperation{{DataID: "order", Operations: "R"}}}}
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	rows := GenerateBusinessDataCrossReference(model)
	if len(rows) != 1 || rows[0].DataID != "order" {
		t.Fatalf("expected business->data mapping via function, got %+v", rows)
	}
}

func TestGenerateDataDataCrossReference(t *testing.T) {
	model := NewThreeElementModel("probe", "Probe")
	model.DataElements = []DataElement{
		{ID: "order", Name: "Order"},
		{ID: "customer", Name: "Customer", Relations: []DataRelation{{ID: "r1", TargetID: "order", Kind: "one-to-many"}}},
	}
	rows := GenerateDataDataCrossReference(model)
	if len(rows) != 1 || rows[0].SourceID != "customer" || rows[0].TargetID != "order" {
		t.Fatalf("expected data->data relation, got %+v", rows)
	}
}
