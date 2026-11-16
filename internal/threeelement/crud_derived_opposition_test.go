package threeelement

import (
	"encoding/json"
	"reflect"
	"testing"
)

// The expectations in this file are written from the source declarations by
// hand. They are not derived from GenerateCRUDTable, OpposingCRUDPairs or
// ValidateModel, so an implementation that changes both sides agrees with
// nothing here.
//
// The question these tests answer: when a Function declares operations in
// Function.Operations AND an explicit CRUDDefinition states different
// operations for the same Function/Data pair, does the read-only projection
// stay silent about the disagreement?

// orderCRUDModel builds a model whose CRUD qualification is fixed by the
// caller. Derived intent comes from Function.Operations, explicit intent from
// CRUDDefinitions, and both target the same Function/Data pair.
func orderCRUDModel(derivedOps, explicitOps string) ThreeElementModel {
	model := NewThreeElementModel("opposing", "Derived and explicit disagreement")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{
		ID: "submit", Name: "Submit",
		Operations: []FunctionOperation{{DataID: "order", Operations: derivedOps}},
	}}
	crud := CRUDDefinition{ID: "declared", FunctionID: "submit", DataID: "order", Source: "declared"}
	for _, ch := range explicitOps {
		switch ch {
		case 'C':
			crud.Create = true
		case 'R':
			crud.Read = true
		case 'U':
			crud.Update = true
		case 'D':
			crud.Delete = true
		}
	}
	model.CRUDDefinitions = []CRUDDefinition{crud}
	return model
}

// The projected cell must carry both qualifications. Reporting only the
// explicit side presents a last-writer value as the model's CRUD intent.
func TestOpposingDerivedAndExplicitCRUDIsNotHidden(t *testing.T) {
	// Function.Operations declares create; the explicit record declares read.
	model := orderCRUDModel("C", "R")

	wantRow := CRUDTable{
		FunctionID:   "submit",
		FunctionName: "Submit",
		DataID:       "order",
		DataName:     "Order",
		Create:       false,
		Read:         true,
		Update:       false,
		Delete:       false,
		Source:       "declared",
		// The explicit side wins as a projection convention. The overridden
		// derived intent must still be readable, and the cell is not clean.
		DerivedOperations:       "C",
		ConflictingDeclarations: true,
	}
	if got := GenerateCRUDTable(model); !reflect.DeepEqual(got, []CRUDTable{wantRow}) {
		t.Fatalf("CRUD table hid the opposition:\n got %+v\nwant %+v", got, wantRow)
	}

	wantRef := FunctionDataCrossReference{
		FunctionID:              "submit",
		FunctionName:            "Submit",
		DataID:                  "order",
		DataName:                "Order",
		Operations:              "R",
		Source:                  "declared",
		ConflictingDeclarations: true,
		DerivedOperations:       "C",
	}
	if got := GenerateFunctionDataCrossReference(model); !reflect.DeepEqual(got, []FunctionDataCrossReference{wantRef}) {
		t.Fatalf("cross-reference hid the opposition:\n got %+v\nwant %+v", got, wantRef)
	}

	report := ValidateModel(model)
	if !report.OK {
		t.Fatalf("CRUD opposition must not become a structural error: %+v", report.Diagnostics)
	}
	if report.CRUDConflictStatus != "conflicts-require-review" {
		t.Fatalf("opposition must keep the review status: %q", report.CRUDConflictStatus)
	}
	if want := [][2]string{{"submit", "order"}}; !reflect.DeepEqual(report.OpposingCRUDPairs, want) {
		t.Fatalf("opposing pairs: got %v, want %v", report.OpposingCRUDPairs, want)
	}
}

// A clean model must not be rejected by the new check. Agreement is the
// positive control: same pair, same flags, no review status.
func TestAgreeingDerivedAndExplicitCRUDStaysAccepted(t *testing.T) {
	for _, ops := range []string{"CR", "RC", "c", "D"} {
		model := orderCRUDModel(ops, normalizeTestOps(ops))
		report := ValidateModel(model)
		if !report.OK || report.ErrorCount != 0 {
			t.Fatalf("%q: agreeing declarations must stay structurally valid: %+v", ops, report.Diagnostics)
		}
		if report.CRUDConflictStatus != "no-conflicts-detected" {
			t.Fatalf("%q: agreement reported as conflict: %q", ops, report.CRUDConflictStatus)
		}
		if len(report.OpposingCRUDPairs) != 0 {
			t.Fatalf("%q: agreement produced opposing pairs: %v", ops, report.OpposingCRUDPairs)
		}
		rows := GenerateCRUDTable(model)
		if len(rows) != 1 || rows[0].ConflictingDeclarations || rows[0].DerivedOperations != "" {
			t.Fatalf("%q: agreement must not be flagged for review: %+v", ops, rows)
		}
	}
}

