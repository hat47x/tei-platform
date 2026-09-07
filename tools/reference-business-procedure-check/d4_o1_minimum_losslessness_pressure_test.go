package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/workspace"
)

type d4O1StepProbe struct {
	Body                        d4BusinessProcedureStepProbe `json:"body"`
	ProtectedApplicationPattern string                       `json:"protectedApplicationPattern,omitempty"`
}

type d4O1ProcedureProbe struct {
	ID      string          `json:"id"`
	Purpose string          `json:"purpose"`
	Trigger candidateRef    `json:"trigger"`
	Actor   candidateRef    `json:"actor"`
	Steps   []d4O1StepProbe `json:"steps"`
}

func TestD4AloneIsLossyForAcceptedProtectedTargetOnlyChange(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	beforeD4, err := deriveD4BusinessProcedureProbe(f)
	if err != nil {
		t.Fatal(err)
	}
	afterD4, err := deriveD4BusinessProcedureProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	beforeTarget := "order_entry.submit_order"
	afterTarget := "order_entry.order_record"
	if beforeTarget == afterTarget {
		t.Fatal("test setup: protected target changeがありません")
	}
	if !reflect.DeepEqual(beforeD4, afterD4) {
		t.Fatal("test setup: target-only pressureなのにD4 bodyまで変化しました")
	}

	before, err := deriveD4O1ProcedureProbe(f, map[string][]string{"BP-06": {beforeTarget}})
	if err != nil {
		t.Fatal(err)
	}
	after, err := deriveD4O1ProcedureProbe(f, map[string][]string{"BP-06": {afterTarget}})
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(before, after) {
		t.Fatal("D4 + narrow O1でもprotected target-only meaning差を失いました")
	}
	beforeSteps := d4O1StepsBySourceLocalKeyProbe(f, before)
	afterSteps := d4O1StepsBySourceLocalKeyProbe(f, after)
	if !reflect.DeepEqual(beforeSteps["BP-06"].Body, afterSteps["BP-06"].Body) {
		t.Fatal("protected target-only changeをBusiness Action body / realization changeへ変換しました")
	}
}

func TestD4O1PreservesBP06TargetWithoutInventingApplicationRealization(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	model, err := deriveD4O1ProcedureProbe(f, map[string][]string{"BP-06": {"order_entry.submit_order"}})
	if err != nil {
		t.Fatal(err)
	}
	steps := d4O1StepsBySourceLocalKeyProbe(f, model)
	bp06 := steps["BP-06"]

	if len(bp06.Body.ApplicationPatterns) != 0 {
		t.Fatalf("protected targetをBP-06 direct realizationへ変換しました: %+v", bp06.Body.ApplicationPatterns)
	}
	if !reflect.DeepEqual(bp06.Body.Capabilities, []string{"security.authorize"}) {
		t.Fatalf("supporting Authorization Capabilityを失いました: %+v", bp06.Body.Capabilities)
	}
	if bp06.ProtectedApplicationPattern != "order_entry.submit_order" {
		t.Fatalf("protected target exact identityを保持できません: %q", bp06.ProtectedApplicationPattern)
	}
	if err := checkExpectedDiagnostics(applicationPatternTargetDiagnostics(loaded.World, "BP-06", []string{bp06.ProtectedApplicationPattern}), ""); err != nil {
		t.Fatalf("protected targetをnormalized Worldでexact resolutionできません: %v", err)
	}
}

func TestD4O1NarrowAxisDoesNotBecomeGenericRoleReferenceModel(t *testing.T) {
	typ := reflect.TypeOf(d4O1StepProbe{})
	want := []string{"Body", "ProtectedApplicationPattern"}
	if got := fieldNames(typ); !reflect.DeepEqual(got, want) {
		t.Fatalf("D4 + O1 minimum candidateが膨張しました: got=%v want=%v", got, want)
	}
	for _, forbidden := range []string{"Role", "Kind", "SemanticRefs", "Targets", "Concerns", "Supports", "Policy", "Permission", "Principal"} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("narrow O1へgeneric role/policy axis %qを追加しました", forbidden)
		}
	}
}

func TestD4O1FailsClosedOnUnknownStepOrUnprovenMultipleTargets(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := deriveD4O1ProcedureProbe(f, map[string][]string{"BP-missing": {"order_entry.submit_order"}}); err == nil {
		t.Fatal("unknown Stepへのprotected targetを黙ってdropしました")
	}
	if _, err := deriveD4O1ProcedureProbe(f, map[string][]string{"BP-06": {"order_entry.submit_order", "order_entry.order_record"}}); err == nil {
		t.Fatal("current Evidenceで未証明のmultiple protected targetsを先回りして受理しました")
	}
}

func TestD4O1DoesNotRequireTargetsForUnrelatedSteps(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	model, err := deriveD4O1ProcedureProbe(f, map[string][]string{"BP-06": {"order_entry.submit_order"}})
	if err != nil {
		t.Fatal(err)
	}
	steps := d4O1StepsBySourceLocalKeyProbe(f, model)
	for _, stepID := range []string{"BP-00", "BP-01", "BP-02", "BP-03", "BP-04", "BP-05", "BP-07", "BP-08", "BP-09"} {
		if steps[stepID].ProtectedApplicationPattern != "" {
			t.Fatalf("unrelated Step %sへsymmetry目的のprotected targetを要求しました: %q", stepID, steps[stepID].ProtectedApplicationPattern)
		}
	}
}

func deriveD4O1ProcedureProbe(f fixture, targetsByStep map[string][]string) (d4O1ProcedureProbe, error) {
	base, err := deriveD4BusinessProcedureProbe(f)
	if err != nil {
		return d4O1ProcedureProbe{}, err
	}
	known := map[string]struct{}{}
	result := d4O1ProcedureProbe{
		ID:      base.ID,
		Purpose: base.Purpose,
		Trigger: base.Trigger,
		Actor:   base.Actor,
		Steps:   make([]d4O1StepProbe, 0, len(base.Steps)),
	}
	for i, body := range base.Steps {
		stepID := f.Steps[i].ID
		known[stepID] = struct{}{}
		targets := targetsByStep[stepID]
		if len(targets) > 1 {
			return d4O1ProcedureProbe{}, errProcedureMeaning("multiple protected Application Pattern targets are not proven at step " + stepID)
		}
		target := ""
		if len(targets) == 1 {
			target = targets[0]
			if target == "" {
				return d4O1ProcedureProbe{}, errProcedureMeaning("protected Application Pattern target is empty at step " + stepID)
			}
		}
		result.Steps = append(result.Steps, d4O1StepProbe{
			Body:                        body,
			ProtectedApplicationPattern: target,
		})
	}
	for stepID := range targetsByStep {
		if _, ok := known[stepID]; !ok {
			return d4O1ProcedureProbe{}, errProcedureMeaning("protected Application Pattern target step not found: " + stepID)
		}
	}
	return result, nil
}

func d4O1StepsBySourceLocalKeyProbe(f fixture, model d4O1ProcedureProbe) map[string]d4O1StepProbe {
	result := make(map[string]d4O1StepProbe, len(model.Steps))
	for i, item := range model.Steps {
		result[f.Steps[i].ID] = item
	}
	return result
}
