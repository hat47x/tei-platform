package partition

import (
	"testing"

	"tei/internal/threeelement"
)

// These are source-model projection characterizations, not an Accepted
// Subsystem Owner, direct Function I/O, or runtime write witness.
// A CRUDDefinition with Source="declared" is still a caller declaration.
func TestThreeElementExplicitOnlyCRUDIsProjectedAsDeclaredOnly(t *testing.T) {
	model := threeelement.NewThreeElementModel("candidate", "Candidate")
	model.DataElements = []threeelement.DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []threeelement.FunctionElement{{ID: "submit", Name: "Submit"}}
	model.CRUDDefinitions = []threeelement.CRUDDefinition{{
		ID: "submit-order", FunctionID: "submit", DataID: "order",
		Update: true, Source: "declared",
	}}

	if diagnostics := model.Validate(); len(diagnostics) != 0 {
		t.Fatalf("positive control: the referenced Function/Data exist: %+v", diagnostics)
	}
	// Explicit-only CRUD is now projected. A visible cell still proves only
	// that the declaration was read, never that a write is authorized.
	rows := threeelement.GenerateCRUDTable(model)
	if len(rows) != 1 || !rows[0].Update || rows[0].Read || rows[0].Source != "declared" {
		t.Fatalf("explicit-only CRUD must be projected with its declared flags: %+v", rows)
	}
	if rows[0].ConflictingDeclarations {
		t.Fatalf("a single declaration is not a conflict: %+v", rows[0])
	}
	refs := threeelement.GenerateFunctionDataCrossReference(model)
	if len(refs) != 1 || refs[0].Operations != "U" || refs[0].Source != "declared" {
		t.Fatalf("explicit-only cross-reference must be projected: %+v", refs)
	}

	// Adding a FunctionOperation that opposes the declaration makes the cell
	// reviewable, but must not change which side is projected.
	model.FunctionElements[0].Operations = []threeelement.FunctionOperation{{
		DataID: "order", Operations: "R",
	}}
	rows = threeelement.GenerateCRUDTable(model)
	if len(rows) != 1 || !rows[0].Update || rows[0].Read || rows[0].Source != "declared" {
		t.Fatalf("explicit CRUD should override FunctionOperation once a row is reachable: %+v", rows)
	}
	if !rows[0].ConflictingDeclarations || rows[0].DerivedOperations != "R" {
		t.Fatalf("the overridden derived intent must stay visible: %+v", rows[0])
	}
	refs = threeelement.GenerateFunctionDataCrossReference(model)
	if len(refs) != 1 || refs[0].Operations != "U" || refs[0].Source != "declared" {
		t.Fatalf("explicit CRUD should override derived cross-reference: %+v", refs)
	}
	if !refs[0].ConflictingDeclarations || refs[0].DerivedOperations != "R" {
		t.Fatalf("the overridden derived intent must stay visible: %+v", refs[0])
	}
	report := threeelement.ValidateModel(model)
	if report.CRUDConflictStatus != "conflicts-require-review" {
		t.Fatalf("a projected override still requires review: %+v", report)
	}
}

