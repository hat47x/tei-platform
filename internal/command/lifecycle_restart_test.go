package command

import "testing"

// RCAS-02 (#2564): the Lifecycle is an in-memory stage store. A process restart
// or a different worker starts with no record of a previous unknown attempt, so
// it accepts a new dispatch of the same command. The stage store therefore does
// not provide restart exclusion; a durable owner/generation or the Provider's
// operation ledger has to. This test pins that limitation.
func TestLifecycleRestartLosesUnknownStage(t *testing.T) {
	first := NewLifecycle()
	if _, err := first.Dispatch("cmd-1", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if err := first.MarkUnknown("cmd-1", "owner-a", 1, "process exited before observation"); err != nil {
		t.Fatal(err)
	}
	if first.MayReplay("cmd-1") {
		t.Fatal("the original instance reported the unknown command as replayable")
	}

	// A new instance stands for a restarted process or another worker.
	restarted := NewLifecycle()
	if _, ok := restarted.Attempt("cmd-1"); ok {
		t.Fatal("a new Lifecycle unexpectedly retained the previous attempt")
	}
	if !restarted.MayReplay("cmd-1") {
		t.Fatal("the restarted instance refused a dispatch it has no record of")
	}
	if _, err := restarted.Dispatch("cmd-1", "owner-a", 1); err != nil {
		t.Fatalf("restarted instance refused the same owner/generation: %v", err)
	}
	attempt, _ := restarted.Attempt("cmd-1")
	if attempt.Stage != StageDispatched {
		t.Fatalf("restarted attempt=%+v", attempt)
	}
}
