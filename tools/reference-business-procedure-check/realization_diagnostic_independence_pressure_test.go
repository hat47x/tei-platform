package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/workspace"
)

func TestCrossViewCheckerMeaningAndDiagnosticsDoNotDependOnRealizationLabels(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	before := evaluate(loaded.World, f)
	if err := checkExpectedDiagnostics(before.Diagnostics, ""); err != nil {
		t.Fatalf("baselineにblocking diagnosticがあります: %v\n%+v", err, before.Diagnostics)
	}

	changed := f
	changed.Steps = append([]step(nil), f.Steps...)
	for i := range changed.Steps {
		changed.Steps[i].Realization = "authoring-review-label-" + changed.Steps[i].ID
	}

	after := evaluate(loaded.World, changed)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("realization authoring labelだけでcross-view report / diagnosticsが変化しました:\nbefore=%+v\nafter=%+v", before, after)
	}
}

func TestCrossViewCheckerStillRejectsBrokenSemanticRefWhenRealizationLabelChanges(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	changed.Steps = append([]step(nil), f.Steps...)
	for i := range changed.Steps {
		if changed.Steps[i].ID == "BP-06" {
			changed.Steps[i].Realization = "not-security-looking-anymore"
			changed.Steps[i].Capabilities = []string{"security.missing_authority"}
		}
	}

	result := evaluate(loaded.World, changed)
	if err := checkExpectedDiagnostics(result.Diagnostics, "BUSINESS_CAPABILITY_REF_NOT_FOUND"); err != nil {
		t.Fatalf("realization labelから意味をguessせずexplicit semantic ref failureを検出できません: %v\n%+v", err, result.Diagnostics)
	}
}
