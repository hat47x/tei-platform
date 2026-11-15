package operations

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestOrgAxisAsOfBitemporalRectangleCandidateOptIn compares two independent
// time axes for a single Membership occurrence. It is a physical, disposable
// PostgreSQL probe, NOT an accepted TEI temporal model or Product Query.
func TestOrgAxisAsOfBitemporalRectangleCandidateOptIn(t *testing.T) {
	dsn := os.Getenv("TEI_ORG_AXIS_PG_DSN")
	if dsn == "" {
		t.Skip("bitemporal rectangle candidate NOT RUN: TEI_ORG_AXIS_PG_DSN unset")
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
		t.Fatalf("refusing candidate probe outside isolated DB %q: %v", dbName, err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `CREATE TEMP TABLE org_membership_bitemporal_candidate (
		membership_id text NOT NULL,
		revision_id text PRIMARY KEY,
		valid_from timestamptz NOT NULL,
		valid_to timestamptz,
		known_from timestamptz NOT NULL,
		known_to timestamptz,
		allocation numeric(8,2) NOT NULL,
		CHECK (valid_to IS NULL OR valid_from < valid_to),
		CHECK (known_to IS NULL OR known_from < known_to)
	) ON COMMIT DROP`)
	if err != nil {
		t.Fatal(err)
	}
	const insert = `INSERT INTO org_membership_bitemporal_candidate
		(membership_id, revision_id, valid_from, valid_to, known_from, known_to, allocation)
		VALUES ($1, $2, $3::timestamptz, $4::timestamptz, $5::timestamptz, $6::timestamptz, $7)`
	// A2 is the same Membership corrected at known-time 2027-07-03.
	// A3 is a distinct occurrence whose two valid periods do not overlap,
	// although both versions are known during exactly the same period.
	for _, r := range []struct {
		id, rev, validFrom string
		validTo            any
		knownFrom          string
		knownTo            any
		allocation         string
	}{
		{"A2", "A2-old", "2027-01-01", nil, "2027-04-01", "2027-07-03", "0.60"},
		{"A2", "A2-new", "2027-01-01", nil, "2027-07-03", nil, "0.90"},
		{"A3", "A3-early", "2027-01-01", "2027-06-01", "2027-04-01", nil, "0.40"},
		{"A3", "A3-later", "2027-06-01", nil, "2027-04-01", nil, "0.80"},
	} {
		if _, err := tx.ExecContext(ctx, insert, r.id, r.rev, r.validFrom, r.validTo, r.knownFrom, r.knownTo, r.allocation); err != nil {
			t.Fatalf("insert %s: %v", r.rev, err)
		}
	}
	const asOf = `SELECT revision_id FROM org_membership_bitemporal_candidate
		WHERE membership_id = $1
		AND valid_from <= $2::timestamptz AND (valid_to IS NULL OR $2::timestamptz < valid_to)
		AND known_from <= $3::timestamptz AND (known_to IS NULL OR $3::timestamptz < known_to)
		ORDER BY revision_id`
	check := func(id, validAt, knownAt string, want []string) {
		t.Helper()
		rows, err := tx.QueryContext(ctx, asOf, id, validAt, knownAt)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var actual []string
		for rows.Next() {
			var rev string
			if err := rows.Scan(&rev); err != nil {
				t.Fatal(err)
			}
			actual = append(actual, rev)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(actual, want) {
			t.Fatalf("%s valid=%s known=%s: got %v, want %v", id, validAt, knownAt, actual, want)
		}
	}
	// Overlapping known-time intervals do not imply ambiguity if their valid
	// intervals are disjoint. Known-time-only exclusion would reject A3.
	t.Run("known overlap but disjoint valid periods", func(t *testing.T) {
		check("A3", "2027-05-31", "2027-07-04", []string{"A3-early"})
		check("A3", "2027-06-01", "2027-07-04", []string{"A3-later"})
	})
	t.Run("corrected A2 before and at known boundary", func(t *testing.T) {
		check("A2", "2027-05-15", "2027-07-02", []string{"A2-old"})
		check("A2", "2027-05-15", "2027-07-03", []string{"A2-new"})
	})
	// A third physical revision is permitted by the candidate PK, and its
	// valid and known periods BOTH overlap A2-new at the same query point.
	// This is ambiguous; no arbitrary row can be silently selected.
	if _, err := tx.ExecContext(ctx, insert,
		"A2", "A2-conflicting", "2027-05-01", "2027-08-01",
		"2027-06-01", "2027-08-01", "0.70"); err != nil {
		t.Fatal(err)
	}
	t.Run("both axes overlap yields competing versions", func(t *testing.T) {
		check("A2", "2027-07-01", "2027-07-04", []string{"A2-conflicting", "A2-new"})
	})
	t.Run("either axis outside overlap resolves single version", func(t *testing.T) {
		check("A2", "2027-04-30", "2027-07-04", []string{"A2-new"})
		check("A2", "2027-08-01", "2027-07-04", []string{"A2-new"})
		check("A2", "2027-07-01", "2027-08-01", []string{"A2-new"})
	})
}
