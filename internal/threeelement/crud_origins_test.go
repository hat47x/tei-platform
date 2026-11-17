package threeelement

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCRUDCellOriginsRetainEveryOperationBehindOneCell(t *testing.T) {
	model := NewThreeElementModel("review", "Read-only review")
	model.DataElements = []DataElement{{ID: "order", Name: "Order data"}}
	model.BusinessElements = []BusinessElement{{ID: "order", Name: "Unrelated business"}}
	model.FunctionElements = []FunctionElement{{
		ID: "submit", Name: "Submit",
		Operations: []FunctionOperation{
			{DataID: "order", Operations: "r"},
			{DataID: "order", Operations: "U"},
			{DataID: "order", Operations: "R"},
		},
	}}
	before, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	cells := GenerateCRUDCellOrigins(model)
	want := []CRUDCellOrigin{{
		FunctionID: "submit", DataID: "order",
		Facts: []CRUDFactOccurrence{
			{Kind: "function-operation", FunctionPosition: 1, OperationPosition: 1, Operations: "r"},
			{Kind: "function-operation", FunctionPosition: 1, OperationPosition: 2, Operations: "U"},
			{Kind: "function-operation", FunctionPosition: 1, OperationPosition: 3, Operations: "R"},
		},
	}}
	if !reflect.DeepEqual(cells, want) {
		t.Fatalf("model-local declarations lost or coalesced: got %+v, want %+v", cells, want)
	}
	after, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("read-only drill-down modified the model")
	}

	// A projection can show one RU cell while retaining all three literal
	// declarations for review. No row is a direct-write occurrence.
	rows := GenerateCRUDTable(model)
	refs := GenerateFunctionDataCrossReference(model)
	if len(rows) != 1 || !rows[0].Read || !rows[0].Update ||
		len(refs) != 1 || refs[0].Operations != "RU" {
		t.Fatalf("display cell and original declarations disagree: rows=%+v refs=%+v", rows, refs)
	}
}

func TestCRUDCellOriginsKeepConflictingDefinitionsAndModelPositions(t *testing.T) {
	model := NewThreeElementModel("review", "Review conflicting declarations")
	model.DataElements = []DataElement{{ID: "order", Name: "Order data"}, {ID: "customer", Name: "Customer data"}}
	model.FunctionElements = []FunctionElement{
		{ID: "lookup", Name: "Lookup", Operations: []FunctionOperation{{DataID: "customer", Operations: "R"}}},
		{ID: "submit", Name: "Submit", Operations: []FunctionOperation{{DataID: "order", Operations: "R"}}},
	}
	model.CRUDDefinitions = []CRUDDefinition{
		{ID: "decl-1", FunctionID: "submit", DataID: "order", Update: true, Source: "declared"},
		{ID: "decl-2", FunctionID: "submit", DataID: "order", Delete: true, Source: "inferred"},
	}
	cells := GenerateCRUDCellOrigins(model)
	if len(cells) != 2 || cells[0].FunctionID != "lookup" ||
		cells[0].DataID != "customer" || cells[1].FunctionID != "submit" ||
		cells[1].DataID != "order" {
		t.Fatalf("cell order or identity is wrong: %+v", cells)
	}
	want := []CRUDFactOccurrence{
		{Kind: "function-operation", FunctionPosition: 2, OperationPosition: 1, Operations: "R"},
		{Kind: "crud-definition", DefinitionPosition: 1, DefinitionID: "decl-1", Operations: "U", SourceLabel: "declared"},
		{Kind: "crud-definition", DefinitionPosition: 2, DefinitionID: "decl-2", Operations: "D", SourceLabel: "inferred"},
	}
	if !reflect.DeepEqual(cells[1].Facts, want) {
		t.Fatalf("a selected last declaration hid earlier facts: got %+v, want %+v", cells[1].Facts, want)
	}
	if len(model.ContradictoryCRUDs()) != 1 {
		t.Fatal("fixture must include genuinely contradictory declarations")
	}

	// Changing the order of the two declarations changes local positions,
	// not the retained set of facts or their declaration IDs. No position is
	// qualified as a persistent Source locator.
	model.CRUDDefinitions[0], model.CRUDDefinitions[1] = model.CRUDDefinitions[1], model.CRUDDefinitions[0]
	reordered := GenerateCRUDCellOrigins(model)
	if len(reordered) != 2 || len(reordered[1].Facts) != 3 ||
		reordered[1].Facts[1].DefinitionID != "decl-2" ||
		reordered[1].Facts[1].DefinitionPosition != 1 ||
		reordered[1].Facts[2].DefinitionID != "decl-1" ||
		reordered[1].Facts[2].DefinitionPosition != 2 {
		t.Fatalf("origin positions did not follow the source snapshot: %+v", reordered)
	}
}

func TestCRUDCellOriginsPreserveUnresolvedPairsForReview(t *testing.T) {
	model := NewThreeElementModel("unresolved", "Unknown targets")
	model.FunctionElements = []FunctionElement{{
		ID: "submit", Operations: []FunctionOperation{{DataID: "not-loaded", Operations: "U"}},
	}}
	cells := GenerateCRUDCellOrigins(model)
	if len(cells) != 1 || cells[0].DataID != "not-loaded" ||
		len(cells[0].Facts) != 1 || cells[0].Facts[0].Operations != "U" {
		t.Fatalf("unresolved declaration disappeared: %+v", cells)
	}
	// This read-only projection records a declared target, not successful
	// resolution against a complete Data collection or an authorized write.
}
