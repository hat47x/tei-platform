package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/workspace"
)

type d4ApplicationPatternTargetOverlayProbe struct {
	Body                      d4BusinessProcedureStepProbe `json:"body"`
	ApplicationPatternTargets []string                     `json:"applicationPatternTargets,omitempty"`
}

func TestD4TargetOverlaySeparatesProtectedTargetFromFunctionRealization(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	candidate, err := deriveD4ApplicationPatternTargetOverlayProbe(f, "BP-06", []string{"order_entry.submit_order"})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.Body.ApplicationPatterns) != 0 {
		t.Fatalf("protected targetをD4 direct realizationへ変換しました: %+v", candidate.Body.ApplicationPatterns)
	}
	if !reflect.DeepEqual(candidate.Body.Capabilities, []string{"security.authorize"}) {
		t.Fatalf("supporting Authorization authorityが失われました: %v", candidate.Body.Capabilities)
	}
	if !reflect.DeepEqual(candidate.ApplicationPatternTargets, []string{"order_entry.submit_order"}) {
		t.Fatalf("protected Application targetが保持されていません: %v", candidate.ApplicationPatternTargets)
	}
	if err := checkExpectedDiagnostics(applicationPatternTargetDiagnostics(loaded.World, "BP-06", candidate.ApplicationPatternTargets), ""); err != nil {
		t.Fatalf("D4 target overlayをnormalized Worldでexact resolutionできません: %v", err)
	}

	result := evaluate(loaded.World, f)
	row, ok := targetPressureFunctionRowByStep(result, "BP-06")
	if !ok || !row.NoApplicationFunction {
		t.Fatalf("target overlayだけでBP-06をApplication Function realizationへ誤分類しました: %+v found=%v", row, ok)
	}
}

func TestD4TargetOverlayFailsClosedOnMissingOrAmbiguousTarget(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	missingCandidate, err := deriveD4ApplicationPatternTargetOverlayProbe(f, "BP-06", []string{"order_entry.missing_submit_target"})
	if err != nil {
		t.Fatal(err)
	}
	missing := applicationPatternTargetDiagnostics(loaded.World, "BP-06", missingCandidate.ApplicationPatternTargets)
	if err := checkExpectedDiagnostics(missing, "BUSINESS_APPLICATION_PATTERN_TARGET_REF_NOT_FOUND"); err != nil {
		t.Fatalf("D4 compositionでmissing protected targetをfail closedできません: %v\n%+v", err, missing)
	}

	ambiguousCandidate, err := deriveD4ApplicationPatternTargetOverlayProbe(f, "BP-06", []string{"order_entry.submit_order"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ambiguousCandidate.ApplicationPatternTargets, []string{"order_entry.submit_order"}) {
		t.Fatalf("ambiguous probe前にprotected target identityが変化しました: %v", ambiguousCandidate.ApplicationPatternTargets)
	}
	var ambiguous report
	checkReference(&ambiguous, "BP-06", "APPLICATION_PATTERN_TARGET", ambiguousCandidate.ApplicationPatternTargets[0], 2)
	sortDiagnostics(ambiguous.Diagnostics)
	if err := checkExpectedDiagnostics(ambiguous.Diagnostics, "BUSINESS_APPLICATION_PATTERN_TARGET_REF_AMBIGUOUS"); err != nil {
		t.Fatalf("D4 compositionでambiguous protected targetをfail closedできません: %v\n%+v", err, ambiguous.Diagnostics)
	}
}

func TestD4TargetOnlyChangeDoesNotChangeD4BodyButChangesOverlayMeaning(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	before, err := deriveD4ApplicationPatternTargetOverlayProbe(f, "BP-06", []string{"order_entry.submit_order"})
	if err != nil {
		t.Fatal(err)
	}
	after, err := deriveD4ApplicationPatternTargetOverlayProbe(f, "BP-06", []string{"order_entry.order_record"})
	if err != nil {
		t.Fatal(err)
	}
	if err := checkExpectedDiagnostics(applicationPatternTargetDiagnostics(loaded.World, "BP-06", after.ApplicationPatternTargets), ""); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before.Body, after.Body) {
		t.Fatal("protected targetだけの変更でD4 Step bodyまで変更しました")
	}
	if reflect.DeepEqual(before, after) {
		t.Fatal("protected Application target changeを独立cross-view meaning差として観測できません")
	}
}

func TestD4TargetOverlayDoesNotDeriveTargetFromAdjacentRealization(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := deriveD4ApplicationPatternTargetOverlayProbe(f, "BP-06", []string{"order_entry.submit_order"})
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
	after, err := deriveD4ApplicationPatternTargetOverlayProbe(changed, "BP-06", []string{"order_entry.submit_order"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("adjacent BP-07 realizationからBP-06 protected targetをderiveしました: before=%+v after=%+v", before, after)
	}
}

func TestD4TargetOverlayDoesNotIntroduceGenericRefRoleTaxonomy(t *testing.T) {
	overlayType := reflect.TypeOf(d4ApplicationPatternTargetOverlayProbe{})
	want := []string{"Body", "ApplicationPatternTargets"}
	if got := fieldNames(overlayType); !reflect.DeepEqual(got, want) {
		t.Fatalf("D4 target overlayがgeneric ref modelへ膨張しました: got=%v want=%v", got, want)
	}
	for _, forbidden := range []string{"Kind", "Role", "Relation", "Policy", "Effect", "Principal", "Permission"} {
		if _, found := overlayType.FieldByName(forbidden); found {
			t.Fatalf("D4 target overlayへ未証明のgeneric role/policy field %qを追加しました", forbidden)
		}
	}
}

func deriveD4ApplicationPatternTargetOverlayProbe(f fixture, stepID string, targets []string) (d4ApplicationPatternTargetOverlayProbe, error) {
	body, err := deriveD4BusinessProcedureProbe(f)
	if err != nil {
		return d4ApplicationPatternTargetOverlayProbe{}, err
	}
	index := -1
	for i, sourceStep := range f.Steps {
		if sourceStep.ID == stepID {
			index = i
			break
		}
	}
	if index < 0 || index >= len(body.Steps) {
		return d4ApplicationPatternTargetOverlayProbe{}, errProcedureMeaning("step not found: " + stepID)
	}
	return d4ApplicationPatternTargetOverlayProbe{
		Body:                      body.Steps[index],
		ApplicationPatternTargets: normalizedMeaningRefs(targets),
	}, nil
}
