package main

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

type canonicalBusinessProcedureMeaningProbe struct {
	ID      string                                `json:"id"`
	Purpose string                                `json:"purpose"`
	Trigger candidateRef                          `json:"trigger"`
	Actor   candidateRef                          `json:"actor"`
	Steps   []canonicalBusinessProcedureStepProbe `json:"steps"`
}

type canonicalBusinessProcedureStepProbe struct {
	ID                  string   `json:"id"`
	BusinessAction      string   `json:"businessAction"`
	Realization         string   `json:"realization"`
	ApplicationPatterns []string `json:"applicationPatterns,omitempty"`
	Data                []string `json:"data,omitempty"`
	Relations           []string `json:"relations,omitempty"`
	Validations         []string `json:"validations,omitempty"`
	Capabilities        []string `json:"capabilities,omitempty"`
}

func TestCanonicalBusinessProcedureMeaningProbeKeepsThinBusinessSemantics(t *testing.T) {
	procedureType := reflect.TypeOf(canonicalBusinessProcedureMeaningProbe{})
	wantProcedureFields := []string{"ID", "Purpose", "Trigger", "Actor", "Steps"}
	if got := fieldNames(procedureType); !reflect.DeepEqual(got, wantProcedureFields) {
		t.Fatalf("Procedure meaning probeのfield境界が変化しました: got=%v want=%v", got, wantProcedureFields)
	}

	stepType := reflect.TypeOf(canonicalBusinessProcedureStepProbe{})
	wantStepFields := []string{"ID", "BusinessAction", "Realization", "ApplicationPatterns", "Data", "Relations", "Validations", "Capabilities"}
	if got := fieldNames(stepType); !reflect.DeepEqual(got, wantStepFields) {
		t.Fatalf("Procedure step meaning probeへruntime workflow detailが侵入しました: got=%v want=%v", got, wantStepFields)
	}
}

func TestCanonicalBusinessProcedureMeaningProbeSeesBusinessOnlyAcceptedChange(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := deriveCanonicalBusinessProcedureMeaningProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	found := false
	for i := range changed.Steps {
		if changed.Steps[i].ID == "BP-04" {
			changed.Steps[i].BusinessAction = "decide whether surrounding business circumstances permit submission"
			found = true
		}
	}
	if !found {
		t.Fatal("test setupがBP-04を見つけられません")
	}
	after, err := deriveCanonicalBusinessProcedureMeaningProbe(changed)
	if err != nil {
		t.Fatal(err)
	}

	if before.ID != after.ID {
		t.Fatal("Business wording changeでProcedure identityまで変化しました")
	}
	if reflect.DeepEqual(before, after) {
		t.Fatal("Business-only accepted changeをCanonical meaning差として観測できません")
	}
}

