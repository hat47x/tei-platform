package threeelement

import (
	"reflect"
	"strings"
	"testing"
)

// These expected duplicate identities are independently specified here. The
// production Index must never choose the first or last duplicate by accident.
func TestDuplicateKindIDsAreInvalidAndOmittedFromLookup(t *testing.T) {
	model := NewThreeElementModel("duplicate-fixture", "Duplicate identity fixture")
	model.DataElements = []DataElement{
		{ID: "order", Name: "Original Order", Attributes: []DataAttribute{{ID: "original", Name: "Original", Type: "text"}}},
		{ID: "account", Name: "Unique Account"},
		{ID: "order", Name: "Replaced Order", Attributes: []DataAttribute{{ID: "replacement", Name: "Replacement", Type: "text"}}},
		{ID: "order", Name: "Third Order"},
	}
	model.FunctionElements = []FunctionElement{
		{ID: "save", Name: "Save first"},
		{ID: "unique-fn", Name: "Unique function"},
		{ID: "save", Name: "Save second"},
	}
	model.BusinessElements = []BusinessElement{
		{ID: "approve", Name: "Approve first"},
		{ID: "unique-bus", Name: "Unique business"},
		{ID: "approve", Name: "Approve second"},
	}

	report := ValidateModel(model)
	if report.OK {
		t.Fatalf("duplicate model unexpectedly OK: %+v", report.Diagnostics)
	}
	want := map[string]int{
		"DUPLICATE_DATA_ID":     2,
		"DUPLICATE_FUNCTION_ID": 1,
		"DUPLICATE_BUSINESS_ID": 1,
	}
	got := map[string]int{}
	for _, diagnostic := range report.Diagnostics {
		if _, expected := want[diagnostic.Code]; !expected {
			continue
		}
		if diagnostic.Severity != "error" || len(diagnostic.SubjectIDs) != 1 {
			t.Fatalf("duplicate identity must be error with subject ID: %+v", diagnostic)
		}
		if !strings.Contains(diagnostic.Message, "positions ") {
			t.Fatalf("duplicate diagnosis must identify source positions: %+v", diagnostic)
		}
		got[diagnostic.Code]++
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("duplicate diagnostics: got %v, want %v; all: %+v", got, want, report.Diagnostics)
	}
	if report.ErrorCount != 4 {
		t.Fatalf("expected four duplicate errors, got %d: %+v", report.ErrorCount, report.Diagnostics)
	}

	data, functions, businesses := model.Index()
	if _, exists := data["order"]; exists {
		t.Fatal("ambiguous data ID was silently resolved")
	}
	if _, exists := functions["save"]; exists {
		t.Fatal("ambiguous function ID was silently resolved")
	}
	if _, exists := businesses["approve"]; exists {
		t.Fatal("ambiguous business ID was silently resolved")
	}
	if got := data["account"].Name; got != "Unique Account" {
		t.Fatalf("unique data lost: %q", got)
	}
	if got := functions["unique-fn"].Name; got != "Unique function" {
		t.Fatalf("unique function lost: %q", got)
	}
	if got := businesses["unique-bus"].Name; got != "Unique business" {
		t.Fatalf("unique business lost: %q", got)
	}

	// Calling Validate or Index must not change the source definitions.
	if got := model.DataElements[0].Name; got != "Original Order" {
		t.Fatalf("source was modified: %q", got)
	}
	if got := model.DataElements[2].Name; got != "Replaced Order" {
		t.Fatalf("second source definition was modified: %q", got)
	}
	if again := ValidateModel(model); !reflect.DeepEqual(again.Diagnostics, report.Diagnostics) {
		t.Fatalf("duplicate validation is not deterministic: first %v, second %v", report.Diagnostics, again.Diagnostics)
	}
}

// A bare ID is not globally unique: qualified Kind is part of identity.
// Same ID in Data, Function and Business is valid when each kind has one entry.
func TestIdenticalIDsAcrossDifferentKindsRemainValid(t *testing.T) {
	model := NewThreeElementModel("qualified-fixture", "Kind-qualified fixture")
	model.DataElements = []DataElement{{ID: "shared", Name: "Data shared"}}
	model.FunctionElements = []FunctionElement{{ID: "shared", Name: "Function shared"}}
	model.BusinessElements = []BusinessElement{{ID: "shared", Name: "Business shared"}}

	report := ValidateModel(model)
	if !report.OK || report.ErrorCount != 0 {
		t.Fatalf("qualified cross-kind IDs must be valid: %+v", report.Diagnostics)
	}
	data, functions, businesses := model.Index()
	if data["shared"].Name != "Data shared" ||
		functions["shared"].Name != "Function shared" ||
		businesses["shared"].Name != "Business shared" {
		t.Fatalf("qualified values collapsed: data=%v, functions=%v, businesses=%v",
			data, functions, businesses)
	}
}

// An empty ID is a separate error; its repeated presence must not create a
// misleading duplicate subject, and never make the overall model valid.
func TestEmptyIdentityIsNotMisreportedAsDuplicate(t *testing.T) {
	model := NewThreeElementModel("empty-fixture", "Empty IDs fixture")
	model.DataElements = []DataElement{{Name: "Unnamed A"}, {Name: "Unnamed B"}}
	report := ValidateModel(model)
	if report.OK {
		t.Fatal("empty IDs must fail validation")
	}
	for _, d := range report.Diagnostics {
		if d.Code == "DUPLICATE_DATA_ID" {
			t.Fatalf("empty ID was reported as a duplicate qualified subject: %+v", d)
		}
	}
}
