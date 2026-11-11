// Package lifecyclepg is a PostgreSQL-backed stage store for command.Lifecycle.
//
// Every mutation takes a row lock with SELECT ... FOR UPDATE inside one
// transaction, so two processes cannot both read the same "not dispatched"
// state and both write. It applies the same fail-closed rules as the in-memory
// lifecycle.
package lifecyclepg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"

	"tei/internal/command"
)

var schemaPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// Store is a PostgreSQL-backed command.StageLifecycle.
type Store struct {
	db        *sql.DB
	qualified string
}

// Open connects to PostgreSQL and ensures the fixed schema and table exist.
// The schema name is supplied by deployment, never by a request.
func Open(dsn, schema string) (*Store, error) {
	if dsn == "" || !schemaPattern.MatchString(schema) {
		return nil, fmt.Errorf("command lifecycle: a PostgreSQL DSN and a fixed schema name are required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("command lifecycle: open postgres: %w", err)
	}
	store := &Store{db: db, qualified: schema + ".command_lifecycle"}
	ctx := context.Background()
	statements := []string{
		`CREATE SCHEMA IF NOT EXISTS ` + schema,
		`CREATE TABLE IF NOT EXISTS ` + store.qualified + ` (
			command_id TEXT PRIMARY KEY,
			owner TEXT NOT NULL,
			generation BIGINT NOT NULL,
			stage TEXT NOT NULL,
			outcome TEXT NOT NULL DEFAULT '',
			reason TEXT NOT NULL DEFAULT '',
			takeover_from BIGINT NOT NULL DEFAULT 0
		)`,
	}
	if err := ensure(ctx, db, statements); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("command lifecycle: prepare schema: %w", err)
	}
	return store, nil
}

// ensure runs idempotent DDL and retries the catalog races that PostgreSQL can
// report when two processes create the same schema or table at the same time.
func ensure(ctx context.Context, db *sql.DB, statements []string) error {
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		err = nil
		for _, statement := range statements {
			if _, err = db.ExecContext(ctx, statement); err != nil {
				break
			}
		}
		if err == nil {
			return nil
		}
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) {
			return err
		}
		switch pgErr.Code {
		case "23505", "42P06", "42P07", "40P01":
			time.Sleep(time.Duration(25*(attempt+1)) * time.Millisecond)
		default:
			return err
		}
	}
	return err
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
		 FROM `+s.qualified+` WHERE command_id = $1 AND stage <> ''`, commandID)
	return scanAttempt(row)
}

// mutate holds a row lock (SELECT ... FOR UPDATE) across the read, the rule and
// the write. A placeholder row is inserted first so a brand-new command also
// has a row to lock; PostgreSQL has no gap locks, so two concurrent inserts
// would otherwise both proceed.
func (s *Store) mutate(commandID string, apply func(*command.Lifecycle) error) (command.Attempt, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return command.Attempt{}, fmt.Errorf("command lifecycle: begin: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO `+s.qualified+`(command_id, owner, generation, stage)
		 VALUES ($1, '', 0, '') ON CONFLICT (command_id) DO NOTHING`, commandID); err != nil {
		return command.Attempt{}, fmt.Errorf("command lifecycle: claim row: %w", err)
	}
	existing, err := scanAttempt(tx.QueryRowContext(ctx,
		`SELECT command_id, owner, generation, stage, outcome, reason, takeover_from
		 FROM `+s.qualified+` WHERE command_id = $1 FOR UPDATE`, commandID))
	var attempts []command.Attempt
	switch {
	case err == nil:
		if existing.Stage != "" {
			attempts = append(attempts, existing)
		}
	case errors.Is(err, sql.ErrNoRows):
		// The row was deleted between the insert and the lock; fail closed.
		return command.Attempt{}, fmt.Errorf("command lifecycle: row disappeared during lock")
	default:
		return command.Attempt{}, fmt.Errorf("command lifecycle: lock row: %w", err)
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
		`UPDATE `+s.qualified+`
		 SET owner = $2, generation = $3, stage = $4, outcome = $5, reason = $6, takeover_from = $7
		 WHERE command_id = $1`,
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
