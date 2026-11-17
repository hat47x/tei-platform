package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestDeclaredDecimalPrecisionSurvivesToTheDatabaseAndBack closes the one
// item plan/NOW.md left out of the DDL acceptance: a column whose definition
// states a precision and scale is not only printed as numeric(p,s), but is
// created with that width in PostgreSQL and read back through the same `tei
// view` entry.
//
// Two things are checked, and neither is a comparison of the product with
// itself:
//
//  1. The catalog reports the declared numeric_precision/numeric_scale. The
//     expected width is written here from the definition this test edits
//     (12,3 for quantity, 12,2 for unit_price), not read out of the printed
//     SQL.
//  2. The rows read back from that column answer the declared rules the same
//     way testdata/destination-order-change/expected-before.json says they
//     do. That fixture was written by hand from the meaning of the rules, so
//     a value that the narrower column silently rounded would show up there.
//
// A scale-bearing value is deliberately present: the seed's zero quantity is
// stored as 0.000 in a numeric(12,3) column, and the rule must still call it
// not-satisfied. Reading "0.000" as "no value" or truncating to 0 would both
// change the answer.
func TestDeclaredDecimalPrecisionSurvivesToTheDatabaseAndBack(t *testing.T) {
	const databaseName = "tei_destination_order_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL destination not run")
	}
	if os.Getenv("TEI_VIEW_PG_CONFIRM_ISOLATED") != databaseName {
		t.Fatal("refusing to modify database without explicit disposable database confirmation")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot initialize isolated PostgreSQL database")
	}
	t.Cleanup(func() { _ = db.Close() })
	var actualName string
	if err := db.QueryRow("select current_database()").Scan(&actualName); err != nil || actualName != databaseName {
		t.Fatalf("isolated PostgreSQL database check failed: actual=%q err=%v", actualName, err)
	}

	for _, table := range []string{"public.order_lines", "public.orders"} {
		if _, err := db.Exec("drop table if exists " + table + " cascade"); err != nil {
			t.Fatalf("cannot clear %s: %v", table, err)
		}
	}
	t.Cleanup(func() {
		for _, table := range []string{"public.order_lines", "public.orders"} {
			if _, err := db.Exec("drop table if exists " + table + " cascade"); err != nil {
				t.Errorf("cannot clean up %s: %v", table, err)
			}
		}
	})

	// The definition is where the width is declared; the reference Workspace
	// is accepted Source, so this test edits a copy of it rather than the
	// committed meaning.
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	persistencePath := filepath.Join(root, "databases", "sales-core", "persistence", "order-line.tei")
	rawPersistence, err := os.ReadFile(persistencePath)
	if err != nil {
		t.Fatal(err)
	}
	// The expected widths below are what these two declarations state.
	const (
		quantityColumn    = "column sales.order_line.quantity -> quantity storage decimal required"
		quantityDeclared  = "column sales.order_line.quantity -> quantity storage decimal(12,3) required"
		unitPriceColumn   = "column sales.order_line.unit_price -> unit_price storage decimal required"
		unitPriceDeclared = "column sales.order_line.unit_price -> unit_price storage decimal(12,2) required"
	)
	if strings.Count(string(rawPersistence), quantityColumn) != 1 ||
		strings.Count(string(rawPersistence), unitPriceColumn) != 1 {
		t.Fatalf("the reference persistence mapping changed shape:\n%s", rawPersistence)
	}
	if strings.Contains(string(rawPersistence), "decimal(") {
		t.Fatalf("the reference mapping already declares a width; this test would not be testing the declaration:\n%s", rawPersistence)
	}
	edited := strings.Replace(string(rawPersistence), quantityColumn, quantityDeclared, 1)
	edited = strings.Replace(edited, unitPriceColumn, unitPriceDeclared, 1)
	if err := os.WriteFile(persistencePath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	binary := buildCLIForLifecycle(t)

	// The width must reach the SQL the product prints; a statement that says
	// plain numeric would create a column that cannot hold the declared scale.
	printed := runCLIProcessWithStreams(t, binary,
		"desired-schema", "-workspace", root,
		"-database", "sales_core", "-provider", "postgres", "-format", "sql")
	if printed.code != 0 {
		t.Fatalf("desired-schema failed: code=%d stderr=%s", printed.code, printed.stderr)
	}
	for _, want := range []string{
		`"quantity" pg_catalog.numeric(12,3)`,
		`"unit_price" pg_catalog.numeric(12,2)`,
	} {
		if !strings.Contains(printed.stdout, want) {
			t.Fatalf("the printed DDL does not declare %s:\n%s", want, printed.stdout)
		}
	}
	for _, statement := range splitSQLStatements(t, printed.stdout) {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("a printed statement did not apply: %v\nstatement:\n%s", err, statement)
		}
	}

	// The database is the authority on the width that was actually created.
	// These numbers come from the declarations above, not from the printed
	// text: a renderer that dropped the parenthesis would still pass the
	// string check and fail here.
	wantWidths := map[string]struct {
		precision int
		scale     int
	}{
		"quantity":   {12, 3},
		"unit_price": {12, 2},
	}
	rows, err := db.Query(`select column_name, numeric_precision, numeric_scale
		from information_schema.columns
		where table_schema = 'public' and table_name = 'order_lines'
			and column_name in ('quantity', 'unit_price')`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	seen := map[string]bool{}
	for rows.Next() {
		var name string
		var precision, scale sql.NullInt64
		if err := rows.Scan(&name, &precision, &scale); err != nil {
			t.Fatal(err)
		}
		want, known := wantWidths[name]
		if !known {
			t.Fatalf("unexpected column in the catalog check: %s", name)
		}
		if !precision.Valid || !scale.Valid ||
			int(precision.Int64) != want.precision || int(scale.Int64) != want.scale {
			t.Fatalf("column %s was created with precision=%v scale=%v; the definition declares %d,%d",
				name, precision, scale, want.precision, want.scale)
		}
		seen[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !seen["quantity"] || !seen["unit_price"] {
		t.Fatalf("the catalog did not report both declared columns: %v", seen)
	}

	fixtures := filepath.Join("..", "..", "testdata", "destination-order-change")
	seed, err := os.ReadFile(filepath.Join(fixtures, "seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(seed)); err != nil {
		t.Fatalf("the seeded rows did not fit the precision-declared table: %v", err)
	}

	// The seed's zero quantity is stored as 0.000 here. Its rule answer must
	// still be not-satisfied, and the row that holds it must be the one the
	// independent fixture names.
	rawExpected, err := os.ReadFile(filepath.Join(fixtures, "expected-before.json"))
	if err != nil {
		t.Fatal(err)
	}
	var expected struct {
		Orders map[string]struct {
			PerLine  map[string]map[string]string `json:"perLine"`
			PerOrder map[string]string            `json:"perOrder"`
		} `json:"orders"`
	}
	if err := json.Unmarshal(rawExpected, &expected); err != nil {
		t.Fatal(err)
	}
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace did not validate: code=%d stdout=%s stderr=%s",
			validated.code, validated.stdout, validated.stderr)
	}

	for _, orderID := range []string{"1001", "1002"} {
		t.Run(orderID, func(t *testing.T) {
			got := runCLIProcessWithStreams(t, binary,
				"view", "-workspace", root, "-application", "order_entry",
				"-pattern", "order_entry.order_lines",
				"-environment", "development", "-input", "order_id="+orderID,
				"-format", "json",
			)
			if got.code != 0 {
				t.Fatalf("CLI failed: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
			}
			if strings.Contains(got.stdout, dsn) {
				t.Fatal("the database connection string appeared in the JSON output")
			}
			var report struct {
				OK     bool `json:"ok"`
				Result struct {
					Coverage string `json:"coverage"`
					Rows     []struct {
						Key    string `json:"key"`
						Fields map[string]struct {
							State string `json:"state"`
							Value any    `json:"value"`
						} `json:"fields"`
					} `json:"rows"`
					PerLine map[string]map[string]string `json:"perRow"`
				} `json:"result"`
			}
			if err := json.Unmarshal([]byte(got.stdout), &report); err != nil {
				t.Fatalf("invalid CLI JSON: %v (%s)", err, got.stdout)
			}
			if !report.OK || report.Result.Coverage != "partial" {
				t.Fatalf("CLI claimed complete coverage or failed: %+v", report)
			}
			want := expected.Orders[orderID]
			if !reflect.DeepEqual(report.Result.PerLine, want.PerLine) {
				t.Fatalf("order=%s perLine=%v, expected independently %v",
					orderID, report.Result.PerLine, want.PerLine)
			}
			// A value in the precision-declared column must come back present,
			// with its state, rather than as "not-fetched".
			for _, row := range report.Result.Rows {
				quantity, held := row.Fields["sales.order_line.quantity"]
				if !held {
					t.Fatalf("order=%s row=%s did not report the quantity column at all", orderID, row.Key)
				}
				if quantity.State != "present" && quantity.State != "null" {
					t.Fatalf("order=%s row=%s quantity state=%q; the seeded column holds a value",
						orderID, row.Key, quantity.State)
				}
			}
		})
	}
}
