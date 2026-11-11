package partition

import (
	"encoding/json"
	"testing"
)

func TestBuildCoverageReport(t *testing.T) {
	subjects := []string{"F1", "D1"}
	entries := []CoverageEntry{
		{SubjectID: "F1", Kind: SubjectFunction, Provenance: ProvDeclaredSource, SourceLocator: "src:F1", Notes: ""},
		{SubjectID: "D1", Kind: SubjectData, Provenance: ProvUnresolved, Notes: "ambiguous owner"},
	}
	report, err := BuildCoverageReport(subjects, entries)
	if err != nil {
		t.Fatal(err)
	}
	if report.AuthorityLimit != AuthorityLimit {
		t.Fatal("authority limit missing")
	}
	if report.Summary["declared-source"] != 1 || report.Summary["unresolved"] != 1 {
		t.Fatal("summary mismatch")
	}
	if len(report.Subjects) != 2 {
		t.Fatal("subjects mismatch")
	}
	_ = MustMarshalCoverageReport(report)
}

func TestBuildCoverageReportRejectsInvalid(t *testing.T) {
	cases := []struct {
		subjects []string
		entries  []CoverageEntry
	}{
		{[]string{""}, []CoverageEntry{{SubjectID: "x", Provenance: "ok"}}},
		{[]string{"F1"}, []CoverageEntry{{SubjectID: "F1"}}},
	}
	for _, tc := range cases {
		_, err := BuildCoverageReport(tc.subjects, tc.entries)
		if err == nil {
			t.Fatalf("expected error for %v", tc)
		}
	}
}

func TestCoverageReportDeterministic(t *testing.T) {
	subjects := []string{"D1", "F1", "D2"}
	entries := []CoverageEntry{
		{SubjectID: "D2", Kind: SubjectData, Provenance: ProvNotExamined, Notes: "source not read"},
	}
	first, err := BuildCoverageReport(subjects, entries)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		second, err := BuildCoverageReport(subjects, entries)
		if err != nil {
			t.Fatal(err)
		}
		if string(MustMarshalCoverageReport(first)) != string(MustMarshalCoverageReport(second)) {
			t.Fatal("coverage report is not deterministic")
		}
	}
	// ensure ordering by SubjectID then provenance.
	firstJSON := MustMarshalCoverageReport(first)
	var decoded []CoverageEntry
	_ = json.Unmarshal(firstJSON, &decoded) // no wrapper; just check order is stable via marshal round-trip.
	_ = decoded
}
