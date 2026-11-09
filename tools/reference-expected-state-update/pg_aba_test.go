package expectedstateupdate

import (
	"context"
	"testing"
)

// RCAS-01 (ABA) for #2564: value-equality CAS alone does not bind one logical
// command to one effect.
//
// Sequence: apply with expected=secret-before -> the response is lost -> a third
// party legitimately restores secret-before -> a takeover retries the same
// logical command with a NEW idempotency key. The CAS matches again and applies
// a second effect, overwriting the third party's value. This test pins the
// counterexample; it is NOT evidence that the reference prevents ABA.
//
// The positive control shows that reusing the SAME idempotency key after the
// takeover does not write again (IDEMPOTENT_REPLAY). The defect therefore
// depends on both value-equality CAS and a changed idempotency key.
func TestPGAbaReapplyAfterRestoreWithNewIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	store := openPGStore(t, 0)
	if err := store.Seed(ctx, "order:1", "secret-before"); err != nil {
		t.Fatal(err)
	}

	stage := NewRuntimeStage()
	if _, err := stage.Dispatch("cmd-aba", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	first := UpdateRequest{
		Key: "order:1", ExpectedPresent: true, Expected: "secret-before", NewValue: "secret-after",
		IdempotencyKey: "update-a", Authorized: true, Valid: true,
	}
	applied, err := store.Apply(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if applied.Outcome != OutcomeApplied {
		t.Fatalf("first outcome=%q", applied.Outcome)
	}
	if value, _, err := store.Value(ctx, "order:1"); err != nil || value != "secret-after" {
		t.Fatalf("after first apply value=%q err=%v", value, err)
	}

	// The response is lost before the caller can observe it.
	if err := stage.MarkUnknown("cmd-aba", "owner-a", 1, "response lost after commit"); err != nil {
		t.Fatal(err)
	}

	// A separate, legitimate transaction restores the old value. This is the
	// A -> B -> A in ABA, not a replay of the first command.
	if _, err := store.db.ExecContext(ctx,
		`UPDATE `+PGRecordTable+` SET value = 'secret-before' WHERE key = 'order:1'`); err != nil {
		t.Fatal(err)
	}

	if _, err := stage.Takeover("cmd-aba", "owner-b", 2); err != nil {
		t.Fatal(err)
	}

	// Positive control: the SAME idempotency key is an idempotent replay and
	// does not write. The restored value survives.
	replay := first
	replayed, err := store.Apply(ctx, replay)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Outcome != OutcomeIdempotentReplay {
		t.Fatalf("same-key retry outcome=%q, want %s", replayed.Outcome, OutcomeIdempotentReplay)
	}
	if value, _, err := store.Value(ctx, "order:1"); err != nil || value != "secret-before" {
		t.Fatalf("same-key retry wrote: value=%q err=%v", value, err)
	}

	// The counterexample: a NEW idempotency key makes the value CAS match again.
	retry := first
	retry.IdempotencyKey = "update-b"
	second, err := store.Apply(ctx, retry)
	if err != nil {
		t.Fatal(err)
	}
	if second.Outcome != OutcomeApplied {
		t.Fatalf("ABA retry outcome=%q reason=%q, want %s", second.Outcome, second.Reason, OutcomeApplied)
	}
	if second.Observation.Applied != 1 || second.Observation.Comparisons != 1 {
		t.Fatalf("ABA retry observation=%+v", second.Observation)
	}
	// The third party's restore was overwritten: one logical command produced
	// two effects and lost the intervening write.
	if value, _, err := store.Value(ctx, "order:1"); err != nil || value != "secret-after" {
		t.Fatalf("after ABA retry value=%q err=%v", value, err)
	}
	var receipts int
	if err := store.db.QueryRowContext(ctx,
		`SELECT count(*) FROM `+PGAppliedTable+` WHERE key = 'order:1'`).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if receipts != 2 {
		t.Fatalf("idempotency receipts=%d, want 2 (one logical command, two effects)", receipts)
	}
	t.Logf("ABA reproduced: first=%s sameKeyReplay=%s newKeyRetry=%s receipts=%d",
		applied.Outcome, replayed.Outcome, second.Outcome, receipts)

	// The lifecycle recorded the takeover as an observed APPLIED: the stage
	// model did not detect the ABA either.
	if err := stage.Observe("cmd-aba", "owner-b", 2, second.Outcome); err != nil {
		t.Fatal(err)
	}
	final, _ := stage.Attempt("cmd-aba")
	if final.Stage != StageObserved || final.Outcome != OutcomeApplied || final.TakeoverFrom != 1 || final.Generation != 2 {
		t.Fatalf("final attempt=%+v", final)
	}
}
