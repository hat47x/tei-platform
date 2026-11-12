package threeelement

import (
	"encoding/json"
	"reflect"
	"testing"
)

// The expected conflicts are authored independently of ValidateModel and
// ContradictoryCRUDs. Structural OK must not be misrepresented as CRUD
// consistency or Owner acceptance; the unresolved conflict has its own status.
func TestContradictoryCRUDDefinitionsRequireSeparateReview(t *testing.T) {
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
	if !report.OK || report.ErrorCount != 0 {
		t.Fatalf("CRUD conflicts must not be unilaterally escalated to structural errors: %+v", report)
	}
	if report.OKScope != "structural-diagnostics-only" {
		t.Fatalf("structural OK must have explicit scope, got %q", report.OKScope)
	}
	if report.CRUDConflictStatus != "conflicts-require-review" {
		t.Fatalf("conflict status lost: %q", report.CRUDConflictStatus)
	}

	wantIDs := []string{"order-read", "invoice-update", "order-create-again"}
	gotIDs := make([]string, 0, len(report.ContradictoryCRUDs))
	for _, definition := range report.ContradictoryCRUDs {
		gotIDs = append(gotIDs, definition.ID)
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("first-conflict-only / last-writer-wins bug: got %v, want %v", gotIDs, wantIDs)
	}

	// The underlying source is still structurally well-formed; selecting
	// error/warning severity for conflicting business declarations belongs
	// to the Owner. A separate machine-readable status makes this explicit.
	for _, diagnostic := range model.Validate() {
		if diagnostic.Code == "CONTRADICTORY_CRUD_DEFINITION" {
			t.Fatalf("unaccepted severity policy leaked into structural diagnostics: %+v", diagnostic)
		}
	}
	if again := ValidateModel(model); !reflect.DeepEqual(again.ContradictoryCRUDs, report.ContradictoryCRUDs) {
		t.Fatalf("conflict enumeration changed across identical calls: first %v, second %v",
			report.ContradictoryCRUDs, again.ContradictoryCRUDs)
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
func TestIdenticalCRUDDefinitionsHaveNoConflictStatus(t *testing.T) {
	model := NewThreeElementModel("crud-agree", "Agreeing declarations")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit"}}
	model.CRUDDefinitions = []CRUDDefinition{
		{ID: "entry-a", FunctionID: "submit", DataID: "order", Create: true, Read: true},
		{ID: "entry-b", FunctionID: "submit", DataID: "order", Create: true, Read: true},
	}
	report := ValidateModel(model)
	if !report.OK || report.ErrorCount != 0 || len(report.ContradictoryCRUDs) != 0 ||
		report.CRUDConflictStatus != "no-conflicts-detected" {
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
	if !report.OK || len(report.ContradictoryCRUDs) != 0 ||
		report.CRUDConflictStatus != "no-conflicts-detected" {
		t.Fatalf("distinct qualified pairs must remain independently valid: %+v", report)
	}
}