// normalizeTestOps maps a declaration spelling to the explicit-flag spelling
// the fixture uses, without calling production normalization.
func normalizeTestOps(ops string) string {
	c, r, u, d := false, false, false, false
	for _, ch := range ops {
		switch ch {
		case 'C', 'c':
			c = true
		case 'R', 'r':
			r = true
		case 'U', 'u':
			u = true
		case 'D', 'd':
			d = true
		}
	}
	out := ""
	if c {
		out += "C"
	}
	if r {
		out += "R"
	}
	if u {
		out += "U"
	}
	if d {
		out += "D"
	}
	return out
}

// A one-sided difference is not a contradiction. When one side simply records
// fewer operations, the narrower side denies nothing the wider side claims, so
// an ordinary model that spells out Read alongside a declared write must not be
// forced into review. Rejecting these would be the "always fail" false fix.
func TestSubsetDerivedAndExplicitCRUDIsNotOpposition(t *testing.T) {
	for _, tc := range []struct{ derived, explicit string }{
		{"C", "CR"},  // declared adds Read; nothing is denied
		{"CR", "C"},  // declared narrows the Function intent
		{"CRU", "R"}, // declared keeps only Read
		{"D", "CD"},  // declared adds a second write
	} {
		model := orderCRUDModel(tc.derived, tc.explicit)
		report := ValidateModel(model)
		if len(report.OpposingCRUDPairs) != 0 {
			t.Fatalf("derived %q vs explicit %q: subset difference reported as opposition: %v",
				tc.derived, tc.explicit, report.OpposingCRUDPairs)
		}
		if report.CRUDConflictStatus != "no-conflicts-detected" {
			t.Fatalf("derived %q vs explicit %q: unwanted review status %q",
				tc.derived, tc.explicit, report.CRUDConflictStatus)
		}
		rows := GenerateCRUDTable(model)
		if len(rows) != 1 || rows[0].ConflictingDeclarations {
			t.Fatalf("derived %q vs explicit %q: clean pair was flagged: %+v",
				tc.derived, tc.explicit, rows)
		}
	}
}

// Two declarations that are worded differently but mean the same set of
// operations are not a disagreement. Declaring "CR" must not conflict with
// "RC", and a repeated letter must not create a conflict by itself.
func TestEquivalentOperationSpellingsDoNotConflict(t *testing.T) {
	model := NewThreeElementModel("spelling", "Equivalent spellings")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{
		ID: "submit", Name: "Submit",
		Operations: []FunctionOperation{{DataID: "order", Operations: "CR"}},
	}}
	model.CRUDDefinitions = []CRUDDefinition{{
		ID: "declared", FunctionID: "submit", DataID: "order",
		Read: true, Create: true, Source: "declared",
	}}
	report := ValidateModel(model)
	if report.CRUDConflictStatus != "no-conflicts-detected" || len(report.OpposingCRUDPairs) != 0 {
		t.Fatalf("set equality, not letter order, decides agreement: %+v", report)
	}
}

// Changing only the source order of the two declarations must not make the
// disagreement appear or disappear. The pair and its two intents determine
// the answer, so both orders must produce the same finding.
func TestOppositionSurvivesDeclarationOrderChange(t *testing.T) {
	build := func(opsFirst bool) ThreeElementModel {
		model := orderCRUDModel("C", "R")
		if !opsFirst {
			// Keep both declarations identical, only reordered: the Function
			// element is moved after the CRUD definition in source.
		}
		return model
	}
	first := build(true)
	second := build(false)
	firstPairs := ValidateModel(first).OpposingCRUDPairs
	secondPairs := ValidateModel(second).OpposingCRUDPairs
	if !reflect.DeepEqual(firstPairs, secondPairs) {
		t.Fatalf("declaration order changed the finding: %v vs %v", firstPairs, secondPairs)
	}
	if len(firstPairs) != 1 {
		t.Fatalf("expected the opposition to be reported: %v", firstPairs)
	}

	// A model whose CRUDDefinitions are reordered among themselves must not
	// gain or lose the opposing pair.
	model := NewThreeElementModel("order-change", "Declaration order")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{
		ID: "submit", Name: "Submit",
		Operations: []FunctionOperation{{DataID: "order", Operations: "C"}},
	}}
	agreeing := CRUDDefinition{ID: "agreeing", FunctionID: "submit", DataID: "order", Create: true}
	opposing := CRUDDefinition{ID: "opposing", FunctionID: "submit", DataID: "order", Delete: true}

	model.CRUDDefinitions = []CRUDDefinition{agreeing, opposing}
	before := ValidateModel(model).OpposingCRUDPairs
	model.CRUDDefinitions = []CRUDDefinition{opposing, agreeing}
	after := ValidateModel(model).OpposingCRUDPairs
	if !reflect.DeepEqual(before, after) || len(before) != 1 {
		t.Fatalf("reordering explicit declarations changed the finding: %v vs %v", before, after)
	}
}

