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
