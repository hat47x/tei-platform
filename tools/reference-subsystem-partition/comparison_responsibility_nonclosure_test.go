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
