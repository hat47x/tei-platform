package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

type d4BusinessProcedureProbe struct {
	ID      string                         `json:"id"`
	Purpose string                         `json:"purpose"`
	Trigger candidateRef                   `json:"trigger"`
	Actor   candidateRef                   `json:"actor"`
	Steps   []d4BusinessProcedureStepProbe `json:"steps"`
}

type d4BusinessProcedureStepProbe struct {
	BusinessAction      string   `json:"businessAction"`
	ApplicationPatterns []string `json:"applicationPatterns,omitempty"`
	Validations         []string `json:"validations,omitempty"`
	Capabilities        []string `json:"capabilities,omitempty"`
}

func TestD4KeepsOnlyCurrentEvidenceBackedRefKinds(t *testing.T) {
	stepType := reflect.TypeOf(d4BusinessProcedureStepProbe{})
	want := []string{"BusinessAction", "ApplicationPatterns", "Validations", "Capabilities"}
	if got := fieldNames(stepType); !reflect.DeepEqual(got, want) {
		t.Fatalf("D4 ref-kind boundaryが変化しました: got=%v want=%v", got, want)
	}
	for _, forbidden := range []string{"Data", "Relations", "Realization", "Role", "Kind"} {
		if _, found := stepType.FieldByName(forbidden); found {
			t.Fatalf("D4 minimum candidateへ未証明または別axisのfield %qを追加しました", forbidden)
		}
	}
}

func TestD4PreservesCurrentEvidenceBackedBusinessRefs(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	model, err := deriveD4BusinessProcedureProbe(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Steps) != len(f.Steps) {
		t.Fatalf("D4でordered Business stepsが失われました: got=%d want=%d", len(model.Steps), len(f.Steps))
	}

	steps := d4StepsBySourceLocalKey(f, model)
	if !reflect.DeepEqual(steps["BP-03"].ApplicationPatterns, []string{"order_entry.order_lines"}) || len(steps["BP-03"].Validations) != 0 {
		t.Fatalf("BP-03のPattern利用とValidation非重複境界をD4で保持できません: %+v", steps["BP-03"])
	}
	wantValidations := []string{"sales.order.lines.line_number_unique", "sales.order_line.quantity_positive"}
	if !reflect.DeepEqual(steps["BP-05"].Validations, wantValidations) {
		t.Fatalf("BP-05 deterministic Validation refsがD4で失われました: got=%v want=%v", steps["BP-05"].Validations, wantValidations)
	}
	if !reflect.DeepEqual(steps["BP-06"].Capabilities, []string{"security.authorize"}) || len(steps["BP-06"].ApplicationPatterns) != 0 {
		t.Fatalf("BP-06 Authorization authority / no-realization境界をD4で保持できません: %+v", steps["BP-06"])
	}
	if !reflect.DeepEqual(steps["BP-07"].ApplicationPatterns, []string{"order_entry.submit_order"}) || len(steps["BP-07"].Capabilities) != 0 {
		t.Fatalf("BP-07 Pattern-owned Capability非重複境界をD4で保持できません: %+v", steps["BP-07"])
	}
	for _, id := range []string{"BP-04", "BP-08", "BP-09"} {
		step := steps[id]
		if len(step.ApplicationPatterns) != 0 || len(step.Validations) != 0 || len(step.Capabilities) != 0 {
			t.Fatalf("residual Business Action %sへD4がfake refを追加しました: %+v", id, step)
		}
	}
}

func TestD4FailsClosedIfFutureSourceNeedsDirectDataRef(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := f
	changed.Steps = append([]step(nil), f.Steps...)
	for i := range changed.Steps {
		if changed.Steps[i].ID == "BP-04" {
			changed.Steps[i].Data = []string{"sales.order"}
		}
	}
	if _, err := deriveD4BusinessProcedureProbe(changed); err == nil {
		t.Fatal("future direct Data refをD4 projectionが黙ってdropしました")
	}
}

func TestD4FailsClosedIfFutureSourceNeedsDirectRelationRef(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := f
	changed.Steps = append([]step(nil), f.Steps...)
	for i := range changed.Steps {
		if changed.Steps[i].ID == "BP-04" {
			changed.Steps[i].Relations = []string{"sales.order.lines"}
		}
	}
	if _, err := deriveD4BusinessProcedureProbe(changed); err == nil {
		t.Fatal("future direct Relation refをD4 projectionが黙ってdropしました")
	}
}

func TestD4KeepsAuthorityKindEvenWhenLiteralIdentityMatches(t *testing.T) {
	patternRef := d4BusinessProcedureStepProbe{BusinessAction: "probe", ApplicationPatterns: []string{"same.literal.id"}}
	capabilityRef := d4BusinessProcedureStepProbe{BusinessAction: "probe", Capabilities: []string{"same.literal.id"}}
	if reflect.DeepEqual(patternRef, capabilityRef) {
		t.Fatal("同じliteral ID textをApplication Pattern / Capabilityのkind差なしに一袋へ畳みました")
	}
}

func deriveD4BusinessProcedureProbe(f fixture) (d4BusinessProcedureProbe, error) {
	base, err := deriveCanonicalBusinessProcedureBodyWithoutStepIDProbe(f)
	if err != nil {
		return d4BusinessProcedureProbe{}, err
	}
	result := d4BusinessProcedureProbe{
		ID:      base.ID,
		Purpose: base.Purpose,
		Trigger: base.Trigger,
		Actor:   base.Actor,
		Steps:   make([]d4BusinessProcedureStepProbe, 0, len(base.Steps)),
	}
	for i, item := range base.Steps {
		if len(item.Data) != 0 {
			return d4BusinessProcedureProbe{}, errProcedureMeaning("D4 unsupported direct Data ref at step " + f.Steps[i].ID)
		}
		if len(item.Relations) != 0 {
			return d4BusinessProcedureProbe{}, errProcedureMeaning("D4 unsupported direct Relation ref at step " + f.Steps[i].ID)
		}
		result.Steps = append(result.Steps, d4BusinessProcedureStepProbe{
			BusinessAction:      item.BusinessAction,
			ApplicationPatterns: append([]string(nil), item.ApplicationPatterns...),
			Validations:         append([]string(nil), item.Validations...),
			Capabilities:        append([]string(nil), item.Capabilities...),
		})
	}
	return result, nil
}

func d4StepsBySourceLocalKey(f fixture, model d4BusinessProcedureProbe) map[string]d4BusinessProcedureStepProbe {
	result := make(map[string]d4BusinessProcedureStepProbe, len(model.Steps))
	for i, item := range model.Steps {
		result[f.Steps[i].ID] = item
	}
	return result
}