func TestCanonicalBusinessProcedureMeaningProbeSurvivesPureFileMove(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	beforeFixture, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := deriveCanonicalBusinessProcedureMeaningProbe(beforeFixture)
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(t.TempDir(), "sales", "moved-order-handling.json")
	if err := os.MkdirAll(filepath.Dir(moved), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(moved, content, 0o600); err != nil {
		t.Fatal(err)
	}
	afterFixture, err := readFixture(moved)
	if err != nil {
		t.Fatal(err)
	}
	after, err := deriveCanonicalBusinessProcedureMeaningProbe(afterFixture)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(before, after) {
		t.Fatalf("pure file moveでCanonical Business meaningが変化しました: before=%+v after=%+v", before, after)
	}
}

func TestCanonicalBusinessProcedureMeaningProbeRetainsManualActionWithoutFakeFunction(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	meaning, err := deriveCanonicalBusinessProcedureMeaningProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	step, ok := canonicalMeaningStep(meaning, "BP-04")
	if !ok {
		t.Fatal("BP-04 manual Business ActionがCanonical meaningから失われました")
	}
	if step.BusinessAction == "" || step.Realization != "manual-unresolved" {
		t.Fatalf("manual Business meaningが保持されていません: %+v", step)
	}
	if len(step.ApplicationPatterns) != 0 || len(step.Capabilities) != 0 {
		t.Fatalf("manual Business Actionへfake Application Function / Capabilityを付与しました: %+v", step)
	}
}

func TestCanonicalBusinessProcedureMeaningProbeKeepsExplicitRefsWithoutCopyingPatternOwnedMeaning(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	meaning, err := deriveCanonicalBusinessProcedureMeaningProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	customer, ok := canonicalMeaningStep(meaning, "BP-02")
	if !ok {
		t.Fatal("BP-02がありません")
	}
	if !reflect.DeepEqual(customer.ApplicationPatterns, []string{"order_entry.customer_reference"}) {
		t.Fatalf("BP-02 Pattern referenceが不正です: %+v", customer)
	}
	if len(customer.Data) != 0 || len(customer.Relations) != 0 || len(customer.Validations) != 0 || len(customer.Capabilities) != 0 {
		t.Fatalf("Pattern-owned meaningをBusiness Procedureへコピーしました: %+v", customer)
	}

	lineReview, ok := canonicalMeaningStep(meaning, "BP-03")
	if !ok {
		t.Fatal("BP-03がありません")
	}
	if !reflect.DeepEqual(lineReview.ApplicationPatterns, []string{"order_entry.order_lines"}) {
		t.Fatalf("BP-03 Pattern referenceが不正です: %+v", lineReview)
	}
	if len(lineReview.Data) != 0 || len(lineReview.Relations) != 0 || len(lineReview.Validations) != 0 || len(lineReview.Capabilities) != 0 {
		t.Fatalf("order-line reviewへ別Business ActionのValidation authorityまたはPattern-owned meaningを重複保持しました: %+v", lineReview)
	}

	constraintCheck, ok := canonicalMeaningStep(meaning, "BP-05")
	if !ok {
		t.Fatal("BP-05がありません")
	}
	wantValidations := []string{"sales.order.lines.line_number_unique", "sales.order_line.quantity_positive"}
	if !reflect.DeepEqual(constraintCheck.Validations, wantValidations) {
		t.Fatalf("deterministic constraint Business Actionのexplicit Validation referenceが保持されていません: got=%v want=%v", constraintCheck.Validations, wantValidations)
	}
}

func TestCanonicalBusinessProcedureMeaningProbeNormalizesSetLikeReferenceOrder(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := deriveCanonicalBusinessProcedureMeaningProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	for i := range changed.Steps {
		if changed.Steps[i].ID == "BP-05" {
			for left, right := 0, len(changed.Steps[i].Validations)-1; left < right; left, right = left+1, right-1 {
				changed.Steps[i].Validations[left], changed.Steps[i].Validations[right] = changed.Steps[i].Validations[right], changed.Steps[i].Validations[left]
			}
		}
	}
	after, err := deriveCanonicalBusinessProcedureMeaningProbe(changed)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(before, after) {
		t.Fatal("set-like explicit referenceの記述順だけでCanonical meaningが変化しました")
	}
}

func deriveCanonicalBusinessProcedureMeaningProbe(f fixture) (canonicalBusinessProcedureMeaningProbe, error) {
	if f.Procedure.ID == "" {
		return canonicalBusinessProcedureMeaningProbe{}, errProcedureMeaning("Procedure ID is empty")
	}
	if f.Procedure.IdentityStatus != referenceLocalStable {
		return canonicalBusinessProcedureMeaningProbe{}, errProcedureMeaning("Procedure identity is not reference-local-stable")
	}

	result := canonicalBusinessProcedureMeaningProbe{
		ID:      f.Procedure.ID,
		Purpose: f.Procedure.Purpose,
		Trigger: f.Procedure.Trigger,
		Actor:   f.Procedure.Actor,
		Steps:   make([]canonicalBusinessProcedureStepProbe, 0, len(f.Steps)),
	}
	seenSteps := map[string]struct{}{}
	for _, item := range f.Steps {
		if item.ID == "" {
			return canonicalBusinessProcedureMeaningProbe{}, errProcedureMeaning("step ID is empty")
		}
		if _, duplicate := seenSteps[item.ID]; duplicate {
			return canonicalBusinessProcedureMeaningProbe{}, errProcedureMeaning("duplicate step ID " + item.ID)
		}
		seenSteps[item.ID] = struct{}{}
		result.Steps = append(result.Steps, canonicalBusinessProcedureStepProbe{
			ID:                  item.ID,
			BusinessAction:      item.BusinessAction,
			Realization:         item.Realization,
			ApplicationPatterns: normalizedMeaningRefs(item.ApplicationPatterns),
			Data:                normalizedMeaningRefs(item.Data),
			Relations:           normalizedMeaningRefs(item.Relations),
			Validations:         normalizedMeaningRefs(item.Validations),
			Capabilities:        normalizedMeaningRefs(item.Capabilities),
		})
	}
	return result, nil
}

func normalizedMeaningRefs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func canonicalMeaningStep(meaning canonicalBusinessProcedureMeaningProbe, id string) (canonicalBusinessProcedureStepProbe, bool) {
	for _, item := range meaning.Steps {
		if item.ID == id {
			return item, true
		}
	}
	return canonicalBusinessProcedureStepProbe{}, false
}

func fieldNames(value reflect.Type) []string {
	result := make([]string, 0, value.NumField())
	for i := 0; i < value.NumField(); i++ {
		result = append(result, value.Field(i).Name)
	}
	return result
}

type procedureMeaningError string

func (e procedureMeaningError) Error() string { return string(e) }

func errProcedureMeaning(message string) error { return procedureMeaningError(message) }
