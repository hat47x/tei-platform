package generatorextensionpoints

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func loadFixture(t *testing.T) Input {
	t.Helper()
	input, err := LoadInput(filepath.Join("testdata", "input-bound.json"))
	if err != nil {
		t.Fatalf("LoadInput: %v", err)
	}
	return input
}

func extensionReasons(report Report) []string {
	for _, extension := range report.Extensions {
		if extension.Point == "compute-derived" {
			return extension.Reasons
		}
	}
	return nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func reasonsContain(values []string, fragment string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, fragment) {
			return true
		}
	}
	return false
}

func TestBoundExtensionPasses(t *testing.T) {
	report := Check(loadFixture(t))
	if report.Status != StatusPassed {
		t.Fatalf("status = %q, want %q; problems=%v extensions=%+v", report.Status, StatusPassed, report.Problems, report.Extensions)
	}
	if len(report.Extensions) != 1 || report.Extensions[0].Status != ExtensionBound {
		t.Fatalf("extensions = %+v", report.Extensions)
	}
}

func TestRegenerationKeepsExtensionOutsideArtifact(t *testing.T) {
	report := Check(loadFixture(t))
	if report.Regeneration.Rounds != 3 {
		t.Errorf("rounds = %d, want 3", report.Regeneration.Rounds)
	}
	if !report.Regeneration.ArtifactStable || !report.Regeneration.ExtensionsStable {
		t.Errorf("regeneration = %+v", report.Regeneration)
	}
	if !report.Regeneration.ExtensionsOutsideArtifact {
		t.Errorf("extension must stay outside the artifact: %+v", report.Regeneration)
	}
	if contains(report.Regeneration.ArtifactPaths, "case/derived.go") {
		t.Errorf("case extension was placed inside the artifact: %v", report.Regeneration.ArtifactPaths)
	}
	if !contains(report.Regeneration.ExtensionPaths, "case/derived.go") {
		t.Errorf("extension path missing: %v", report.Regeneration.ExtensionPaths)
	}
}

func TestDefaultEqualsNoExtension(t *testing.T) {
	if !DefaultEqualsNoExtension(loadFixture(t)) {
		t.Fatal("a generator run with no extension must equal the declared default artifact")
	}
}

func TestContractVersionChangeIsConflict(t *testing.T) {
	input := loadFixture(t)
	input.CaseExtensions[0].Contract.Version = "2"
	report := Check(input)
	if report.Status != StatusConflict {
		t.Fatalf("status = %q, want conflict", report.Status)
	}
	if !reasonsContain(extensionReasons(report), "contract-version-changed") {
		t.Errorf("reasons = %v", extensionReasons(report))
	}
}

func TestContractOrderChangeIsConflict(t *testing.T) {
	input := loadFixture(t)
	input.CaseExtensions[0].Contract.Order = 99
	report := Check(input)
	if report.Status != StatusConflict {
		t.Fatalf("status = %q, want conflict", report.Status)
	}
	if !reasonsContain(extensionReasons(report), "contract-order-changed") {
		t.Errorf("reasons = %v", extensionReasons(report))
	}
}

func TestContractInputsChangeIsConflict(t *testing.T) {
	input := loadFixture(t)
	input.CaseExtensions[0].Contract.Inputs = []string{"amount", "tax"}
	report := Check(input)
	if report.Status != StatusConflict {
		t.Fatalf("status = %q, want conflict", report.Status)
	}
	if !reasonsContain(extensionReasons(report), "contract-inputs-changed") {
		t.Errorf("reasons = %v", extensionReasons(report))
	}
}

func TestDeletedExtensionPointIsConflict(t *testing.T) {
	input := loadFixture(t)
	input.Generator.ExtensionPoints[0].ID = "compute-derived-v2"
	report := Check(input)
	if report.Status != StatusConflict {
		t.Fatalf("status = %q, want conflict", report.Status)
	}
	if !contains(extensionReasons(report), "extension-point-not-declared") {
		t.Errorf("reasons = %v", extensionReasons(report))
	}
}

