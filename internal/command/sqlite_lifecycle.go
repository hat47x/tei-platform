package command

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const sqliteLifecycleSchema = `CREATE TABLE IF NOT EXISTS command_lifecycle (
	command_id TEXT PRIMARY KEY,
	owner TEXT NOT NULL,
	generation INTEGER NOT NULL,
	stage TEXT NOT NULL,
	outcome TEXT NOT NULL DEFAULT '',
	reason TEXT NOT NULL DEFAULT '',
	takeover_from INTEGER NOT NULL DEFAULT 0
)`

// SQLiteLifecycle is a SQLite-backed Command stage store. Every mutation runs
// in an immediate write transaction (`_txlock=immediate`), so two processes
// cannot both read the same "not dispatched" state and both write. It applies
// the same fail-closed rules as the in-memory Lifecycle.
type SQLiteLifecycle struct {
	db *sql.DB
}

// OpenSQLiteLifecycle opens or creates the stage store at path. A busy timeout
// makes a concurrent writer wait for the other process instead of failing
// randomly.
func OpenSQLiteLifecycle(path string) (*SQLiteLifecycle, error) {
	if path == "" {
		return nil, fmt.Errorf("command lifecycle: sqlite path is required")
	}
	dsn := fmt.Sprintf("file:%s?_txlock=immediate&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)",
		filepath.ToSlash(path))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("command lifecycle: open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	store := &SQLiteLifecycle{db: db}
	if _, err := db.ExecContext(context.Background(), sqliteLifecycleSchema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("command lifecycle: create schema: %w", err)
	}
	return store, nil
}

// Close releases the database handle.
func (s *SQLiteLifecycle) Close() error { return s.db.Close() }

// Attempt returns the persisted attempt for a command.
func (s *SQLiteLifecycle) Attempt(commandID string) (Attempt, bool) {
	attempt, err := s.load(context.Background(), commandID)
	if err != nil {
		return Attempt{}, false
	}
	return attempt, true
}

// MayReplay reports whether the command may be dispatched without a takeover.
func (s *SQLiteLifecycle) MayReplay(commandID string) bool {
	attempt, ok := s.Attempt(commandID)
	if !ok {
		return true
	}
	return attempt.Stage != StageUnknown && attempt.Stage != StageObserved
}

// Dispatch records a new attempt and persists it atomically.
func (s *SQLiteLifecycle) Dispatch(commandID, owner string, generation int) (Attempt, error) {
	return s.mutate(commandID, func(lifecycle *Lifecycle) error {
		_, err := lifecycle.Dispatch(commandID, owner, generation)
		return err
	})
}

// Takeover starts a new attempt for an unknown command and persists it.
func (s *SQLiteLifecycle) Takeover(commandID, owner string, generation int) (Attempt, error) {
	return s.mutate(commandID, func(lifecycle *Lifecycle) error {
		_, err := lifecycle.Takeover(commandID, owner, generation)
		return err
	})
}

// Observe records the outcome of the current attempt and persists it.
func (s *SQLiteLifecycle) Observe(commandID, owner string, generation int, outcome string) error {
	_, err := s.mutate(commandID, func(lifecycle *Lifecycle) error {
		return lifecycle.Observe(commandID, owner, generation, outcome)
	})
	return err
}

// MarkUnknown records that the attempt has no reliable outcome and persists it.
func (s *SQLiteLifecycle) MarkUnknown(commandID, owner string, generation int, reason string) error {
	_, err := s.mutate(commandID, func(lifecycle *Lifecycle) error {
		return lifecycle.MarkUnknown(commandID, owner, generation, reason)
	})
	return err
}

func (s *SQLiteLifecycle) load(ctx context.Context, commandID string) (Attempt, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT command_id, owner, generation, stage, outcome, reason, takeover_from
		 FROM command_lifecycle WHERE command_id = ?`, commandID)
	return scanAttempt(row)
}

// mutate runs one rule under BEGIN IMMEDIATE so a concurrent process cannot
// interleave a read and a write on the same command.
func (s *SQLiteLifecycle) mutate(commandID string, apply func(*Lifecycle) error) (Attempt, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Attempt{}, fmt.Errorf("command lifecycle: begin: %w", err)
	}
	defer tx.Rollback()
	lifecycle := NewLifecycle()
	existing, err := scanAttempt(tx.QueryRowContext(ctx,
		`SELECT command_id, owner, generation, stage, outcome, reason, takeover_from
		 FROM command_lifecycle WHERE command_id = ?`, commandID))
	switch {
	case err == nil:
		lifecycle.attempts[commandID] = existing
	case errors.Is(err, sql.ErrNoRows):
		// No attempt yet.
	default:
		return Attempt{}, fmt.Errorf("command lifecycle: read: %w", err)
	}
	if err := apply(lifecycle); err != nil {
		return Attempt{}, err
	}
	attempt, ok := lifecycle.Attempt(commandID)
	if !ok {
		return Attempt{}, fmt.Errorf("command lifecycle: mutation produced no attempt")
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
		return Attempt{}, fmt.Errorf("command lifecycle: write: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Attempt{}, fmt.Errorf("command lifecycle: commit: %w", err)
	}
	return attempt, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAttempt(row rowScanner) (Attempt, error) {
	var attempt Attempt
	var stage string
	var generation, takeoverFrom int64
	if err := row.Scan(&attempt.CommandID, &attempt.Owner, &generation, &stage,
		&attempt.Outcome, &attempt.Reason, &takeoverFrom); err != nil {
		return Attempt{}, err
	}
	attempt.Generation = int(generation)
	attempt.TakeoverFrom = int(takeoverFrom)
	attempt.Stage = Stage(stage)
	return attempt, nil
}
