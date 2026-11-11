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
}
