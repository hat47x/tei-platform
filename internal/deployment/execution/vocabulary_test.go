package execution

import "testing"

// A run trace is read after the fact: its event kinds and run status are the
// vocabulary of what a trace can say, so each value is listed here (#4032).
func TestExecutionVocabulariesAreCovered(t *testing.T) {
	t.Run("event kinds", func(t *testing.T) {
		kinds := []EventKind{
			EventDatabaseReleaseApplied, EventDatabaseGateSatisfied, EventApplicationStarted,
			EventTrafficShifted, EventRolloutCheckpoint, EventDrainStarted,
			EventDrainCompleted, EventApplicationRetired,
		}
		want := []string{
			"database-release-applied", "database-gate-satisfied", "application-started",
			"traffic-shifted", "rollout-checkpoint", "drain-started",
			"drain-completed", "application-retired",
		}
		if len(kinds) != len(want) {
			t.Fatalf("an event kind was dropped: %d values", len(kinds))
		}
		seen := map[string]bool{}
		for i, kind := range kinds {
			if string(kind) != want[i] {
				t.Fatalf("kinds[%d]=%q want %q", i, kind, want[i])
			}
			if seen[string(kind)] {
				t.Fatalf("event kind %q is declared twice", kind)
			}
			seen[string(kind)] = true
		}
	})
	t.Run("run statuses", func(t *testing.T) {
		statuses := []RunStatus{RunSucceeded, RunBlocked}
		want := []string{"succeeded", "blocked"}
		for i, status := range statuses {
			if string(status) != want[i] {
				t.Fatalf("statuses[%d]=%q want %q", i, status, want[i])
			}
		}
		if RunSucceeded == RunBlocked {
			t.Fatal("success and blocked share a value")
		}
	})
}
