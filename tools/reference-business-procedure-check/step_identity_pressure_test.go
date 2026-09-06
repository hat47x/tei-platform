package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

type canonicalBusinessProcedureBodyWithoutStepIDProbe struct {
	ID      string                                         `json:"id"`
	Purpose string                                         `json:"purpose"`
	Trigger candidateRef                                   `json:"trigger"`
	Actor   candidateRef                                   `json:"actor"`
	Steps   []canonicalBusinessProcedureStepWithoutIDProbe `json:"steps"`
}

type canonicalBusinessProcedureStepWithoutIDProbe struct {
	BusinessAction      string   `json:"businessAction"`
	ApplicationPatterns []string `json:"applicationPatterns,omitempty"`
	Data                []string `json:"data,omitempty"`
	Relations           []string `json:"relations,omitempty"`
	Validations         []string `json:"validations,omitempty"`
	Capabilities        []string `json:"capabilities,omitempty"`
}

func TestProcedureBodyMeaningCanBeProjectedWithoutStepID(t *testing.T) {
	stepType := reflect.TypeOf(canonicalBusinessProcedureStepWithoutIDProbe{})
	want := []string{"BusinessAction", "ApplicationPatterns", "Data", "Relations", "Validations", "Capabilities"}
	if got := fieldNames(stepType); !reflect.DeepEqual(got, want) {
		t.Fatalf("Step identity subtraction後のbody field境界が変化しました: got=%v want=%v", got, want)
	}
}

func TestReferenceLocalStepKeyRenameNeedNotChangeProcedureBodyMeaning(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := deriveCanonicalBusinessProcedureBodyWithoutStepIDProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	found := false
	for i := range changed.Steps {
		if changed.Steps[i].ID == "BP-04" {
			changed.Steps[i].ID = "BP-04-renamed"
			found = true
		}
	}
	if !found {
		t.Fatal("test setupがBP-04を見つけられません")
	}
	after, err := deriveCanonicalBusinessProcedureBodyWithoutStepIDProbe(changed)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(before, after) {
		t.Fatal("Reference-local Step key renameだけでProcedure body meaningを変更しました")
	}
}

func TestOrderedBusinessActionChangeStillChangesBodyMeaningWithoutStepID(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := deriveCanonicalBusinessProcedureBodyWithoutStepIDProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	changed.Steps = append([]step(nil), f.Steps...)
	changed.Steps[0], changed.Steps[1] = changed.Steps[1], changed.Steps[0]
	after, err := deriveCanonicalBusinessProcedureBodyWithoutStepIDProbe(changed)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(before, after) {
		t.Fatal("Step IDを外した結果、Business Actionのordered sequenceまで失いました")
	}
}

func TestExistingHandoffRequiresStableStepAddressEvenIfBodyDoesNot(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}

	primary := fixtures["order-entry.order-handling"]
	found := false
	for i := range primary.Steps {
		if primary.Steps[i].ID == "BP-09" {
			primary.Steps[i].ID = "BP-09-renamed"
			found = true
		}
	}
	if !found {
		t.Fatal("test setupがBP-09を見つけられません")
	}
	fixtures["order-entry.order-handling"] = primary

	if _, err := deriveCanonicalBusinessHandoffProbes(set, fixtures); err == nil {
		t.Fatal("handoff endpointのstale Step keyを名称類似や位置で補完しました")
	}
}

func TestStepAddressabilityDoesNotImplyRuntimeWorkflowIdentity(t *testing.T) {
	endpointType := reflect.TypeOf(canonicalBusinessHandoffEndpointProbe{})
	for _, forbidden := range []string{"Runtime", "Execution", "Instance", "Token", "State"} {
		if _, found := endpointType.FieldByName(forbidden); found {
			t.Fatalf("Business handoff endpointへruntime workflow identity %qが侵入しました", forbidden)
		}
	}
}

func deriveCanonicalBusinessProcedureBodyWithoutStepIDProbe(f fixture) (canonicalBusinessProcedureBodyWithoutStepIDProbe, error) {
	base, err := deriveCanonicalBusinessProcedureMeaningWithoutRealizationProbe(f)
	if err != nil {
		return canonicalBusinessProcedureBodyWithoutStepIDProbe{}, err
	}
	result := canonicalBusinessProcedureBodyWithoutStepIDProbe{
		ID:      base.ID,
		Purpose: base.Purpose,
		Trigger: base.Trigger,
		Actor:   base.Actor,
		Steps:   make([]canonicalBusinessProcedureStepWithoutIDProbe, 0, len(base.Steps)),
	}
	for _, item := range base.Steps {
		result.Steps = append(result.Steps, canonicalBusinessProcedureStepWithoutIDProbe{
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
