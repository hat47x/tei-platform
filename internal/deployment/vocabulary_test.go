package deployment

import "testing"

// ActionKind, EventKind, EvidenceKind and ApplicationLifecycle are the vocabulary
// of a deployment plan and its trace. A reader branches on them, so each value is
// listed here rather than only declared (#4032).
func TestDeploymentVocabulariesAreCovered(t *testing.T) {
	t.Run("action kinds", func(t *testing.T) {
		assertVocabulary(t, "action kind", []string{
			string(ActionDatabaseRelease), string(ActionDatabaseGate), string(ActionApplicationStart),
			string(ActionTrafficShift), string(ActionRolloutHealthGate), string(ActionStopAcceptingWork),
			string(ActionDrainCompleteGate), string(ActionApplicationRetire),
		}, []string{
			"database-release", "database-gate", "application-start", "traffic-shift",
			"rollout-health-gate", "stop-accepting-work", "drain-complete-gate", "application-retire",
		})
	})

	t.Run("evidence kinds", func(t *testing.T) {
		if string(EvidenceRolloutHealth) != "rollout-health" {
			t.Fatalf("EvidenceRolloutHealth=%q", EvidenceRolloutHealth)
		}
	})

	t.Run("application lifecycles", func(t *testing.T) {
		assertVocabulary(t, "lifecycle", []string{
			string(LifecycleStarting), string(LifecycleServing),
			string(LifecycleDraining), string(LifecycleQuiescent),
		}, []string{"starting", "serving", "draining", "quiescent"})
		// A declared lifecycle has to be accepted by Valid, an unknown one not.
		for _, lifecycle := range []ApplicationLifecycle{
			LifecycleStarting, LifecycleServing, LifecycleDraining, LifecycleQuiescent,
		} {
			if !lifecycle.Valid() {
				t.Fatalf("%q is declared but not accepted by Valid", lifecycle)
			}
		}
		if ApplicationLifecycle("archived").Valid() {
			t.Fatal("an undeclared lifecycle was accepted")
		}
	})
}

// assertVocabulary checks that every declared value keeps its literal and that no
// value is declared twice.
func assertVocabulary(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: a value was dropped (%d of %d)", name, len(got), len(want))
	}
	seen := map[string]bool{}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s[%d]=%q want %q", name, i, got[i], want[i])
		}
		if seen[got[i]] {
			t.Fatalf("%s %q is declared twice", name, got[i])
		}
		seen[got[i]] = true
	}
}
