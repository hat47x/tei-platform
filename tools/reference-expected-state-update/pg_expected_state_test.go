package expectedstateupdate

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// These tests are a REAL PostgreSQL probe for #2564 item 1: compare-and-swap in
// a database, not in a synthetic state map. They are opt-in and require a
// dedicated, disposable database. The tests TRUNCATE the two fixed tables, so
// never point them at a live application database.
//
//	CREATE DATABASE tei_expected_state_e2e;
//	TEI_EXPECTED_STATE_PG_DSN='postgres://.../tei_expected_state_e2e?sslmode=disable' \
//	  TEI_EXPECTED_STATE_PG_CONFIRM_ISOLATED=tei_expected_state_e2e \
//	  go test ./tools/reference-expected-state-update/ -run TestPG -v
func openPGStore(t *testing.T, level sql.IsolationLevel) *PGStore {
	t.Helper()
	dsn := os.Getenv("TEI_EXPECTED_STATE_PG_DSN")
	if dsn == "" {
		t.Skip("real PostgreSQL evidence unavailable: TEI_EXPECTED_STATE_PG_DSN unset")
	}
	if os.Getenv("TEI_EXPECTED_STATE_PG_CONFIRM_ISOLATED") != "tei_expected_state_e2e" {
		t.Fatal("refusing to truncate tables without the explicit disposable database confirmation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("failed to initialize PostgreSQL driver")
	}
	t.Cleanup(func() { db.Close() })
	store := NewPGStore(db)
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.Reset(ctx); err != nil {
		t.Fatal(err)
	}
	if level != 0 {
		store = store.WithIsolation(level)
	}
	return store
}

func pgRequest(newValue, idempotencyKey string) UpdateRequest {
	return UpdateRequest{
		Key: "order:1", ExpectedPresent: true, Expected: "old", NewValue: newValue,
		IdempotencyKey: idempotencyKey, Authorized: true, Valid: true,
	}
}

func TestPGConflictIsABusinessResultAndKeepsTheFirstUpdate(t *testing.T) {
	ctx := context.Background()
	store := openPGStore(t, 0)
	if err := store.Seed(ctx, "order:1", "old"); err != nil {
		t.Fatal(err)
	}

	first, err := store.Apply(ctx, pgRequest("A", "update-a"))
	if err != nil {
		t.Fatal(err)
	}
	if first.Outcome != OutcomeApplied || first.ExecutionError {
		t.Fatalf("first outcome=%q executionError=%v", first.Outcome, first.ExecutionError)
	}

	second, err := store.Apply(ctx, pgRequest("B", "update-b"))
	if err != nil {
		t.Fatal(err)
	}
	if second.Outcome != OutcomeConflict || second.ExecutionError {
		t.Fatalf("second outcome=%q executionError=%v", second.Outcome, second.ExecutionError)
	}
	if second.Reason != ReasonExpectedMismatch {
		t.Fatalf("second reason=%q", second.Reason)
	}
	// The raw current value must not be echoed back: the comparison happens in
	// the database and the result carries no current value.
	if second.CurrentValue != "" {
		t.Fatalf("conflict leaked the current value: %q", second.CurrentValue)
	}
	if got := second.Observation.LostUpdatesPrevented; got != 1 {
		t.Fatalf("lostUpdatesPrevented=%d", got)
	}
	// Only the supplied expected value is counted; the current value is not
	// read back into the observation.
	if got := second.Observation.ComparisonInputBytes; got != len("old") {
		t.Fatalf("comparisonInputBytes=%d, want %d", got, len("old"))
	}

	value, ok, err := store.Value(ctx, "order:1")
	if err != nil || !ok {
		t.Fatalf("read final value: ok=%v err=%v", ok, err)
	}
	if value != "A" {
		t.Fatalf("final value=%q, want A", value)
	}
}

func TestPGIdempotentReplayDoesNotApplyTwice(t *testing.T) {
	ctx := context.Background()
	store := openPGStore(t, 0)
	if err := store.Seed(ctx, "order:1", "old"); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Apply(ctx, pgRequest("A", "update-a")); err != nil {
		t.Fatal(err)
	}
	replay, err := store.Apply(ctx, pgRequest("A", "update-a"))
	if err != nil {
		t.Fatal(err)
	}
	if replay.Outcome != OutcomeIdempotentReplay || replay.ExecutionError {
		t.Fatalf("replay outcome=%q executionError=%v", replay.Outcome, replay.ExecutionError)
	}
	if replay.Before != "old" || replay.NewValue != "A" {
		t.Fatalf("replay before=%q newValue=%q", replay.Before, replay.NewValue)
	}

	reused, err := store.Apply(ctx, pgRequest("C", "update-a"))
	if err != nil {
		t.Fatal(err)
	}
	if reused.Outcome != OutcomeRejected || reused.Reason != ReasonIdempotencyKeyReused {
		t.Fatalf("reused outcome=%q reason=%q", reused.Outcome, reused.Reason)
	}

	value, _, err := store.Value(ctx, "order:1")
	if err != nil {
		t.Fatal(err)
	}
	if value != "A" {
		t.Fatalf("final value=%q, want A", value)
	}
}

func TestPGDeniedAndInvalidDoNotClaimTheIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	store := openPGStore(t, 0)
	if err := store.Seed(ctx, "order:1", "old"); err != nil {
		t.Fatal(err)
	}

	denied := pgRequest("A", "key-denied")
	denied.Authorized = false
	if result, err := store.Apply(ctx, denied); err != nil || result.Outcome != OutcomeDenied {
		t.Fatalf("denied outcome=%v err=%v", result.Outcome, err)
	}
	invalid := pgRequest("A", "key-validation")
	invalid.Valid = false
	if result, err := store.Apply(ctx, invalid); err != nil || result.Outcome != OutcomeInvalid {
		t.Fatalf("invalid outcome=%v err=%v", result.Outcome, err)
	}

	// The rejected keys must be reusable: nothing was claimed.
	if result, err := store.Apply(ctx, pgRequest("A", "key-denied")); err != nil || result.Outcome != OutcomeApplied {
		t.Fatalf("reuse of denied key outcome=%v err=%v", result.Outcome, err)
	}
}

