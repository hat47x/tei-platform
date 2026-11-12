package threeelement

import (
	"reflect"
	"testing"
)

// The expected Function/Data pairs are specified independently of the
// projection implementation. No FunctionOperation exists in this fixture.
func TestExplicitOnlyCRUDIsPresentInBothReadOnlyProjections(t *testing.T) {
	model := NewThreeElementModel("explicit-only", "Explicit CRUD only")
	model.DataElements = []DataElement{
		{ID: "order", Name: "Order data"},
		{ID: "invoice", Name: "Invoice data"},
		{ID: "orphan", Name: "Unused data"},
	}
	model.BusinessElements = []BusinessElement{
		{ID: "order", Name: "Business order with the same bare ID"},
	}
	model.FunctionElements = []FunctionElement{
		{ID: "submit", Name: "Submit order"},
		{ID: "bill", Name: "Bill order"},
	}
	model.CRUDDefinitions = []CRUDDefinition{
		{ID: "explicit-order", FunctionID: "submit", DataID: "order", Create: true, Source: "declared"},
		{ID: "explicit-invoice", FunctionID: "bill", DataID: "invoice", Read: true, Update: true, Source: "declared"},
	}
	wantCRUD := []CRUDTable{
		{FunctionID: "bill", FunctionName: "Bill order", DataID: "invoice", DataName: "Invoice data", Read: true, Update: true, Source: "declared"},
		{FunctionID: "submit", FunctionName: "Submit order", DataID: "order", DataName: "Order data", Create: true, Source: "declared"},
	}
	if got := GenerateCRUDTable(model); !reflect.DeepEqual(got, wantCRUD) {
		t.Fatalf("explicit-only CRUD cells silently lost: got %+v, want %+v", got, wantCRUD)
	}
	wantRefs := []FunctionDataCrossReference{
		{FunctionID: "bill", FunctionName: "Bill order", DataID: "invoice", DataName: "Invoice data", Operations: "RU", Source: "declared"},
		{FunctionID: "submit", FunctionName: "Submit order", DataID: "order", DataName: "Order data", Operations: "C", Source: "declared"},
	}
	if got := GenerateFunctionDataCrossReference(model); !reflect.DeepEqual(got, wantRefs) {
		t.Fatalf("explicit-only Function/Data references silently lost: got %+v, want %+v", got, wantRefs)
	}
	if got := model.UnusedData(); !reflect.DeepEqual(got, []string{"orphan"}) {
		t.Fatalf("explicit-only Data must count as referenced; unused=%v", got)
	}
	report := ValidateModel(model)
	if !report.OK || report.CRUDConflictStatus != "no-conflicts-detected" {
		t.Fatalf("valid explicit-only model must remain admissible: %+v", report)
	}
}

// An explicit pair already covered by a FunctionOperation must be emitted
// only once. A different explicit-only pair must still appear, in stable order.
func TestMixedDerivedAndExplicitOnlyCRUDHasNoDuplicatePairs(t *testing.T) {
	model := NewThreeElementModel("mixed", "Derived and explicit")
	model.DataElements = []DataElement{
		{ID: "order", Name: "Order"},
		{ID: "invoice", Name: "Invoice"},
		{ID: "shipment", Name: "Shipment"},
	}
	model.FunctionElements = []FunctionElement{
		{ID: "submit", Name: "Submit", Operations: []FunctionOperation{{DataID: "order", Operations: "R"}}},
		{ID: "ship", Name: "Ship", Operations: []FunctionOperation{{DataID: "shipment", Operations: "D"}}},
	}
	model.CRUDDefinitions = []CRUDDefinition{
		{ID: "override", FunctionID: "submit", DataID: "order", Create: true, Source: "declared"},
		{ID: "only", FunctionID: "submit", DataID: "invoice", Update: true, Source: "declared"},
	}
	wantPairs := [][2]string{{"ship", "shipment"}, {"submit", "invoice"}, {"submit", "order"}}
	rows := GenerateCRUDTable(model)
	if len(rows) != len(wantPairs) {
		t.Fatalf("expected 3 unique pairs, got %+v", rows)
	}
	refs := GenerateFunctionDataCrossReference(model)
	if len(refs) != len(wantPairs) {
		t.Fatalf("expected 3 Function/Data pairs, got %+v", refs)
	}
	for i, pair := range wantPairs {
		if rows[i].FunctionID != pair[0] || rows[i].DataID != pair[1] ||
			refs[i].FunctionID != pair[0] || refs[i].DataID != pair[1] {
			t.Fatalf("unstable or duplicated pair %d: rows=%+v refs=%+v", i, rows, refs)
		}
	}
	if !rows[2].Create || rows[2].Read || refs[2].Operations != "C" {
		t.Fatalf("explicit declaration must override Operation R: rows=%+v refs=%+v", rows, refs)
	}
	if !rows[1].Update || refs[1].Operations != "U" ||
		!rows[0].Delete || refs[0].Operations != "D" {
		t.Fatalf("explicit-only and derived flags must coexist: rows=%+v refs=%+v", rows, refs)
	}
}

// A last-writer projection is not evidence of accepted CRUD meaning. The
// same source pair must carry an explicit conflict marker to direct callers.
func TestExplicitOnlyConflictingCRUDIsVisibleAsUnresolved(t *testing.T) {
	model := NewThreeElementModel("conflicting", "Contradictory explicit-only")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit"}}
	model.CRUDDefinitions = []CRUDDefinition{
		{ID: "create", FunctionID: "submit", DataID: "order", Create: true, Source: "declared"},
		{ID: "read", FunctionID: "submit", DataID: "order", Read: true, Source: "declared"},
	}
	report := ValidateModel(model)
	if !report.OK || report.CRUDConflictStatus != "conflicts-require-review" ||
		len(report.ContradictoryCRUDs) != 1 || report.ContradictoryCRUDs[0].ID != "read" {
		t.Fatalf("structural OK and CRUD review status must be separate: %+v", report)
	}
	rows := GenerateCRUDTable(model)
	refs := GenerateFunctionDataCrossReference(model)
	if len(rows) != 1 || !rows[0].ConflictingDeclarations ||
		len(refs) != 1 || !refs[0].ConflictingDeclarations {
		t.Fatalf("unresolved contradiction was silently presented as a clean cell: rows=%+v refs=%+v", rows, refs)
	}
	if rows[0].DataID != "order" || refs[0].DataID != "order" {
		t.Fatalf("qualified target was lost: rows=%+v refs=%+v", rows, refs)
	}
}

// A recorded but invalid explicit reference remains inspectable by ID.
// Neither the projection nor usage statistics may invent a resolved Function.
func TestExplicitOnlyUnknownFunctionDoesNotBecomeResolvedUsage(t *testing.T) {
	model := NewThreeElementModel("unresolved", "Unknown declared Function")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.CRUDDefinitions = []CRUDDefinition{{
		ID: "unresolved", FunctionID: "missing", DataID: "order", Read: true, Source: "declared",
	}}
	report := ValidateModel(model)
	if report.OK {
		t.Fatalf("unknown Function must not be structurally admitted: %+v", report)
	}
	rows := GenerateCRUDTable(model)
	if len(rows) != 1 || rows[0].FunctionID != "missing" ||
		rows[0].FunctionName != "" || rows[0].DataName != "Order" {
		t.Fatalf("unknown Function must remain inspectable but unresolved: %+v", rows)
	}
	if got := model.UnusedData(); !reflect.DeepEqual(got, []string{"order"}) {
		t.Fatalf("invalid Function must not prove Data usage: %v", got)
	}
}
