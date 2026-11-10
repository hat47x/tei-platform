package threeelement

import (
	"testing"
)

func TestGenerateCRUDTableUsesExplicitDefinition(t *testing.T) {
	model := NewThreeElementModel("probe", "Probe")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit", Operations: []FunctionOperation{{DataID: "order", Operations: "R"}}}}
	model.CRUDDefinitions = []CRUDDefinition{{ID: "crud1", FunctionID: "submit", DataID: "order", Create: true, Read: true, Update: true, Delete: true, Source: "declared"}}
	rows := GenerateCRUDTable(model)
	if len(rows) != 1 {
		t.Fatalf("expected 1 CRUD row, got %d", len(rows))
	}
	if !rows[0].Create || !rows[0].Read || !rows[0].Update || !rows[0].Delete {
		t.Fatalf("expected explicit CRUD to override derived flags: %+v", rows[0])
	}
	if rows[0].Source != "declared" {
		t.Fatalf("expected source declared, got %s", rows[0].Source)
	}
}

func TestGenerateBusinessFunctionCrossReference(t *testing.T) {
	model := NewThreeElementModel("probe", "Probe")
	model.BusinessElements = []BusinessElement{{ID: "sales", Name: "Sales", Activities: []BusinessActivity{{ID: "a1", Name: "Enter order", FunctionIDs: []string{"submit"}}}}}
	model.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit"}}
	rows := GenerateBusinessFunctionCrossReference(model)
	if len(rows) != 1 || rows[0].FunctionID != "submit" {
		t.Fatalf("expected business->function mapping, got %+v", rows)
	}
}

func TestGenerateFunctionDataCrossReference(t *testing.T) {
	model := NewThreeElementModel("probe", "Probe")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit", Operations: []FunctionOperation{{DataID: "order", Operations: "C"}}}}
	rows := GenerateFunctionDataCrossReference(model)
	if len(rows) != 1 || rows[0].Operations != "C" {
		t.Fatalf("expected function->data mapping, got %+v", rows)
	}
}
