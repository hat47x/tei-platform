package operations

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/semantic/validationeval"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

const (
	markedExampleSource = "semantic/organization/transfer-request.tei.md"
	markedExampleRule = "organization.transfer_request.allocation_positive"
)

const markedExampleTable = "\n<!-- tei: example of organization.transfer_request -->\n" +
	"| allocation | organization.transfer_request.allocation_positive |\n" +
	"| --- | --- |\n" +
	"| 2.5 | satisfied |\n" +
	"| 0 | not-satisfied |\n" +
	"| | undecided |\n" +
	"| -1 | not-satisfied |\n"

// The product Reader must keep the marked Data definition and the adjacent
// Rule in canonical Meaning, but treat a record example as independent
// authoring evidence. Editing an expected answer must change the diagnostic,
// not the Rule or the Data's canonical model.
func TestMarkedDataTableRuleExampleDiagnosesAnExpectedCellWithoutChangingMeaning(t *testing.T) {
	fixture := filepath.Join("..", "workspace", "testdata", "tei-md-data-table")
	root := t.TempDir()
	copyFixture := func(path string) []byte {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		return raw
	}
	copyFixture("tei.workspace.yaml")
	original := copyFixture(markedExampleSource)
	destination := filepath.Join(root, filepath.FromSlash(markedExampleSource))
	dataSubject := sourceindex.Subject{Kind: "semantic-data", ID: "organization.transfer_request"}
	ruleSubject := sourceindex.Subject{Kind: "semantic-validation", ID: markedExampleRule}

	beforeData, err := ReadWorkspaceSubject(root, dataSubject)
	if err != nil {
		t.Fatal(err)
	}
	beforeRule, err := ReadWorkspaceSubject(root, ruleSubject)
	if err != nil {
		t.Fatal(err)
	}
	if baseline, err := workspace.Load(root); err != nil || len(baseline.RuleExamples) != 0 {
		t.Fatalf("unmarked reference table must stay prose: examples=%+v err=%v", baseline.RuleExamples, err)
	}

	validSource := append(append([]byte(nil), original...), []byte(markedExampleTable)...)
	if err := os.WriteFile(destination, validSource, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("selected product Reader did not load marked example: %v", err)
	}
	if len(loaded.AuthoringSources) != 1 ||
		loaded.AuthoringSources[0].Language.LanguageID != "tei.authoring" ||
		loaded.AuthoringSources[0].Language.ContractVersion != "v1" ||
		loaded.AuthoringSources[0].Path != markedExampleSource ||
		loaded.AuthoringSources[0].Definitions != 2 {
		t.Fatalf("record example was treated as a Definition or wrong Reader selected: %+v", loaded.AuthoringSources)
	}
	if len(loaded.RuleExamples) != 1 ||
		loaded.RuleExamples[0].Data != dataSubject.ID ||
		loaded.RuleExamples[0].File != markedExampleSource ||
		len(loaded.RuleExamples[0].Rows) != 4 ||
		len(loaded.RuleExamples[0].Expectations) != 4 {
		t.Fatalf("the example's independent authoring role was not retained: %+v", loaded.RuleExamples)
	}
	if diagnostics := CheckRuleExamples(loaded, validationeval.BuiltinImplementations()); len(diagnostics) != 0 {
		t.Fatalf("independently written positive, zero, NULL and negative expectations disagree: %+v", diagnostics)
	}
	accepted, err := ValidateWorkspace(root)
	if err != nil || accepted.HasErrors() || accepted.Direct != nil || len(accepted.Diagnostics) != 0 {
		t.Fatalf("valid product example failed validation: %+v, err=%v", accepted, err)
	}
	afterData, err := ReadWorkspaceSubject(root, dataSubject)
	if err != nil {
		t.Fatal(err)
	}
	afterRule, err := ReadWorkspaceSubject(root, ruleSubject)
	if err != nil {
		t.Fatal(err)
	}
	for _, comparison := range []struct {
		name string
		before, after workspaceSubjectReadView
	}{
		{"data", workspaceSubjectReadView{beforeData.Inspection.Canonical, beforeData.Inspection.WorldFingerprint,
			beforeData.Inspection.SourceIndexFingerprint, beforeData.Inspection.Sources.References},
			workspaceSubjectReadView{afterData.Inspection.Canonical, afterData.Inspection.WorldFingerprint,
				afterData.Inspection.SourceIndexFingerprint, afterData.Inspection.Sources.References}},
		{"rule", workspaceSubjectReadView{beforeRule.Inspection.Canonical, beforeRule.Inspection.WorldFingerprint,
			beforeRule.Inspection.SourceIndexFingerprint, beforeRule.Inspection.Sources.References},
			workspaceSubjectReadView{afterRule.Inspection.Canonical, afterRule.Inspection.WorldFingerprint,
				afterRule.Inspection.SourceIndexFingerprint, afterRule.Inspection.Sources.References}},
	} {
		if !bytes.Equal(comparison.before.canonical, comparison.after.canonical) ||
			comparison.before.world != comparison.after.world ||
			comparison.before.source == comparison.after.source ||
			!reflect.DeepEqual(comparison.before.references, comparison.after.references) {
			t.Fatalf("example changed canonical %s meaning or reused stale Source: %+v", comparison.name, comparison)
		}
	}
	if accepted.WorldFingerprint != afterRule.Inspection.WorldFingerprint ||
		accepted.SourceIndexFingerprint != afterRule.Inspection.SourceIndexFingerprint {
		t.Fatal("product Validate and Inspector disagree on the current example Source")
	}

	// Change only a row's authored expected answer. The Data and Rule are
	// unchanged; the failure must point to that cell, not the Rule definition.
	const correct = "| 0 | not-satisfied |"
	const wrong = "| 0 | satisfied |"
	if bytes.Count(validSource, []byte(correct)) != 1 {
		t.Fatal("the independently authored expected row is no longer unique")
	}
	wrongSource := bytes.Replace(validSource, []byte(correct), []byte(wrong), 1)
	if err := os.WriteFile(destination, wrongSource, 0o644); err != nil {
		t.Fatal(err)
	}
	mismatch, err := ValidateWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	if mismatch.Direct != nil || !mismatch.HasErrors() || len(mismatch.Diagnostics) != 1 {
		t.Fatalf("one wrong example must create exactly one rule-example finding: %+v", mismatch)
	}
	finding := mismatch.Diagnostics[0]
	if finding.Diagnostic.Code != RuleExampleMismatchCode ||
		finding.Diagnostic.Subject == nil ||
		finding.Diagnostic.Subject.Kind != ruleSubject.Kind ||
		finding.Diagnostic.Subject.ID != ruleSubject.ID ||
		!strings.Contains(finding.Diagnostic.Message, "answers not-satisfied") ||
		len(finding.Sources) != 1 ||
		finding.Sources[0].Role != sourceindex.RoleAuthoring ||
		finding.Sources[0].ArtifactPath != markedExampleSource {
		t.Fatalf("wrong example was not attributed to its authoring expectation: %+v", finding)
	}
	anchor := bytes.Index(wrongSource, []byte(wrong)) + len("| 0 | ")
	if anchor < len("| 0 | ") || finding.Sources[0].Range.Start.Offset != anchor {
		t.Fatalf("example finding points away from its expectation cell: %+v, want %d", finding.Sources[0].Range, anchor)
	}
	if err := finding.Sources[0].Range.Validate(wrongSource); err != nil {
		t.Fatal(err)
	}
	currentRule, err := ReadWorkspaceSubject(root, ruleSubject)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(currentRule.Inspection.Canonical, afterRule.Inspection.Canonical) ||
		currentRule.Inspection.WorldFingerprint != afterRule.Inspection.WorldFingerprint ||
		currentRule.Inspection.SourceIndexFingerprint == afterRule.Inspection.SourceIndexFingerprint ||
		mismatch.WorldFingerprint != afterRule.Inspection.WorldFingerprint ||
		mismatch.SourceIndexFingerprint != currentRule.Inspection.SourceIndexFingerprint {
		t.Fatal("example correction changed Meaning or retained stale Source Index")
	}
}

type workspaceSubjectReadView struct {
	canonical []byte
	world string
	source string
	references []sourceindex.Reference
}
