package lifecyclesqlite

import (
	"path/filepath"
	"testing"
)

// RC-05 (#1174): the durable store must answer MayReplay like the in-memory
// Lifecycle. A dispatched attempt without a recorded outcome is not
// replayable, and neither is a command the store could not read.
func TestMayReplayRefusesDispatchedAndUnreadableAttempts(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "lifecycle.db"))
	if err != nil {
		t.Fatal(err)
	}
	if !store.MayReplay("cmd-1") {
		t.Fatal("an absent command was not initially dispatchable")
	}
	if _, err := store.Dispatch("cmd-1", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if store.MayReplay("cmd-1") {
		t.Fatal("a dispatched attempt without a recorded outcome was reported as replayable")
	}
	if _, err := store.Dispatch("cmd-1", "owner-a", 1); err == nil {
		t.Fatal("MayReplay and Dispatch disagree: a dispatched attempt was re-dispatched")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if store.MayReplay("cmd-1") || store.MayReplay("never-dispatched") {
		t.Fatal("an unreadable store reported a command as replayable")
	}
}
