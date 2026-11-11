package partition

import (
	"testing"
)

func TestBuildComparisonReport(t *testing.T) {
	candidates := []Candidate{
		{ID: "cand-002", Assignments: []Assignment{{Function: "F1", Subsystem: "sales"}}, Evaluation: Evaluation{CrossWriteCount: 1, CrossWriteWeight: 1, CrossReadCount: 2, Cohesion: 0.9, Sizes: map[string]int{"sales": 1}}},
		{ID: "cand-001", Assignments: []Assignment{{Function: "F1", Subsystem: "sales"}}, Evaluation: Evaluation{CrossWriteCount: 1, CrossWriteWeight: 1, CrossReadCount: 2, Cohesion: 0.9, Sizes: map[string]int{"sales": 1}}},
	}
	report, err := BuildComparisonReport(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(report.Rows))
	}
	if report.Rows[0].ID != "cand-001" {
		t.Fatalf("expected first row cand-001, got %s", report.Rows[0].ID)
	}
	if report.Rows[1].ID != "cand-002" {
		t.Fatalf("expected second row cand-002, got %s", report.Rows[1].ID)
	}
	if report.SchemaVersion != "tei.subsystem-partition.comparison/v0" {
		t.Fatalf("unexpected schema version %s", report.SchemaVersion)
	}
	if report.AuthorityLimit != AuthorityLimit {
		t.Fatalf("missing authority limit")
	}
}

func TestBuildComparisonReportOrdering(t *testing.T) {
	candidates := []Candidate{
		{ID: "cand-003", Assignments: []Assignment{{Function: "F1", Subsystem: "sales"}}, Evaluation: Evaluation{CrossWriteCount: 3, CrossWriteWeight: 3, CrossReadCount: 6, Cohesion: 0.727273, Sizes: map[string]int{"sales": 5}}},
		{ID: "cand-001", Assignments: []Assignment{{Function: "F1", Subsystem: "sales"}}, Evaluation: Evaluation{CrossWriteCount: 1, CrossWriteWeight: 1, CrossReadCount: 6, Cohesion: 0.909091, Sizes: map[string]int{"sales": 4}}},
		{ID: "cand-002", Assignments: []Assignment{{Function: "F1", Subsystem: "sales"}}, Evaluation: Evaluation{CrossWriteCount: 1, CrossWriteWeight: 1, CrossReadCount: 6, Cohesion: 0.909091, Sizes: map[string]int{"sales": 4}}},
		{ID: "cand-004", Assignments: []Assignment{{Function: "F1", Subsystem: "sales"}}, Evaluation: Evaluation{CrossWriteCount: 2, CrossWriteWeight: 2, CrossReadCount: 6, Cohesion: 0.818182, Sizes: map[string]int{"sales": 4}}},
	}
	report, err := BuildComparisonReport(candidates)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"cand-001", "cand-002", "cand-004", "cand-003"}
	if len(report.Rows) != len(want) {
		t.Fatalf("expected %d rows, got %d", len(want), len(report.Rows))
	}
	for i, id := range want {
		if report.Rows[i].ID != id {
			t.Fatalf("row %d: got %s want %s", i, report.Rows[i].ID, id)
		}
	}
}

func TestBuildComparisonReportFromFixture(t *testing.T) {
	p, err := fixtureProblem()
	if err != nil {
		t.Fatal(err)
	}
	candidates, status := GenerateCandidates(p)
	if status != "exhaustive-proven" {
		t.Fatalf("expected exhaustive-proven, got %s", status)
	}
	report, err := BuildComparisonReport(candidates)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Rows) != len(candidates) {
		t.Fatalf("row count %d != candidate count %d", len(report.Rows), len(candidates))
	}
	for i := 1; i < len(report.Rows); i++ {
		prev, cur := report.Rows[i-1], report.Rows[i]
		if cur.CrossWriteCount < prev.CrossWriteCount {
			t.Fatalf("rows not sorted by cross write count at %d", i)
		}
		if cur.CrossWriteCount == prev.CrossWriteCount && cur.ID < prev.ID {
			t.Fatalf("ties not sorted by id at %d", i)
		}
	}
	for _, row := range report.Rows {
		if row.Status == "" {
			t.Fatalf("row %s missing status", row.ID)
		}
		if row.FunctionCount == 0 {
			t.Fatalf("row %s missing function count", row.ID)
		}
	}
}

func snapshotFromCandidate(label string, c Candidate, coverage map[string]ProvenanceKind) PartitionSnapshot {
	writes := make([]Write, 0)
	_ = writes
	return PartitionSnapshot{
		Label:       label,
		Assignments: c.Assignments,
		Coverage:    coverage,
	}
}

