package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/workspace"
)

func TestPatternDerivedProjectionCannotReplaceManualBusinessAuthoring(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	result := evaluate(loaded.World, f)
	if err := checkExpectedDiagnostics(result.Diagnostics, ""); err != nil {
		t.Fatalf("baselineにblocking diagnosticがあります: %v", err)
	}

	authoringSteps := map[string]step{}
	for _, item := range f.Steps {
		authoringSteps[item.ID] = item
	}
	derivedSteps := map[string]bool{}
	for _, row := range result.BusinessActionPatternMeaningRows {
		derivedSteps[row.StepID] = true
	}

	for _, stepID := range []string{"BP-04", "BP-09"} {
		item, ok := authoringSteps[stepID]
		if !ok {
			t.Fatalf("manual Business Action %sがauthoring fixtureから失われています", stepID)
		}
		if item.Realization == "application-use" {
			t.Fatalf("%sがmanual/external residualではありません: %+v", stepID, item)
		}
		if derivedSteps[stepID] {
			t.Fatalf("%sをPattern-owned meaningから導出できるものとして扱っています", stepID)
		}
	}
}

func TestBusinessWordingChangeDoesNotPretendApplicationMeaningChanged(t *testing.T) {
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
	changed := false
	for i := range f.Steps {
		if f.Steps[i].ID == "BP-04" {
			f.Steps[i].BusinessAction = "decide whether surrounding business circumstances permit submission"
			changed = true
		}
	}
	if !changed {
		t.Fatal("test setupがBP-04を見つけられません")
	}
	after := evaluate(loaded.World, f)

	if !reflect.DeepEqual(before.BusinessActionPatternMeaningRows, after.BusinessActionPatternMeaningRows) {
		t.Fatalf("manual Business wording changeでPattern-derived Application meaningまで変化しました\nbefore=%+v\nafter=%+v", before.BusinessActionPatternMeaningRows, after.BusinessActionPatternMeaningRows)
	}
	if reflect.DeepEqual(before.BusinessActionFunctionRows, after.BusinessActionFunctionRows) {
		t.Fatal("Business wording changeがBusiness-side Projectionへ現れていません")
	}
	if !reflect.DeepEqual(before.Diagnostics, after.Diagnostics) {
		t.Fatalf("manual wording changeだけでcross-view diagnosticが変化しました\nbefore=%+v\nafter=%+v", before.Diagnostics, after.Diagnostics)
	}
}

func TestPurposeAndActorCandidatesRemainAuthoringMeaning(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	if f.Procedure.Purpose == "" {
		t.Fatal("Procedure purposeがauthoring fixtureにありません")
	}
	if f.Procedure.Trigger.Status == "" {
		t.Fatal("Procedure trigger candidateがauthoring fixtureにありません")
	}
	if f.Procedure.Actor.Status == "" || f.Procedure.Actor.Label == "" {
		t.Fatal("Procedure actor candidateがauthoring fixtureにありません")
	}
}
