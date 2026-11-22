package threeelement

import (
	"errors"
	"testing"
)

func TestInspectCRUDCellAtSnapshotKeepsOneCellAndAllOperations(t *testing.T) {
	model := NewThreeElementModel("order-entry", "Order")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{
		ID: "submit", Name: "Submit",
		Operations: []FunctionOperation{
			{DataID: "order", Operations: "R"},
			{DataID: "order", Operations: "U"},
		},
	}}
	original := ModelFingerprint(model)
	got, err := InspectCRUDCellAtSnapshot(model, original, "submit", "order")
	if err != nil {
		t.Fatal(err)
	}
	if got.ModelFingerprint != original || got.SourceQualification != "model-local-only" ||
		!got.Row.Read || !got.Row.Update || len(got.Facts) != 2 ||
		got.Facts[0].Operations != "R" || got.Facts[1].Operations != "U" {
		t.Fatalf("lost model-local facts or falsely claimed Source qualification: %+v", got)
	}
	got.Facts[0].Operations = "D"
	if model.FunctionElements[0].Operations[0].Operations != "R" {
		t.Fatal("inspector modified the model")
	}

	model.FunctionElements[0].Operations[1].Operations = "D"
	if _, err := InspectCRUDCellAtSnapshot(model, original, "submit", "order"); !errors.Is(err, ErrCRUDSnapshotStale) {
		t.Fatalf("same IDs with changed operations should be stale: %v", err)
	}
}

func TestInspectCRUDCellAtSnapshotRequiresFingerprintAndQualifiedPair(t *testing.T) {
	model := NewThreeElementModel("scope-a", "A")
	model.DataElements = []DataElement{{ID: "shared", Name: "Data"}}
	model.FunctionElements = []FunctionElement{{
		ID: "shared", Name: "Function",
		Operations: []FunctionOperation{{DataID: "shared", Operations: "R"}},
	}}
	fingerprint := ModelFingerprint(model)
	for _, invalid := range []string{"", " " + fingerprint} {
		if _, err := InspectCRUDCellAtSnapshot(model, invalid, "shared", "shared"); !errors.Is(err, ErrCRUDSnapshotRequired) {
			t.Fatalf("invalid expected fingerprint %q was accepted: %v", invalid, err)
		}
	}
	if _, err := InspectCRUDCellAtSnapshot(model, fingerprint, "shared", "missing"); !errors.Is(err, ErrCRUDCellNotFound) {
		t.Fatalf("unknown Data was not refused: %v", err)
	}
	// Identical spelling across element kinds is valid.
	if _, err := InspectCRUDCellAtSnapshot(model, fingerprint, "shared", "shared"); err != nil {
		t.Fatal(err)
	}
	otherScope := model
	otherScope.ModelID = "scope-b"
	if _, err := InspectCRUDCellAtSnapshot(otherScope, fingerprint, "shared", "shared"); !errors.Is(err, ErrCRUDSnapshotStale) {
		t.Fatalf("identical pair in another model scope was accepted: %v", err)
	}
}

func TestInspectCRUDCellAtSnapshotDoesNotHideExplicitConflicts(t *testing.T) {
	model := NewThreeElementModel("conflict", "Conflict")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit"}}
	model.CRUDDefinitions = []CRUDDefinition{
		{ID: "first", FunctionID: "submit", DataID: "order", Create: true},
		{ID: "second", FunctionID: "submit", DataID: "order", Delete: true},
	}
	got, err := InspectCRUDCellAtSnapshot(model, ModelFingerprint(model), "submit", "order")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Row.ConflictingDeclarations || len(got.Facts) != 2 ||
		got.Facts[0].DefinitionID != "first" || got.Facts[1].DefinitionID != "second" {
		t.Fatalf("conflicting declarations were hidden: %+v", got)
	}
}

func TestInspectCRUDCellAtSnapshotRejectsAmbiguousKindIdentity(t *testing.T) {
	model := NewThreeElementModel("duplicates", "Duplicates")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{
		{ID: "submit", Operations: []FunctionOperation{{DataID: "order", Operations: "R"}}},
		{ID: "submit", Operations: []FunctionOperation{{DataID: "order", Operations: "U"}}},
	}
	if _, err := InspectCRUDCellAtSnapshot(model, ModelFingerprint(model), "submit", "order"); !errors.Is(err, ErrCRUDCellAmbiguous) {
		t.Fatalf("duplicated Function identity was silently accepted: %v", err)
	}
	model.FunctionElements = model.FunctionElements[:1]
	model.DataElements = append(model.DataElements, DataElement{ID: "order", Name: "Other"})
	if _, err := InspectCRUDCellAtSnapshot(model, ModelFingerprint(model), "submit", "order"); !errors.Is(err, ErrCRUDCellAmbiguous) {
		t.Fatalf("duplicated Data identity was silently accepted: %v", err)
	}
}
