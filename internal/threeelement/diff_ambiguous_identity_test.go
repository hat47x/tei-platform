package threeelement

import "testing"

// A diff compares declarations by identity. Two declarations that share one
// identity can only be compared as one, so the reduction has to be reported or
// the diff answers for a declaration it did not choose. These expectations are
// written from the contract, not from byID: the models below are built by hand.
func TestDiffModelsReportsIdentitiesItHadToReduce(t *testing.T) {
	before := NewThreeElementModel("before", "Before")
	before.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	before.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit"}}
	before.CrossReferences = []CrossReference{
		{ID: "x1", Kind: "function-to-data", FromKind: "function", FromID: "submit", ToKind: "data", ToID: "order"},
		{ID: "x1", Kind: "function-to-data", FromKind: "function", FromID: "submit", ToKind: "data", ToID: "order"},
	}
	before.CRUDDefinitions = []CRUDDefinition{
		{ID: "c1", FunctionID: "submit", DataID: "order", Read: true},
		{ID: "c1", FunctionID: "submit", DataID: "order", Update: true},
	}

	// The after model is the same content with unique identities: no ambiguity.
	after := NewThreeElementModel("after", "After")
	after.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	after.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit"}}
	after.CrossReferences = []CrossReference{
		{ID: "x1", Kind: "function-to-data", FromKind: "function", FromID: "submit", ToKind: "data", ToID: "order"},
		{ID: "x2", Kind: "function-to-data", FromKind: "function", FromID: "submit", ToKind: "data", ToID: "order"},
	}
	after.CRUDDefinitions = []CRUDDefinition{
		{ID: "c1", FunctionID: "submit", DataID: "order", Read: true},
		{ID: "c2", FunctionID: "submit", DataID: "order", Update: true},
	}

	diff := DiffModels(before, after)
	if len(diff.AmbiguousIdentities) != 2 {
		t.Fatalf("expected the before side to report two reduced identities, got %+v", diff.AmbiguousIdentities)
	}
	for _, entry := range diff.AmbiguousIdentities {
		if entry.Side != "before" || entry.Count != 2 {
			t.Fatalf("wrong side or count: %+v", entry)
		}
		if entry.Category != "cross-reference" && entry.Category != "crud" {
			t.Fatalf("unexpected category: %+v", entry)
		}
	}
	// The entries must be ordered and stable, so a Consumer can compare them.
	first, second := diff.AmbiguousIdentities[0], diff.AmbiguousIdentities[1]
	if first.Side != second.Side || first.Category > second.Category {
		t.Fatalf("ambiguous identities are not ordered: %+v", diff.AmbiguousIdentities)
	}
	// The same id under different categories stays two entries, not one.
	if first.Category == second.Category {
		t.Fatalf("two categories collapsed into one: %+v", diff.AmbiguousIdentities)
	}
}

// A model with unique identities must report nothing: the ambiguity list is a
// finding about the input, not a blanket warning.
func TestDiffModelsReportsNoAmbiguityForUniqueIdentities(t *testing.T) {
	model := NewThreeElementModel("unique", "Unique")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit"}}
	model.CrossReferences = []CrossReference{
		{ID: "x1", Kind: "function-to-data", FromKind: "function", FromID: "submit", ToKind: "data", ToID: "order"},
		{ID: "x2", Kind: "function-to-data", FromKind: "function", FromID: "submit", ToKind: "data", ToID: "order"},
	}
	model.CRUDDefinitions = []CRUDDefinition{
		{ID: "c1", FunctionID: "submit", DataID: "order", Read: true},
		{ID: "c2", FunctionID: "submit", DataID: "order", Create: true},
	}

	// The same model compared with itself is the sharpest case: no entries yet
	// two declarations per pair.
	diff := DiffModels(model, model)
	if len(diff.AmbiguousIdentities) != 0 {
		t.Fatalf("unique identities reported as ambiguous: %+v", diff.AmbiguousIdentities)
	}
	if len(diff.Changed) != 0 || len(diff.Added) != 0 || len(diff.Removed) != 0 {
		t.Fatalf("self-comparison reported changes: %+v", diff)
	}
}

// Both sides can declare the same ambiguous identity; each side is named
// separately, because the finding is about which model overloads the name.
func TestDiffModelsNamesEachSideThatIsAmbiguous(t *testing.T) {
	build := func(modelID string) ThreeElementModel {
		model := NewThreeElementModel(modelID, modelID)
		model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
		model.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit"}}
		model.CrossReferences = []CrossReference{
			{ID: "x1", Kind: "function-to-data", FromKind: "function", FromID: "submit", ToKind: "data", ToID: "order"},
			{ID: "x1", Kind: "function-to-data", FromKind: "function", FromID: "submit", ToKind: "data", ToID: "order"},
		}
		return model
	}

	diff := DiffModels(build("a"), build("b"))
	if len(diff.AmbiguousIdentities) != 2 {
		t.Fatalf("expected one entry per side, got %+v", diff.AmbiguousIdentities)
	}
	if diff.AmbiguousIdentities[0].Side == diff.AmbiguousIdentities[1].Side {
		t.Fatalf("both sides were not named: %+v", diff.AmbiguousIdentities)
	}
}
