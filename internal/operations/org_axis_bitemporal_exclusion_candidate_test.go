package operations

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// The physical exclusion constraint is a PostgreSQL-specific candidate for
// ONE owner-accepted temporal rule: one revision per membership occurrence
// and (validAt, knownAt). This test does not adopt that rule for TEI Core.
// Both the extension and temporary table are created inside a transaction
// that rolls back, so no persistent schema or data is modified.
func TestOrgAxisAsOfExclusionConstraintCandidateOptIn(t *testing.T) {
	dsn := os.Getenv("TEI_ORG_AXIS_PG_DSN")
	if dsn == "" {
		t.Skip("bitemporal exclusion candidate NOT RUN: TEI_ORG_AXIS_PG_DSN unset")
	}
	const isolated = "tei_org_axis_e2e"
	if os.Getenv("TEI_ORG_AXIS_PG_CONFIRM_ISOLATED") != isolated {
		t.Fatal("isolated database confirmation absent")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var dbName string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&dbName); err != nil || dbName != isolated {
		t.Fatalf("refusing exclusion candidate outside isolated DB %q: %v", dbName, err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	// btree_gist supplies the equality operator for text Membership IDs.
	// Creation within the rollback-only transaction avoids persistent DDL.
	if _, err := tx.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS btree_gist"); err != nil {
		t.Fatalf("PostgreSQL exclusion candidate requires btree_gist: %v", err)
	}
	const ddl = "CREATE TEMP TABLE org_membership_exclusion_candidate (" +
		"membership_id text NOT NULL," +
		"revision_id text PRIMARY KEY," +
		"valid_from timestamptz NOT NULL," +
		"valid_to timestamptz," +
		"known_from timestamptz NOT NULL," +
		"known_to timestamptz," +
		"CHECK (valid_to IS NULL OR valid_from < valid_to)," +
		"CHECK (known_to IS NULL OR known_from < known_to)," +
		"CONSTRAINT org_membership_two_axis_excl EXCLUDE USING GIST (" +
		"membership_id WITH =," +
		"tstzrange(valid_from, valid_to, '[)') WITH &&," +
		"tstzrange(known_from, known_to, '[)') WITH &&)" +
		") ON COMMIT DROP"
	if _, err := tx.ExecContext(ctx, ddl); err != nil {
		t.Fatalf("create isolated candidate exclusion constraint: %v", err)
	}
	const insert = "INSERT INTO org_membership_exclusion_candidate " +
		"(membership_id, revision_id, valid_from, valid_to, known_from, known_to) " +
		"VALUES ($1, $2, $3::timestamptz, $4::timestamptz, $5::timestamptz, $6::timestamptz)"
	add := func(membership, revision, validFrom string, validTo any, knownFrom string, knownTo any) {
		t.Helper()
		if _, err := tx.ExecContext(ctx, insert, membership, revision, validFrom, validTo, knownFrom, knownTo); err != nil {
			t.Fatalf("insert %s should coexist: %v", revision, err)
		}
	}
	// Same valid period, adjacent known periods: legitimate correction.
	add("A2", "A2-old", "2027-01-01", nil, "2027-04-01", "2027-07-03")
	add("A2", "A2-new", "2027-01-01", nil, "2027-07-03", nil)
	// Same known period, adjacent valid periods: legitimate effective change.
	add("A3", "A3-early", "2027-01-01", "2027-06-01", "2027-04-01", nil)
	add("A3", "A3-later", "2027-06-01", nil, "2027-04-01", nil)
	// Different Membership occurrence, same (validAt, knownAt): legitimate.
	add("A4", "A4-independent", "2027-01-01", nil, "2027-07-03", nil)

	const asOf = "SELECT revision_id FROM org_membership_exclusion_candidate " +
		"WHERE membership_id = $1 " +
		"AND tstzrange(valid_from, valid_to, '[)') @> $2::timestamptz " +
		"AND tstzrange(known_from, known_to, '[)') @> $3::timestamptz " +
		"ORDER BY revision_id"
	check := func(t *testing.T, membership, validAt, knownAt string, expected []string) {
		t.Helper()
		rows, err := tx.QueryContext(ctx, asOf, membership, validAt, knownAt)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var got []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			got = append(got, id)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("%s valid=%s known=%s: got %v, expected %v", membership, validAt, knownAt, got, expected)
		}
	}
	t.Run("both legal time-boundary changes survive", func(t *testing.T) {
		check(t, "A2", "2027-05-15", "2027-07-02", []string{"A2-old"})
		check(t, "A2", "2027-05-15", "2027-07-03", []string{"A2-new"})
		check(t, "A3", "2027-05-31", "2027-07-04", []string{"A3-early"})
		check(t, "A3", "2027-06-01", "2027-07-04", []string{"A3-later"})
		check(t, "A4", "2027-05-15", "2027-07-04", []string{"A4-independent"})
	})

	// PostgreSQL aborts the transaction after a constraint violation. Use a
	// savepoint and roll it back to prove both typed rejection and recovery.
	if _, err := tx.ExecContext(ctx, "SAVEPOINT before_two_axis_conflict"); err != nil {
		t.Fatal(err)
	}
	_, err = tx.ExecContext(ctx, insert,
		"A2", "A2-conflicting", "2027-05-01", "2027-08-01",
		"2027-07-02", "2027-07-04")
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23P01" ||
		pgErr.ConstraintName != "org_membership_two_axis_excl" {
		t.Fatalf("two-axis overlap must be rejected by its named exclusion constraint: %v", err)
	}
	if _, err := tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT before_two_axis_conflict"); err != nil {
		t.Fatalf("recover from expected exclusion violation: %v", err)
	}
	if _, err := tx.ExecContext(ctx, "RELEASE SAVEPOINT before_two_axis_conflict"); err != nil {
		t.Fatal(err)
	}
	t.Run("conflict rejected without losing valid revisions", func(t *testing.T) {
		check(t, "A2", "2027-07-01", "2027-07-04", []string{"A2-new"})
		check(t, "A2", "2027-05-15", "2027-07-02", []string{"A2-old"})
		check(t, "A3", "2027-06-01", "2027-07-04", []string{"A3-later"})
	})
	// A new correction must close the previous known interval first. This is
	// a transaction-local ordering probe, not a crash recovery or Host commit.
	add("A5", "A5-old", "2027-01-01", nil, "2027-04-01", nil)
	if _, err := tx.ExecContext(ctx,
		"UPDATE org_membership_exclusion_candidate SET known_to = $1::timestamptz WHERE revision_id = $2",
		"2027-07-03", "A5-old"); err != nil {
		t.Fatalf("close old known interval before correction: %v", err)
	}
	add("A5", "A5-new", "2027-01-01", nil, "2027-07-03", nil)
	t.Run("old known interval closes before a correction is inserted", func(t *testing.T) {
		check(t, "A5", "2027-05-15", "2027-07-02", []string{"A5-old"})
		check(t, "A5", "2027-05-15", "2027-07-03", []string{"A5-new"})
	})
}
