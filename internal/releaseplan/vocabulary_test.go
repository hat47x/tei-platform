package releaseplan_test

import (
	"testing"

	"tei/internal/releaseplan"
)

// Each of these slices is quoted by a validator diagnostic. The set Valid
// accepts and the set the message names must be the same (#4094).
func TestPhasesSliceAndValidAgree(t *testing.T) {
	for _, value := range releaseplan.Phases {
		if !value.Valid() {
			t.Errorf("Phases lists %q but Valid rejects it", value)
		}
	}
	for _, invalid := range []releaseplan.Phase{"", "Expand", "expand-and-populate", "cutover"} {
		if invalid.Valid() {
			t.Errorf("Valid accepts %q, which is not in Phases", invalid)
		}
	}
}

func TestRolloutStrategiesSliceAndValidAgree(t *testing.T) {
	for _, value := range releaseplan.RolloutStrategies {
		if !value.Valid() {
			t.Errorf("RolloutStrategies lists %q but Valid rejects it", value)
		}
	}
	// An empty strategy means "not declared" and is accepted, but it is not a
	// value a diagnostic should offer, so it stays out of the slice.
	if (releaseplan.RolloutStrategy("")).Valid() != true {
		t.Error("an undeclared strategy must be accepted")
	}
	for _, value := range releaseplan.RolloutStrategies {
		if value == "" {
			t.Error("RolloutStrategies must not list the empty string")
		}
	}
	for _, invalid := range []releaseplan.RolloutStrategy{"Canary", "blue_green", "rolling"} {
		if invalid.Valid() {
			t.Errorf("Valid accepts %q, which is not in RolloutStrategies", invalid)
		}
	}
}

func TestStepKindsSliceAndValidAgree(t *testing.T) {
	for _, value := range releaseplan.StepKinds {
		if !value.Valid() {
			t.Errorf("StepKinds lists %q but Valid rejects it", value)
		}
	}
	for _, invalid := range []releaseplan.StepKind{"", "database", "application", "DatabaseRelease"} {
		if invalid.Valid() {
			t.Errorf("Valid accepts %q, which is not in StepKinds", invalid)
		}
	}
}
