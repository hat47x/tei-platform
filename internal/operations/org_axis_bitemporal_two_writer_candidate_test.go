package operations

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestOrgAxisAsOfTwoWriterExclusionCandidateOptIn exercises one physical
// PostgreSQL conflict across two connections. It does not adopt a TEI temporal
// rule, implement a Provider operation, or prove Host receipt/retry semantics.
// The non-TEMP table is needed so both sessions can observe the same rows; the
// isolated database is explicitly guarded and the probe schema is removed.
func TestOrgAxisAsOfTwoWriterExclusionCandidateOptIn(t *testing.T) {
	dsn := os.Getenv("TEI_ORG_AXIS_PG_DSN")
	if dsn == "" {
		t.Skip("two-writer PostgreSQL exclusion candidate NOT RUN: TEI_ORG_AXIS_PG_DSN unset")
	}
	const isolated = "tei_org_axis_e2e"
	if os.Getenv("TEI_ORG_AXIS_PG_CONFIRM_ISOLATED") != isolated {
		t.Fatal("two-writer candidate refuses unconfirmed database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(5)
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	var database string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&database); err != nil || database != isolated {
		t.Fatalf("refusing to modify database %q: %v", database, err)
	}
	const probeSchema = "tei_org_axis_two_writer_probe"
	var existing sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT to_regnamespace($1)::text", probeSchema).Scan(&existing); err != nil {
		t.Fatal(err)
	}
	if existing.Valid {
		t.Fatalf("refusing to replace preexisting probe schema %s", probeSchema)
	}
	if _, err := db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS btree_gist"); err != nil {
		t.Fatalf("two-writer probe needs btree_gist in disposable database: %v", err)
	}
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+probeSchema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		if _, err := db.ExecContext(cleanup, "DROP SCHEMA "+probeSchema+" CASCADE"); err != nil {
			t.Errorf("two-writer isolated probe cleanup failed: %v", err)
		}
	}()
	const table = probeSchema + ".revisions"
	const ddl = "CREATE TABLE " + table + " (" +
		"membership_id text NOT NULL, revision_id text PRIMARY KEY," +
		"valid_from timestamptz NOT NULL, valid_to timestamptz," +
		"known_from timestamptz NOT NULL, known_to timestamptz," +
		"CHECK (valid_to IS NULL OR valid_from < valid_to)," +
		"CHECK (known_to IS NULL OR known_from < known_to)," +
		"CONSTRAINT org_two_writer_axes_excl EXCLUDE USING GIST (" +
		"membership_id WITH =," +
		"tstzrange(valid_from, valid_to, '[)') WITH &&," +
		"tstzrange(known_from, known_to, '[)') WITH &&))"
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		t.Fatalf("create two-writer exclusion candidate: %v", err)
	}
	const insert = "INSERT INTO " + table +
		" (membership_id, revision_id, valid_from, valid_to, known_from, known_to) " +
		" VALUES ($1, $2, $3::timestamptz, $4::timestamptz, $5::timestamptz, $6::timestamptz)"
	writer1, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer writer1.Rollback()
	if _, err := writer1.ExecContext(ctx, insert,
		"A2", "A2-first", "2027-01-01", nil, "2027-04-01", "2027-07-03"); err != nil {
		t.Fatal(err)
	}

	conn2, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close()
	writer2, err := conn2.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer writer2.Rollback()
	var secondPID int
	if err := writer2.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&secondPID); err != nil {
		t.Fatal(err)
	}
	secondResult := make(chan error, 1)
	go func() {
		_, err := writer2.ExecContext(ctx, insert,
			"A2", "A2-competing", "2027-05-01", nil, "2027-06-01", "2027-08-01")
		secondResult <- err
	}()
	// Observe the *second backend* actually waiting on a PostgreSQL lock while
	// writer1 still owns the uncommitted conflicting row. Starting a goroutine
	// alone is not proof that the two SQL statements overlapped.
	blocked := false
	until := time.Now().Add(8 * time.Second)
	for time.Now().Before(until) {
		select {
		case err := <-secondResult:
			t.Fatalf("competing write finished while first was uncommitted: %v", err)
		default:
		}
		var wait sql.NullString
		if err := db.QueryRowContext(ctx,
			"SELECT wait_event_type FROM pg_stat_activity WHERE pid = $1", secondPID).Scan(&wait); err != nil {
			t.Fatalf("observe concurrent writer %d: %v", secondPID, err)
		}
		if wait.Valid && wait.String == "Lock" {
			blocked = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !blocked {
		t.Fatal("second PostgreSQL backend never entered a lock wait before the first committed")
	}
	if err := writer1.Commit(); err != nil {
		t.Fatalf("first writer must commit its valid version: %v", err)
	}
	select {
	case err := <-secondResult:
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23P01" ||
			pgErr.ConstraintName != "org_two_writer_axes_excl" {
			t.Fatalf("second writer must lose with named two-axis exclusion: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("second writer did not finish after first commit: %v", ctx.Err())
	}
	if err := writer2.Rollback(); err != nil {
		t.Fatalf("rollback rejected conflicting second transaction: %v", err)
	}

	t.Run("only the winner remains after both transactions", func(t *testing.T) {
		var count int
		var revision string
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*), MIN(revision_id) FROM "+table+" WHERE membership_id = $1", "A2").
			Scan(&count, &revision); err != nil {
			t.Fatal(err)
		}
		if count != 1 || revision != "A2-first" {
			t.Fatalf("conflict admitted or erased correct version: count=%d revision=%s", count, revision)
		}
	})
	t.Run("a legal adjacent correction and independent membership still commit", func(t *testing.T) {
		if _, err := db.ExecContext(ctx, insert,
			"A2", "A2-next", "2027-01-01", nil, "2027-07-03", nil); err != nil {
			t.Fatalf("adjacent known period must remain valid: %v", err)
		}
		if _, err := db.ExecContext(ctx, insert,
			"A4", "A4-independent", "2027-01-01", nil, "2027-04-01", nil); err != nil {
			t.Fatalf("different Membership occurrence must coexist: %v", err)
		}
		for _, tc := range []struct {
			validAt, knownAt string
			want             string
		}{
			{"2027-05-15", "2027-07-02", "A2-first"},
			{"2027-05-15", "2027-07-03", "A2-next"},
		} {
			var got string
			err := db.QueryRowContext(ctx,
				"SELECT revision_id FROM "+table+
					" WHERE membership_id = 'A2' AND "+
					"tstzrange(valid_from,valid_to,'[)') @> $1::timestamptz AND "+
					"tstzrange(known_from,known_to,'[)') @> $2::timestamptz",
				tc.validAt, tc.knownAt).Scan(&got)
			if err != nil || got != tc.want {
				t.Fatalf("valid=%s known=%s got=%s err=%v expected=%s", tc.validAt, tc.knownAt, got, err, tc.want)
			}
		}
	})
}
