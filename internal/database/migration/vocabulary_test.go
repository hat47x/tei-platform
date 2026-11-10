package migration

import "testing"

// The lists are quoted by validation diagnostics, so each list and its Valid must
// be one answer (#4094).
func TestMigrationVocabularyListsMatchValid(t *testing.T) {
	recoveries := []RecoveryStrategy{RecoveryForwardFix, RecoveryExplicitRollback, RecoveryNotReversible}
	if len(recoveries) != len(RecoveryStrategies) {
		t.Fatalf("RecoveryStrategies has %d entries but %d are declared", len(RecoveryStrategies), len(recoveries))
	}
	for _, v := range RecoveryStrategies {
		if !v.Valid() {
			t.Errorf("RecoveryStrategies lists %q but Valid() rejects it", v)
		}
	}
	for _, bad := range []RecoveryStrategy{"", "ForwardFix", "rollback"} {
		if bad.Valid() {
			t.Errorf("Valid() accepted recovery %q", bad)
		}
	}

	steps := []StepKind{StepSchema, StepData, StepVerify}
	if len(steps) != len(StepKinds) {
		t.Fatalf("StepKinds has %d entries but %d are declared", len(StepKinds), len(steps))
	}
	for _, v := range StepKinds {
		if !v.Valid() {
			t.Errorf("StepKinds lists %q but Valid() rejects it", v)
		}
	}
	for _, bad := range []StepKind{"", "Schema", "migrate"} {
		if bad.Valid() {
			t.Errorf("Valid() accepted step kind %q", bad)
		}
	}
}
