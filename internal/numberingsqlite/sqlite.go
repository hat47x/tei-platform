// Package numberingsqlite is a SQLite-backed numbering.CounterStore.
//
// It keeps the same version check as the in-memory store: Load returns the
// counter and its version, and Save writes only when the version still matches.
// The write runs in an immediate transaction, so two processes cannot both read
// the same counter and both issue from it.
//
// X-TEA wrote the counter with an optimistic update whose result it never
// inspected (D:Session.java L1995–2002), so a concurrent caller silently lost an
// issuance. A version check here turns that into a reported conflict instead.
package numberingsqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"tei/internal/numbering"
)

const schema = `CREATE TABLE IF NOT EXISTS numbering_counter (
	rule_id TEXT PRIMARY KEY,
	current INTEGER NOT NULL,
	version INTEGER NOT NULL
)`

// Store is a SQLite-backed numbering.CounterStore.
type Store struct {
	db *sql.DB
}

// Open opens or creates the counter store at path.
//
// A busy timeout makes a concurrent writer wait for the other process instead of
// failing randomly, and the journal mode is set explicitly so the wait is the
// one SQLite promises.
func Open(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("numbering store: sqlite path is required")
	}
	dsn := fmt.Sprintf("file:%s?_txlock=immediate", filepath.ToSlash(path))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("numbering store: open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	ctx := context.Background()
	if err := retryBusy(ctx, db, "PRAGMA busy_timeout=5000"); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := retryBusy(ctx, db, "PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := retryBusy(ctx, db, schema); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close closes the store.
func (s *Store) Close() error { return s.db.Close() }

// Load returns the counter and its version. A rule with no row reads as zero at
// version zero, which is the state a rule starts in.
func (s *Store) Load(ctx context.Context, ruleID string) (numbering.State, int, error) {
	if ruleID == "" {
		return numbering.State{}, 0, fmt.Errorf("numbering store: rule id is required")
	}
	var current sql.NullInt64
	var version sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`SELECT current, version FROM numbering_counter WHERE rule_id = ?`, ruleID).
		Scan(&current, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return numbering.State{Current: 0}, 0, nil
	}
	if err != nil {
		return numbering.State{}, 0, fmt.Errorf("numbering store: load %s: %w", ruleID, err)
	}
	return numbering.State{Current: int(current.Int64)}, int(version.Int64), nil
}

// Save writes the counter when the version still matches.
//
// The update is conditional on the version in the same statement as the write, so
// a concurrent writer either sees the old version and fails, or the other one
// does: the two cannot both succeed.
func (s *Store) Save(ctx context.Context, ruleID string, expectedVersion int, state numbering.State) error {
	if ruleID == "" {
		return fmt.Errorf("numbering store: rule id is required")
	}
	if state.Current < 0 {
		return fmt.Errorf("numbering store: counter %d is negative", state.Current)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("numbering store: begin %s: %w", ruleID, err)
	}
	defer func() { _ = tx.Rollback() }()

	if expectedVersion == 0 {
		// The first write for a rule inserts it. The insert is refused when a row
		// appeared meanwhile, so a first write also races safely.
		result, err := tx.ExecContext(ctx,
			`INSERT INTO numbering_counter (rule_id, current, version) VALUES (?, ?, 1)`,
			ruleID, state.Current)
		if err != nil {
			return fmt.Errorf("%w: rule %s", numbering.ErrNumberingConflict, ruleID)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("numbering store: insert %s: %w", ruleID, err)
		}
		if affected != 1 {
			return fmt.Errorf("%w: rule %s", numbering.ErrNumberingConflict, ruleID)
		}
		return tx.Commit()
	}

	result, err := tx.ExecContext(ctx,
		`UPDATE numbering_counter SET current = ?, version = version + 1 WHERE rule_id = ? AND version = ?`,
		state.Current, ruleID, expectedVersion)
	if err != nil {
		return fmt.Errorf("numbering store: save %s: %w", ruleID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("numbering store: save %s: %w", ruleID, err)
	}
	if affected != 1 {
		return fmt.Errorf("%w: rule %s moved from version %d", numbering.ErrNumberingConflict, ruleID, expectedVersion)
	}
	return tx.Commit()
}

// retryBusy runs a statement until SQLite stops reporting that the database is
// locked, so opening two processes at once does not fail at random.
func retryBusy(ctx context.Context, db *sql.DB, statement string) error {
	var last error
	for attempt := 0; attempt < 20; attempt++ {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			last = err
			if !isBusy(err) {
				return fmt.Errorf("numbering store: %s: %w", statement, err)
			}
			time.Sleep(time.Duration(50*(attempt+1)) * time.Millisecond)
			continue
		}
		return nil
	}
	return fmt.Errorf("numbering store: %s: %w", statement, last)
}

// isBusy reports whether the error is SQLite's "database is locked" or "busy".
func isBusy(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "database is locked") || strings.Contains(message, "SQLITE_BUSY")
}
