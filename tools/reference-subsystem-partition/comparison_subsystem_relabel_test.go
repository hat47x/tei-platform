package partition

import "testing"

// A bijective relabeling of caller-defined subsystem identifiers can retain
// the entire candidate Function/Data/write topology. ComparePartitions cannot
// distinguish that representation change from actual reassignment, because it
// has no accepted old-to-new subsystem correspondence.
func TestComparePartitionsCountsSubsystemIDReencodingAsFunctionMovement(t *testing.T) {
	before := PartitionSnapshot{
		Label: "legacy",
		Assignments: []Assignment{
			{Function: "submit", Subsystem: "sales"},
			{Function: "bill", Subsystem: "billing"},
		},
		Writes: []Write{
			{ID: "w-order", Function: "submit", Data: "order", Owner: "sales", Operation: "U", Weight: 1},
			{ID: "w-invoice", Function: "bill", Data: "invoice", Owner: "billing", Operation: "C", Weight: 1},
		},
	}
	after := PartitionSnapshot{
		Label: "candidate",
		Assignments: []Assignment{
			{Function: "submit", Subsystem: "order-management"},
			{Function: "bill", Subsystem: "invoicing"},
		},
		Writes: []Write{
			{ID: "w-order", Function: "submit", Data: "order", Owner: "order-management", Operation: "U", Weight: 1},
			{ID: "w-invoice", Function: "bill", Data: "invoice", Owner: "invoicing", Operation: "C", Weight: 1},
		},
	}
	report, err := ComparePartitions(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if report.MovedFunctionCount != 2 || !report.DiffModelsConsistent ||
		report.BeforeCrossWrites != 0 || report.AfterCrossWrites != 0 ||
		report.BoundaryDelta != 0 {
		t.Fatalf("current ID-string comparison must expose its limited meaning: %+v", report)
	}
	if len(report.AddedData) != 0 || len(report.RemovedData) != 0 ||
		len(report.AddedFunctions) != 0 || len(report.RemovedFunctions) != 0 {
		t.Fatalf("bijective subsystem ID relabeling must not modify Function/Data sets: %+v", report)
	}
	// The synthetic bijection {sales -> order-management,
	// billing -> invoicing} is a caller hypothesis, NOT proof of
	// continued Subsystem Identity, inherited Grant, or accepted rename.
}

// Snapshot.Label is a label for the comparison snapshot, not the
// canonical display name for a particular Subsystem or its identity.
func TestComparePartitionsSnapshotLabelChangeIsNotSubsystemRename(t *testing.T) {
	assignments := []Assignment{{Function: "submit", Subsystem: "sales"}}
	writes := []Write{{ID: "w-order", Function: "submit", Data: "order", Owner: "sales", Operation: "U", Weight: 1}}
	before := PartitionSnapshot{
		Label: "販売管理 (legacy observation)", Assignments: assignments, Writes: writes,
	}
	after := PartitionSnapshot{
		Label: "受注管理 (candidate view)", Assignments: assignments, Writes: writes,
	}
	report, err := ComparePartitions(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if report.MovedFunctionCount != 0 || report.BoundaryDelta != 0 ||
		!report.DiffModelsConsistent {
		t.Fatalf("snapshot display label must not be treated as structure: %+v", report)
	}
	if report.BeforeLabel != before.Label || report.AfterLabel != after.Label {
		t.Fatalf("snapshot labels must remain descriptive metadata: %+v", report)
	}
	// Even an unchanged structure does not prove owner acceptance of a
	// Subsystem display-name change or make Source/API/Grant changes safe.
}