func TestThreeElementConflictingDeclarationsAreNotSafeToCollapse(t *testing.T) {
	model := threeelement.NewThreeElementModel("candidate", "Candidate")
	model.DataElements = []threeelement.DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []threeelement.FunctionElement{{
		ID: "submit", Name: "Submit",
		Operations: []threeelement.FunctionOperation{{DataID: "order", Operations: "R"}},
	}}
	model.CRUDDefinitions = []threeelement.CRUDDefinition{
		{ID: "first", FunctionID: "submit", DataID: "order", Update: true, Source: "declared"},
		{ID: "second", FunctionID: "submit", DataID: "order", Delete: true, Source: "declared"},
	}

	// A separate helper detects a contradiction, but Validate's diagnostic
	// result does not include that result. Do not treat Validate OK as an
	// accepted unique Writer edge.
	if conflicts := model.ContradictoryCRUDs(); len(conflicts) != 1 {
		t.Fatalf("conflict must remain observable through its dedicated helper: %+v", conflicts)
	}
	if diagnostics := model.Validate(); len(diagnostics) != 0 {
		t.Fatalf("characterization changed: Validate now reports this conflict: %+v", diagnostics)
	}
	// The full report exposes the conflicting CRUDDefinition separately, but
	// OK is based only on Diagnostic errors, not on that conflict list.
	report := threeelement.ValidateModel(model)
	if !report.OK || report.ErrorCount != 0 || len(report.ContradictoryCRUDs) != 1 {
		t.Fatalf("full report must expose contradiction without an error status: %+v", report)
	}

	a := threeelement.GenerateCRUDTable(model)
	if len(a) != 1 || !a[0].Delete || a[0].Update {
		t.Fatalf("current projection should select the last declared pair: %+v", a)
	}
	x := threeelement.GenerateFunctionDataCrossReference(model)
	if len(x) != 1 || x[0].Operations != "D" {
		t.Fatalf("current cross-reference should collapse to last declaration: %+v", x)
	}

	model.CRUDDefinitions[0], model.CRUDDefinitions[1] =
		model.CRUDDefinitions[1], model.CRUDDefinitions[0]
	b := threeelement.GenerateCRUDTable(model)
	y := threeelement.GenerateFunctionDataCrossReference(model)
	if len(b) != 1 || !b[0].Update || b[0].Delete ||
		len(y) != 1 || y[0].Operations != "U" {
		t.Fatalf("declaration order changed the selected result without an accepted owner decision: rows=%+v refs=%+v", b, y)
	}

	// Neither one-row projection may be converted directly to partition.Write.
	// This test has no accepted Function Directness or Primary Data Owner.
}

// Repeated Function.Operations on one Data pair contribute a set of CRUD
// letters to a single review cell; they do not establish multiple write
// occurrences, a Primary Data Owner, or an actual runtime effect.
func TestThreeElementRepeatedFunctionOperationsDoNotDefineDistinctWrites(t *testing.T) {
	model := threeelement.NewThreeElementModel("candidate", "Candidate")
	model.DataElements = []threeelement.DataElement{{ID: "order", Name: "Order"}}
	model.FunctionElements = []threeelement.FunctionElement{{
		ID: "submit", Name: "Submit",
		Operations: []threeelement.FunctionOperation{
			{DataID: "order", Operations: "R"},
			{DataID: "order", Operations: "U"},
		},
	}}
	if diagnostics := model.Validate(); len(diagnostics) != 0 {
		t.Fatalf("positive control: repeated pair is structurally valid: %+v", diagnostics)
	}
	rows := threeelement.GenerateCRUDTable(model)
	if len(rows) != 1 || !rows[0].Read || !rows[0].Update ||
		rows[0].Create || rows[0].Delete || rows[0].Source != "derived" {
		t.Fatalf("Function/Data pair must preserve both declared letters in one review cell: %+v", rows)
	}
	refs := threeelement.GenerateFunctionDataCrossReference(model)
	if len(refs) != 1 || refs[0].Operations != "RU" {
		t.Fatalf("cross-reference must retain the pair's full read-only intent: %+v", refs)
	}

	model.FunctionElements[0].Operations[0], model.FunctionElements[0].Operations[1] =
		model.FunctionElements[0].Operations[1], model.FunctionElements[0].Operations[0]
	reordered := threeelement.GenerateFunctionDataCrossReference(model)
	reorderedRows := threeelement.GenerateCRUDTable(model)
	if len(reordered) != 1 || reordered[0].Operations != "RU" ||
		len(reorderedRows) != 1 || !reorderedRows[0].Read || !reorderedRows[0].Update {
		t.Fatalf("declaration order must not change the projected Function/Data intent: refs=%+v rows=%+v", reordered, reorderedRows)
	}
	// Neither the single cell nor the unioned CRUD letters assert the
	// existence of a direct write, a transaction, or an accepted Data Owner.
}
