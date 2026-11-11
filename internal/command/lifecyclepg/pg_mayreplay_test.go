package lifecyclepg

import (
	"testing"

	"tei/internal/command"
)

// RC-05 (#1174): the durable store must answer MayReplay like the in-memory
// Lifecycle. A dispatched attempt whose outcome was never recorded (the
// process died, or the Observe write failed after the provider committed) is
// not replayable, and neither is a command the store could not read.
func TestMayReplayRefusesDispatchedAndUnreadableAttempts(t *testing.T) {
	database := newTestDB(t)
	store := database.open(t)
	if !store.MayReplay("cmd-1") {
		t.Fatal("an absent command was not initially dispatchable")
	}
	if _, err := store.Dispatch("cmd-1", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	// A different connection sees only the persisted dispatched stage.
	reopened := database.open(t)
	if attempt, ok := reopened.Attempt("cmd-1"); !ok || attempt.Stage != command.StageDispatched {
		t.Fatalf("reopened attempt=%+v ok=%v", attempt, ok)
	}
	if reopened.MayReplay("cmd-1") {
		t.Fatal("a dispatched attempt without a recorded outcome was reported as replayable")
	}
	if _, err := reopened.Dispatch("cmd-1", "owner-a", 1); err == nil {
		t.Fatal("MayReplay and Dispatch disagree: a dispatched attempt was re-dispatched")
	}

	// A store that cannot read must not answer "never dispatched".
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
	if reopened.MayReplay("cmd-1") {
		t.Fatal("an unreadable store reported a dispatched command as replayable")
	}
	if reopened.MayReplay("never-dispatched") {
		t.Fatal("an unreadable store reported an unknown command as replayable")
	}
}
