package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestCurrentReferenceUsesOnlyEvidenceBackedExplicitRefs(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	model, err := deriveNormalizedBusinessProcedureWithStepAddressProbe(f)
	if err != nil {
		t.Fatal(err)
	}
	steps := addressedD3StepsByLocalKey(model)

	wantPatterns := map[string][]string{
		"BP-00": {"order_entry.recent_orders"},
		"BP-01": {"order_entry.order_record"},
		"BP-02": {"order_entry.customer_reference"},
		"BP-03": {"order_entry.order_lines"},
		"BP-07": {"order_entry.submit_order"},
	}
	for id, want := range wantPatterns {
		if !reflect.DeepEqual(steps[id].ApplicationPatterns, want) {
			t.Fatalf("%sのevidence-backed Application Pattern refs=%v want=%v", id, steps[id].ApplicationPatterns, want)
		}
	}

	wantValidations := []string{"sales.order.lines.line_number_unique", "sales.order_line.quantity_positive"}
	if !reflect.DeepEqual(steps["BP-05"].Validations, wantValidations) {
		t.Fatalf("BP-05のdeterministic Validation refs=%v want=%v", steps["BP-05"].Validations, wantValidations)
	}
	if !reflect.DeepEqual(steps["BP-06"].Capabilities, []string{"security.authorize"}) {
		t.Fatalf("BP-06のAuthorization authority ref=%v", steps["BP-06"].Capabilities)
	}

	for _, id := range []string{"BP-04", "BP-08", "BP-09"} {
		step := steps[id]
		if len(step.ApplicationPatterns) != 0 || len(step.Data) != 0 || len(step.Relations) != 0 || len(step.Validations) != 0 || len(step.Capabilities) != 0 {
			t.Fatalf("residual Business Action %sへ未証明のexplicit refを付与しました: %+v", id, step)
		}
	}
}

func TestOrderLineReviewDoesNotDuplicateDeterministicValidationAuthority(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	model, err := deriveNormalizedBusinessProcedureWithStepAddressProbe(f)
	if err != nil {
		t.Fatal(err)
	}
	steps := addressedD3StepsByLocalKey(model)

	if len(steps["BP-03"].Validations) != 0 {
		t.Fatalf("BP-03 line reviewへBP-05のValidation authorityを重複保持しました: %v", steps["BP-03"].Validations)
	}
	if len(steps["BP-05"].Validations) == 0 {
		t.Fatal("BP-05 deterministic constraint checkからValidation authorityが失われました")
	}
}

func TestCurrentReferenceDoesNotProveDirectDataOrRelationRefsAreD3Minimum(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}

	for _, step := range f.Steps {
		if len(step.Data) != 0 || len(step.Relations) != 0 {
			t.Fatalf("current ReferenceだけではD3 minimumと証明していないdirect Data/Relation refがあります: step=%s data=%v relations=%v", step.ID, step.Data, step.Relations)
		}
	}
}

func TestCopyingPatternOwnedCapabilityCreatesSpuriousBusinessMeaningChange(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := deriveNormalizedBusinessProcedureWithStepAddressProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	changed.Steps = append([]step(nil), f.Steps...)
	for i := range changed.Steps {
		if changed.Steps[i].ID == "BP-07" {
			changed.Steps[i].Capabilities = []string{"sales.order.submit"}
		}
	}
	after, err := deriveNormalizedBusinessProcedureWithStepAddressProbe(changed)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(before, after) {
		t.Fatal("test setup failed: copied Pattern-owned Capabilityがmeaningへ現れません")
	}

	// order_entry.submit_order already owns the sales.order.submit Capability.
	// Copying it into the Business source does not add a new Business decision;
	// it only creates a second authored representation that can drift independently.
}
