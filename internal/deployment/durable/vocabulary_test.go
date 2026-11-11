package durable

import "testing"

// The durable run record is written to disk and read by an operator after the
// fact, so its event, mode, status and recovery vocabularies are the contract of
// what a stored log can say. Each value is listed here rather than only declared
// (#4032).
func TestDurableVocabulariesAreCovered(t *testing.T) {
	t.Run("action modes", func(t *testing.T) {
		assertVocabulary(t, "mode", []string{
			string(ModeCommand), string(ModeObservation), string(ModeEvidence),
		}, []string{"command", "observation", "evidence"})
	})

	t.Run("action statuses", func(t *testing.T) {
		assertVocabulary(t, "status", []string{
			string(ActionPending), string(ActionCompleted),
		}, []string{"pending", "completed"})
	})

	t.Run("event kinds", func(t *testing.T) {
		assertVocabulary(t, "event", []string{
			string(EventActionCompleted), string(EventEvidenceAccepted), string(EventEvidenceRejected),
			string(EventCommandAttempted), string(EventRetryAuthorized), string(EventRunPaused),
			string(EventRunResumed), string(EventRunAborted), string(EventRecoveryPlanned),
			string(EventRecoveryRunLinked),
		}, []string{
			"action-completed", "evidence-accepted", "evidence-rejected", "command-attempted",
			"retry-authorized", "run-paused", "run-resumed", "run-aborted", "recovery-planned",
			"recovery-run-linked",
		})
	})

	t.Run("recovery strategies", func(t *testing.T) {
		assertVocabulary(t, "strategy", []string{
			string(RecoveryRetry), string(RecoveryForwardFix), string(RecoveryRollback), string(RecoveryManual),
		}, []string{"retry", "forward-fix", "rollback", "manual"})
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
