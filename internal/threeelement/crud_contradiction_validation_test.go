package threeelement

import (
	"encoding/json"
	"reflect"
	"testing"
)

// The expectation is authored independently of ValidateModel and
// ContradictoryCRUDs: different flags for the SAME Function/Data pair are
// invalid, regardless of which definition was encountered last.
func TestContradictoryCRUDDefinitionsFailModelValidation(t *testing.T) {
	model := NewThreeElementModel("crud-conflicts", "Independent contradiction fixture")
	model.DataElements = []DataElement{
		{ID: "order", Name: "Order"},
		{ID: "invoice", Name: "Invoice"},
	}
	model.FunctionElements = []FunctionElement{
		{ID: "submit", Name: "Submit"},
		{ID: "bill", Name: "Bill"},
	}
	model.CRUDDefinitions = []CRUDDefinition{
		{ID: "order-create", FunctionID: "submit", DataID: "order", Create: true, Source: "declared"},
		{ID: "invoice-read", FunctionID: "bill", DataID: "invoice", Read: true, Source: "declared"},
		{ID: "order-read", FunctionID: "submit", DataID: "order", Read: true, Source: "declared"},
		{ID: "invoice-update", FunctionID: "bill", DataID: "invoice", Update: true, Source: "declared"},
		{ID: "order-create-again", FunctionID: "submit", DataID: "order", Create: true, Source: "declared"},
	}

	before, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	report := ValidateModel(model)
	if report.OK {
		t.Fatalf("conflicting CRUD definitions must not be valid: %+v", report)
	}
	if report.ErrorCount != 3 {
		t.Fatalf("each conflicting later definition is an error: want 3, got %d; diagnostics=%+v",
			report.ErrorCount, report.Diagnostics)
	}

	wantIDs := []string{"order-read", "invoice-update", "order-create-again"}
	gotIDs := make([]string, 0, len(report.ContradictoryCRUDs))
	for _, definition := range report.ContradictoryCRUDs {
		gotIDs = append(gotIDs, definition.ID)
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("first-conflict-only / last-writer-wins bug: got %v, want %v", gotIDs, wantIDs)
	}

	wantDiagnosticSubjects := map[string][3]string{
		"order-read":         {"order-read", "submit", "order"},
		"invoice-update":     {"invoice-update", "bill", "invoice"},
		"order-create-again": {"order-create-again", "submit", "order"},
	}
	seen := map[string]bool{}
	for _, diagnostic := range model.Validate() {
		if diagnostic.Code != "CONTRADICTORY_CRUD_DEFINITION" {
			continue
		}
		if diagnostic.Severity != "error" || len(diagnostic.SubjectIDs) != 3 {
			t.Fatalf("contradiction must have error severity and all typed IDs: %+v", diagnostic)
		}
		var subjects [3]string
		copy(subjects[:], diagnostic.SubjectIDs)
		if expected, exists := wantDiagnosticSubjects[subjects[0]]; !exists || expected != subjects {
			t.Fatalf("wrong contradiction subject: %v, expected one of %+v", subjects, wantDiagnosticSubjects)
		}
		if seen[subjects[0]] {
			t.Fatalf("same contradiction emitted twice: %v", subjects)
		}
		seen[subjects[0]] = true
	}
	if len(seen) != len(wantDiagnosticSubjects) {
		t.Fatalf("direct Validate omitted contradictions: got %v, want %v", seen, wantDiagnosticSubjects)
	}
	after, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("validation changed source model:\nbefore %s\nafter  %s", before, after)
	}
}

// Identical declarations for the same pair are not contradictory merely
// because they have two source records. A separate duplicate-record policy
// must not be accidentally introduced as part of the conflict fix.
func TestIdenticalCRUDDefinitionsDoNotBecomeConflictErrors(t *testing.T) {
	model := NewThreeElementModel("crud-agree", "Agreeing declarations")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit"}}
	model.CRUDDefinitions = []CRUDDefinition{
		{ID: "entry-a", FunctionID: "submit", DataID: "order", Create: true, Read: true},
		{ID: "entry-b", FunctionID: "submit", DataID: "order", Create: true, Read: true},
	}
	report := ValidateModel(model)
	if !report.OK || report.ErrorCount != 0 || len(report.ContradictoryCRUDs) != 0 {
		t.Fatalf("agreeing flags are not a contradiction: %+v", report)
	}
}

// A completely different function-data pair must not collide with a
// declaration just because the CRUD flags or bare IDs have similar text.
func TestCRUDConflictIsQualifiedByFunctionAndDataPair(t *testing.T) {
	model := NewThreeElementModel("crud-pairs", "Qualified CRUD pairs")
	model.DataElements = []DataElement{
		{ID: "order", Name: "Order"},
		{ID: "invoice", Name: "Invoice"},
	}
	model.FunctionElements = []FunctionElement{
		{ID: "submit", Name: "Submit"},
		{ID: "bill", Name: "Bill"},
	}
	model.CRUDDefinitions = []CRUDDefinition{
		{ID: "one", FunctionID: "submit", DataID: "order", Create: true},
		{ID: "two", FunctionID: "bill", DataID: "order", Read: true},
		{ID: "three", FunctionID: "submit", DataID: "invoice", Update: true},
	}
	report := ValidateModel(model)
	if !report.OK || len(report.ContradictoryCRUDs) != 0 {
		t.Fatalf("distinct qualified pairs must remain independently valid: %+v", report)
	}
}
