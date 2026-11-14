package partition

import "testing"

// This is a characterization of what ComparePartitions can establish from
// caller-supplied IDs. It does not grant Accepted Owner, Source closure,
// directness, or identity continuity across a system migration.
func TestComparePartitionsStableElementSetsDoNotProveStableResponsibility(t *testing.T) {
	before := PartitionSnapshot{
		Label:       "before",
		Assignments: []Assignment{{Function: "checkout", Subsystem: "sales"}},
		Writes: []Write{{
			ID: "write-order", Function: "checkout", Data: "order",
			Owner: "sales", Operation: "U", Source: "source-before", Weight: 1,
		}},
	}

	cases := []struct {
		name                string
		after               Write
		wantAfterCross      int
		wantAfterInternalOp string
	}{
		{
			name: "crud-operation-changed-with-unchanged-name-and-membership",
			after: Write{
				ID: "write-order", Function: "checkout", Data: "order",
				Owner: "sales", Operation: "D", Source: "source-after", Weight: 1,
			},
			wantAfterCross: 0, wantAfterInternalOp: "D",
		},
		{
			name: "candidate-data-owner-changed-with-unchanged-name-and-membership",
			after: Write{
				ID: "write-order", Function: "checkout", Data: "order",
				Owner: "billing", Operation: "U", Source: "source-after", Weight: 1,
			},
			wantAfterCross: 1,
		},
		{
			name: "write-occurrence-and-source-changed-with-unchanged-name-and-membership",
			after: Write{
				ID: "new-occurrence", Function: "checkout", Data: "order",
				Owner: "sales", Operation: "U", Source: "revised-source", Weight: 1,
			},
			wantAfterCross: 0, wantAfterInternalOp: "U",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			after := PartitionSnapshot{
				Label:       "after",
				Assignments: []Assignment{{Function: "checkout", Subsystem: "sales"}},
				Writes:      []Write{tc.after},
			}
			got, err := ComparePartitions(before, after)
			if err != nil {
				t.Fatal(err)
			}
			if !got.DiffModelsConsistent || got.MovedFunctionCount != 0 ||
				len(got.AddedFunctions) != 0 || len(got.RemovedFunctions) != 0 ||
				len(got.AddedData) != 0 || len(got.RemovedData) != 0 {
				t.Fatalf("current set-only comparison should stay stable: %+v", got)
			}
			if got.BeforeCrossWrites != 0 || got.AfterCrossWrites != tc.wantAfterCross {
				t.Fatalf("positive control: boundary counters changed unexpectedly: %+v", got)
			}
			if tc.wantAfterInternalOp != "" && got.Reclassification.Internal[tc.wantAfterInternalOp] != 1 {
				t.Fatalf("positive control: expected after-only operation %s: %+v", tc.wantAfterInternalOp, got)
			}
			// Model-set consistency is not proof of unchanged direct CRUD,
			// Primary Owner, source-locator, or occurrence identity.
			// Before/After labels are snapshot titles, not subsystem names.
		})
	}
}

// Subsystem strings in the reference evaluator are structural membership IDs,
// NOT display names. Relabeling those keys as though they were UI labels is
// interpreted as a move, even when the caller also relabels the Owner.
func TestComparePartitionsChangingSubsystemKeyIsNotDisplayRename(t *testing.T) {
	before := PartitionSnapshot{
		Label:       "before",
		Assignments: []Assignment{{Function: "checkout", Subsystem: "sales"}},
		Writes: []Write{{
			ID: "write-order", Function: "checkout", Data: "order",
			Owner: "sales", Operation: "U", Source: "same-source", Weight: 1,
		}},
	}
	// A caller has renamed *structural keys*, not merely a display label.
	after := PartitionSnapshot{
		Label:       "after",
		Assignments: []Assignment{{Function: "checkout", Subsystem: "orders"}},
		Writes: []Write{{
			ID: "write-order", Function: "checkout", Data: "order",
			Owner: "orders", Operation: "U", Source: "same-source", Weight: 1,
		}},
	}
	report, err := ComparePartitions(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if !report.DiffModelsConsistent || report.MovedFunctionCount != 1 ||
		len(report.MovedFunctions) != 1 || report.MovedFunctions[0] != "checkout" ||
		report.BeforeCrossWrites != 0 || report.AfterCrossWrites != 0 {
		t.Fatalf("relabeling structural Subsystem keys is observed as a move: %+v", report)
	}

	// When only assignment was relabeled and the former Owner key remains,
	// a spurious-looking boundary also appears. This is why formal rename
	// must preserve qualified subsystem IDs and change labels elsewhere.
	after.Writes[0].Owner = "sales"
	mismatch, err := ComparePartitions(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if mismatch.MovedFunctionCount != 1 || mismatch.AfterCrossWrites != 1 ||
		!mismatch.DiffModelsConsistent {
		t.Fatalf("Owner/assignment key mismatch should count cross-boundary: %+v", mismatch)
	}
	// This test does NOT define or accept a new Subsystem name field or
	// automatically infer lineage, current authority, or rename semantics.
}

// A smaller CrossWriteCount is not necessarily an engineering improvement:
// dropping evidence from a partial after-snapshot produces exactly the same
// arithmetic as a real boundary repair. Likewise, discovering a prior edge
// can look like deterioration. Source closure is not represented by an empty
// Coverage map, and DiffModelsConsistent compares only element ID sets.
func TestComparePartitionsBoundaryDeltaDoesNotProveStructuralImprovement(t *testing.T) {
	assignment := []Assignment{{Function: "checkout", Subsystem: "sales"}}
	readAnchor := []ReadEdge{{ID: "read-order", Function: "checkout", Data: "order"}}
	crossWrite := []Write{{
		ID: "write-order", Function: "checkout", Data: "order",
		Owner: "billing", Operation: "U", Source: "source-order", Weight: 1,
	}}
	tests := []struct {
		name         string
		beforeWrites []Write
		afterWrites  []Write
		wantBefore   int
		wantAfter    int
	}{
		{
			name: "omitted-cross-write-looks-like-boundary-repair",
			beforeWrites: crossWrite, afterWrites: nil,
			wantBefore: 1, wantAfter: 0,
		},
		{
			name: "newly-observed-existing-cross-write-looks-like-deterioration",
			beforeWrites: nil, afterWrites: crossWrite,
			wantBefore: 0, wantAfter: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			before := PartitionSnapshot{
				Label: "before", Assignments: assignment,
				Writes: tc.beforeWrites, ReadEdges: readAnchor,
			}
			after := PartitionSnapshot{
				Label: "after", Assignments: assignment,
				Writes: tc.afterWrites, ReadEdges: readAnchor,
			}
			report, err := ComparePartitions(before, after)
			if err != nil {
				t.Fatal(err)
			}
			if report.BeforeCrossWrites != tc.wantBefore ||
				report.AfterCrossWrites != tc.wantAfter ||
				report.BoundaryDelta != tc.wantAfter-tc.wantBefore {
				t.Fatalf("positive control: counts must follow supplied edges: %+v", report)
			}
			if !report.DiffModelsConsistent ||
				report.MovedFunctionCount != 0 ||
				report.NotExaminedCount != 0 ||
				report.UnresolvedCount != 0 {
				t.Fatalf("ID sets and absent coverage metadata do not expose the lost observation: %+v", report)
			}
			// These reports are compatible with both a real engineering change
			// and a changed observation window. Only a separate qualified
			// Source/Reader coverage claim can distinguish those worlds.
		})
	}
}
