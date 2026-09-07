package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/ir"
	"tei/internal/workspace"
)

type applicationPatternTargetAwareStepProbe struct {
	Body                      canonicalBusinessProcedureStepWithoutIDProbe `json:"body"`
	ApplicationPatternTargets []string                                     `json:"applicationPatternTargets,omitempty"`
}

func TestApplicationPatternTargetPressureSeparatesTargetFromRealization(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	candidate, err := deriveApplicationPatternTargetAwareStepProbe(f, "BP-06", []string{"order_entry.submit_order"})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.Body.ApplicationPatterns) != 0 {
		t.Fatalf("protected targetをBP-06のApplication Function realizationへ変換しました: %+v", candidate.Body.ApplicationPatterns)
	}
	if !reflect.DeepEqual(candidate.Body.Capabilities, []string{"security.authorize"}) {
		t.Fatalf("supporting Authorization authorityが失われました: %+v", candidate.Body.Capabilities)
	}
	if !reflect.DeepEqual(candidate.ApplicationPatternTargets, []string{"order_entry.submit_order"}) {
		t.Fatalf("protected Application targetが保持されていません: %+v", candidate.ApplicationPatternTargets)
	}

	diagnostics := applicationPatternTargetDiagnostics(loaded.World, "BP-06", candidate.ApplicationPatternTargets)
	if err := checkExpectedDiagnostics(diagnostics, ""); err != nil {
		t.Fatalf("protected Application targetをnormalized Worldでexact resolutionできません: %v\n%+v", err, diagnostics)
	}

	result := evaluate(loaded.World, f)
	row, ok := targetPressureFunctionRowByStep(result, "BP-06")
	if !ok {
		t.Fatal("BP-06のBusiness Action / Function rowがありません")
	}
	if !row.NoApplicationFunction {
		t.Fatalf("target refだけでBP-06をApplication Function realizationへ誤分類しました: %+v", row)
	}
}

func TestApplicationPatternTargetPressureFailsClosedOnMissingOrAmbiguousTarget(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	missing := applicationPatternTargetDiagnostics(loaded.World, "BP-06", []string{"order_entry.missing_submit_target"})
	if err := checkExpectedDiagnostics(missing, "BUSINESS_APPLICATION_PATTERN_TARGET_REF_NOT_FOUND"); err != nil {
		t.Fatalf("missing protected targetをfail closedできません: %v\n%+v", err, missing)
	}

	var ambiguous report
	checkReference(&ambiguous, "BP-06", "APPLICATION_PATTERN_TARGET", "order_entry.submit_order", 2)
	sortDiagnostics(ambiguous.Diagnostics)
	if err := checkExpectedDiagnostics(ambiguous.Diagnostics, "BUSINESS_APPLICATION_PATTERN_TARGET_REF_AMBIGUOUS"); err != nil {
		t.Fatalf("ambiguous protected targetをfail closedできません: %v\n%+v", err, ambiguous.Diagnostics)
	}
}

func TestApplicationPatternTargetOnlyChangeIsCrossViewMeaningChangeWithoutRealizationChange(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	before, err := deriveApplicationPatternTargetAwareStepProbe(f, "BP-06", []string{"order_entry.submit_order"})
	if err != nil {
		t.Fatal(err)
	}
	after, err := deriveApplicationPatternTargetAwareStepProbe(f, "BP-06", []string{"order_entry.order_record"})
	if err != nil {
		t.Fatal(err)
	}

	if err := checkExpectedDiagnostics(applicationPatternTargetDiagnostics(loaded.World, "BP-06", before.ApplicationPatternTargets), ""); err != nil {
		t.Fatal(err)
	}
	if err := checkExpectedDiagnostics(applicationPatternTargetDiagnostics(loaded.World, "BP-06", after.ApplicationPatternTargets), ""); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before.Body, after.Body) {
		t.Fatal("protected targetだけの変更でBP-06のBusiness body / realization meaningまで変更しました")
	}
	if reflect.DeepEqual(before, after) {
		t.Fatal("protected Application target changeをcross-view meaning差として観測できません")
	}
	if len(after.Body.ApplicationPatterns) != 0 {
		t.Fatal("target changeをApplication Function realization changeへ変換しました")
	}
}

func TestApplicationPatternTargetPressureDoesNotDeriveTargetFromAdjacentStep(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	before, err := deriveApplicationPatternTargetAwareStepProbe(f, "BP-06", []string{"order_entry.submit_order"})
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	changed.Steps = append([]step(nil), f.Steps...)
	for i := range changed.Steps {
		if changed.Steps[i].ID == "BP-07" {
			changed.Steps[i].ApplicationPatterns = []string{"order_entry.order_record"}
		}
	}
	after, err := deriveApplicationPatternTargetAwareStepProbe(changed, "BP-06", []string{"order_entry.submit_order"})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(before, after) {
		t.Fatalf("adjacent BP-07 realizationからBP-06 targetをderiveしました: before=%+v after=%+v", before, after)
	}
}

func TestApplicationPatternTargetPressureDoesNotImportAuthorizationPolicyShape(t *testing.T) {
	candidateType := reflect.TypeOf(applicationPatternTargetAwareStepProbe{})
	want := []string{"Body", "ApplicationPatternTargets"}
	if got := fieldNames(candidateType); !reflect.DeepEqual(got, want) {
		t.Fatalf("target-aware test-only candidateが膨張しました: got=%v want=%v", got, want)
	}
	for _, forbidden := range []string{"Policy", "Role", "Organization", "Effect", "Principal", "Permission"} {
		if _, found := candidateType.FieldByName(forbidden); found {
			t.Fatalf("Business target pressureへAuthorization policy field %qを逆流させました", forbidden)
		}
	}
}

func deriveApplicationPatternTargetAwareStepProbe(f fixture, stepID string, targets []string) (applicationPatternTargetAwareStepProbe, error) {
	model, err := deriveNormalizedBusinessProcedureWithStepAddressProbe(f)
	if err != nil {
		return applicationPatternTargetAwareStepProbe{}, err
	}
	steps := addressedD3StepsByLocalKey(model)
	body, ok := steps[stepID]
	if !ok {
		return applicationPatternTargetAwareStepProbe{}, errProcedureMeaning("step not found: " + stepID)
	}
	return applicationPatternTargetAwareStepProbe{
		Body:                      body,
		ApplicationPatternTargets: normalizedMeaningRefs(targets),
	}, nil
}

func applicationPatternTargetDiagnostics(world ir.World, stepID string, targets []string) []diagnostic {
	counts := patternCounts(world)
	result := report{}
	for _, ref := range targets {
		checkReference(&result, stepID, "APPLICATION_PATTERN_TARGET", ref, counts[ref])
	}
	sortDiagnostics(result.Diagnostics)
	return result.Diagnostics
}

func targetPressureFunctionRowByStep(result report, stepID string) (functionRow, bool) {
	for _, row := range result.BusinessActionFunctionRows {
		if row.StepID == stepID {
			return row, true
		}
	}
	return functionRow{}, false
}
