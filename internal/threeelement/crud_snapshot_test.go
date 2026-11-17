package threeelement

import "testing"

func TestCRUDSnapshotProjectionBindsRowsAndOriginsToOneModel(t *testing.T) {
	model := NewThreeElementModel("snapshot", "Snapshot")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{
		ID: "submit", Name: "Submit",
		Operations: []FunctionOperation{
			{DataID: "order", Operations: "R"},
			{DataID: "order", Operations: "U"},
		},
	}}
	first := GenerateCRUDSnapshotProjection(model)
	if first.ModelFingerprint != ModelFingerprint(model) {
		t.Fatal("projection fingerprint differs from its input snapshot")
	}
	if len(first.Rows) != 1 || len(first.Origins) != 1 || len(first.Origins[0].Facts) != 2 {
		t.Fatalf("expected one cell with two original facts: %+v", first)
	}
	if first.Rows[0].FunctionID != first.Origins[0].FunctionID ||
		first.Rows[0].DataID != first.Origins[0].DataID {
		t.Fatal("cell and origins refer to different Function/Data pairs")
	}
	model.FunctionElements[0].Operations[1].Operations = "D"
	second := GenerateCRUDSnapshotProjection(model)
	if first.ModelFingerprint == second.ModelFingerprint {
		t.Fatal("changing an operation must invalidate the snapshot binding")
	}
	if second.Origins[0].Facts[1].Operations != "D" ||
		first.Origins[0].Facts[1].Operations != "U" {
		t.Fatal("projection facts must reflect their own input snapshots")
	}
}

func TestCRUDSnapshotProjectionPreservesConflictingDeclarations(t *testing.T) {
	model := NewThreeElementModel("conflict", "Conflict")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit"}}
	model.CRUDDefinitions = []CRUDDefinition{
		{ID: "one", FunctionID: "submit", DataID: "order", Create: true, Source: "declared"},
		{ID: "two", FunctionID: "submit", DataID: "order", Delete: true, Source: "declared"},
	}
	projection := GenerateCRUDSnapshotProjection(model)
	if len(projection.Rows) != 1 || len(projection.Origins) != 1 {
		t.Fatalf("expected one cell and one origin group: %+v", projection)
	}
	if len(projection.Origins[0].Facts) != 2 ||
		projection.Origins[0].Facts[0].DefinitionID != "one" ||
		projection.Origins[0].Facts[1].DefinitionID != "two" {
		t.Fatalf("conflicting declarations were lost: %+v", projection.Origins)
	}
}
