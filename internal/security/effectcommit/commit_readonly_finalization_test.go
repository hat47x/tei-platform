package effectcommit

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// This tests the existing memoryProvider control flow, not a real PostgreSQL
// read-only transaction. A failed Tx.Commit in a replay/lookup does not by
// itself establish anything about the original write's historical outcome.
func TestReadOnlyReplayAndLookupTxFinalizationErrorRetainsHistoricalReceipt(t *testing.T) {
	db := newMemoryProvider()
	runner := Runner{Provider: db}
	req := fixtureRequest()
	grant := fixtureGrant(req)

	initial, err := runner.Execute(context.Background(), req, grant)
	if err != nil || initial.Status != StatusCommitted || initial.Receipt.ResourceVersion != 1 {
		t.Fatalf("initial commit: result=%+v err=%v", initial, err)
	}
	if db.version != 1 || db.value != "value" || len(db.records) != 1 {
		t.Fatalf("initial effect not persisted: version=%d value=%q records=%d", db.version, db.value, len(db.records))
	}

	txExitFailure := errors.New("read-only transaction exit failed")
	db.commitApplied = false
	db.commitError = txExitFailure
	wantEvents := []string{"begin", "authority", "lookup", "commit"}

	t.Run("execute-replay", func(t *testing.T) {
		db.events = nil
		got, err := runner.Execute(context.Background(), req, grant)
		var uncertain *CommitUncertainError
		if got != (Result{}) || !errors.As(err, &uncertain) || !errors.Is(err, txExitFailure) {
			t.Fatalf("replay Commit response was silently qualified: result=%+v err=%v", got, err)
		}
		if !reflect.DeepEqual(db.events, wantEvents) {
			t.Fatalf("replay unexpectedly wrote resource/receipt: events=%v want=%v", db.events, wantEvents)
		}
		if db.version != 1 || db.value != "value" || len(db.records) != 1 || db.records[req.OperationID] != initial.Receipt {
			t.Fatalf("read-only replay changed the historical effect: version=%d value=%q records=%v", db.version, db.value, db.records)
		}
	})

	t.Run("lookup-present", func(t *testing.T) {
		db.events = nil
		got, err := runner.Lookup(context.Background(), req, grant)
		var uncertain *CommitUncertainError
		if got != (Result{}) || !errors.Is(err, txExitFailure) || errors.As(err, &uncertain) {
			t.Fatalf("Lookup should preserve raw finalization error, not issue receipt: result=%+v err=%v", got, err)
		}
		if !reflect.DeepEqual(db.events, wantEvents) {
			t.Fatalf("lookup unexpectedly wrote resource/receipt: events=%v want=%v", db.events, wantEvents)
		}
		if db.version != 1 || db.value != "value" || len(db.records) != 1 || db.records[req.OperationID] != initial.Receipt {
			t.Fatalf("lookup finalization changed the historical effect: version=%d value=%q records=%v", db.version, db.value, db.records)
		}
	})

	t.Run("lookup-absent-not-negative-proof", func(t *testing.T) {
		other := req
		other.OperationID = "unrecorded-operation"
		db.events = nil
		got, err := runner.Lookup(context.Background(), other, fixtureGrant(other))
		if got != (Result{}) || !errors.Is(err, txExitFailure) {
			t.Fatalf("failed lookup must not be reported as not-recorded: result=%+v err=%v", got, err)
		}
		if !reflect.DeepEqual(db.events, wantEvents) || len(db.records) != 1 {
			t.Fatalf("failed lookup caused a write or was not read-only: events=%v records=%v", db.events, db.records)
		}
	})

	db.commitError = nil
	recovered, err := runner.Lookup(context.Background(), req, grant)
	if err != nil || recovered.Status != StatusReplayed || recovered.Receipt != initial.Receipt || db.version != 1 || len(db.records) != 1 {
		t.Fatalf("fresh Lookup could not recover original receipt: result=%+v err=%v version=%d records=%d", recovered, err, db.version, len(db.records))
	}
}
