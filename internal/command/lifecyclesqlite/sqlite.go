// Package lifecyclesqlite is a SQLite-backed stage store for command.Lifecycle.
//
// It applies the same fail-closed rules as the in-memory lifecycle and runs
// every mutation in an immediate write transaction, so two processes cannot
// both read the same "not dispatched" state and both write. SQLite provides a
// database-level write lock, not row-level locking.
package lifecyclesqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"tei/internal/command"
)

const schema = `CREATE TABLE IF NOT EXISTS command_lifecycle (
	command_id TEXT PRIMARY KEY,
	owner TEXT NOT NULL,
	generation INTEGER NOT NULL,
	stage TEXT NOT NULL,
	outcome TEXT NOT NULL DEFAULT '',
	reason TEXT NOT NULL DEFAULT '',
	takeover_from INTEGER NOT NULL DEFAULT 0
)`

// Store is a SQLite-backed command.StageLifecycle.
type Store struct {
	db *sql.DB
}

// Open opens or creates the stage store at path. A busy timeout makes a
// concurrent writer wait for the other process instead of failing randomly.
func Open(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("command lifecycle: sqlite path is required")
	}
	dsn := fmt.Sprintf("file:%s?_txlock=immediate", filepath.ToSlash(path))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("command lifecycle: open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	ctx := context.Background()
	// The busy timeout, journal mode and schema are set explicitly and retried:
	// two processes may open the same file at the same time, and SQLite can
	// return SQLITE_BUSY while the other process is changing the journal mode
	// or creating the table.
	if err := retryBusy(ctx, db, "PRAGMA busy_timeout=5000"); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := retryBusy(ctx, db, "PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, err
	}
	store := &Store{db: db}
	if err := retryBusy(ctx, db, schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("command lifecycle: create schema: %w", err)
	}
	return store, nil
}

func retryBusy(ctx context.Context, db *sql.DB, statement string) error {
	var err error
	for attempt := 0; attempt < 40; attempt++ {
		if _, err = db.ExecContext(ctx, statement); err == nil {
			return nil
		}
		if !isBusy(err) {
			return err
		}
		time.Sleep(time.Duration(25+attempt*10) * time.Millisecond)
	}
	return err
}

func isBusy(err error) bool {
	message := err.Error()
	return strings.Contains(message, "SQLITE_BUSY") || strings.Contains(message, "database is locked")
}

// Close releases the database handle.
func (s *Store) Close() error { return s.db.Close() }

// Attempt returns the persisted attempt for a command.
func (s *Store) Attempt(commandID string) (command.Attempt, bool) {
	attempt, err := s.load(context.Background(), commandID)
	if err != nil {
		return command.Attempt{}, false
	}
	return attempt, true
}

// MayReplay reports whether the command may be dispatched without a takeover.
//
// It answers true only when the store positively read "no attempt". An attempt
// in any stage refuses: dispatched is in flight (or was abandoned after its
// effect may have committed), unknown needs a takeover, observed is final. A
// failed read also refuses, because "the store could not answer" is not "the
// command was never dispatched"; reading it as replayable would let a caller
// re-run a command whose effect may already be committed.
func (s *Store) MayReplay(commandID string) bool {
	_, err := s.load(context.Background(), commandID)
	return errors.Is(err, sql.ErrNoRows)
}

// Dispatch records a new attempt and persists it atomically.
func (s *Store) Dispatch(commandID, owner string, generation int) (command.Attempt, error) {
	return s.mutate(commandID, func(lifecycle *command.Lifecycle) error {
		_, err := lifecycle.Dispatch(commandID, owner, generation)
		return err
	})
}

// Takeover starts a new attempt for an unknown command and persists it.
func (s *Store) Takeover(commandID, owner string, generation int) (command.Attempt, error) {
	return s.mutate(commandID, func(lifecycle *command.Lifecycle) error {
		_, err := lifecycle.Takeover(commandID, owner, generation)
		return err
	})
}

// Observe records the outcome of the current attempt and persists it.
func (s *Store) Observe(commandID, owner string, generation int, outcome string) error {
	_, err := s.mutate(commandID, func(lifecycle *command.Lifecycle) error {
		return lifecycle.Observe(commandID, owner, generation, outcome)
	})
	return err
}

// MarkUnknown records that the attempt has no reliable outcome and persists it.
func (s *Store) MarkUnknown(commandID, owner string, generation int, reason string) error {
	_, err := s.mutate(commandID, func(lifecycle *command.Lifecycle) error {
		return lifecycle.MarkUnknown(commandID, owner, generation, reason)
	})
	return err
}

func (s *Store) load(ctx context.Context, commandID string) (command.Attempt, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT command_id, owner, generation, stage, outcome, reason, takeover_from
		 FROM command_lifecycle WHERE command_id = ?`, commandID)
	return scanAttempt(row)
}

// mutate runs one rule under BEGIN IMMEDIATE so a concurrent process cannot
// interleave a read and a write on the same command.
func (s *Store) mutate(commandID string, apply func(*command.Lifecycle) error) (command.Attempt, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return command.Attempt{}, fmt.Errorf("command lifecycle: begin: %w", err)
	}
	defer tx.Rollback()
	var attempts []command.Attempt
	existing, err := scanAttempt(tx.QueryRowContext(ctx,
		`SELECT command_id, owner, generation, stage, outcome, reason, takeover_from
		 FROM command_lifecycle WHERE command_id = ?`, commandID))
	switch {
	case err == nil:
		attempts = append(attempts, existing)
	case errors.Is(err, sql.ErrNoRows):
		// No attempt yet.
	default:
		return command.Attempt{}, fmt.Errorf("command lifecycle: read: %w", err)
	}
	lifecycle := command.LifecycleFromAttempts(attempts)
	if err := apply(lifecycle); err != nil {
		return command.Attempt{}, err
	}
	attempt, ok := lifecycle.Attempt(commandID)
	if !ok {
		return command.Attempt{}, fmt.Errorf("command lifecycle: mutation produced no attempt")
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO command_lifecycle(command_id, owner, generation, stage, outcome, reason, takeover_from)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(command_id) DO UPDATE SET
		   owner = excluded.owner,
		   generation = excluded.generation,
		   stage = excluded.stage,
		   outcome = excluded.outcome,
		   reason = excluded.reason,
		   takeover_from = excluded.takeover_from`,
		attempt.CommandID, attempt.Owner, attempt.Generation, string(attempt.Stage),
		attempt.Outcome, attempt.Reason, attempt.TakeoverFrom); err != nil {
		return command.Attempt{}, fmt.Errorf("command lifecycle: write: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return command.Attempt{}, fmt.Errorf("command lifecycle: commit: %w", err)
	}
	return attempt, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAttempt(row rowScanner) (command.Attempt, error) {
	var attempt command.Attempt
	var stage string
	var generation, takeoverFrom int64
	if err := row.Scan(&attempt.CommandID, &attempt.Owner, &generation, &stage,
		&attempt.Outcome, &attempt.Reason, &takeoverFrom); err != nil {
		return command.Attempt{}, err
	}
	attempt.Generation = int(generation)
	attempt.TakeoverFrom = int(takeoverFrom)
	attempt.Stage = command.Stage(stage)
	return attempt, nil
}

var _ command.StageLifecycle = (*Store)(nil)
