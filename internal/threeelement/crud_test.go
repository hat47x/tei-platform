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

// TestCRUDDataNamesUseTheDataIndex exercises a cross-kind ID collision.
// Data and Business IDs belong to different maps; using the Business map to
// render a Data name silently substitutes an unrelated subject.
func TestCRUDDataNamesUseTheDataIndex(t *testing.T) {
	model := NewThreeElementModel("cross-kind", "Cross-kind lookup")
	model.DataElements = []DataElement{{ID: "order", Name: "Order data"}}
	model.BusinessElements = []BusinessElement{{ID: "order", Name: "Order business"}}
	model.FunctionElements = []FunctionElement{{
		ID: "submit", Name: "Submit",
		Operations: []FunctionOperation{{DataID: "order", Operations: "R"}},
	}}

	derived := GenerateCRUDTable(model)
	if len(derived) != 1 || derived[0].DataID != "order" || derived[0].DataName != "Order data" {
		t.Fatalf("CRUD table used a non-Data index: %+v", derived)
	}
	refs := GenerateFunctionDataCrossReference(model)
	if len(refs) != 1 || refs[0].DataName != "Order data" || refs[0].Operations != "R" {
		t.Fatalf("Function/Data reference used a non-Data index: %+v", refs)
	}

	// An explicit CRUD definition changes operation flags, never the target
	// Data's identity/name. The Business element has the same plain ID.
	model.CRUDDefinitions = []CRUDDefinition{{
		ID: "explicit-submit", FunctionID: "submit", DataID: "order",
		Create: true, Source: "declared",
	}}
	explicit := GenerateCRUDTable(model)
	if len(explicit) != 1 || !explicit[0].Create || explicit[0].DataName != "Order data" {
		t.Fatalf("explicit CRUD has an incorrect Data subject: %+v", explicit)
	}
	refs = GenerateFunctionDataCrossReference(model)
	if len(refs) != 1 || refs[0].Operations != "C" || refs[0].DataName != "Order data" {
		t.Fatalf("explicit Function/Data reference has an incorrect subject: %+v", refs)
	}
}

// TestCRUDGapsUseTheDataIndex checks the Data name in an existing gap report.
// Whether a cross-reference should require CRUD is a separate policy question;
// this test only pins the lookup identity for a gap that the current rule emits.
func TestCRUDGapsUseTheDataIndex(t *testing.T) {
	model := NewThreeElementModel("cross-kind-gap", "Cross-kind gap")
	model.DataElements = []DataElement{{ID: "order", Name: "Order data"}}
	model.BusinessElements = []BusinessElement{{ID: "order", Name: "Order business"}}
	model.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit"}}
	model.CrossReferences = []CrossReference{{
		ID: "submit-order", Kind: "function-to-data",
		FromID: "submit", FromKind: "function", ToID: "order", ToKind: "data",
	}}
	gaps := GenerateCRUDGaps(model)
	if len(gaps) != 1 || gaps[0].FunctionID != "submit" ||
		gaps[0].DataID != "order" || gaps[0].DataName != "Order data" {
		t.Fatalf("CRUD gap used a non-Data index: %+v", gaps)
	}
}