func TestPGMissingRecordIsDistinctFromConflict(t *testing.T) {
	ctx := context.Background()
	store := openPGStore(t, 0)

	result, err := store.Apply(ctx, pgRequest("A", "update-a"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeMissing || result.Reason != ReasonMissingRecord || result.ExecutionError {
		t.Fatalf("outcome=%q reason=%q executionError=%v", result.Outcome, result.Reason, result.ExecutionError)
	}
	// A missing record must not leave an idempotency claim behind.
	if err := store.Seed(ctx, "order:1", "old"); err != nil {
		t.Fatal(err)
	}
	if result, err := store.Apply(ctx, pgRequest("A", "update-a")); err != nil || result.Outcome != OutcomeApplied {
		t.Fatalf("after seeding: outcome=%v err=%v", result.Outcome, err)
	}
}

// Two connections send the same update against the same expected value. Exactly
// one may apply; the other must observe a conflict, and the first update stays.
func TestPGTwoConnectionsApplyExactlyOneUpdate(t *testing.T) {
	ctx := context.Background()
	store := openPGStore(t, 0)
	if err := store.Seed(ctx, "order:1", "old"); err != nil {
		t.Fatal(err)
	}

	type outcome struct {
		result Result
		err    error
	}
	results := make([]outcome, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			result, err := store.Apply(ctx, pgRequest(string(rune('A'+i)), "update-"+string(rune('a'+i))))
			results[i] = outcome{result, err}
		}(i)
	}
	close(start)
	wg.Wait()

	applied, conflicts := 0, 0
	for _, item := range results {
		if item.err != nil {
			t.Fatalf("concurrent apply failed: %v", item.err)
		}
		switch item.result.Outcome {
		case OutcomeApplied:
			applied++
		case OutcomeConflict:
			conflicts++
		default:
			t.Fatalf("unexpected outcome %q", item.result.Outcome)
		}
	}
	if applied != 1 || conflicts != 1 {
		t.Fatalf("applied=%d conflicts=%d, want 1/1", applied, conflicts)
	}
}

// A single-statement CAS at READ COMMITTED needs no application-level retry:
// the blocked writer re-evaluates the predicate after the row lock is released.
func TestPGReadCommittedCASNeedsNoRetry(t *testing.T) {
	ctx := context.Background()
	store := openPGStore(t, 0)
	if err := store.Seed(ctx, "order:1", "old"); err != nil {
		t.Fatal(err)
	}

	const writers = 8
	results := make([]Result, writers)
	errs := make([]error, writers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], errs[i] = store.Apply(ctx, pgRequest("W"+string(rune('0'+i)), "writer-"+string(rune('0'+i))))
		}(i)
	}
	close(start)
	wg.Wait()

	applied := 0
	for i := 0; i < writers; i++ {
		if errs[i] != nil {
			t.Fatalf("writer %d hit an execution error and would need a retry: %v", i, errs[i])
		}
		switch results[i].Outcome {
		case OutcomeApplied:
			applied++
		case OutcomeConflict:
		default:
			t.Fatalf("writer %d outcome %q", i, results[i].Outcome)
		}
	}
	if applied != 1 {
		t.Fatalf("applied=%d, want exactly 1", applied)
	}
}

