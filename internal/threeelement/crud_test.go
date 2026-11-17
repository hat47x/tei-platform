package threeelement

import (
	"reflect"
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


// TestMultipleFunctionOperationsFormOneCRUDCell keeps the Function/Data
// correspondence at one row per pair. A first-match projection would omit U;
// a one-row-per-Operation projection would invent duplicate Data subjects.
func TestMultipleFunctionOperationsFormOneCRUDCell(t *testing.T) {
	model := NewThreeElementModel("multi-operation", "Multiple CRUD intentions")
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

	rows := GenerateCRUDTable(model)
	if len(rows) != 1 {
		t.Fatalf("one Function/Data pair should yield one CRUD cell, got %+v", rows)
	}
	if rows[0].DataName != "Order data" || !rows[0].Read || !rows[0].Update ||
		rows[0].Create || rows[0].Delete || rows[0].Source != "derived" {
		t.Fatalf("CRUD cell omitted or misattributed intent: %+v", rows[0])
	}

	refs := GenerateFunctionDataCrossReference(model)
	if len(refs) != 1 || refs[0].DataName != "Order data" ||
		refs[0].Operations != "RU" || refs[0].Source != "derived" {
		t.Fatalf("Function/Data projection omitted unioned operations: %+v", refs)
	}

	// Reordering operations should not change the meaning or the read-only
	// projection. The render order is not a business sequence.
	model.FunctionElements[0].Operations = []FunctionOperation{
		{DataID: "order", Operations: "U"},
		{DataID: "order", Operations: "R"},
		{DataID: "order", Operations: "r"},
	}
	if reordered := GenerateCRUDTable(model); !reflect.DeepEqual(rows, reordered) {
		t.Fatalf("CRUD meaning depends on source operation order: before=%+v after=%+v", rows, reordered)
	}
	if reordered := GenerateFunctionDataCrossReference(model); !reflect.DeepEqual(refs, reordered) {
		t.Fatalf("Function/Data meaning depends on operation order: before=%+v after=%+v", refs, reordered)
	}
}

// TestMultipleFunctionOperationsRemainVisibleUnderExplicitCRUD makes sure
// the selected display flags never erase the separate Function.Operations
// facts. A declaration is not authorization to perform a real DB write.
func TestMultipleFunctionOperationsRemainVisibleUnderExplicitCRUD(t *testing.T) {
	model := NewThreeElementModel("explicit-multi", "Explicit and derived")
	model.DataElements = []DataElement{{ID: "order", Name: "Order data"}}
	model.FunctionElements = []FunctionElement{{
		ID: "submit", Name: "Submit",
		Operations: []FunctionOperation{
			{DataID: "order", Operations: "R"},
			{DataID: "order", Operations: "U"},
		},
	}}
	model.CRUDDefinitions = []CRUDDefinition{{
		ID: "declared", FunctionID: "submit", DataID: "order",
		Read: true, Source: "declared",
	}}

	rows := GenerateCRUDTable(model)
	if len(rows) != 1 || !rows[0].Read || rows[0].Update ||
		rows[0].DerivedOperations != "RU" || rows[0].Source != "declared" ||
		rows[0].ConflictingDeclarations {
		t.Fatalf("explicit projection lost the additional Function intent: %+v", rows)
	}
	refs := GenerateFunctionDataCrossReference(model)
	if len(refs) != 1 || refs[0].Operations != "R" ||
		refs[0].DerivedOperations != "RU" || refs[0].Source != "declared" ||
		refs[0].ConflictingDeclarations {
		t.Fatalf("Function/Data cross-reference lost the derived intent: %+v", refs)
	}

	// An opposing explicit declaration is reported, not silently reconciled.
	model.CRUDDefinitions[0].Read = false
	model.CRUDDefinitions[0].Create = true
	rows = GenerateCRUDTable(model)
	refs = GenerateFunctionDataCrossReference(model)
	if len(rows) != 1 || !rows[0].ConflictingDeclarations ||
		!rows[0].Create || rows[0].Read || rows[0].Update ||
		rows[0].DerivedOperations != "RU" {
		t.Fatalf("opposing CRUD intent was hidden: %+v", rows)
	}
	if len(refs) != 1 || !refs[0].ConflictingDeclarations ||
		refs[0].Operations != "C" || refs[0].DerivedOperations != "RU" {
		t.Fatalf("opposing Function/Data intent was hidden: %+v", refs)
	}
}
