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

// TestDesiredSchemaSQLCreatesTheTableTheReadNeeds checks the step the guide
// tells a reader to run: take the Create statements from `tei desired-schema
// -format sql`, apply them to an empty database, put the same detail rows in,
// and read them back through the same `tei view` entry.
//
// The expected answers are the same independently written fixture the
// hand-written table test uses, so this does not compare the program with
// itself. What it establishes is narrower and new: the printed statements are
// enough to create the table the read needs. Before this, the guide's SQL and
// the test's CREATE TABLE were two different statements, and only the second
// was ever executed.
//
// The definitions declare no precision or scale, so the columns are plain
// `numeric`. The answers must not depend on that: the seed holds integers and
// the rules read them as exact numbers either way.
func TestDesiredSchemaSQLCreatesTheTableTheReadNeeds(t *testing.T) {
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

	// Start from an empty schema so nothing from an earlier test can stand in
	// for the statements under test.
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

	binary := buildCLIForLifecycle(t)

	// The statements come from the product entry, not from this test.
	printed := runCLIProcessWithStreams(t, binary,
		"desired-schema", "-workspace", filepath.Join("..", "..", "reference", "workspace"),
		"-database", "sales_core", "-provider", "postgres", "-format", "sql")
	if printed.code != 0 {
		t.Fatalf("desired-schema failed: code=%d stderr=%s", printed.code, printed.stderr)
	}
	statements := splitSQLStatements(t, printed.stdout)
	if len(statements) == 0 {
		t.Fatal("desired-schema printed no statements")
	}
	// The table the read needs must be created here; a set of statements that
	// creates something else would leave this test reading a table it built
	// itself.
	createsOrderLines := false
	for _, statement := range statements {
		if strings.Contains(statement, `"public"."order_lines"`) {
			createsOrderLines = true
			break
		}
	}
	if !createsOrderLines {
		t.Fatalf("the printed statements never mention public.order_lines:\n%s", printed.stdout)
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("a printed statement did not apply: %v\nstatement:\n%s", err, statement)
		}
	}

	// The columns the read maps must exist with the types the statements
	// declared. This is read from the catalog, not from the printed text.
	type column struct {
		name string
		typ  string
		null string
	}
	wantColumns := []column{
		{"order_line_id", "bigint", "NO"},
		{"line_number", "bigint", "NO"},
		{"item_code", "character varying", "NO"},
		{"quantity", "numeric", "NO"},
		{"unit_price", "numeric", "NO"},
		{"order_id", "bigint", "NO"},
	}
	rows, err := db.Query(`select column_name, data_type, is_nullable
		from information_schema.columns
		where table_schema = 'public' and table_name = 'order_lines'
		order by ordinal_position`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var observed []column
	for rows.Next() {
		var c column
		if err := rows.Scan(&c.name, &c.typ, &c.null); err != nil {
			t.Fatal(err)
		}
		observed = append(observed, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(observed) != len(wantColumns) {
		t.Fatalf("public.order_lines has %d columns, expected %d: %+v", len(observed), len(wantColumns), observed)
	}
	for index, want := range wantColumns {
		if observed[index] != want {
			t.Errorf("column %d: got %+v, want %+v", index, observed[index], want)
		}
	}

	// The primary key must be the one the definition declares.
	var keyColumns int
	err = db.QueryRow(`select count(*) from information_schema.table_constraints tc
		join information_schema.key_column_usage kcu
			on kcu.constraint_name = tc.constraint_name and kcu.table_schema = tc.table_schema
		where tc.table_schema = 'public' and tc.table_name = 'order_lines'
			and tc.constraint_type = 'PRIMARY KEY' and kcu.column_name = 'order_line_id'`).Scan(&keyColumns)
	if err != nil {
		t.Fatal(err)
	}
	if keyColumns != 1 {
		t.Fatalf("order_line_id is not the declared primary key (matched %d columns)", keyColumns)
	}

	fixtures := filepath.Join("..", "..", "testdata", "destination-order-change")
	seed, err := os.ReadFile(filepath.Join(fixtures, "seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(seed)); err != nil {
		t.Fatalf("the seeded rows did not fit the generated table: %v", err)
	}

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

	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace did not validate: code=%d stdout=%s stderr=%s",
			validated.code, validated.stdout, validated.stderr)
	}

	for _, orderID := range []string{"1001", "1002", "1003"} {
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
						Key string `json:"key"`
					} `json:"rows"`
					PerLine           map[string]map[string]string `json:"perRow"`
					PerOrder          map[string]string            `json:"perRelation"`
					PerOrderUndecided map[string]string            `json:"perRelationUndecided"`
				} `json:"result"`
			}
			if err := json.Unmarshal([]byte(got.stdout), &report); err != nil {
				t.Fatalf("invalid CLI JSON: %v (%s)", err, got.stdout)
			}
			if !report.OK || report.Result.Coverage != "complete" {
				t.Fatalf("CLI failed to qualify the declared complete read: %+v", report)
			}
			want := expected.Orders[orderID]
			if !reflect.DeepEqual(report.Result.PerLine, want.PerLine) {
				t.Fatalf("order=%s perLine=%v, expected independently %v",
					orderID, report.Result.PerLine, want.PerLine)
			}
			if len(report.Result.Rows) != len(want.PerLine) {
				t.Fatalf("order=%s rows=%d, want=%d", orderID, len(report.Result.Rows), len(want.PerLine))
			}
			seen := map[string]bool{}
			for _, row := range report.Result.Rows {
				if _, expectedRow := want.PerLine[row.Key]; !expectedRow || seen[row.Key] {
					t.Fatalf("order=%s: missing, unexpected or duplicate row key %q", orderID, row.Key)
				}
				seen[row.Key] = true
			}
			// The declared covers range is checked against the actual Query
			// and relation storage. Preserve the independent relation-wide
			// expectations, including empty orders, without undecided rows.
			if !reflect.DeepEqual(report.Result.PerOrder, want.PerOrder) || len(report.Result.PerOrderUndecided) != 0 {
				t.Fatalf("order=%s perRelation=%v undecided=%v, independently expected %v",
					orderID, report.Result.PerOrder, report.Result.PerOrderUndecided, want.PerOrder)
			}
		})
	}
}

// splitSQLStatements splits the printed SQL into the statements psql would
// receive. The renderer already terminates each statement with one semicolon.
func splitSQLStatements(t *testing.T, sql string) []string {
	t.Helper()
	var statements []string
	for _, statement := range strings.Split(sql, ";") {
		trimmed := strings.TrimSpace(statement)
		if trimmed == "" {
			continue
		}
		statements = append(statements, trimmed)
	}
	return statements
}