func TestComparePartitionsPlacementOnly(t *testing.T) {
	p, err := fixtureProblem()
	if err != nil {
		t.Fatal(err)
	}
	candidates, _ := GenerateCandidates(p)
	if len(candidates) < 2 {
		t.Fatalf("need at least 2 candidates, got %d", len(candidates))
	}
	coverage := map[string]ProvenanceKind{
		"sales.order":      ProvDeclaredSource,
		"billing.invoice":  ProvParsedExtracted,
		"inventory.stock":  ProvDeclaredSource,
		"crm.customer":     ProvDeclaredSource,
		"unknown.subject":  ProvUnresolved,
		"unexamined.thing": ProvNotExamined,
	}
	before := snapshotFromCandidate("before", candidates[0], coverage)
	before.Writes = p.Writes
	before.ReadEdges = p.ReadEdges
	after := snapshotFromCandidate("after", candidates[1], coverage)
	after.Writes = p.Writes
	after.ReadEdges = p.ReadEdges

	cmp, err := ComparePartitions(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if cmp.SchemaVersion != "tei.subsystem-partition.comparison-partition/v0" {
		t.Fatalf("unexpected schema version %s", cmp.SchemaVersion)
	}
	if cmp.AuthorityLimit != AuthorityLimit {
		t.Fatalf("missing authority limit")
	}
	if !cmp.DiffModelsConsistent {
		t.Fatalf("same fixture candidates must be diff-models consistent")
	}
	if len(cmp.AddedFunctions) != 0 || len(cmp.RemovedFunctions) != 0 {
		t.Fatalf("same fixture must not add or remove functions: +%v -%v", cmp.AddedFunctions, cmp.RemovedFunctions)
	}
	if len(cmp.AddedData) != 0 || len(cmp.RemovedData) != 0 {
		t.Fatalf("same fixture must not add or remove data: +%v -%v", cmp.AddedData, cmp.RemovedData)
	}
	if cmp.MovedFunctionCount != len(cmp.MovedFunctions) {
		t.Fatalf("moved count mismatch")
	}
	if cmp.UnresolvedCount != 1 {
		t.Fatalf("expected 1 unresolved subject, got %d", cmp.UnresolvedCount)
	}
	if cmp.NotExaminedCount != 1 {
		t.Fatalf("expected 1 not-examined subject, got %d", cmp.NotExaminedCount)
	}
	if cmp.AfterCrossWrites == 0 {
		t.Fatalf("fixture writes must produce cross writes")
	}
	if len(cmp.Reclassification.Cross) == 0 || len(cmp.Reclassification.Internal) == 0 {
		t.Fatalf("reclassification must split internal and cross: %+v", cmp.Reclassification)
	}
	for _, note := range cmp.Notes {
		if note == "function-or-data-set-differs-between-snapshots" {
			t.Fatalf("unexpected inconsistency note for same-fixture comparison")
		}
	}
}

func TestComparePartitionsDetectsSetChange(t *testing.T) {
	before := PartitionSnapshot{
		Label:       "before",
		Assignments: []Assignment{{Function: "F1", Subsystem: "sales"}},
		Writes:      []Write{{ID: "w1", Function: "F1", Data: "sales.order", Owner: "sales", Operation: "C", Weight: 1}},
	}
	after := PartitionSnapshot{
		Label:       "after",
		Assignments: []Assignment{{Function: "F2", Subsystem: "sales"}},
		Writes:      []Write{{ID: "w2", Function: "F2", Data: "billing.invoice", Owner: "billing", Operation: "C", Weight: 1}},
	}
	cmp, err := ComparePartitions(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if cmp.DiffModelsConsistent {
		t.Fatalf("function set change must be reported as inconsistent")
	}
	if len(cmp.AddedFunctions) != 1 || cmp.AddedFunctions[0] != "F2" {
		t.Fatalf("expected added F2, got %v", cmp.AddedFunctions)
	}
	if len(cmp.RemovedFunctions) != 1 || cmp.RemovedFunctions[0] != "F1" {
		t.Fatalf("expected removed F1, got %v", cmp.RemovedFunctions)
	}
	if len(cmp.AddedData) != 1 || cmp.AddedData[0] != "billing.invoice" {
		t.Fatalf("expected added billing.invoice, got %v", cmp.AddedData)
	}
	if len(cmp.RemovedData) != 1 || cmp.RemovedData[0] != "sales.order" {
		t.Fatalf("expected removed sales.order, got %v", cmp.RemovedData)
	}
	if cmp.MovedFunctionCount != 0 {
		t.Fatalf("no shared function means no movement, got %d", cmp.MovedFunctionCount)
	}
}
