package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func realDecisionIndexFixture(t *testing.T) decisionIndexProjection {
	t.Helper()
	path := filepath.Join("..", "..", "reference", "workspace", ".tei", "decisions",
		"a043abc84702feb4a4558e2fbfd6fa17bd60867075f65d126d648998a11f2523.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record decisionRecordExcerpt
	if err := json.Unmarshal([]byte(`{"decision":`+string(raw)+`}`), &record); err != nil {
		t.Fatal(err)
	}
	record.Path = ".tei/decisions/a043abc84702feb4a4558e2fbfd6fa17bd60867075f65d126d648998a11f2523.json"
	record.Status = "superseded"
	record.SourceEdits = "intact"
	record.ReviewBindingStatus = "verified"
	record.DecisionActorStatus = "unresolved"
	record.DecisionAuthorityStatus = "unresolved"
	return decisionIndexProjection{
		Schema: decisionIndexSchema, ModelRevision: "sha256:report-revision-not-authority",
		Records: []decisionRecordExcerpt{record},
	}
}

func attachFixtureIndex(t *testing.T, report *WorkspaceAssessment, index decisionIndexProjection) error {
	t.Helper()
	encoded, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	return AttachDecisionHistory(report, bytes.NewReader(encoded))
}

func TestActualValidationLinksHistoricalDecisionWithoutHumanAuthority(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	got, err := InspectValidation(root, "order_entry", "sales.order_line.line_number_positive")
	if err != nil {
		t.Fatal(err)
	}
	if err := attachFixtureIndex(t, &got, realDecisionIndexFixture(t)); err != nil {
		t.Fatal(err)
	}
	if got.DecisionHistoryStatus != "reported-source-matched" || len(got.DecisionHistoryLinks) != 1 {
		t.Fatalf("expected a source-bound historical Decision link: %+v", got)
	}
	if got.DecisionHistoryLinks[0].RecordStatus != "superseded" || got.DecisionHistoryLinks[0].ReviewBindingStatus != "verified" {
		t.Fatalf("Decision Index status was not preserved: %+v", got.DecisionHistoryLinks)
	}
	if got.DecisionHistoryLinks[0].DecisionAuthorityStatus != "unresolved" {
		t.Fatal("an unverifiable actor was silently made authoritative")
	}
	if got.RuleMeaning != "canonical-validation-declared" || got.ExecutionAllowed {
		t.Fatal("historical Decision changed canonical meaning or granted execution")
	}
	hasHumanGap := false
	for _, reason := range got.Unresolved {
		hasHumanGap = hasHumanGap || reason == "human-acceptance-not-verified"
	}
	if !hasHumanGap {
		t.Fatal("historical Decision improperly cleared human acceptance gap")
	}
}

func TestHistoryRejectsUnrelatedAndChangedSources(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	cases := []struct {
		name     string
		mutate   func(*decisionIndexProjection)
		expected string
	}{
		{"unrelated", func(idx *decisionIndexProjection) {
			idx.Records[0].Decision.SemanticDiff[0].ID = "sales.order_line.another_rule"
		}, "not-matched"},
		{"wrong-source-bytes", func(idx *decisionIndexProjection) {
			idx.Records[0].Decision.SourceEdits[1].AfterFingerprint = "sha256:another-source"
		}, "recorded-subject-source-not-current"},
		{"source-index-denied", func(idx *decisionIndexProjection) {
			idx.Records[0].SourceEdits = "diverged"
		}, "recorded-subject-source-not-current"},
		{"rejected", func(idx *decisionIndexProjection) {
			idx.Records[0].Decision.Decision = "rejected"
		}, "not-matched"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, err := InspectValidation(root, "order_entry", "sales.order_line.line_number_positive")
			if err != nil {
				t.Fatal(err)
			}
			index := realDecisionIndexFixture(t)
			tc.mutate(&index)
			if err := attachFixtureIndex(t, &report, index); err != nil {
				t.Fatal(err)
			}
			if report.DecisionHistoryStatus != tc.expected || len(report.DecisionHistoryLinks) != 0 {
				t.Fatalf("unqualified historical Decision was linked: %+v", report)
			}
			if report.ExecutionAllowed {
				t.Fatal("historical Decision granted execution")
			}
		})
	}
}

func TestDecisionIndexRejectsWrongSchemaAndTrailingJSON(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	report, err := InspectValidation(root, "order_entry", "sales.order_line.line_number_positive")
	if err != nil {
		t.Fatal(err)
	}
	for _, encoded := range []string{
		`{"schema":"wrong","modelRevision":"R1","records":[]}`,
		`{"schema":"tei.reference.decision-index/v0","modelRevision":"R1","records":[]} {}`,
		`{"schema":"tei.reference.decision-index/v0","modelRevision":"","records":[]}`,
	} {
		candidate := report
		if err := AttachDecisionHistory(&candidate, bytes.NewBufferString(encoded)); err == nil {
			t.Fatalf("invalid index accepted: %s", encoded)
		}
	}
}

func TestDecisionHistoryCLIReportIsReadOnlyAndSourceBound(t *testing.T) {
	index := realDecisionIndexFixture(t)
	data, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	inputFile := filepath.Join(t.TempDir(), "shared-decision-index.json")
	if err := os.WriteFile(inputFile, data, 0o600); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join("..", "..", "reference", "workspace")
	var output bytes.Buffer
	err = runCLI([]string{
		"-workspace", root,
		"-application", "order_entry",
		"-subject-kind", "semantic-validation",
		"-subject-id", "sales.order_line.line_number_positive",
		"-decision-index", inputFile,
	}, &output)
	if err != nil {
		t.Fatal(err)
	}
	var report WorkspaceAssessment
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.DecisionHistoryStatus != "reported-source-matched" || len(report.DecisionHistoryLinks) != 1 ||
		report.ExecutionAllowed || report.RuleMeaning != "canonical-validation-declared" {
		t.Fatalf("unexpected Decision History projection: %+v", report)
	}
	output.Reset()
	if err := runCLI([]string{
		"-workspace", root, "-application", "order_entry",
		"-subject-kind", "application-pattern", "-subject-id", "order_entry.submit_order",
		"-decision-index", inputFile,
	}, &output); err == nil {
		t.Fatal("Decision Index was silently accepted for unsupported subject kind")
	}
	if output.Len() != 0 {
		t.Fatal("invalid Decision Index routing emitted a successful-looking report")
	}
}
