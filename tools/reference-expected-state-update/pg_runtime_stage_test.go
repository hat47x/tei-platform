package expectedstateupdate

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// This is the end-to-end trial for #2564 -> #1129: a real PostgreSQL CAS whose
// response is lost, an unknown stage, a refused replay, a takeover with a higher
// generation, and an independent observation. It is opt-in and requires the
// disposable database named by TEI_EXPECTED_STATE_PG_CONFIRM_ISOLATED.
func TestPGRuntimeStageUnknownTakeoverEndToEnd(t *testing.T) {
	store := openPGStore(t, 0)
	ctx := context.Background()
	if err := store.Seed(ctx, "order:1", "secret-before"); err != nil {
		t.Fatal(err)
	}
	dsn := os.Getenv("TEI_EXPECTED_STATE_PG_DSN")
	reader, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("failed to open the independent reader pool")
	}
	t.Cleanup(func() { reader.Close() })
	independentStatus := func() string {
		t.Helper()
		var value string
		if err := reader.QueryRowContext(ctx, `SELECT value FROM `+PGRecordTable+` WHERE key = 'order:1'`).Scan(&value); err != nil {
			t.Fatalf("independent read: %v", err)
		}
		return value
	}

	stage := NewRuntimeStage()
	first := UpdateRequest{
		Key: "order:1", ExpectedPresent: true, Expected: "secret-before", NewValue: "secret-after",
		IdempotencyKey: "update-a", Authorized: true, Valid: true,
	}
	if _, err := stage.Dispatch("cmd-1", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	applied, err := store.Apply(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if applied.Outcome != OutcomeApplied {
		t.Fatalf("first attempt outcome=%q", applied.Outcome)
	}
	// The response is lost before the caller can observe it.
	if err := stage.MarkUnknown("cmd-1", "owner-a", 1, "connection reset after commit"); err != nil {
		t.Fatal(err)
	}
	if stage.MayReplay("cmd-1") {
		t.Fatal("unknown command was reported as replayable")
	}
	if _, err := stage.Dispatch("cmd-1", "owner-a", 1); err == nil {
		t.Fatal("the same owner/generation replayed an unknown command")
	}
	if _, err := stage.Takeover("cmd-1", "owner-b", 1); err == nil {
		t.Fatal("takeover with an equal generation was accepted")
	}

	// A new owner takes over with a higher generation. The CAS sees the value
	// already changed, so the retry is a business conflict, not a second write.
	if _, err := stage.Takeover("cmd-1", "owner-b", 2); err != nil {
		t.Fatal(err)
	}
	retry := first
	retry.IdempotencyKey = "update-b"
	conflict, err := store.Apply(ctx, retry)
	if err != nil {
		t.Fatal(err)
	}
	if conflict.Outcome != OutcomeConflict || conflict.ExecutionError {
		t.Fatalf("retry outcome=%q executionError=%v, want a business conflict", conflict.Outcome, conflict.ExecutionError)
	}
	if err := stage.Observe("cmd-1", "owner-b", 2, conflict.Outcome); err != nil {
		t.Fatal(err)
	}
	current, _ := stage.Attempt("cmd-1")
	if current.Generation != 2 || current.TakeoverFrom != 1 || current.Stage != StageObserved || current.Outcome != OutcomeConflict {
		t.Fatalf("unexpected final attempt: %+v", current)
	}

	// The independent read sees exactly one application of the first update.
	if value := independentStatus(); value != "secret-after" {
		t.Fatalf("independent read=%q, want the first applied update", value)
	}

	// The run is recordable and reproducible without raw values.
	record, err := RecordRun([]UpdateRequest{first, retry}, []Result{applied, conflict})
	if err != nil {
		t.Fatal(err)
	}
	var encoded bytes.Buffer
	if err := EncodeRecord(&encoded, record); err != nil {
		t.Fatal(err)
	}
	text := encoded.String()
	for _, secret := range []string{"secret-before", "secret-after"} {
		if strings.Contains(text, secret) {
			t.Fatalf("record stored the raw value %q", secret)
		}
	}
	decoded, err := DecodeRecord(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyRecord([]UpdateRequest{first, retry}, []Result{applied, conflict}, decoded); err != nil {
		t.Fatalf("run record was not reproducible: %v", err)
	}
}
