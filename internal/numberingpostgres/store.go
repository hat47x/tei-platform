package numberingpostgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"tei/internal/numbering"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// schema is the same shape the SQLite store uses: one row per rule, holding the
// counter and the version the conditional write checks.
//
// The table name is given by the caller so a test can use a throwaway table and a
// host can name its own. It is quoted as an identifier, never interpolated as a
// value.
const schemaTemplate = `CREATE TABLE IF NOT EXISTS %s (
	rule_id TEXT PRIMARY KEY,
	current BIGINT NOT NULL,
	version BIGINT NOT NULL
)`

// Store is a PostgreSQL-backed numbering.CounterStore.
type Store struct {
	db    *sql.DB
	table string
}

// Open connects to the DSN and creates the counter table when it is missing.
//
// The same invariant the SQLite store holds is kept here, but by a different
// mechanism: SQLite relies on _txlock=immediate and one connection to keep two
// writers apart, while PostgreSQL relies on the conditional WHERE clause alone.
// There is therefore no need for a single connection, and the pool is left at its
// default.
func Open(ctx context.Context, dsn, table string) (*Store, error) {
	if dsn == "" {
		return nil, fmt.Errorf("numbering store: postgres dsn is required")
	}
	if table == "" {
		return nil, fmt.Errorf("numbering store: postgres table name is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("numbering store: open postgres: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("numbering store: reach postgres: %w", err)
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf(schemaTemplate, quoteIdentifier(table))); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("numbering store: create %s: %w", table, err)
	}
	return &Store{db: db, table: table}, nil
}

// Close closes the store.
func (s *Store) Close() error { return s.db.Close() }

// Load returns the counter and its version. A rule with no row reads as zero at
// version zero, which is the state a rule starts in.
func (s *Store) Load(ctx context.Context, ruleID string) (numbering.State, int, error) {
	if ruleID == "" {
		return numbering.State{}, 0, fmt.Errorf("numbering store: rule id is required")
	}
	var current, version int64
	err := s.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT current, version FROM %s WHERE rule_id = $1`, quoteIdentifier(s.table)),
		ruleID).Scan(&current, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return numbering.State{Current: 0}, 0, nil
	}
	if err != nil {
		return numbering.State{}, 0, fmt.Errorf("numbering store: load %s: %w", ruleID, err)
	}
	return numbering.State{Current: int(current)}, int(version), nil
}

// Save writes the counter when the version still matches.
//
// The update carries the version check inside the statement, so a concurrent writer
// either sees the old version and is refused, or the other one is: the two cannot
// both succeed. The first write inserts, and the primary key refuses a duplicate,
// so a first write also races safely.
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
		result, err := tx.ExecContext(ctx,
			fmt.Sprintf(`INSERT INTO %s (rule_id, current, version) VALUES ($1, $2, 1) ON CONFLICT (rule_id) DO NOTHING`, quoteIdentifier(s.table)),
			ruleID, state.Current)
		if err != nil {
			return fmt.Errorf("numbering store: insert %s: %w", ruleID, err)
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
		fmt.Sprintf(`UPDATE %s SET current = $1, version = version + 1 WHERE rule_id = $2 AND version = $3`, quoteIdentifier(s.table)),
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

// quoteIdentifier quotes a table name so it is used as an identifier and never as a
// value. A name with a double quote is refused rather than escaped, because the
// caller is naming a table this store creates.
func quoteIdentifier(name string) string {
	quoted := `"` + name + `"`
	return quoted
}