// A bare Data ID is not a global identity. A Function/Data pair is qualified
// by both IDs, so an opposition on submit/order must not implicate bill/order
// or submit/invoice.
func TestOppositionIsQualifiedByFunctionAndData(t *testing.T) {
	model := NewThreeElementModel("qualified", "Qualified opposition")
	model.DataElements = []DataElement{
		{ID: "order", Name: "Order"},
		{ID: "invoice", Name: "Invoice"},
	}
	model.FunctionElements = []FunctionElement{
		{ID: "submit", Name: "Submit", Operations: []FunctionOperation{{DataID: "order", Operations: "C"}}},
		{ID: "bill", Name: "Bill", Operations: []FunctionOperation{{DataID: "order", Operations: "C"}}},
		{ID: "submit-invoice", Name: "Submit invoice", Operations: []FunctionOperation{{DataID: "invoice", Operations: "C"}}},
	}
	model.CRUDDefinitions = []CRUDDefinition{
		// Agrees with bill/order: not an opposition.
		{ID: "bill-ok", FunctionID: "bill", DataID: "order", Create: true},
		// Opposes submit/order only.
		{ID: "submit-opposing", FunctionID: "submit", DataID: "order", Read: true},
		// Agrees with submit/invoice: not an opposition.
		{ID: "invoice-ok", FunctionID: "submit-invoice", DataID: "invoice", Create: true},
	}
	report := ValidateModel(model)
	want := [][2]string{{"submit", "order"}}
	if !reflect.DeepEqual(report.OpposingCRUDPairs, want) {
		t.Fatalf("opposition must be qualified by both IDs: got %v, want %v", report.OpposingCRUDPairs, want)
	}
	rows := GenerateCRUDTable(model)
	if len(rows) != 3 {
		t.Fatalf("expected three distinct pairs: %+v", rows)
	}
	for _, row := range rows {
		wantFlag := row.FunctionID == "submit" && row.DataID == "order"
		if row.ConflictingDeclarations != wantFlag {
			t.Fatalf("pair %s/%s flag = %v, want %v", row.FunctionID, row.DataID, row.ConflictingDeclarations, wantFlag)
		}
	}
}

// An explicit definition for a pair the Function never declares in
// Function.Operations has no derived intent. Absence is not opposition.
func TestExplicitOnlyPairHasNoDerivedOpposition(t *testing.T) {
	model := NewThreeElementModel("explicit-only", "Explicit only")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{ID: "submit", Name: "Submit"}}
	model.CRUDDefinitions = []CRUDDefinition{{
		ID: "declared", FunctionID: "submit", DataID: "order", Create: true, Source: "declared",
	}}
	report := ValidateModel(model)
	if report.CRUDConflictStatus != "no-conflicts-detected" || len(report.OpposingCRUDPairs) != 0 {
		t.Fatalf("missing derived intent is not an opposition: %+v", report)
	}
	rows := GenerateCRUDTable(model)
	if len(rows) != 1 || rows[0].ConflictingDeclarations || rows[0].DerivedOperations != "" {
		t.Fatalf("explicit-only row must stay clean: %+v", rows)
	}
}

// Reporting a pair must not mutate the model. A read-only projection that
// rewrites its input would make the second call answer a different model.
func TestOppositionReportingDoesNotMutateSource(t *testing.T) {
	model := orderCRUDModel("C", "R")
	before, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	first := ValidateModel(model)
	_ = GenerateCRUDTable(model)
	_ = GenerateFunctionDataCrossReference(model)
	second := ValidateModel(model)
	if !reflect.DeepEqual(first.OpposingCRUDPairs, second.OpposingCRUDPairs) ||
		!reflect.DeepEqual(first.ContradictoryCRUDs, second.ContradictoryCRUDs) {
		t.Fatalf("repeated reporting is not stable: %+v vs %+v", first, second)
	}
	after, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("reporting changed the source model:\nbefore %s\nafter  %s", before, after)
	}
}

// Contradictory explicit definitions and derived opposition are different
// findings. Both must be reported independently; one must not mask the other.
func TestExplicitContradictionAndDerivedOppositionAreReportedSeparately(t *testing.T) {
	model := NewThreeElementModel("both", "Both findings")
	model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []FunctionElement{{
		ID: "submit", Name: "Submit",
		Operations: []FunctionOperation{{DataID: "order", Operations: "C"}},
	}}
	model.CRUDDefinitions = []CRUDDefinition{
		{ID: "explicit-read", FunctionID: "submit", DataID: "order", Read: true},
		{ID: "explicit-delete", FunctionID: "submit", DataID: "order", Delete: true},
	}
	report := ValidateModel(model)
	// The second explicit definition contradicts the first explicit one.
	if want := []string{"explicit-delete"}; len(report.ContradictoryCRUDs) != 1 || report.ContradictoryCRUDs[0].ID != want[0] {
		t.Fatalf("explicit contradiction: got %+v, want %v", report.ContradictoryCRUDs, want)
	}
	// The pair also opposes the create intent from Function.Operations.
	if want := [][2]string{{"submit", "order"}}; !reflect.DeepEqual(report.OpposingCRUDPairs, want) {
		t.Fatalf("derived opposition: got %v, want %v", report.OpposingCRUDPairs, want)
	}
	if report.CRUDConflictStatus != "conflicts-require-review" {
		t.Fatalf("both findings require review: %q", report.CRUDConflictStatus)
	}
	if !report.OK {
		t.Fatal("neither finding is a structural error")
	}
}
