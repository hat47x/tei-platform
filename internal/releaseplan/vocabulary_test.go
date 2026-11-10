package releaseplan

import "testing"

// The lists are quoted by validation diagnostics (#4094).
func TestReleaseplanVocabularyListsMatchValid(t *testing.T) {
	phases := []Phase{PhaseExpand, PhasePopulate, PhaseTransition, PhaseRollout, PhaseContract, PhaseVerify, PhaseOther}
	if len(phases) != len(Phases) {
		t.Fatalf("Phases has %d entries but %d are declared", len(Phases), len(phases))
	}
	for _, v := range Phases {
		if !v.Valid() {
			t.Errorf("Phases lists %q but Valid() rejects it", v)
		}
	}
	for _, bad := range []Phase{"", "Expand", "drain"} {
		if bad.Valid() {
			t.Errorf("Valid() accepted phase %q", bad)
		}
	}

	strategies := []RolloutStrategy{RolloutDirect, RolloutCanary, RolloutBlueGreen, RolloutGradual}
	if len(strategies) != len(RolloutStrategies) {
		t.Fatalf("RolloutStrategies has %d entries but %d are declared", len(RolloutStrategies), len(strategies))
	}
	for _, v := range RolloutStrategies {
		if !v.Valid() {
			t.Errorf("RolloutStrategies lists %q but Valid() rejects it", v)
		}
	}
	// The empty strategy means "not declared" and is accepted, but it must not be
	// listed: a diagnostic should not tell an author to write an empty value.
	if !RolloutStrategy("").Valid() {
		t.Error("the empty rollout strategy must stay valid")
	}
	for _, v := range RolloutStrategies {
		if v == "" {
			t.Error("the empty rollout strategy must not appear in RolloutStrategies")
		}
	}
	for _, bad := range []RolloutStrategy{"Canary", "wave"} {
		if bad.Valid() {
			t.Errorf("Valid() accepted rollout strategy %q", bad)
		}
	}

	steps := []StepKind{StepDatabaseRelease, StepApplicationRelease, StepApplicationRetire, StepDatabaseGate}
	if len(steps) != len(StepKinds) {
		t.Fatalf("StepKinds has %d entries but %d are declared", len(StepKinds), len(steps))
	}
	for _, v := range StepKinds {
		if !v.Valid() {
			t.Errorf("StepKinds lists %q but Valid() rejects it", v)
		}
	}
	for _, bad := range []StepKind{"", "Database-Release", "gate"} {
		if bad.Valid() {
			t.Errorf("Valid() accepted step kind %q", bad)
		}
	}
}
