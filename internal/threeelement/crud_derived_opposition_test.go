package threeelement

import (
	"encoding/json"
	"reflect"
	"strings"
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
	for _, ch := range strings.ToUpper(explicitOps) {
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

// A narrowing or widening is not reviewed, but the overridden intent must still
// be readable. Without this, a Consumer of the CRUD table alone would conclude
// that a function performs exactly the declared operations and nothing else,
// while the Function model says more. Keeping the intent visible serves the
// no-unnecessary-rejection rule and the no-hidden-overwrite rule at once.
func TestRefinementKeepsOverriddenDerivedIntentVisible(t *testing.T) {
	for _, tc := range []struct{ derived, explicit string }{
		{"CRUD", "R"}, // declared narrows the Function intent
		{"C", "CR"},   // declared widens the Function intent
	} {
		model := orderCRUDModel(tc.derived, tc.explicit)
		rows := GenerateCRUDTable(model)
		if len(rows) != 1 {
			t.Fatalf("derived %q explicit %q: expected one row, got %+v", tc.derived, tc.explicit, rows)
		}
		if rows[0].DerivedOperations != wantNormalized(tc.derived) {
			t.Fatalf("derived %q explicit %q: overridden intent erased: %+v",
				tc.derived, tc.explicit, rows[0])
		}
		if rows[0].ConflictingDeclarations {
			t.Fatalf("derived %q explicit %q: a refinement must not be flagged for review: %+v",
				tc.derived, tc.explicit, rows[0])
		}
		refs := GenerateFunctionDataCrossReference(model)
		if len(refs) != 1 || refs[0].DerivedOperations != wantNormalized(tc.derived) {
			t.Fatalf("derived %q explicit %q: cross-reference erased the intent: %+v",
				tc.derived, tc.explicit, refs)
		}
		if refs[0].ConflictingDeclarations {
			t.Fatalf("derived %q explicit %q: cross-reference flagged a refinement: %+v",
				tc.derived, tc.explicit, refs[0])
		}
		if status := ValidateModel(model).CRUDConflictStatus; status != "no-conflicts-detected" {
			t.Fatalf("derived %q explicit %q: refinement forced review: %q", tc.derived, tc.explicit, status)
		}
	}
}

// A declaration that states no operations must not silently zero an operation
// set the Function declared. Asserting nothing is not a refinement of anything.
func TestEmptyExplicitDefinitionDoesNotSilentlyEraseIntent(t *testing.T) {
	model := orderCRUDModel("CRUD", "")
	rows := GenerateCRUDTable(model)
	if len(rows) != 1 {
		t.Fatalf("expected one row: %+v", rows)
	}
	if rows[0].Create || rows[0].Read || rows[0].Update || rows[0].Delete {
		t.Fatalf("the empty declaration still controls the projected flags: %+v", rows[0])
	}
	if rows[0].DerivedOperations != "CRUD" {
		t.Fatalf("the erased intent must stay visible: %+v", rows[0])
	}
	if !rows[0].ConflictingDeclarations {
		t.Fatalf("an empty override of a declared set requires review: %+v", rows[0])
	}
	report := ValidateModel(model)
	if report.CRUDConflictStatus != "conflicts-require-review" {
		t.Fatalf("empty override was reported as clean: %q", report.CRUDConflictStatus)
	}
	if !report.OK {
		t.Fatal("an empty declaration is still only a warning, not a structural error")
	}
}

// The Business→Data projection must resolve the Data name from the Data index
// and use the same operation resolution as the Function→Data section. Two
// sections of one result disagreeing about the same pair is a Consumer trap.
func TestBusinessDataCrossReferenceUsesDataIndexAndResolvedOperations(t *testing.T) {
	model := NewThreeElementModel("crossref", "Cross-reference identity")
	model.DataElements = []DataElement{{ID: "order", Name: "Order data"}}
	// A business element with the same bare ID but a different name must not
	// supply the Data name.
	model.BusinessElements = []BusinessElement{{
		ID: "order", Name: "Sales partner",
		Activities: []BusinessActivity{{ID: "a1", Name: "Accept", FunctionIDs: []string{"submit"}}},
	}}
	model.FunctionElements = []FunctionElement{{
		ID: "submit", Name: "Submit",
		Operations: []FunctionOperation{{DataID: "order", Operations: "CRUD"}},
	}}
	model.CRUDDefinitions = []CRUDDefinition{{
		ID: "declared", FunctionID: "submit", DataID: "order", Read: true, Source: "declared",
	}}

	rows := GenerateBusinessDataCrossReference(model)
	if len(rows) != 1 {
		t.Fatalf("expected one Business→Data row: %+v", rows)
	}
	if rows[0].DataName != "Order data" {
		t.Fatalf("Business→Data used a non-Data index: %q", rows[0].DataName)
	}
	// The declared read must be reported here too, not the raw CRUD intent.
	if rows[0].Operations != "R" {
		t.Fatalf("Business→Data disagrees with Function→Data about the same pair: %q", rows[0].Operations)
	}
	refs := GenerateFunctionDataCrossReference(model)
	if len(refs) != 1 || refs[0].Operations != rows[0].Operations {
		t.Fatalf("one result carries two answers for the same pair: businessData=%q functionData=%+v",
			rows[0].Operations, refs)
	}
}

// wantNormalized spells the canonical operation order for a fixture input.
func wantNormalized(ops string) string {
	out := ""
	for _, ch := range []string{"C", "R", "U", "D"} {
		if strings.Contains(ops, ch) {
			out += ch
		}
	}
	return out
}

// Equivalent operation spellings are not a disagreement. "CR" and "RC" name the
// same set, a lowercase spelling names the same set, and repeating a letter does
// not make a distinct claim. Only the set of operations decides agreement.
func TestEquivalentOperationSpellingsDoNotConflict(t *testing.T) {
	for _, spelling := range []struct {
		derived  string
		explicit string
	}{
		{"CR", "CR"},
		{"CR", "RC"},
		{"RC", "CR"},
		{"CR", "RCCR"},
		{"cr", "CR"},
		{"D", "d"},
	} {
		model := orderCRUDModel(spelling.derived, spelling.explicit)
		report := ValidateModel(model)
		if report.CRUDConflictStatus != "no-conflicts-detected" || len(report.OpposingCRUDPairs) != 0 {
			t.Fatalf("derived %q explicit %q: set equality, not spelling, decides agreement: %+v",
				spelling.derived, spelling.explicit, report)
		}
		rows := GenerateCRUDTable(model)
		if len(rows) != 1 || rows[0].DerivedOperations != "" {
			t.Fatalf("derived %q explicit %q: equal intent must not be carried as an override: %+v",
				spelling.derived, spelling.explicit, rows)
		}
	}
}

// Changing only the source order of the declarations must not make the
// disagreement appear or disappear. The pair and its two intents determine
// the answer, so every permutation must produce the same finding.
func TestOppositionSurvivesDeclarationOrderChange(t *testing.T) {
	agreeing := CRUDDefinition{ID: "agreeing", FunctionID: "submit", DataID: "order", Create: true}
	opposing := CRUDDefinition{ID: "opposing", FunctionID: "submit", DataID: "order", Delete: true}
	derivedOp := FunctionOperation{DataID: "order", Operations: "C"}

	// Duplicated derived intents must also not change the answer: the union of
	// operations for a pair is what the Function claims, independent of how many
	// times the pair is listed.
	build := func(order []CRUDDefinition, derivedOps []FunctionOperation) ThreeElementModel {
		model := NewThreeElementModel("order-change", "Declaration order")
		model.DataElements = []DataElement{{ID: "order", Name: "Order"}}
		model.FunctionElements = []FunctionElement{{
			ID: "submit", Name: "Submit", Operations: derivedOps,
		}}
		model.CRUDDefinitions = order
		return model
	}

	variants := []struct {
		name       string
		order      []CRUDDefinition
		derivedOps []FunctionOperation
	}{
		{"agreeing-first", []CRUDDefinition{agreeing, opposing}, []FunctionOperation{derivedOp}},
		{"opposing-first", []CRUDDefinition{opposing, agreeing}, []FunctionOperation{derivedOp}},
		{"repeated-derived", []CRUDDefinition{agreeing, opposing}, []FunctionOperation{derivedOp, derivedOp}},
	}

	var want [][2]string
	for i, variant := range variants {
		got := ValidateModel(build(variant.order, variant.derivedOps)).OpposingCRUDPairs
		if len(got) != 1 || got[0] != [2]string{"submit", "order"} {
			t.Fatalf("%s: lost the finding: %v", variant.name, got)
		}
		if i == 0 {
			want = got
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: changed the finding: %v vs %v", variant.name, got, want)
		}
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
