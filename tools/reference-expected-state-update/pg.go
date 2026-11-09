package expectedstateupdate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// PostgreSQL table names used by the real-database verification. They are
// deliberately fixed and created only in a disposable database: the tests
// TRUNCATE them.
const (
	PGRecordTable  = "public.tei_expected_state"
	PGAppliedTable = "public.tei_expected_state_applied"
)

// PGStore is the real-database path for hypothesis H1 (#2564): the update is a
// single compare-and-swap statement, and zero affected rows is a business
// conflict rather than an execution error.
//
// The Owner decision for #2564 item 3 is server-side comparison of the supplied
// expected value: the caller sends the expected value over the protected request
// path, the database compares it inside the CAS statement, and the raw current
// value is never echoed in the result or the observation.
//
// It is a verification reference, not a product persistence layer. It does not
// implement digest comparison (the adopted design does not require it),
// distributed coordination or automatic retry.
type PGStore struct {
	db   *sql.DB
	opts *sql.TxOptions
}

// NewPGStore wraps an existing connection pool. The caller owns the pool.
func NewPGStore(db *sql.DB) *PGStore {
	return &PGStore{db: db}
}

// WithIsolation returns a store that runs each apply in a transaction with the
// given isolation level. It exists so the verification can show why the
// default (READ COMMITTED) single-statement CAS needs no retry, while a
// read-then-write at REPEATABLE READ would.
func (s *PGStore) WithIsolation(level sql.IsolationLevel) *PGStore {
	return &PGStore{db: s.db, opts: &sql.TxOptions{Isolation: level}}
}

// EnsureSchema creates the two tables if they do not exist.
func (s *PGStore) EnsureSchema(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS ` + PGRecordTable + ` (
			key text PRIMARY KEY,
			value text NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS ` + PGAppliedTable + ` (
			idempotency_key text PRIMARY KEY,
			key text NOT NULL,
			before text NOT NULL,
			new_value text NOT NULL
		)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("ensure expected-state schema: %w", err)
		}
	}
	return nil
}

// Reset empties both tables. It is only for the disposable verification
// database named by TEI_EXPECTED_STATE_PG_CONFIRM_ISOLATED.
func (s *PGStore) Reset(ctx context.Context) error {
	for _, table := range []string{PGRecordTable, PGAppliedTable} {
		if _, err := s.db.ExecContext(ctx, `TRUNCATE TABLE `+table); err != nil {
			return fmt.Errorf("reset %s: %w", table, err)
		}
	}
	return nil
}

// Seed inserts one record, replacing an existing value.
func (s *PGStore) Seed(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO `+PGRecordTable+` (key, value) VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, key, value)
	return err
}

// Value reads a record directly. The verification uses it to assert the final
// state; the update path itself never reads before writing.
func (s *PGStore) Value(ctx context.Context, key string) (string, bool, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM `+PGRecordTable+` WHERE key = $1`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

// Apply runs one request against the database.
//
// Ordering matters: shape checks and authorization happen before any write, the
// idempotency key is claimed with a primary-key insert so duplicate sends
// serialize, and the record update is a single CAS statement. A rollback
// releases the claim when the compare fails.
func (s *PGStore) Apply(ctx context.Context, req UpdateRequest) (Result, error) {
	mode, invalid, ok := validateRequest(req)
	if !ok {
		return invalid, nil
	}
	if mode == CompareDigest {
		// #2564 item 3 adopted server-side comparison of the supplied expected
		// value. Digest comparison is therefore not part of the adopted design;
		// refuse instead of silently comparing raw values.
		return Result{}, fmt.Errorf("digest comparison is not adopted for the PostgreSQL store: see #2564")
	}
	result := Result{Schema: Schema, Key: req.Key}
	if !req.Authorized {
		result.Outcome, result.Reason = OutcomeDenied, ReasonPermission
		result.Observation.PermissionRejections = 1
		return result, nil
	}
	if !req.Valid {
		result.Outcome, result.Reason = OutcomeInvalid, ReasonValidation
		result.Observation.ValidationRejections = 1
		return result, nil
	}

	tx, err := s.db.BeginTx(ctx, s.opts)
	if err != nil {
		return Result{}, fmt.Errorf("begin expected-state transaction: %w", err)
	}
	defer tx.Rollback()

	var claimedKey, claimedNew string
	err = tx.QueryRowContext(ctx,
		`INSERT INTO `+PGAppliedTable+` (idempotency_key, key, before, new_value)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (idempotency_key) DO NOTHING
		 RETURNING key, new_value`,
		req.IdempotencyKey, req.Key, req.Expected, req.NewValue).Scan(&claimedKey, &claimedNew)
	if errors.Is(err, sql.ErrNoRows) {
		var existingKey, existingBefore, existingNew string
		if err := tx.QueryRowContext(ctx,
			`SELECT key, before, new_value FROM `+PGAppliedTable+` WHERE idempotency_key = $1`,
			req.IdempotencyKey).Scan(&existingKey, &existingBefore, &existingNew); err != nil {
			return Result{}, fmt.Errorf("read idempotency record: %w", err)
		}
		if existingKey == req.Key && existingNew == req.NewValue && existingBefore == req.Expected {
			result.Outcome = OutcomeIdempotentReplay
			result.Before = existingBefore
			result.NewValue = existingNew
			result.Observation.IdempotentReplays = 1
			return result, nil
		}
		result.Outcome, result.Reason = OutcomeRejected, ReasonIdempotencyKeyReused
		result.Observation.IdempotencyKeyRejections = 1
		return result, nil
	}
	if err != nil {
		return Result{}, fmt.Errorf("claim idempotency key: %w", err)
	}

	res, err := tx.ExecContext(ctx,
		`UPDATE `+PGRecordTable+` SET value = $3 WHERE key = $1 AND value = $2`,
		req.Key, req.Expected, req.NewValue)
	if err != nil {
		return Result{}, fmt.Errorf("compare-and-swap update: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return Result{}, fmt.Errorf("compare-and-swap row count: %w", err)
	}

	result.Observation.Comparisons = 1
	if affected == 1 {
		if err := tx.Commit(); err != nil {
			return Result{}, fmt.Errorf("commit expected-state transaction: %w", err)
		}
		result.Outcome = OutcomeApplied
		result.Before = req.Expected
		result.NewValue = req.NewValue
		result.Observation.Applied = 1
		result.Observation.ComparisonInputBytes = 2 * len(req.Expected)
		return result, nil
	}

	// Zero affected rows: the record is gone or the value changed. The caller
	// gets a business result, not an execution error. The current value is not
	// echoed: the Owner decision for #2564 item 3 is server-side comparison of
	// the supplied expected value, so the raw current value must not reach the
	// result, the observation, or a log.
	var exists int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM `+PGRecordTable+` WHERE key = $1`, req.Key).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		result.Outcome, result.Reason = OutcomeMissing, ReasonMissingRecord
		result.Observation.MissingRecords = 1
		return result, nil
	}
	if err != nil {
		return Result{}, fmt.Errorf("read record presence after conflict: %w", err)
	}
	result.Outcome, result.Reason = OutcomeConflict, ReasonExpectedMismatch
	result.Observation.ConflictRejections = 1
	result.Observation.LostUpdatesPrevented = 1
	result.Observation.ComparisonInputBytes = len(req.Expected)
	return result, nil
}
