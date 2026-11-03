package expectedstateupdate

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// This test wires the real-PostgreSQL compare-and-swap path to the observation
// record (#2564 items 1, 3 and 4). It is opt-in and requires the disposable
// database named by TEI_EXPECTED_STATE_PG_CONFIRM_ISOLATED.
func TestPGRunIsRecordedAndReproducedWithoutRawValues(t *testing.T) {
	store := openPGStore(t, 0)
	ctx := context.Background()
	if err := store.Seed(ctx, "order:1", "secret-before"); err != nil {
		t.Fatal(err)
	}

	requests := []UpdateRequest{
		{Key: "order:1", ExpectedPresent: true, Expected: "secret-before", NewValue: "secret-after", IdempotencyKey: "update-a", Authorized: true, Valid: true},
		// Same expected state: the second write must observe a conflict.
		{Key: "order:1", ExpectedPresent: true, Expected: "secret-before", NewValue: "secret-other", IdempotencyKey: "update-b", Authorized: true, Valid: true},
		// Replay of the first request.
		{Key: "order:1", ExpectedPresent: true, Expected: "secret-before", NewValue: "secret-after", IdempotencyKey: "update-a", Authorized: true, Valid: true},
	}
	results := make([]Result, len(requests))
	for i, request := range requests {
		result, err := store.Apply(ctx, request)
		if err != nil {
			t.Fatalf("apply %d: %v", i, err)
		}
		results[i] = result
	}
	if results[0].Outcome != OutcomeApplied || results[1].Outcome != OutcomeConflict || results[2].Outcome != OutcomeIdempotentReplay {
		t.Fatalf("unexpected outcomes: %q / %q / %q", results[0].Outcome, results[1].Outcome, results[2].Outcome)
	}
	if results[1].CurrentValue != "" {
		t.Fatalf("conflict leaked the current value: %q", results[1].CurrentValue)
	}

	record, err := RecordRun(requests, results)
	if err != nil {
		t.Fatal(err)
	}
	var encoded bytes.Buffer
	if err := EncodeRecord(&encoded, record); err != nil {
		t.Fatal(err)
	}
	text := encoded.String()
	for _, secret := range []string{"secret-before", "secret-after", "secret-other"} {
		if strings.Contains(text, secret) {
			t.Fatalf("record stored the raw value %q:\n%s", secret, text)
		}
	}
	if !strings.Contains(text, `"conflictRejections": 1`) {
		t.Fatalf("record does not carry the conflict observation:\n%s", text)
	}

	decoded, err := DecodeRecord(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Digest != record.Digest {
		t.Fatalf("digest changed across the round trip")
	}
	if err := VerifyRecord(requests, results, decoded); err != nil {
		t.Fatalf("recorded real-DB run was not reproducible: %v", err)
	}

	// The observation record is data, not a second source of truth: the database
	// still holds the first update.
	value, ok, err := store.Value(ctx, "order:1")
	if err != nil || !ok {
		t.Fatalf("read final value: ok=%v err=%v", ok, err)
	}
	if value != "secret-after" {
		t.Fatalf("final value=%q, want the first applied update", value)
	}
}
