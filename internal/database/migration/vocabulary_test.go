package migration_test

import (
	"testing"

	"tei/internal/database/migration"
)

// The validator quotes RecoveryStrategies in its diagnostic, so the slice and
// Valid must accept exactly the same set (#4094).
func TestRecoveryStrategiesSliceAndValidAgree(t *testing.T) {
	for _, value := range migration.RecoveryStrategies {
		if !value.Valid() {
			t.Errorf("RecoveryStrategies lists %q but Valid rejects it", value)
		}
	}
	for _, invalid := range []migration.RecoveryStrategy{"", "rollback", "forward", "NOT-REVERSIBLE"} {
		if invalid.Valid() {
			t.Errorf("Valid accepts %q, which is not in RecoveryStrategies", invalid)
		}
	}
}

func TestStepKindsSliceAndValidAgree(t *testing.T) {
	for _, value := range migration.StepKinds {
		if !value.Valid() {
			t.Errorf("StepKinds lists %q but Valid rejects it", value)
		}
	}
	for _, invalid := range []migration.StepKind{"", "schema-and-data", "Schema", "check"} {
		if invalid.Valid() {
			t.Errorf("Valid accepts %q, which is not in StepKinds", invalid)
		}
	}
}
