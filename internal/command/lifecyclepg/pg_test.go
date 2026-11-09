package lifecyclepg

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"tei/internal/command"
)

type testDB struct {
	dsn    string
	schema string
}

// newTestDB uses a dedicated schema per test. The test drops only that schema.
func newTestDB(t *testing.T) testDB {
	t.Helper()
	dsn := os.Getenv("TEI_COMMAND_PG_DSN")
	if dsn == "" {
		t.Skip("real PostgreSQL evidence unavailable: TEI_COMMAND_PG_DSN unset")
	}
	hash := sha256.Sum256([]byte(t.Name()))
	database := testDB{dsn: dsn, schema: "tei_cmd_" + hex.EncodeToString(hash[:9])}
	t.Cleanup(func() {
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			return
		}
		defer db.Close()
		_, _ = db.Exec(`DROP SCHEMA IF EXISTS ` + database.schema + ` CASCADE`)
	})
	return database
}

func (d testDB) open(t *testing.T) *Store {
	t.Helper()
	store, err := Open(d.dsn, d.schema)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestStorePersistsStagesAcrossReconnect(t *testing.T) {
	database := newTestDB(t)
	first := database.open(t)
	if _, err := first.Dispatch("cmd-1", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if err := first.MarkUnknown("cmd-1", "owner-a", 1, "response lost"); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second := database.open(t)
	defer second.Close()
	attempt, ok := second.Attempt("cmd-1")
	if !ok || attempt.Stage != command.StageUnknown || attempt.Reason != "response lost" {
		t.Fatalf("reopened attempt=%+v ok=%v", attempt, ok)
	}
	if second.MayReplay("cmd-1") {
		t.Fatal("an unknown attempt was reported as replayable after reconnect")
	}
	if _, err := second.Dispatch("cmd-1", "owner-a", 1); err == nil {
		t.Fatal("a same-attempt replay was accepted after reconnect")
	}
	if _, err := second.Takeover("cmd-1", "owner-b", 2); err != nil {
		t.Fatal(err)
	}
	if err := second.Observe("cmd-1", "owner-b", 2, command.OutcomeCommitted); err != nil {
		t.Fatal(err)
	}
	if _, err := second.Dispatch("cmd-1", "owner-c", 3); err == nil {
		t.Fatal("an observed attempt was re-dispatched")
	}
}

// TestRowLockBlocksASecondWriter shows that the store really takes a row lock:
// while one transaction holds SELECT ... FOR UPDATE, another store cannot
// finish its dispatch.
func TestRowLockBlocksASecondWriter(t *testing.T) {
	database := newTestDB(t)
	first := database.open(t)
	defer first.Close()
	if _, err := first.Dispatch("cmd-lock", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	second := database.open(t)
	defer second.Close()

	ctx := context.Background()
	tx, err := first.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var locked string
	if err := tx.QueryRowContext(ctx,
		`SELECT command_id FROM `+first.qualified+` WHERE command_id = $1 FOR UPDATE`, "cmd-lock").Scan(&locked); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := second.Dispatch("cmd-lock", "owner-b", 2)
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("the second writer did not block on the row lock: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
		// The rule rejects the second dispatch, but only after the lock was
		// released.
	case <-time.After(5 * time.Second):
		t.Fatal("the second writer did not proceed after the lock was released")
	}
}
