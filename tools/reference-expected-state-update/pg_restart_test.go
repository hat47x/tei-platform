package expectedstateupdate

import (
	"context"
	"testing"
)

// RCAS-02 (#2564): RuntimeStage is an in-memory map. After a process restart (a
// new RuntimeStage), the previous unknown attempt is gone, so the new instance
// accepts a dispatch of the same command. This test pins the limitation and
// shows what still protects the effect: the PostgreSQL idempotency record, as
// long as the restarted worker reuses the same operation identity.
//
// With a new idempotency key the same retry re-applies; see
// TestPGAbaReapplyAfterRestoreWithNewIdempotencyKey.
func TestPGRestartLosesStageButReceiptStopsSameKeyReapply(t *testing.T) {
	ctx := context.Background()
	store := openPGStore(t, 0)
	if err := store.Seed(ctx, "order:1", "secret-before"); err != nil {
		t.Fatal(err)
	}

	request := UpdateRequest{
		Key: "order:1", ExpectedPresent: true, Expected: "secret-before", NewValue: "secret-after",
		IdempotencyKey: "update-a", Authorized: true, Valid: true,
	}
	first := NewRuntimeStage()
	if _, err := first.Dispatch("cmd-restart", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	applied, err := store.Apply(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if applied.Outcome != OutcomeApplied {
		t.Fatalf("first outcome=%q", applied.Outcome)
	}
	if err := first.MarkUnknown("cmd-restart", "owner-a", 1, "process exited before observation"); err != nil {
		t.Fatal(err)
	}
	if first.MayReplay("cmd-restart") {
		t.Fatal("the original instance reported the unknown command as replayable")
	}

	// A new process has no record of the unknown attempt.
	restarted := NewRuntimeStage()
	if _, ok := restarted.Attempt("cmd-restart"); ok {
		t.Fatal("a restarted RuntimeStage retained the previous attempt")
	}
	if !restarted.MayReplay("cmd-restart") {
		t.Fatal("the restarted instance refused a dispatch it has no record of")
	}
	if _, err := restarted.Dispatch("cmd-restart", "owner-a", 1); err != nil {
		t.Fatalf("the restarted instance refused the same owner/generation: %v", err)
	}

	// Reusing the operation identity is an idempotent replay: the database
	// record, not the stage, prevents a second effect.
	replayed, err := store.Apply(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Outcome != OutcomeIdempotentReplay {
		t.Fatalf("restarted retry outcome=%q, want %s", replayed.Outcome, OutcomeIdempotentReplay)
	}
	if replayed.Observation.Applied != 0 || replayed.Observation.IdempotentReplays != 1 {
		t.Fatalf("restarted retry observation=%+v", replayed.Observation)
	}
	if value, _, err := store.Value(ctx, "order:1"); err != nil || value != "secret-after" {
		t.Fatalf("restarted retry changed the value: %q err=%v", value, err)
	}
	var receipts int
	if err := store.db.QueryRowContext(ctx,
		`SELECT count(*) FROM `+PGAppliedTable+` WHERE key = 'order:1'`).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if receipts != 1 {
		t.Fatalf("idempotency receipts=%d, want 1", receipts)
	}
	if err := restarted.Observe("cmd-restart", "owner-a", 1, replayed.Outcome); err != nil {
		t.Fatal(err)
	}
	final, _ := restarted.Attempt("cmd-restart")
	if final.Stage != StageObserved || final.Outcome != OutcomeIdempotentReplay {
		t.Fatalf("restarted attempt=%+v", final)
	}
	t.Logf("restart lost the unknown stage: newInstanceMayReplay=true providerReceipt=%s receipts=%d",
		replayed.Outcome, receipts)
}
