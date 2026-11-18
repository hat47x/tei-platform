//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/workspace"
	"tei/tools/internal/postgresdb"
)

const (
	readPostgresDSNEnv   = "TEI_WEB_READ_PG_DSN"
	readPostgresNameEnv  = "TEI_WEB_READ_PG_CONFIRM_ISOLATED"
	readPostgresDatabase = "tei_destination_order_e2e"
)

// A read of order 1001 against the real database answers the declared rules
// over the rows PostgreSQL returned, and each answer group is named by the
// position of the read it came from. The seed puts quantity 0 on exactly one
// line, so exactly one row must read not-satisfied, and the answer must sit on
// that row rather than on a leftover position.
func TestPostgresReadRunnerAnswersTheRowsPostgreSQLReturned(t *testing.T) {
	dsn := os.Getenv(readPostgresDSNEnv)
	if dsn == "" {
		t.Skip(readPostgresDSNEnv + " is not set; real PostgreSQL read not run")
	}
	if os.Getenv(readPostgresNameEnv) != readPostgresDatabase {
		t.Fatalf("refusing to modify %q without %s=%s", readPostgresDatabase, readPostgresNameEnv, readPostgresDatabase)
	}
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	// The confirmation environment variable is NOT proof of the actual
	// connected database. Refuse all fixture DDL until PostgreSQL itself
	// reports the exact disposable database name.
	if err := requireIsolatedWebReadDatabase(t.Context(), db, readPostgresDatabase); err != nil {
		t.Fatal(err)
	}
	// A mismatched name must be refused before either DROP or INSERT.
	if err := requireIsolatedWebReadDatabase(t.Context(), db, readPostgresDatabase+"_unexpected"); err == nil {
		t.Fatal("a different database name passed the isolation guard")
	}

	t.Run("the Contract Query transaction is read-only and repeatable-read", func(t *testing.T) {
		// Use a disposable table unrelated to the order fixture. Reading its
		// row count establishes the snapshot; a different connection then
		// commits a second row. The selected Query's transaction must not see
		// that later write and must refuse attempts to write itself.
		const table = "public.tei_web_read_snapshot_probe"
		var preexisting bool
		if err := db.QueryRowContext(t.Context(),
			"SELECT to_regclass($1) IS NOT NULL", table).Scan(&preexisting); err != nil || preexisting {
			t.Fatal("snapshot probe table already exists or could not be checked")
		}
		if _, err := db.ExecContext(t.Context(),
			"CREATE TABLE "+table+" (id bigint PRIMARY KEY)"); err != nil {
			t.Fatal("cannot create isolated snapshot probe")
		}
		t.Cleanup(func() {
			if _, err := db.Exec("DROP TABLE IF EXISTS " + table); err != nil {
				t.Error("cannot clean isolated snapshot probe")
			}
		})
		if _, err := db.ExecContext(t.Context(), "INSERT INTO "+table+" VALUES (1)"); err != nil {
			t.Fatal("cannot seed isolated snapshot probe")
		}

		tx, err := beginWebReadOnlyQueryTx(t.Context(), db)
		if err != nil {
			t.Fatal("cannot begin PostgreSQL web-query transaction")
		}
		defer func() { _ = tx.Rollback() }()
		var isolation, readOnly string
		var initial int
		if err := tx.QueryRowContext(t.Context(),
			"SELECT current_setting('transaction_isolation'), current_setting('transaction_read_only'), (SELECT count(*) FROM "+table+")").
			Scan(&isolation, &readOnly, &initial); err != nil {
			t.Fatal("cannot inspect the actual web-query transaction")
		}
		if isolation != "repeatable read" || readOnly != "on" || initial != 1 {
			t.Fatalf("web-query transaction is not a stable read-only snapshot: isolation=%q readOnly=%q count=%d",
				isolation, readOnly, initial)
		}
		if _, err := db.ExecContext(t.Context(), "INSERT INTO "+table+" VALUES (2)"); err != nil {
			t.Fatal("cannot commit independent snapshot probe write")
		}
		var after, outside int
		if err := tx.QueryRowContext(t.Context(), "SELECT count(*) FROM "+table).Scan(&after); err != nil {
			t.Fatal("cannot reread inside web-query snapshot")
		}
		if err := db.QueryRowContext(t.Context(), "SELECT count(*) FROM "+table).Scan(&outside); err != nil {
			t.Fatal("cannot read from an independent connection")
		}
		if after != 1 || outside != 2 {
			t.Fatalf("different snapshots were conflated: query=%d independent=%d", after, outside)
		}
		if _, err := tx.ExecContext(t.Context(), "INSERT INTO "+table+" VALUES (3)"); err == nil {
			t.Fatal("the web-query transaction accepted a write")
		}
	})
	if err := seedReadFixture(t, db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("drop table if exists public.order_lines"); err != nil {
			t.Errorf("cannot clean up disposable order lines: %v", err)
		}
	})

	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := newPostgresReadRunner(loaded.World, db)
	if err != nil {
		t.Fatal(err)
	}
	read, err := runner.Run(t.Context(), readScenarioInput{From: "2026-09-01T00:00:00Z", RowIndex: 0})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(read.OrderLineRows) == 0 {
		t.Fatal("the database read returned no order lines")
	}
	if len(read.RowAnswers) != len(read.OrderLineRows) {
		t.Fatalf("row answers=%d for %d rows", len(read.RowAnswers), len(read.OrderLineRows))
	}
	for index, row := range read.RowAnswers {
		if row.RowIndex != index {
			t.Fatalf("answer %d names row %d; the position must be the read it came from", index, row.RowIndex)
		}
	}

	// Exactly one seeded line holds quantity 0, so its positive-quantity rule
	// must be the only not-satisfied answer, and it must sit on that line.
	unsatisfied := 0
	for index, row := range read.RowAnswers {
		for _, answer := range row.Answers {
			if answer.Result != "not-satisfied" {
				continue
			}
			unsatisfied++
			if answer.Rule != "sales.order_line.quantity_positive" {
				t.Fatalf("row %d: unexpected rule %q failed", index, answer.Rule)
			}
			quantity, ok := orderLineAttribute(read.OrderLineRows[index], "sales.order_line.quantity")
			if !ok {
				t.Fatalf("row %d has no quantity although it was answered", index)
			}
			// PostgreSQL returns declared numeric(12,3) values as "0.000".
			// Check the numeric meaning, not the representation's scale.
			value, valid := new(big.Rat).SetString(quantity)
			if !valid || value.Sign() != 0 {
				t.Fatalf("row %d is marked not-satisfied but holds nonzero or invalid quantity %q", index, quantity)
			}
		}
	}
	if unsatisfied != 1 {
		t.Fatalf("not-satisfied answers=%d, want exactly the one seeded zero quantity", unsatisfied)
	}

	// Order 1001 has two different rows with the same line_number=2.
	// A witnessed duplicate refutes uniqueness even for a partial read.
	// Unknown coverage prevents proving uniqueness, not disproving it.
	const wantDuplicate = "sales.order.lines.line_number_unique not-satisfied"
	foundDuplicate := false
	for _, line := range read.RelationAnswers {
		if !strings.Contains(line, "sales.order.lines.line_number_unique") {
			continue
		}
		foundDuplicate = true
		if line != wantDuplicate {
			t.Fatalf("witnessed duplicate must refute uniqueness: got %q, want %q", line, wantDuplicate)
		}
	}
	if !foundDuplicate {
		t.Fatal("no relation-wide line_number_unique answer for the database read")
	}

	t.Run("the executed query and PostgreSQL visibility qualify this relation", func(t *testing.T) {
		// The Workspace declares covers, but this positive answer is
		// permitted only after the exact selected Query and physical table
		// were checked inside its own RepeatableRead transaction.
		second, err := runner.Run(t.Context(), readScenarioInput{
			From: "2026-09-01T00:00:00Z", RowIndex: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(second.OrderLineRows) != 2 {
			t.Fatalf("order 1002 returned %d detail rows, want 2", len(second.OrderLineRows))
		}
		const rule = "sales.order.lines.line_number_unique"
		found := false
		for _, answer := range second.RelationAnswers {
			if !strings.HasPrefix(answer, rule+" ") {
				continue
			}
			found = true
			if answer != rule+" satisfied" {
				t.Fatalf("the qualified complete read did not prove uniqueness: %q", answer)
			}
		}
		if !found {
			t.Fatal("the Web read omitted the relation-wide uniqueness answer")
		}
	})

	t.Run("row-level security refuses complete coverage even with covers", func(t *testing.T) {
		// Catalog qualification runs in the same transaction as the
		// read. An RLS-enabled table can hide rows for another role, so
		// the Web must not infer completeness from observed values.
		if _, err := db.ExecContext(t.Context(), "alter table public.order_lines enable row level security"); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if _, err := db.Exec("alter table public.order_lines disable row level security"); err != nil {
				t.Errorf("cannot restore the isolated table's RLS setting: %v", err)
			}
		})
		second, err := runner.Run(t.Context(), readScenarioInput{
			From: "2026-09-01T00:00:00Z", RowIndex: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		const rule = "sales.order.lines.line_number_unique"
		found := false
		for _, answer := range second.RelationAnswers {
			if !strings.HasPrefix(answer, rule+" ") {
				continue
			}
			found = true
			if !strings.Contains(answer, " undecided (") ||
				!strings.Contains(answer, "relation-partially-read") {
				t.Fatalf("an RLS-enabled table was treated as complete: %q", answer)
			}
		}
		if !found {
			t.Fatal("the Web read omitted the relation-wide uniqueness answer")
		}

		// The duplicate that the current role did observe still refutes
		// uniqueness. Unknown coverage prevents a positive proof, not
		// a known counterexample.
		first, err := runner.Run(t.Context(), readScenarioInput{
			From: "2026-09-01T00:00:00Z", RowIndex: 0,
		})
		if err != nil {
			t.Fatal(err)
		}
		found = false
		for _, answer := range first.RelationAnswers {
			if answer == rule+" not-satisfied" {
				found = true
			}
		}
		if !found {
			t.Fatalf("an observed duplicate was lost under RLS: %v", first.RelationAnswers)
		}
	})

}

// requireIsolatedWebReadDatabase checks the ACTUAL PostgreSQL database, not
// merely the caller's confirmation string or the database name parsed from a
// DSN. This must run before the disposable fixture performs any DDL.
func requireIsolatedWebReadDatabase(ctx context.Context, db *sql.DB, expected string) error {
	if expected == "" || db == nil {
		return fmt.Errorf("isolated PostgreSQL database is not configured")
	}
	var actual string
	if err := db.QueryRowContext(ctx, "select current_database()").Scan(&actual); err != nil {
		// Driver errors can include DSN fragments; do not echo them here.
		return fmt.Errorf("cannot verify the isolated PostgreSQL database")
	}
	if actual != expected {
		return fmt.Errorf("refusing to modify a database other than the isolated %s", expected)
	}
	return nil
}

// seedReadFixture creates the tables the accepted persistence mappings declare,
// from the shared destination-order seed, so this test owns its data instead of
// reading what another suite left behind.
//
// The browse Query reads public.orders, so the parent rows are created here too:
// the detail seed alone cannot answer which orders are recent, and a page whose
// selected order is not in the database would join database detail rows to a
// synthetic parent.
func seedReadFixture(t *testing.T, db *sql.DB) error {
	t.Helper()
	seed, err := os.ReadFile(filepath.Join(fixtureRoot(t), "seed.sql"))
	if err != nil {
		return err
	}
	if _, err := db.Exec("drop table if exists public.order_lines"); err != nil {
		return err
	}
	if _, err := db.Exec("drop table if exists public.orders"); err != nil {
		return err
	}
	if _, err := db.Exec(`create table public.orders (
		order_id bigint primary key,
		customer_id bigint not null,
		ordered_at timestamptz not null,
		status varchar not null,
		total_amount numeric not null,
		external_id varchar
	)`); err != nil {
		return err
	}
	// Use the same parent orders as the user-facing first-change guide.
	// The browse Query must not receive private test-only parent fixtures.
	ordersSeed, err := os.ReadFile(filepath.Join(fixtureRoot(t), "seed-orders.sql"))
	if err != nil {
		return err
	}
	if _, err := db.Exec(string(ordersSeed)); err != nil {
		return err
	}
	if _, err := db.Exec(`create table public.order_lines (
		order_line_id bigint primary key,
		order_id bigint not null,
		line_number bigint not null,
		item_code varchar not null,
		quantity numeric(12,3) not null,
		unit_price numeric(12,2) not null
	)`); err != nil {
		return err
	}
	_, err = db.Exec(string(seed))
	return err
}

func fixtureRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve the test source path")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "destination-order-change")
}

// orderLineAttribute reads one attribute out of a projected order-line row.
func orderLineAttribute(row semanticRowProjection, attribute string) (string, bool) {
	for _, field := range row.Attributes {
		if field.Attribute == attribute {
			return fmt.Sprint(field.Value), true
		}
	}
	return "", false
}
