package capabilitydegradation

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func loadFixture(t *testing.T) Input {
	t.Helper()
	input, err := LoadInput(filepath.Join("testdata", "input-existing-db.json"))
	if err != nil {
		t.Fatalf("LoadInput: %v", err)
	}
	return input
}

func operationByID(report Report) map[string]OperationResult {
	out := make(map[string]OperationResult, len(report.Operations))
	for _, operation := range report.Operations {
		out[operation.ID] = operation
	}
	return out
}

func degradationByOperation(report Report) map[string]Degradation {
	out := make(map[string]Degradation, len(report.Degradations))
	for _, degradation := range report.Degradations {
		out[degradation.Operation] = degradation
	}
	return out
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestCapableTableGeneratesAllOperations(t *testing.T) {
	report := Check(loadFixture(t))
	generated := GeneratedOperations(report)
	for _, id := range []string{"orders.read", "orders.insert", "orders.update", "orders.delete"} {
		if !hasString(generated, id) {
			t.Errorf("%s should be generated, got %v", id, generated)
		}
	}
}

func TestMissingOptimisticLockWithholdsUpdateAndDelete(t *testing.T) {
	report := Check(loadFixture(t))
	operations := operationByID(report)
	for _, id := range []string{"legacy.update", "legacy.delete"} {
		if operations[id].Status != OperationWithheld {
			t.Errorf("%s = %q, want withheld", id, operations[id].Status)
		}
		if !hasString(operations[id].Missing, CapabilityConflictDetection) {
			t.Errorf("%s missing = %v, want %s", id, operations[id].Missing, CapabilityConflictDetection)
		}
	}
}

func TestReadAndInsertStillGenerateOnDegradedTable(t *testing.T) {
	report := Check(loadFixture(t))
	operations := operationByID(report)
	for _, id := range []string{"legacy.read", "legacy.insert"} {
		if operations[id].Status != OperationGenerated {
			t.Errorf("%s = %q, want generated", id, operations[id].Status)
		}
	}
}

func TestNoUpdatePathOnTableWithoutConflictDetection(t *testing.T) {
	report := Check(loadFixture(t))
	for _, id := range GeneratedOperations(report) {
		operation := operationByID(report)[id]
		if operation.Table == "legacy_orders" && (operation.Action == ActionUpdate || operation.Action == ActionDelete) {
			t.Errorf("table without conflict detection must not generate %s", id)
		}
	}
}

func TestDegradationRecordsConfirmedAbsence(t *testing.T) {
	report := Check(loadFixture(t))
	degradation, ok := degradationByOperation(report)["legacy.update"]
	if !ok {
		t.Fatal("legacy.update degradation missing")
	}
	if !degradation.Confirmed {
		t.Error("a missing capability on an observed table must be a confirmed fact")
	}
	if degradation.Disposition != OperationWithheld {
		t.Errorf("disposition = %q, want withheld", degradation.Disposition)
	}
	if !hasString(degradation.Missing, CapabilityConflictDetection) {
		t.Errorf("missing = %v", degradation.Missing)
	}
}

func TestExplicitWeakerGuaranteeIsRecorded(t *testing.T) {
	report := Check(loadFixture(t))
	operation := operationByID(report)["legacy.audit-update"]
	if operation.Status != OperationDegraded {
		t.Fatalf("status = %q, want degraded", operation.Status)
	}
	if operation.Guarantee == "" {
		t.Fatal("weaker guarantee is not recorded on the operation")
	}
	degradation, ok := degradationByOperation(report)["legacy.audit-update"]
	if !ok {
		t.Fatal("degradation record missing")
	}
	if degradation.WeakerGuarantee == "" || !degradation.Confirmed {
		t.Errorf("degradation = %+v", degradation)
	}
	if len(degradation.Missing) != 2 {
		t.Errorf("missing = %v, want optimistic-lock-column and audit-column", degradation.Missing)
	}
}

func TestDegradationIsNotCertifiedEquivalent(t *testing.T) {
	report := Check(loadFixture(t))
	if report.CertifiedEquivalent {
		t.Error("degraded result must not be certified as equivalent")
	}
	if report.Equivalence != "not-certified" {
		t.Errorf("equivalence = %q, want not-certified", report.Equivalence)
	}
}

func TestUnobservedTableIsUnresolvedNotConfirmedAbsence(t *testing.T) {
	input := loadFixture(t)
	for i := range input.Tables {
		if input.Tables[i].ID == "legacy_orders" {
			input.Tables[i].Observed = false
		}
	}
	report := Check(input)
	if report.Status != StatusUnresolved {
		t.Fatalf("status = %q, want unresolved", report.Status)
	}
	for _, unresolved := range report.Unresolved {
		if unresolved.Table != "legacy_orders" {
			t.Errorf("unexpected unresolved entry: %+v", unresolved)
		}
	}
	if len(report.Degradations) != 0 {
		t.Errorf("unobserved table must not be recorded as a confirmed degradation: %+v", report.Degradations)
	}
}

func TestUnknownTableIsRejected(t *testing.T) {
	input := loadFixture(t)
	input.Operations = append(input.Operations, Operation{
		ID: "ghost.read", Table: "ghost", Action: ActionRead, Requires: []string{"read"},
	})
	report := Check(input)
	operation := operationByID(report)["ghost.read"]
	if operation.Status != OperationRejected {
		t.Fatalf("status = %q, want rejected", operation.Status)
	}
	if report.Status != StatusBlocked {
		t.Errorf("status = %q, want blocked", report.Status)
	}
}

func TestUpdateWithoutConflictDetectionRequirementIsRejected(t *testing.T) {
	input := loadFixture(t)
	for i := range input.Operations {
		if input.Operations[i].ID == "legacy.update" {
			input.Operations[i].Requires = []string{"update"}
		}
	}
	report := Check(input)
	operation := operationByID(report)["legacy.update"]
	if operation.Status != OperationRejected {
		t.Fatalf("status = %q, want rejected", operation.Status)
	}
}

func TestEmptyRequiresIsRejected(t *testing.T) {
	input := loadFixture(t)
	input.Operations = append(input.Operations, Operation{
		ID: "no.requires", Table: "legacy_orders", Action: ActionRead,
	})
	report := Check(input)
	if operationByID(report)["no.requires"].Status != OperationRejected {
		t.Fatal("an operation without required guarantees must be rejected")
	}
}

func TestDegradationWithoutMissingCapabilityIsRejected(t *testing.T) {
	input := loadFixture(t)
	for i := range input.Operations {
		if input.Operations[i].ID == "orders.update" {
			input.Operations[i].AllowDegradation = true
			input.Operations[i].WeakerGuarantee = "no conflict detection"
		}
	}
	report := Check(input)
	if operationByID(report)["orders.update"].Status != OperationRejected {
		t.Fatal("a degradation with no missing capability must be rejected")
	}
}

func TestCheckDoesNotMutateInput(t *testing.T) {
	input := loadFixture(t)
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var original Input
	if err := json.Unmarshal(raw, &original); err != nil {
		t.Fatal(err)
	}
	_ = Check(input)
	if !reflect.DeepEqual(input, original) {
		t.Fatal("Check changed the input")
	}
}
