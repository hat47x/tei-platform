package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

type canonicalBusinessProcedureMeaningWithoutRealizationProbe struct {
	ID      string                                                  `json:"id"`
	Purpose string                                                  `json:"purpose"`
	Trigger candidateRef                                            `json:"trigger"`
	Actor   candidateRef                                            `json:"actor"`
	Steps   []canonicalBusinessProcedureStepWithoutRealizationProbe `json:"steps"`
}

type canonicalBusinessProcedureStepWithoutRealizationProbe struct {
	ID                  string   `json:"id"`
	BusinessAction      string   `json:"businessAction"`
	ApplicationPatterns []string `json:"applicationPatterns,omitempty"`
	Data                []string `json:"data,omitempty"`
	Relations           []string `json:"relations,omitempty"`
	Validations         []string `json:"validations,omitempty"`
	Capabilities        []string `json:"capabilities,omitempty"`
}

func TestThinCanonicalCandidateCanSubtractRealizationField(t *testing.T) {
	stepType := reflect.TypeOf(canonicalBusinessProcedureStepWithoutRealizationProbe{})
	want := []string{"ID", "BusinessAction", "ApplicationPatterns", "Data", "Relations", "Validations", "Capabilities"}
	if got := fieldNames(stepType); !reflect.DeepEqual(got, want) {
		t.Fatalf("realization subtraction後のstep field境界が変化しました: got=%v want=%v", got, want)
	}
}

func TestRealizationLabelOnlyChangeDoesNotHaveToBeCanonicalBusinessMeaningChange(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := deriveCanonicalBusinessProcedureMeaningWithoutRealizationProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	found := false
	for i := range changed.Steps {
		if changed.Steps[i].ID == "BP-04" {
			changed.Steps[i].Realization = "manual-review-label-reworded"
			found = true
		}
	}
	if !found {
		t.Fatal("test setupがBP-04を見つけられません")
	}
	after, err := deriveCanonicalBusinessProcedureMeaningWithoutRealizationProbe(changed)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(before, after) {
		t.Fatal("realization review labelだけの変更をthin Canonical Business meaning変更へ固定しました")
	}
}

func TestManualBusinessActionSurvivesWithoutRealizationField(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	meaning, err := deriveCanonicalBusinessProcedureMeaningWithoutRealizationProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	step, ok := canonicalMeaningStepWithoutRealization(meaning, "BP-04")
	if !ok {
		t.Fatal("BP-04 manual Business Actionがrealization subtractionで失われました")
	}
	if step.BusinessAction == "" {
		t.Fatal("manual Business Actionのauthoritative wordingが失われました")
	}
	if len(step.ApplicationPatterns) != 0 || len(step.Capabilities) != 0 {
		t.Fatalf("realizationを外した代わりにmanual stepへfake Function / Capabilityを付与しました: %+v", step)
	}
}

func TestBusinessActionChangeStillChangesMeaningAfterRealizationSubtraction(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := deriveCanonicalBusinessProcedureMeaningWithoutRealizationProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	for i := range changed.Steps {
		if changed.Steps[i].ID == "BP-04" {
			changed.Steps[i].BusinessAction = "decide whether surrounding business circumstances permit submission"
		}
	}
	after, err := deriveCanonicalBusinessProcedureMeaningWithoutRealizationProbe(changed)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(before, after) {
		t.Fatal("realization subtraction後にBusiness Action changeまで失いました")
	}
}

func TestExplicitApplicationPatternReferenceStillChangesMeaningAfterRealizationSubtraction(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := deriveCanonicalBusinessProcedureMeaningWithoutRealizationProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	for i := range changed.Steps {
		if changed.Steps[i].ID == "BP-01" {
			changed.Steps[i].ApplicationPatterns = []string{"order_entry.recent_orders"}
		}
	}
	after, err := deriveCanonicalBusinessProcedureMeaningWithoutRealizationProbe(changed)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(before, after) {
		t.Fatal("realization subtraction後にexplicit Application Pattern ref changeまで失いました")
	}
}

func deriveCanonicalBusinessProcedureMeaningWithoutRealizationProbe(f fixture) (canonicalBusinessProcedureMeaningWithoutRealizationProbe, error) {
	base, err := deriveCanonicalBusinessProcedureMeaningProbe(f)
	if err != nil {
		return canonicalBusinessProcedureMeaningWithoutRealizationProbe{}, err
	}
	result := canonicalBusinessProcedureMeaningWithoutRealizationProbe{
		ID:      base.ID,
		Purpose: base.Purpose,
		Trigger: base.Trigger,
		Actor:   base.Actor,
		Steps:   make([]canonicalBusinessProcedureStepWithoutRealizationProbe, 0, len(base.Steps)),
	}
	for _, item := range base.Steps {
		result.Steps = append(result.Steps, canonicalBusinessProcedureStepWithoutRealizationProbe{
			ID:                  item.ID,
			BusinessAction:      item.BusinessAction,
			ApplicationPatterns: append([]string(nil), item.ApplicationPatterns...),
			Data:                append([]string(nil), item.Data...),
			Relations:           append([]string(nil), item.Relations...),
			Validations:         append([]string(nil), item.Validations...),
			Capabilities:        append([]string(nil), item.Capabilities...),
		})
	}
	return result, nil
}

func canonicalMeaningStepWithoutRealization(meaning canonicalBusinessProcedureMeaningWithoutRealizationProbe, id string) (canonicalBusinessProcedureStepWithoutRealizationProbe, bool) {
	for _, item := range meaning.Steps {
		if item.ID == id {
			return item, true
		}
	}
	return canonicalBusinessProcedureStepWithoutRealizationProbe{}, false
}