// The contrast that justifies the single-statement design: a read-then-write at
// REPEATABLE READ does serialization-fail (SQLSTATE 40001) and would require a
// caller retry.
func TestPGRepeatableReadReadThenWriteFailsSerialization(t *testing.T) {
	ctx := context.Background()
	store := openPGStore(t, 0)
	if err := store.Seed(ctx, "order:1", "old"); err != nil {
		t.Fatal(err)
	}

	tx1, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		t.Fatal(err)
	}
	defer tx1.Rollback()
	tx2, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		t.Fatal(err)
	}
	defer tx2.Rollback()

	// Both transactions read before writing, which fixes their snapshots.
	var value1, value2 string
	if err := tx1.QueryRowContext(ctx, `SELECT value FROM `+PGRecordTable+` WHERE key = $1`, "order:1").Scan(&value1); err != nil {
		t.Fatal(err)
	}
	if err := tx2.QueryRowContext(ctx, `SELECT value FROM `+PGRecordTable+` WHERE key = $1`, "order:1").Scan(&value2); err != nil {
		t.Fatal(err)
	}
	if value1 != "old" || value2 != "old" {
		t.Fatalf("unexpected snapshot values %q/%q", value1, value2)
	}
	if _, err := tx1.ExecContext(ctx, `UPDATE `+PGRecordTable+` SET value = 'A' WHERE key = $1 AND value = 'old'`, "order:1"); err != nil {
		t.Fatal(err)
	}
	if err := tx1.Commit(); err != nil {
		t.Fatal(err)
	}

	_, err = tx2.ExecContext(ctx, `UPDATE `+PGRecordTable+` SET value = 'B' WHERE key = $1 AND value = 'old'`, "order:1")
	if err == nil {
		t.Fatal("read-then-write at REPEATABLE READ unexpectedly succeeded without a serialization failure")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "40001" {
		t.Fatalf("error=%v, want SQLSTATE 40001", err)
	}
}

func TestPGDigestComparisonIsRefused(t *testing.T) {
	ctx := context.Background()
	store := openPGStore(t, 0)
	if err := store.Seed(ctx, "order:1", "old"); err != nil {
		t.Fatal(err)
	}

	req := pgRequest("A", "update-a")
	req.Compare = CompareDigest
	req.ExpectedDigest = valueDigest("old")
	if _, err := store.Apply(ctx, req); err == nil {
		t.Fatal("digest comparison was silently accepted by the PostgreSQL store")
	}
}

// 実DBでも、同じキー・同じ対象・同じ新値で期待値だけが違う要求は再送として扱わない(#2564)。
func TestPGIdempotencyKeyWithAnotherExpectationIsRejectedNotReplayed(t *testing.T) {
	ctx := context.Background()
	store := openPGStore(t, 0)
	if err := store.Seed(ctx, "order:1", "old"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Apply(ctx, pgRequest("A", "update-a")); err != nil {
		t.Fatal(err)
	}
	other := pgRequest("A", "update-a")
	other.Expected = "other"
	got, err := store.Apply(ctx, other)
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != OutcomeRejected || got.Reason != ReasonIdempotencyKeyReused {
		t.Fatalf("outcome=%q reason=%q, want %s/%s", got.Outcome, got.Reason, OutcomeRejected, ReasonIdempotencyKeyReused)
	}
	if got.Before != "" {
		t.Fatalf("the first receipt leaked into a different request: before=%q", got.Before)
	}
	if value, _, err := store.Value(ctx, "order:1"); err != nil || value != "A" {
		t.Fatalf("final value=%q err=%v", value, err)
	}
}
