package partition

import (
	"testing"

	"tei/internal/threeelement"
)

// These are source-model projection characterizations, not an Accepted
// Subsystem Owner, direct Function I/O, or runtime write witness.
// A CRUDDefinition with Source="declared" is still a caller declaration.
func TestThreeElementExplicitOnlyCRUDIsNotYetProjected(t *testing.T) {
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
	if rows := threeelement.GenerateCRUDTable(model); len(rows) != 0 {
		t.Fatalf("characterization changed: explicit-only CRUD is now visible; revisit admission contract: %+v", rows)
	}
	if refs := threeelement.GenerateFunctionDataCrossReference(model); len(refs) != 0 {
		t.Fatalf("characterization changed: explicit-only cross-reference is now visible: %+v", refs)
	}

	// Adding a FunctionOperation makes the same explicit declaration visible.
	// Its declared U overrides the FunctionOperation's R in both projections.
	model.FunctionElements[0].Operations = []threeelement.FunctionOperation{{
		DataID: "order", Operations: "R",
	}}
	rows := threeelement.GenerateCRUDTable(model)
	if len(rows) != 1 || !rows[0].Update || rows[0].Read || rows[0].Source != "declared" {
		t.Fatalf("explicit CRUD should override FunctionOperation once a row is reachable: %+v", rows)
	}
	refs := threeelement.GenerateFunctionDataCrossReference(model)
	if len(refs) != 1 || refs[0].Operations != "U" || refs[0].Source != "declared" {
		t.Fatalf("explicit CRUD should override derived cross-reference: %+v", refs)
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

// The two read-only projections also differ when a Function declares the
// same Data pair twice. FunctionOperation has no logical occurrence ID, so
// neither repeated rows nor the first cross-reference is an accepted count of
// independent writes.
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
		t.Fatalf("positive control: current validation allows repeated pair declarations: %+v", diagnostics)
	}
	rows := threeelement.GenerateCRUDTable(model)
	if len(rows) != 2 {
		t.Fatalf("current CRUD table emits one row per FunctionOperation even for the same pair: %+v", rows)
	}
	seenRead, seenUpdate := false, false
	for _, row := range rows {
		seenRead = seenRead || row.Read
		seenUpdate = seenUpdate || row.Update
	}
	if !seenRead || !seenUpdate {
		t.Fatalf("positive control: both source intents must remain in the table: %+v", rows)
	}
	refs := threeelement.GenerateFunctionDataCrossReference(model)
	if len(refs) != 1 || refs[0].Operations != "R" {
		t.Fatalf("current cross-reference keeps only first operation for this pair: %+v", refs)
	}

	model.FunctionElements[0].Operations[0], model.FunctionElements[0].Operations[1] =
		model.FunctionElements[0].Operations[1], model.FunctionElements[0].Operations[0]
	reordered := threeelement.GenerateFunctionDataCrossReference(model)
	if len(reordered) != 1 || reordered[0].Operations != "U" {
		t.Fatalf("same declarations in reversed order change cross-reference intent: %+v", reordered)
	}
	// Explicitly NOT an inference that there are two independent direct-write
	// occurrences or that any Data Primary Owner has been accepted.
}