func TestExtensionInsideArtifactIsConflict(t *testing.T) {
	input := loadFixture(t)
	input.CaseExtensions[0].Path = "generated/order.go"
	report := Check(input)
	if report.Status != StatusConflict {
		t.Fatalf("status = %q, want conflict", report.Status)
	}
	if !contains(extensionReasons(report), "extension-inside-generated-artifact") {
		t.Errorf("reasons = %v", extensionReasons(report))
	}
	if report.Regeneration.ExtensionsOutsideArtifact {
		t.Error("overlap between extension and artifact must be reported")
	}
}

func TestDirectEditOutsideExtensionPointsIsReported(t *testing.T) {
	input := loadFixture(t)
	for i := range input.WorkspaceFiles {
		if input.WorkspaceFiles[i].Path == "generated/order.go" {
			input.WorkspaceFiles[i].Content = "package generated\n\n// edited by hand\n"
		}
	}
	report := Check(input)
	if report.Status != StatusConflict {
		t.Fatalf("status = %q, want conflict", report.Status)
	}
	found := false
	for _, edit := range report.DirectEdits {
		if edit.Path == "generated/order.go" && edit.Status == FileDirectEdit {
			found = true
		}
	}
	if !found {
		t.Fatalf("direct edit not reported: %+v", report.DirectEdits)
	}
}

func TestMissingGeneratedFileIsReported(t *testing.T) {
	input := loadFixture(t)
	var kept []ArtifactFile
	for _, file := range input.WorkspaceFiles {
		if file.Path == "generated/defaults.go" {
			continue
		}
		kept = append(kept, file)
	}
	input.WorkspaceFiles = kept
	report := Check(input)
	found := false
	for _, edit := range report.DirectEdits {
		if edit.Path == "generated/defaults.go" && edit.Status == FileMissing {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing generated file not reported: %+v", report.DirectEdits)
	}
	if report.Status != StatusConflict {
		t.Errorf("status = %q, want conflict", report.Status)
	}
}

func TestExtensionChangeIsAllowed(t *testing.T) {
	input := loadFixture(t)
	for i := range input.WorkspaceFiles {
		if input.WorkspaceFiles[i].Path == "case/derived.go" {
			input.WorkspaceFiles[i].Content = "package caseext\n\nfunc Derived(amount int) int { return amount + 30 }\n"
		}
	}
	report := Check(input)
	if report.Status != StatusPassed {
		t.Fatalf("status = %q, want passed; directEdits=%+v", report.Status, report.DirectEdits)
	}
	found := false
	for _, edit := range report.DirectEdits {
		if edit.Path == "case/derived.go" && edit.Status == FileExtensionChanged {
			found = true
		}
	}
	if !found {
		t.Fatalf("extension change not recorded: %+v", report.DirectEdits)
	}
}

func TestOutcomeEvidenceRecordsExtensionPresence(t *testing.T) {
	report := Check(loadFixture(t))
	if len(report.Outcomes) != 2 {
		t.Fatalf("outcomes = %+v", report.Outcomes)
	}
	got := map[string]OutcomeResult{}
	for _, outcome := range report.Outcomes {
		got[outcome.Point] = outcome
	}
	if !got["compute-derived"].ExtensionPresent {
		t.Error("compute-derived should have an extension")
	}
	if got["before-save"].ExtensionPresent {
		t.Error("before-save should not have an extension")
	}
	if got["compute-derived"].Default != "100" || got["compute-derived"].WithExtension != "120" {
		t.Errorf("outcome evidence = %+v", got["compute-derived"])
	}
}

func TestDuplicateExtensionPointOrderIsUnresolved(t *testing.T) {
	input := loadFixture(t)
	input.Generator.ExtensionPoints = append(input.Generator.ExtensionPoints, ExtensionPoint{
		ID: "extra", Version: "1", Order: 10,
		Inputs: []string{"x"}, Outputs: []string{"y"}, Default: "x",
	})
	report := Check(input)
	if report.Status != StatusUnresolved {
		t.Fatalf("status = %q, want unresolved", report.Status)
	}
}

func TestCheckDoesNotMutateInput(t *testing.T) {
	input := loadFixture(t)
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var original Input
	if err := json.Unmarshal(raw, &original); err != nil {
		t.Fatal(err)
	}
	_ = Check(input)
	if !reflect.DeepEqual(input, original) {
		t.Fatal("Check changed the input")
	}
}
